export namespace model {
	
	export class APIKeyAuth {
	    key: string;
	    value: string;
	    addTo: string;
	
	    static createFrom(source: any = {}) {
	        return new APIKeyAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.addTo = source["addTo"];
	    }
	}
	export class BearerAuth {
	    token: string;
	
	    static createFrom(source: any = {}) {
	        return new BearerAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.token = source["token"];
	    }
	}
	export class BasicAuth {
	    username: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new BasicAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	    }
	}
	export class AuthConfig {
	    type: string;
	    basic: BasicAuth;
	    bearer: BearerAuth;
	    apiKey: APIKeyAuth;
	
	    static createFrom(source: any = {}) {
	        return new AuthConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.basic = this.convertValues(source["basic"], BasicAuth);
	        this.bearer = this.convertValues(source["bearer"], BearerAuth);
	        this.apiKey = this.convertValues(source["apiKey"], APIKeyAuth);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class CollectionNode {
	    nodeId: string;
	    nodeType: string;
	    projectId: string;
	    parentFolderId?: string;
	    sortOrder: number;
	
	    static createFrom(source: any = {}) {
	        return new CollectionNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodeId = source["nodeId"];
	        this.nodeType = source["nodeType"];
	        this.projectId = source["projectId"];
	        this.parentFolderId = source["parentFolderId"];
	        this.sortOrder = source["sortOrder"];
	    }
	}
	export class FormData {
	    key: string;
	    value: string;
	    type: string;
	    filePath?: string;
	    fileName?: string;
	
	    static createFrom(source: any = {}) {
	        return new FormData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.type = source["type"];
	        this.filePath = source["filePath"];
	        this.fileName = source["fileName"];
	    }
	}
	export class RequestBody {
	    type: string;
	    raw: string;
	    json: string;
	    formUrlEncoded: KeyValue[];
	    formData: FormData[];
	
	    static createFrom(source: any = {}) {
	        return new RequestBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.raw = source["raw"];
	        this.json = source["json"];
	        this.formUrlEncoded = this.convertValues(source["formUrlEncoded"], KeyValue);
	        this.formData = this.convertValues(source["formData"], FormData);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class KeyValue {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class RequestItem {
	    id: string;
	    name: string;
	    method: string;
	    url: string;
	    params: KeyValue[];
	    headers: KeyValue[];
	    body: RequestBody;
	    auth: AuthConfig;
	    folderId?: string;
	    sortOrder: number;
	    projectId: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new RequestItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.params = this.convertValues(source["params"], KeyValue);
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.body = this.convertValues(source["body"], RequestBody);
	        this.auth = this.convertValues(source["auth"], AuthConfig);
	        this.folderId = source["folderId"];
	        this.sortOrder = source["sortOrder"];
	        this.projectId = source["projectId"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Folder {
	    id: string;
	    name: string;
	    projectId: string;
	    parentId?: string;
	    sortOrder: number;
	
	    static createFrom(source: any = {}) {
	        return new Folder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.projectId = source["projectId"];
	        this.parentId = source["parentId"];
	        this.sortOrder = source["sortOrder"];
	    }
	}
	export class CollectionData {
	    folders: Folder[];
	    requests: RequestItem[];
	    treeNodes: CollectionNode[];
	
	    static createFrom(source: any = {}) {
	        return new CollectionData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folders = this.convertValues(source["folders"], Folder);
	        this.requests = this.convertValues(source["requests"], RequestItem);
	        this.treeNodes = this.convertValues(source["treeNodes"], CollectionNode);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Variable {
	    id: string;
	    key: string;
	    value: string;
	    enabled: boolean;
	    isSecret: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Variable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.key = source["key"];
	        this.value = source["value"];
	        this.enabled = source["enabled"];
	        this.isSecret = source["isSecret"];
	    }
	}
	export class Environment {
	    id: string;
	    name: string;
	    projectId: string;
	    variables: Variable[];
	
	    static createFrom(source: any = {}) {
	        return new Environment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.projectId = source["projectId"];
	        this.variables = this.convertValues(source["variables"], Variable);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class HistoryEntry {
	    id: string;
	    requestId?: string;
	    name: string;
	    method: string;
	    url: string;
	    statusCode: number;
	    duration: number;
	    size: number;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new HistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.requestId = source["requestId"];
	        this.name = source["name"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.statusCode = source["statusCode"];
	        this.duration = source["duration"];
	        this.size = source["size"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class SizeBreakdown {
	    responseHeaders: number;
	    responseBody: number;
	    responseTotal: number;
	    requestHeaders: number;
	    requestBody: number;
	    requestTotal: number;
	
	    static createFrom(source: any = {}) {
	        return new SizeBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.responseHeaders = source["responseHeaders"];
	        this.responseBody = source["responseBody"];
	        this.responseTotal = source["responseTotal"];
	        this.requestHeaders = source["requestHeaders"];
	        this.requestBody = source["requestBody"];
	        this.requestTotal = source["requestTotal"];
	    }
	}
	export class TimingBreakdown {
	    prepare: number;
	    socketInitialization: number;
	    dnsLookup: number;
	    tcpHandshake: number;
	    sslHandshake: number;
	    waitingTTFB: number;
	    download: number;
	    process: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new TimingBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prepare = source["prepare"];
	        this.socketInitialization = source["socketInitialization"];
	        this.dnsLookup = source["dnsLookup"];
	        this.tcpHandshake = source["tcpHandshake"];
	        this.sslHandshake = source["sslHandshake"];
	        this.waitingTTFB = source["waitingTTFB"];
	        this.download = source["download"];
	        this.process = source["process"];
	        this.total = source["total"];
	    }
	}
	export class NetworkDetails {
	    httpVersion?: string;
	    localAddress?: string;
	    remoteAddress?: string;
	    tlsProtocol?: string;
	    cipherName?: string;
	    certificateCN?: string;
	    issuerCN?: string;
	    validUntil?: string;
	
	    static createFrom(source: any = {}) {
	        return new NetworkDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.httpVersion = source["httpVersion"];
	        this.localAddress = source["localAddress"];
	        this.remoteAddress = source["remoteAddress"];
	        this.tlsProtocol = source["tlsProtocol"];
	        this.cipherName = source["cipherName"];
	        this.certificateCN = source["certificateCN"];
	        this.issuerCN = source["issuerCN"];
	        this.validUntil = source["validUntil"];
	    }
	}
	export class HttpResponse {
	    statusCode: number;
	    statusText: string;
	    headers: Record<string, Array<string>>;
	    body: string;
	    bodyBase64?: string;
	    bodyIsBinary?: boolean;
	    duration: number;
	    size: number;
	    contentType: string;
	    protocol?: string;
	    warnings?: string[];
	    network?: NetworkDetails;
	    timings: TimingBreakdown;
	    sizeDetails: SizeBreakdown;
	
	    static createFrom(source: any = {}) {
	        return new HttpResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.statusCode = source["statusCode"];
	        this.statusText = source["statusText"];
	        this.headers = source["headers"];
	        this.body = source["body"];
	        this.bodyBase64 = source["bodyBase64"];
	        this.bodyIsBinary = source["bodyIsBinary"];
	        this.duration = source["duration"];
	        this.size = source["size"];
	        this.contentType = source["contentType"];
	        this.protocol = source["protocol"];
	        this.warnings = source["warnings"];
	        this.network = this.convertValues(source["network"], NetworkDetails);
	        this.timings = this.convertValues(source["timings"], TimingBreakdown);
	        this.sizeDetails = this.convertValues(source["sizeDetails"], SizeBreakdown);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class Project {
	    id: string;
	    name: string;
	    description?: string;
	    themeColor?: string;
	    createdAt: string;
	    updatedAt: string;
	    schemaVersion: number;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.themeColor = source["themeColor"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.schemaVersion = source["schemaVersion"];
	    }
	}
	
	
	export class SendRequestInput {
	    method: string;
	    url: string;
	    params: KeyValue[];
	    headers: KeyValue[];
	    body: RequestBody;
	    auth: AuthConfig;
	
	    static createFrom(source: any = {}) {
	        return new SendRequestInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.url = source["url"];
	        this.params = this.convertValues(source["params"], KeyValue);
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.body = this.convertValues(source["body"], RequestBody);
	        this.auth = this.convertValues(source["auth"], AuthConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	

}

