export namespace model {
	
	export class Decision {
	    env: string;
	    reason: string;
	    priority: number;
	
	    static createFrom(source: any = {}) {
	        return new Decision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.env = source["env"];
	        this.reason = source["reason"];
	        this.priority = source["priority"];
	    }
	}
	export class FileEntry {
	    name: string;
	    path: string;
	    is_dir: boolean;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new FileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.is_dir = source["is_dir"];
	        this.size = source["size"];
	    }
	}
	export class HealthCheck {
	    name: string;
	    status: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	        this.detail = source["detail"];
	    }
	}
	export class RunRequest {
	    tool: string;
	    args: string;
	    env: string;
	
	    static createFrom(source: any = {}) {
	        return new RunRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.args = source["args"];
	        this.env = source["env"];
	    }
	}
	export class RunResult {
	    execution_id: number;
	    env_used: string;
	    work_dir: string;
	
	    static createFrom(source: any = {}) {
	        return new RunResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.execution_id = source["execution_id"];
	        this.env_used = source["env_used"];
	        this.work_dir = source["work_dir"];
	    }
	}
	export class Tool {
	    id: number;
	    name: string;
	    category: string;
	    description: string;
	    default_env: string;
	    is_high_risk: boolean;
	    dependencies?: string[];
	    icon: string;
	    use_count: number;
	
	    static createFrom(source: any = {}) {
	        return new Tool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.description = source["description"];
	        this.default_env = source["default_env"];
	        this.is_high_risk = source["is_high_risk"];
	        this.dependencies = source["dependencies"];
	        this.icon = source["icon"];
	        this.use_count = source["use_count"];
	    }
	}

}

