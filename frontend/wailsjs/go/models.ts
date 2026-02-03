export namespace main {
	
	export class EvalErrorData {
	    line: number;
	    col: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new EvalErrorData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.line = source["line"];
	        this.col = source["col"];
	        this.message = source["message"];
	    }
	}
	export class MeshData {
	    vertices: number[];
	    normals: number[];
	    indices: number[];
	    partName: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new MeshData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vertices = source["vertices"];
	        this.normals = source["normals"];
	        this.indices = source["indices"];
	        this.partName = source["partName"];
	        this.color = source["color"];
	    }
	}
	export class EvalResult {
	    meshes: MeshData[];
	    errors: EvalErrorData[];
	    warnings: EvalErrorData[];
	
	    static createFrom(source: any = {}) {
	        return new EvalResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.meshes = this.convertValues(source["meshes"], MeshData);
	        this.errors = this.convertValues(source["errors"], EvalErrorData);
	        this.warnings = this.convertValues(source["warnings"], EvalErrorData);
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
	export class FileResult {
	    content: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new FileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.path = source["path"];
	    }
	}

}

