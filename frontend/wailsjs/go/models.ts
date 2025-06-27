export namespace main {
	
	export class SuccessConditionDetails {
	    type: string;
	    jsonPath: string;
	    operator: string;
	    expectedValue: string;
	    actualValue: string;
	    result: boolean;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new SuccessConditionDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.jsonPath = source["jsonPath"];
	        this.operator = source["operator"];
	        this.expectedValue = source["expectedValue"];
	        this.actualValue = source["actualValue"];
	        this.result = source["result"];
	        this.reason = source["reason"];
	    }
	}
	export class DetailedLogEntry {
	    requestId: string;
	    timestamp: string;
	    url: string;
	    method: string;
	    statusCode: number;
	    responseTime: number;
	    response: string;
	    error: string;
	    success: boolean;
	    successConditionDetails?: SuccessConditionDetails;
	    errorType: string;
	    detailedError: string;
	
	    static createFrom(source: any = {}) {
	        return new DetailedLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestId = source["requestId"];
	        this.timestamp = source["timestamp"];
	        this.url = source["url"];
	        this.method = source["method"];
	        this.statusCode = source["statusCode"];
	        this.responseTime = source["responseTime"];
	        this.response = source["response"];
	        this.error = source["error"];
	        this.success = source["success"];
	        this.successConditionDetails = this.convertValues(source["successConditionDetails"], SuccessConditionDetails);
	        this.errorType = source["errorType"];
	        this.detailedError = source["detailedError"];
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
	export class ExecutionLog {
	    taskLogId: string;
	    detailedLogs: DetailedLogEntry[];
	    summary: string;
	    totalRequests: number;
	    successCount: number;
	    failedCount: number;
	    duration: number;
	
	    static createFrom(source: any = {}) {
	        return new ExecutionLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskLogId = source["taskLogId"];
	        this.detailedLogs = this.convertValues(source["detailedLogs"], DetailedLogEntry);
	        this.summary = source["summary"];
	        this.totalRequests = source["totalRequests"];
	        this.successCount = source["successCount"];
	        this.failedCount = source["failedCount"];
	        this.duration = source["duration"];
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
	export class SuccessCondition {
	    enabled: boolean;
	    jsonPath: string;
	    operator: string;
	    expectedValue: string;
	
	    static createFrom(source: any = {}) {
	        return new SuccessCondition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.jsonPath = source["jsonPath"];
	        this.operator = source["operator"];
	        this.expectedValue = source["expectedValue"];
	    }
	}
	
	export class Task {
	    id: string;
	    name: string;
	    url: string;
	    method: string;
	    headers: Record<string, string>;
	    headersText: string;
	    data: string;
	    times: number;
	    threads: number;
	    delayMin: number;
	    delayMax: number;
	    tags: string[];
	    cronExpr: string;
	    successCondition: SuccessCondition;
	    createdAt: number;
	    updatedAt: number;
	    isRunning: boolean;
	    lastRunTime: string;
	    lastRunStatus: string;
	    lastRunResult: string;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.method = source["method"];
	        this.headers = source["headers"];
	        this.headersText = source["headersText"];
	        this.data = source["data"];
	        this.times = source["times"];
	        this.threads = source["threads"];
	        this.delayMin = source["delayMin"];
	        this.delayMax = source["delayMax"];
	        this.tags = source["tags"];
	        this.cronExpr = source["cronExpr"];
	        this.successCondition = this.convertValues(source["successCondition"], SuccessCondition);
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.isRunning = source["isRunning"];
	        this.lastRunTime = source["lastRunTime"];
	        this.lastRunStatus = source["lastRunStatus"];
	        this.lastRunResult = source["lastRunResult"];
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
	export class TaskList {
	    tasks: Record<string, Task>;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new TaskList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tasks = this.convertValues(source["tasks"], Task, true);
	        this.total = source["total"];
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
	export class TaskLogEntry {
	    id: string;
	    timestamp: string;
	    message: string;
	    type: string;
	    status: string;
	    executionLogId: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.timestamp = source["timestamp"];
	        this.message = source["message"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.executionLogId = source["executionLogId"];
	    }
	}
	export class TaskProgress {
	    current: number;
	    total: number;
	    startTime: number;
	    isRunning: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TaskProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.total = source["total"];
	        this.startTime = source["startTime"];
	        this.isRunning = source["isRunning"];
	    }
	}
	export class TaskScheduleInfo {
	    taskId: string;
	    isScheduled: boolean;
	    cronExpr: string;
	    nextRunTime: string;
	    cronDescription: string;
	    status: string;
	    lastRunTime: string;
	    lastRunStatus: string;
	    lastRunResult: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskScheduleInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.isScheduled = source["isScheduled"];
	        this.cronExpr = source["cronExpr"];
	        this.nextRunTime = source["nextRunTime"];
	        this.cronDescription = source["cronDescription"];
	        this.status = source["status"];
	        this.lastRunTime = source["lastRunTime"];
	        this.lastRunStatus = source["lastRunStatus"];
	        this.lastRunResult = source["lastRunResult"];
	    }
	}
	export class TestTaskResult {
	    success: boolean;
	    statusCode: number;
	    statusText: string;
	    responseTime: number;
	    requestHeaders: Record<string, string>;
	    responseHeaders: Record<string, string>;
	    responseBody: string;
	    error: string;
	    requestUrl: string;
	    requestMethod: string;
	    requestBodySize: number;
	    sensitiveHeaders: string[];
	    successConditionDetails?: SuccessConditionDetails;
	
	    static createFrom(source: any = {}) {
	        return new TestTaskResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.statusCode = source["statusCode"];
	        this.statusText = source["statusText"];
	        this.responseTime = source["responseTime"];
	        this.requestHeaders = source["requestHeaders"];
	        this.responseHeaders = source["responseHeaders"];
	        this.responseBody = source["responseBody"];
	        this.error = source["error"];
	        this.requestUrl = source["requestUrl"];
	        this.requestMethod = source["requestMethod"];
	        this.requestBodySize = source["requestBodySize"];
	        this.sensitiveHeaders = source["sensitiveHeaders"];
	        this.successConditionDetails = this.convertValues(source["successConditionDetails"], SuccessConditionDetails);
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

