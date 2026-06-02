package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

var (
	currentLevel = DEBUG
	fileLogger   *log.Logger
	logFile      *os.File
	logPath      string
)

// Init 初始化日志系统，在 ~/.minipost/logs/ 下创建日志文件
func Init() error {
	currentLevel = levelFromEnv(os.Getenv("MINIPOST_LOG_LEVEL"), DEBUG)

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户目录失败: %w", err)
	}

	logDir := filepath.Join(home, ".minipost", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	logFileName := fmt.Sprintf("minipost-%s.log", time.Now().Format("2006-01-02"))
	logPath = filepath.Join(logDir, logFileName)

	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	fileLogger = log.New(logFile, "", 0)

	Info("日志系统初始化完成",
		"logPath", logPath,
		"logLevel", levelNames[currentLevel],
		"goVersion", runtime.Version(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
	)

	return nil
}

// Close 关闭日志文件
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

// SetLevel 设置日志级别
func SetLevel(level Level) {
	currentLevel = level
}

// CurrentLogPath 返回当前日志文件路径，日志系统未初始化时返回空字符串。
func CurrentLogPath() string {
	return logPath
}

func levelFromEnv(value string, fallback Level) Level {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "":
		return fallback
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return fallback
	}
}

func formatMessage(level Level, msg string, kvPairs []interface{}) string {
	var b strings.Builder
	b.WriteString(time.Now().Format("2006-01-02 15:04:05.000"))
	b.WriteString(" [")
	b.WriteString(levelNames[level])
	b.WriteString("] ")

	// 跳过 formatMessage -> logOutput -> Debug/Info/Warn/Error/Fatal
	_, file, line, ok := runtime.Caller(3)
	if ok {
		shortFile := filepath.Base(file)
		b.WriteString(fmt.Sprintf("%s:%d ", shortFile, line))
	}

	b.WriteString(msg)

	// 追加 key=value 对
	for i := 0; i+1 < len(kvPairs); i += 2 {
		b.WriteString(fmt.Sprintf(" %v=%v", kvPairs[i], kvPairs[i+1]))
	}

	return b.String()
}

func logOutput(level Level, msg string, kvPairs ...interface{}) {
	if level < currentLevel {
		return
	}

	formatted := formatMessage(level, msg, kvPairs)

	// 同时输出到 stderr 和日志文件
	fmt.Fprintln(os.Stderr, formatted)
	if fileLogger != nil {
		fileLogger.Println(formatted)
	}
}

func Debug(msg string, kvPairs ...interface{}) {
	logOutput(DEBUG, msg, kvPairs...)
}

func Info(msg string, kvPairs ...interface{}) {
	logOutput(INFO, msg, kvPairs...)
}

func Warn(msg string, kvPairs ...interface{}) {
	logOutput(WARN, msg, kvPairs...)
}

func Error(msg string, kvPairs ...interface{}) {
	logOutput(ERROR, msg, kvPairs...)
}

// Fatal 输出致命错误日志并退出
func Fatal(msg string, kvPairs ...interface{}) {
	logOutput(FATAL, msg, kvPairs...)
	Close()
	os.Exit(1)
}
