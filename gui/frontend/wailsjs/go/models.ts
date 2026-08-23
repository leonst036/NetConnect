export namespace main {
	
	export class SettingsData {
	    serverAddress: string;
	    deviceName: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverAddress = source["serverAddress"];
	        this.deviceName = source["deviceName"];
	    }
	}

}

