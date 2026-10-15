export namespace budget {
	
	export class Issue {
	    severity: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Issue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.severity = source["severity"];
	        this.message = source["message"];
	    }
	}

}

export namespace galaxy {
	
	export class CategoryState {
	    id: string;
	    label: string;
	    group: string;
	    baryonic: boolean;
	    note: string;
	    fraction: number;
	    mass: number;
	    locked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CategoryState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.group = source["group"];
	        this.baryonic = source["baryonic"];
	        this.note = source["note"];
	        this.fraction = source["fraction"];
	        this.mass = source["mass"];
	        this.locked = source["locked"];
	    }
	}
	export class Visual {
	    style: string;
	    color: string;
	    size: number;
	    opacity: number;
	    markers: number;
	
	    static createFrom(source: any = {}) {
	        return new Visual(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.style = source["style"];
	        this.color = source["color"];
	        this.size = source["size"];
	        this.opacity = source["opacity"];
	        this.markers = source["markers"];
	    }
	}
	export class ComponentState {
	    id: string;
	    name: string;
	    matter: string;
	    structure: string;
	    tags: string[];
	    localTags: string[];
	    evidence: string;
	    sources: string[];
	    notes: string;
	    kind: string;
	    params: Record<string, number>;
	    mass: number;
	    share: number;
	    shareLocked: boolean;
	    fractionOfTotal: number;
	    insideR200c: number;
	    maxRadius: number;
	    visual: Visual;
	    markers: number;
	    fixedCount: number;
	    geomKey: string;
	
	    static createFrom(source: any = {}) {
	        return new ComponentState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.matter = source["matter"];
	        this.structure = source["structure"];
	        this.tags = source["tags"];
	        this.localTags = source["localTags"];
	        this.evidence = source["evidence"];
	        this.sources = source["sources"];
	        this.notes = source["notes"];
	        this.kind = source["kind"];
	        this.params = source["params"];
	        this.mass = source["mass"];
	        this.share = source["share"];
	        this.shareLocked = source["shareLocked"];
	        this.fractionOfTotal = source["fractionOfTotal"];
	        this.insideR200c = source["insideR200c"];
	        this.maxRadius = source["maxRadius"];
	        this.visual = this.convertValues(source["visual"], Visual);
	        this.markers = source["markers"];
	        this.fixedCount = source["fixedCount"];
	        this.geomKey = source["geomKey"];
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
	export class PresetInfo {
	    id: string;
	    name: string;
	    morphology: string;
	    kind: string;
	    version: string;
	    description: string;
	    disclaimer: string;
	
	    static createFrom(source: any = {}) {
	        return new PresetInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.morphology = source["morphology"];
	        this.kind = source["kind"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.disclaimer = source["disclaimer"];
	    }
	}
	export class ProfileRequest {
	    components: string[];
	    rMin: number;
	    rMax: number;
	    n: number;
	
	    static createFrom(source: any = {}) {
	        return new ProfileRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.components = source["components"];
	        this.rMin = source["rMin"];
	        this.rMax = source["rMax"];
	        this.n = source["n"];
	    }
	}
	export class Series {
	    id: string;
	    label: string;
	    y: number[];
	
	    static createFrom(source: any = {}) {
	        return new Series(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.y = source["y"];
	    }
	}
	export class Profiles {
	    r: number[];
	    enclosed: Series[];
	    density: Series[];
	    surface: Series[];
	    vc: Series[];
	    baryonFraction: number[];
	    r200c: number;
	    note: string;
	    provenance: string;
	
	    static createFrom(source: any = {}) {
	        return new Profiles(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.r = source["r"];
	        this.enclosed = this.convertValues(source["enclosed"], Series);
	        this.density = this.convertValues(source["density"], Series);
	        this.surface = this.convertValues(source["surface"], Series);
	        this.vc = this.convertValues(source["vc"], Series);
	        this.baryonFraction = source["baryonFraction"];
	        this.r200c = source["r200c"];
	        this.note = source["note"];
	        this.provenance = source["provenance"];
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
	export class Selection {
	    components: string[];
	    tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new Selection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.components = source["components"];
	        this.tags = source["tags"];
	    }
	}
	export class SelectionPart {
	    component: string;
	    matter: string;
	    fraction: number;
	    mass: number;
	
	    static createFrom(source: any = {}) {
	        return new SelectionPart(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.component = source["component"];
	        this.matter = source["matter"];
	        this.fraction = source["fraction"];
	        this.mass = source["mass"];
	    }
	}
	export class SelectionReport {
	    mass: number;
	    fractionOfTotal: number;
	    byMatter: Record<string, number>;
	    parts: SelectionPart[];
	    exact: boolean;
	    insideR200c: number;
	
	    static createFrom(source: any = {}) {
	        return new SelectionReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mass = source["mass"];
	        this.fractionOfTotal = source["fractionOfTotal"];
	        this.byMatter = source["byMatter"];
	        this.parts = this.convertValues(source["parts"], SelectionPart);
	        this.exact = source["exact"];
	        this.insideR200c = source["insideR200c"];
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
	
	export class Source {
	    id: string;
	    citation: string;
	    url: string;
	    usedFor: string;
	
	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.citation = source["citation"];
	        this.url = source["url"];
	        this.usedFor = source["usedFor"];
	    }
	}
	export class TagOwner {
	    component: string;
	    fraction: number;
	    mass: number;
	
	    static createFrom(source: any = {}) {
	        return new TagOwner(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.component = source["component"];
	        this.fraction = source["fraction"];
	        this.mass = source["mass"];
	    }
	}
	export class TagState {
	    id: string;
	    label: string;
	    family: string;
	    note: string;
	    bit: number;
	    mass: number;
	    exact: boolean;
	    owners: TagOwner[];
	
	    static createFrom(source: any = {}) {
	        return new TagState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.family = source["family"];
	        this.note = source["note"];
	        this.bit = source["bit"];
	        this.mass = source["mass"];
	        this.exact = source["exact"];
	        this.owners = this.convertValues(source["owners"], TagOwner);
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
	export class State {
	    preset: PresetInfo;
	    cosmology: units.Cosmology;
	    haloDefinition: string;
	    totalMass: number;
	    r200c: number;
	    v200c: number;
	    concentration: number;
	    mode: string;
	    categories: CategoryState[];
	    unallocated: number;
	    unallocatedMass: number;
	    issues: budget.Issue[];
	    components: ComponentState[];
	    tags: TagState[];
	    baryonFraction: number;
	    cosmicBaryonFraction: number;
	    canUndo: boolean;
	    canRedo: boolean;
	    seed: number;
	    sources: Source[];
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preset = this.convertValues(source["preset"], PresetInfo);
	        this.cosmology = this.convertValues(source["cosmology"], units.Cosmology);
	        this.haloDefinition = source["haloDefinition"];
	        this.totalMass = source["totalMass"];
	        this.r200c = source["r200c"];
	        this.v200c = source["v200c"];
	        this.concentration = source["concentration"];
	        this.mode = source["mode"];
	        this.categories = this.convertValues(source["categories"], CategoryState);
	        this.unallocated = source["unallocated"];
	        this.unallocatedMass = source["unallocatedMass"];
	        this.issues = this.convertValues(source["issues"], budget.Issue);
	        this.components = this.convertValues(source["components"], ComponentState);
	        this.tags = this.convertValues(source["tags"], TagState);
	        this.baryonFraction = source["baryonFraction"];
	        this.cosmicBaryonFraction = source["cosmicBaryonFraction"];
	        this.canUndo = source["canUndo"];
	        this.canRedo = source["canRedo"];
	        this.seed = source["seed"];
	        this.sources = this.convertValues(source["sources"], Source);
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

export namespace main {
	
	export class OpenResult {
	    path: string;
	    state: galaxy.State;
	    ui: string;
	
	    static createFrom(source: any = {}) {
	        return new OpenResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.state = this.convertValues(source["state"], galaxy.State);
	        this.ui = source["ui"];
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

export namespace units {
	
	export class Cosmology {
	    name: string;
	    h0: number;
	    omegaM: number;
	    omegaB: number;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new Cosmology(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.h0 = source["h0"];
	        this.omegaM = source["omegaM"];
	        this.omegaB = source["omegaB"];
	        this.source = source["source"];
	    }
	}

}

