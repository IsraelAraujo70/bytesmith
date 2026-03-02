export namespace backend {
	
	export class AgentInfo {
	    name: string;
	    displayName: string;
	    command: string;
	    description: string;
	    installed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AgentInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.command = source["command"];
	        this.description = source["description"];
	        this.installed = source["installed"];
	    }
	}
	export class AppSettingsInfo {
	    theme: string;
	    defaultAgent: string;
	    defaultCwd: string;
	    autoApprove: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppSettingsInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.defaultAgent = source["defaultAgent"];
	        this.defaultCwd = source["defaultCwd"];
	        this.autoApprove = source["autoApprove"];
	    }
	}
	export class ConnectionInfo {
	    id: string;
	    agentName: string;
	    displayName: string;
	    sessions: string[];
	    integrator: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.agentName = source["agentName"];
	        this.displayName = source["displayName"];
	        this.sessions = source["sessions"];
	        this.integrator = source["integrator"];
	    }
	}
	export class EmbeddedTerminalInfo {
	    id: string;
	    cwd: string;
	    shell: string;
	
	    static createFrom(source: any = {}) {
	        return new EmbeddedTerminalInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.cwd = source["cwd"];
	        this.shell = source["shell"];
	    }
	}
	export class FileEntry {
	    name: string;
	    path: string;
	    isDir: boolean;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new FileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	    }
	}
	export class MessageInfo {
	    id: string;
	    role: string;
	    content: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new MessageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class ResumeHistoricalResult {
	    connectionId: string;
	    sessionId: string;
	    resumed: boolean;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new ResumeHistoricalResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connectionId = source["connectionId"];
	        this.sessionId = source["sessionId"];
	        this.resumed = source["resumed"];
	        this.reason = source["reason"];
	    }
	}
	export class ToolCallDiffSummaryInfo {
	    additions: number;
	    deletions: number;
	    files: number;
	
	    static createFrom(source: any = {}) {
	        return new ToolCallDiffSummaryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.additions = source["additions"];
	        this.deletions = source["deletions"];
	        this.files = source["files"];
	    }
	}
	export class ToolCallPartInfo {
	    type: string;
	    text?: string;
	    path?: string;
	    oldText?: string;
	    newText?: string;
	    terminalId?: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolCallPartInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.text = source["text"];
	        this.path = source["path"];
	        this.oldText = source["oldText"];
	        this.newText = source["newText"];
	        this.terminalId = source["terminalId"];
	    }
	}
	export class ToolCallInfo {
	    id: string;
	    title: string;
	    kind: string;
	    status: string;
	    content: string;
	    parts?: ToolCallPartInfo[];
	    diffSummary?: ToolCallDiffSummaryInfo;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolCallInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.kind = source["kind"];
	        this.status = source["status"];
	        this.content = source["content"];
	        this.parts = this.convertValues(source["parts"], ToolCallPartInfo);
	        this.diffSummary = this.convertValues(source["diffSummary"], ToolCallDiffSummaryInfo);
	        this.timestamp = source["timestamp"];
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
	export class SessionHistoryInfo {
	    id: string;
	    agentName: string;
	    connectionId: string;
	    cwd: string;
	    messages: MessageInfo[];
	    toolCalls: ToolCallInfo[];
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionHistoryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.agentName = source["agentName"];
	        this.connectionId = source["connectionId"];
	        this.cwd = source["cwd"];
	        this.messages = this.convertValues(source["messages"], MessageInfo);
	        this.toolCalls = this.convertValues(source["toolCalls"], ToolCallInfo);
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
	export class SessionListItem {
	    id: string;
	    agentName: string;
	    connectionId: string;
	    cwd: string;
	    messageCount: number;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.agentName = source["agentName"];
	        this.connectionId = source["connectionId"];
	        this.cwd = source["cwd"];
	        this.messageCount = source["messageCount"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class SessionListPage {
	    sessions: SessionListItem[];
	    nextCursor?: string;
	    unsupported?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SessionListPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessions = this.convertValues(source["sessions"], SessionListItem);
	        this.nextCursor = source["nextCursor"];
	        this.unsupported = source["unsupported"];
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
	export class SessionModeInfo {
	    modeId: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionModeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modeId = source["modeId"];
	        this.name = source["name"];
	    }
	}
	export class SessionModelInfo {
	    modelId: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionModelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modelId = source["modelId"];
	        this.name = source["name"];
	    }
	}
	export class SessionModelsInfo {
	    currentModelId: string;
	    models: SessionModelInfo[];
	
	    static createFrom(source: any = {}) {
	        return new SessionModelsInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentModelId = source["currentModelId"];
	        this.models = this.convertValues(source["models"], SessionModelInfo);
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
	export class SessionModesInfo {
	    currentModeId: string;
	    modes: SessionModeInfo[];
	
	    static createFrom(source: any = {}) {
	        return new SessionModesInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentModeId = source["currentModeId"];
	        this.modes = this.convertValues(source["modes"], SessionModeInfo);
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

