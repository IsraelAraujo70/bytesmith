export namespace main {
	
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
	
	    static createFrom(source: any = {}) {
	        return new ConnectionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.agentName = source["agentName"];
	        this.displayName = source["displayName"];
	        this.sessions = source["sessions"];
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
	    role: string;
	    content: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new MessageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class ToolCallInfo {
	    id: string;
	    title: string;
	    kind: string;
	    status: string;
	    content: string;
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
	        this.timestamp = source["timestamp"];
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

}

