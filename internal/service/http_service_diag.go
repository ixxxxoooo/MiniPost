// 请求诊断：耗时分解、网络与 TLS 详情、请求体/头部大小估算、发送错误分类。

package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"minipost/internal/model"
)

type requestTrace struct {
	requestStart time.Time
	getConn      time.Time
	gotConn      time.Time
	localAddr    string
	remoteAddr   string
	dnsStart     time.Time
	dnsDone      time.Time
	connectStart time.Time
	connectDone  time.Time
	tlsStart     time.Time
	tlsDone      time.Time
	wroteRequest time.Time
	firstByte    time.Time
}

func msBetween(start time.Time, end time.Time) float64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return float64(end.Sub(start)) / float64(time.Millisecond)
}

func firstNonZero(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func estimateRequestBodyBytes(body model.RequestBody) int64 {
	switch body.Type {
	case "json":
		return int64(len(body.JSON))
	case "raw":
		return int64(len(body.Raw))
	case "form-urlencoded":
		values := url.Values{}
		for _, kv := range body.FormUrlEncoded {
			if kv.Key == "" {
				continue
			}
			values.Add(kv.Key, kv.Value)
		}
		return int64(len(values.Encode()))
	case "form-data":
		total := 0
		for _, field := range body.FormData {
			total += len(field.Key) + len(field.Value)
			if strings.EqualFold(field.Type, "file") {
				filePath := strings.TrimSpace(field.FilePath)
				if filePath == "" {
					filePath = strings.TrimSpace(field.Value)
				}
				if filePath != "" {
					if stat, err := os.Stat(filePath); err == nil {
						total += int(stat.Size())
					}
				}
			}
		}
		return int64(total)
	default:
		return 0
	}
}

func estimateRequestHeaderBytes(req *http.Request, proto string) int64 {
	if proto == "" {
		proto = "HTTP/1.1"
	}

	size := int64(len(fmt.Sprintf("%s %s %s\r\n", req.Method, req.URL.RequestURI(), proto)))
	hostValue := req.Host
	if hostValue == "" {
		hostValue = req.URL.Host
	}
	hostAlreadyPresent := false
	for key, values := range req.Header {
		if strings.EqualFold(key, "host") {
			hostAlreadyPresent = true
		}
		for _, value := range values {
			size += int64(len(key) + 2 + len(value) + 2)
		}
	}

	if hostValue != "" && !hostAlreadyPresent {
		size += int64(len("Host") + 2 + len(hostValue) + 2)
	}
	size += 2 // 头结束空行
	return size
}

func estimateResponseHeaderBytes(resp *http.Response) int64 {
	proto := resp.Proto
	if proto == "" {
		proto = "HTTP/1.1"
	}
	statusLine := resp.Status
	if statusLine == "" {
		statusLine = fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	size := int64(len(fmt.Sprintf("%s %s\r\n", proto, statusLine)))
	for key, values := range resp.Header {
		for _, value := range values {
			size += int64(len(key) + 2 + len(value) + 2)
		}
	}
	size += 2
	return size
}

func buildTimingBreakdown(trace requestTrace, bodyDone time.Time, done time.Time) model.TimingBreakdown {
	prepareEnd := firstNonZero(trace.getConn, trace.dnsStart, trace.connectStart, trace.wroteRequest, trace.firstByte, bodyDone)
	socketEnd := firstNonZero(trace.dnsStart, trace.connectStart, trace.gotConn, trace.wroteRequest, trace.firstByte, bodyDone)
	waitingStart := firstNonZero(trace.wroteRequest, trace.gotConn, trace.connectDone, trace.dnsDone, trace.getConn, trace.requestStart)
	downloadStart := firstNonZero(trace.firstByte, waitingStart)

	return model.TimingBreakdown{
		Prepare:              msBetween(trace.requestStart, prepareEnd),
		SocketInitialization: msBetween(trace.getConn, socketEnd),
		DNSLookup:            msBetween(trace.dnsStart, trace.dnsDone),
		TCPHandshake:         msBetween(trace.connectStart, trace.connectDone),
		SSLHandshake:         msBetween(trace.tlsStart, trace.tlsDone),
		WaitingTTFB:          msBetween(waitingStart, trace.firstByte),
		Download:             msBetween(downloadStart, bodyDone),
		Process:              msBetween(bodyDone, done),
		Total:                msBetween(trace.requestStart, done),
	}
}

func networkValue(network *model.NetworkDetails, field string) string {
	if network == nil {
		return ""
	}
	switch field {
	case "remoteAddress":
		return network.RemoteAddress
	case "tlsProtocol":
		return network.TLSProtocol
	default:
		return ""
	}
}

func buildNetworkDetails(resp *http.Response, trace requestTrace) *model.NetworkDetails {
	details := &model.NetworkDetails{
		HTTPVersion:   fmt.Sprintf("%d.%d", resp.ProtoMajor, resp.ProtoMinor),
		LocalAddress:  trace.localAddr,
		RemoteAddress: trace.remoteAddr,
	}

	if resp.TLS != nil {
		details.TLSProtocol = tlsVersionLabel(resp.TLS.Version)
		details.CipherName = tls.CipherSuiteName(resp.TLS.CipherSuite)
		if details.CipherName == "" && resp.TLS.CipherSuite != 0 {
			details.CipherName = fmt.Sprintf("0x%04x", resp.TLS.CipherSuite)
		}

		if len(resp.TLS.PeerCertificates) > 0 {
			leaf := resp.TLS.PeerCertificates[0]
			if leaf.Subject.CommonName != "" {
				details.CertificateCN = leaf.Subject.CommonName
			} else if len(leaf.DNSNames) > 0 {
				details.CertificateCN = leaf.DNSNames[0]
			}
			details.IssuerCN = leaf.Issuer.CommonName
			details.ValidUntil = leaf.NotAfter.UTC().Format(time.RFC3339)
		}
	}

	if details.HTTPVersion == "" &&
		details.LocalAddress == "" &&
		details.RemoteAddress == "" &&
		details.TLSProtocol == "" &&
		details.CipherName == "" &&
		details.CertificateCN == "" &&
		details.IssuerCN == "" &&
		details.ValidUntil == "" {
		return nil
	}
	return details
}

func tlsVersionLabel(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLSv1.0"
	case tls.VersionTLS11:
		return "TLSv1.1"
	case tls.VersionTLS12:
		return "TLSv1.2"
	case tls.VersionTLS13:
		return "TLSv1.3"
	default:
		return ""
	}
}

func normalizeSocketAddress(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	raw := strings.TrimSpace(addr.String())
	if raw == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(raw)
	if err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(raw, "[]")
}

func classifyRequestSendError(err error, requestURL string) (code string, message string, detail string) {
	lower := strings.ToLower(err.Error())
	endpoint, host := resolveRequestEndpoint(requestURL, true)
	if host == "" {
		endpoint = requestURL
	}
	rawDetail := stripRequestPrefix(err.Error(), requestURL)

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) || strings.Contains(lower, "no such host") {
		target := host
		if target == "" {
			target = requestURL
		}
		return "DNS_LOOKUP_FAILED", "域名解析失败", fmt.Sprintf("getaddrinfo ENOTFOUND %s", target)
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "REQUEST_TIMEOUT", "请求超时", fmt.Sprintf("connect ETIMEDOUT %s", endpoint)
	}
	if strings.Contains(lower, "deadline exceeded") || strings.Contains(lower, "timeout") {
		return "REQUEST_TIMEOUT", "请求超时", fmt.Sprintf("connect ETIMEDOUT %s", endpoint)
	}

	if strings.Contains(lower, "connection refused") || strings.Contains(lower, "econnrefused") {
		return "CONNECTION_REFUSED", "连接被目标服务拒绝", fmt.Sprintf("connect ECONNREFUSED %s", endpoint)
	}

	if strings.Contains(lower, "x509") || strings.Contains(lower, "tls") || strings.Contains(lower, "certificate") {
		return "TLS_HANDSHAKE_FAILED", "TLS/证书校验失败", rawDetail
	}

	if strings.Contains(lower, "eof") {
		// 某些网络环境下，无法连通目标时 Go 可能返回 EOF，这里统一转换为更可读的连接拒绝样式。
		return "CONNECTION_REFUSED", "连接被目标服务拒绝", fmt.Sprintf("connect ECONNREFUSED %s", endpoint)
	}

	return "REQUEST_FAILED", "请求发送失败", rawDetail
}

func stripRequestPrefix(rawError string, requestURL string) string {
	prefix := fmt.Sprintf(`Get "%s": `, requestURL)
	if strings.HasPrefix(rawError, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(rawError, prefix))
	}
	return rawError
}

func resolveRequestEndpoint(requestURL string, preferIP bool) (endpoint string, host string) {
	u, err := url.Parse(requestURL)
	if err != nil || u.Hostname() == "" {
		return requestURL, ""
	}

	host = u.Hostname()
	port := u.Port()
	if port == "" {
		if strings.EqualFold(u.Scheme, "https") {
			port = "443"
		} else {
			port = "80"
		}
	}

	if preferIP {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		addrs, lookupErr := net.DefaultResolver.LookupIPAddr(ctx, host)
		if lookupErr == nil && len(addrs) > 0 {
			return fmt.Sprintf("%s:%s", addrs[0].IP.String(), port), host
		}
	}

	return fmt.Sprintf("%s:%s", host, port), host
}

func detectTLSWarning(requestURL string, tlsState *tls.ConnectionState) string {
	if tlsState == nil || len(tlsState.PeerCertificates) == 0 {
		return ""
	}

	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return ""
	}

	leaf := tlsState.PeerCertificates[0]
	verifyOpts := x509.VerifyOptions{
		DNSName:       parsedURL.Hostname(),
		Intermediates: x509.NewCertPool(),
	}
	for _, cert := range tlsState.PeerCertificates[1:] {
		verifyOpts.Intermediates.AddCert(cert)
	}
	if roots, rootsErr := x509.SystemCertPool(); rootsErr == nil && roots != nil {
		verifyOpts.Roots = roots
	}

	if _, err := leaf.Verify(verifyOpts); err != nil {
		lower := strings.ToLower(err.Error())
		switch {
		case strings.Contains(lower, "expired"):
			return "Certificate has expired"
		case strings.Contains(lower, "not yet valid"):
			return "Certificate is not yet valid"
		case strings.Contains(lower, "unknown authority"):
			return "Certificate is not trusted"
		default:
			return "TLS certificate verification warning"
		}
	}

	return ""
}
