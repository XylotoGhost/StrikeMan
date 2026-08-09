export namespace main {
	
	export class Server {
	    name: string;
	    host: string;
	    port: number;
	    password?: string;
	    collectionId: string;
	    stickyAdmin?: boolean;
	    sticky?: Record<string, boolean>;
	    warmupSeconds?: number;
	
	    static createFrom(source: any = {}) {
	        return new Server(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.password = source["password"];
	        this.collectionId = source["collectionId"];
	        this.stickyAdmin = source["stickyAdmin"];
	        this.sticky = source["sticky"];
	        this.warmupSeconds = source["warmupSeconds"];
	    }
	}
	export class Config {
	    servers: Server[];
	    default: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers = this.convertValues(source["servers"], Server);
	        this.default = source["default"];
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
	export class WorkshopMap {
	    id: string;
	    title: string;
	    tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new WorkshopMap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.tags = source["tags"];
	    }
	}
	export class MapList {
	    official: string[];
	    wingman: string[];
	    wingmanOnly: string[];
	    workshop: WorkshopMap[];
	
	    static createFrom(source: any = {}) {
	        return new MapList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.official = source["official"];
	        this.wingman = source["wingman"];
	        this.wingmanOnly = source["wingmanOnly"];
	        this.workshop = this.convertValues(source["workshop"], WorkshopMap);
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
	export class Player {
	    userId: string;
	    name: string;
	    ping: string;
	    bot: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Player(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userId = source["userId"];
	        this.name = source["name"];
	        this.ping = source["ping"];
	        this.bot = source["bot"];
	    }
	}
	export class Preset {
	    id: string;
	    name: string;
	    description: string;
	    wingman: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.wingman = source["wingman"];
	    }
	}
	
	export class ServerInfo {
	    version: string;
	    build: number;
	    upToDate: boolean;
	    latest: number;
	    updateNote: string;
	    checkFailed: boolean;
	    uptimeSecs: number;
	    addon: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.build = source["build"];
	        this.upToDate = source["upToDate"];
	        this.latest = source["latest"];
	        this.updateNote = source["updateNote"];
	        this.checkFailed = source["checkFailed"];
	        this.uptimeSecs = source["uptimeSecs"];
	        this.addon = source["addon"];
	    }
	}
	export class Status {
	    connected: boolean;
	    error: string;
	    hostname: string;
	    map: string;
	    humans: number;
	    bots: number;
	    gameType: number;
	    gameMode: number;
	    limitTeams: number;
	    maxRounds: number;
	    version: string;
	    toggles: Record<string, number>;
	    warmupTime: number;
	    warmupOnline: number;
	    players: Player[];
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.error = source["error"];
	        this.hostname = source["hostname"];
	        this.map = source["map"];
	        this.humans = source["humans"];
	        this.bots = source["bots"];
	        this.gameType = source["gameType"];
	        this.gameMode = source["gameMode"];
	        this.limitTeams = source["limitTeams"];
	        this.maxRounds = source["maxRounds"];
	        this.version = source["version"];
	        this.toggles = source["toggles"];
	        this.warmupTime = source["warmupTime"];
	        this.warmupOnline = source["warmupOnline"];
	        this.players = this.convertValues(source["players"], Player);
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
	export class Toggle {
	    id: string;
	    label: string;
	    hint: string;
	    admin: boolean;
	    cvar: string;
	
	    static createFrom(source: any = {}) {
	        return new Toggle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.hint = source["hint"];
	        this.admin = source["admin"];
	        this.cvar = source["cvar"];
	    }
	}

}

