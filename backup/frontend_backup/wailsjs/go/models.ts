export namespace main {
	
	export class DeleteLogsResult {
	    count: number;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteLogsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.path = source["path"];
	    }
	}
	export class Task {
	    id: string;
	    name: string;
	    url: string;
	    method: string;
	    cookie: string;
	    headers: string;
	    data: string;
	    useVirtualIP: boolean;
	    times: number;
	    threads: number;
	    scheduledTime: string;
	    cronExpression: string;
	    delayMin: number;
	    delayMax: number;
	    tags: string[];
	    createdAt: string;
	    updatedAt: string;
	    isRunning: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.method = source["method"];
	        this.cookie = source["cookie"];
	        this.headers = source["headers"];
	        this.data = source["data"];
	        this.useVirtualIP = source["useVirtualIP"];
	        this.times = source["times"];
	        this.threads = source["threads"];
	        this.scheduledTime = source["scheduledTime"];
	        this.cronExpression = source["cronExpression"];
	        this.delayMin = source["delayMin"];
	        this.delayMax = source["delayMax"];
	        this.tags = source["tags"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.isRunning = source["isRunning"];
	    }
	}
	export class TaskLogsResult {
	    logs: Record<string, string[]>;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskLogsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.logs = source["logs"];
	        this.path = source["path"];
	    }
	}
	export class TaskProgress {
	    currentRequest: number;
	    totalRequests: number;
	    // Go type: time
	    startTime: any;
	    elapsedTime: number;
	    delayInfo: number[];
	
	    static createFrom(source: any = {}) {
	        return new TaskProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentRequest = source["currentRequest"];
	        this.totalRequests = source["totalRequests"];
	        this.startTime = this.convertValues(source["startTime"], null);
	        this.elapsedTime = source["elapsedTime"];
	        this.delayInfo = source["delayInfo"];
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

