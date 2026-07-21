export namespace config {
	
	export class ProjectGroup {
	    id: string;
	    name: string;
	    collapsed: boolean;
	    members: string[];
	
	    static createFrom(source: any = {}) {
	        return new ProjectGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.collapsed = source["collapsed"];
	        this.members = source["members"];
	    }
	}
	export class StatusColours {
	    scheduled: string;
	    inProgress: string;
	    done: string;
	    redundant: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusColours(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scheduled = source["scheduled"];
	        this.inProgress = source["inProgress"];
	        this.done = source["done"];
	        this.redundant = source["redundant"];
	    }
	}
	export class Settings {
	    statusColours: StatusColours;
	    decisionColour: string;
	    endpointColour: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.statusColours = this.convertValues(source["statusColours"], StatusColours);
	        this.decisionColour = source["decisionColour"];
	        this.endpointColour = source["endpointColour"];
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

export namespace layout {
	
	export class Edge {
	    id: string;
	    source: string;
	    target: string;
	    kind: string;
	    taskId: string;
	
	    static createFrom(source: any = {}) {
	        return new Edge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source = source["source"];
	        this.target = source["target"];
	        this.kind = source["kind"];
	        this.taskId = source["taskId"];
	    }
	}

}

export namespace model {
	
	export class Project {
	    id: string;
	    name: string;
	    description: string;
	    colour: string;
	    icon: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.colour = source["colour"];
	        this.icon = source["icon"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class ProximityBond {
	    id: string;
	    endpointAId: string;
	    endpointBId: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new ProximityBond(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.endpointAId = source["endpointAId"];
	        this.endpointBId = source["endpointBId"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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

export namespace service {
	
	export class NodeView {
	    id: string;
	    kind: string;
	    title: string;
	    bodyMarkdown: string;
	    icon: string;
	    parentId?: string;
	    childId?: string;
	    decisionType?: string;
	    orderIndex: number;
	    status: string;
	    decisionCount: number;
	    decisionsCollapsed: boolean;
	    x: number;
	    y: number;
	
	    static createFrom(source: any = {}) {
	        return new NodeView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.bodyMarkdown = source["bodyMarkdown"];
	        this.icon = source["icon"];
	        this.parentId = source["parentId"];
	        this.childId = source["childId"];
	        this.decisionType = source["decisionType"];
	        this.orderIndex = source["orderIndex"];
	        this.status = source["status"];
	        this.decisionCount = source["decisionCount"];
	        this.decisionsCollapsed = source["decisionsCollapsed"];
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class ProjectView {
	    project: model.Project;
	    nodes: NodeView[];
	    edges: layout.Edge[];
	    bonds: model.ProximityBond[];
	
	    static createFrom(source: any = {}) {
	        return new ProjectView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.project = this.convertValues(source["project"], model.Project);
	        this.nodes = this.convertValues(source["nodes"], NodeView);
	        this.edges = this.convertValues(source["edges"], layout.Edge);
	        this.bonds = this.convertValues(source["bonds"], model.ProximityBond);
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
	export class SidebarState {
	    projects: model.Project[];
	    groups: config.ProjectGroup[];
	
	    static createFrom(source: any = {}) {
	        return new SidebarState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projects = this.convertValues(source["projects"], model.Project);
	        this.groups = this.convertValues(source["groups"], config.ProjectGroup);
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

