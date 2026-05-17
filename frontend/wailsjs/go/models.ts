export namespace db {
	
	export class SearchResult {
	    path: string;
	    filename: string;
	    tags: string[];
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.filename = source["filename"];
	        this.tags = source["tags"];
	        this.summary = source["summary"];
	    }
	}

}

export namespace llm {
	
	export class AISearchResult {
	    path: string;
	    filename: string;
	    summary: string;
	    aiReason: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new AISearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.filename = source["filename"];
	        this.summary = source["summary"];
	        this.aiReason = source["aiReason"];
	        this.score = source["score"];
	    }
	}

}

