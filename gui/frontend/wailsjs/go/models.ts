export namespace auth {
	
	export class DeviceCodeResponse {
	    device_code: string;
	    user_code: string;
	    verification_uri: string;
	    verification_uri_complete: string;
	    expires_in: number;
	    interval: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceCodeResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device_code = source["device_code"];
	        this.user_code = source["user_code"];
	        this.verification_uri = source["verification_uri"];
	        this.verification_uri_complete = source["verification_uri_complete"];
	        this.expires_in = source["expires_in"];
	        this.interval = source["interval"];
	        this.error = source["error"];
	    }
	}
	export class DeviceTokenResponse {
	    status?: string;
	    token?: string;
	    target_id?: string;
	    username?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceTokenResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.token = source["token"];
	        this.target_id = source["target_id"];
	        this.username = source["username"];
	        this.error = source["error"];
	    }
	}

}

export namespace main {
	
	export class SettingsData {
	    serverAddress: string;
	    deviceName: string;
	    username?: string;
	    isAuthenticated: boolean;
	    autoStart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SettingsData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverAddress = source["serverAddress"];
	        this.deviceName = source["deviceName"];
	        this.username = source["username"];
	        this.isAuthenticated = source["isAuthenticated"];
	        this.autoStart = source["autoStart"];
	    }
	}

}

