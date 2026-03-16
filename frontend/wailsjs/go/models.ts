export namespace main {
	
	export class AppInfo {
	    name: string;
	    version: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.status = source["status"];
	    }
	}
	export class BusinessUnit {
	    id: string;
	    clientId: string;
	    name: string;
	    rfc: string;
	    address: string;
	    notes: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new BusinessUnit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.clientId = source["clientId"];
	        this.name = source["name"];
	        this.rfc = source["rfc"];
	        this.address = source["address"];
	        this.notes = source["notes"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class CalendarEvent {
	    id: string;
	    title: string;
	    startTime: string;
	    endTime: string;
	    location: string;
	    description: string;
	    color: string;
	    allDay: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CalendarEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.startTime = source["startTime"];
	        this.endTime = source["endTime"];
	        this.location = source["location"];
	        this.description = source["description"];
	        this.color = source["color"];
	        this.allDay = source["allDay"];
	    }
	}
	export class CalendarResult {
	    events: CalendarEvent[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new CalendarResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.events = this.convertValues(source["events"], CalendarEvent);
	        this.error = source["error"];
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
	export class ChatMessage {
	    role: string;
	    content: any;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	    }
	}
	export class ChatSessionInfo {
	    key: string;
	    name: string;
	    icon: string;
	    systemPrompt: string;
	    lastMessage: string;
	    lastTime: string;
	    unread: number;
	
	    static createFrom(source: any = {}) {
	        return new ChatSessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.systemPrompt = source["systemPrompt"];
	        this.lastMessage = source["lastMessage"];
	        this.lastTime = source["lastTime"];
	        this.unread = source["unread"];
	    }
	}
	export class Client {
	    id: string;
	    name: string;
	    contactName: string;
	    contactEmail: string;
	    phone: string;
	    notes: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Client(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.contactName = source["contactName"];
	        this.contactEmail = source["contactEmail"];
	        this.phone = source["phone"];
	        this.notes = source["notes"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class CreateEventResult {
	    success: boolean;
	    id?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateEventResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.id = source["id"];
	        this.error = source["error"];
	    }
	}
	export class DeleteEventResult {
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteEventResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class EmailDetail {
	    id: string;
	    from: string;
	    to: string;
	    subject: string;
	    body: string;
	    date: string;
	    isRead: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EmailDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.from = source["from"];
	        this.to = source["to"];
	        this.subject = source["subject"];
	        this.body = source["body"];
	        this.date = source["date"];
	        this.isRead = source["isRead"];
	    }
	}
	export class EmailDetailResult {
	    email: EmailDetail;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new EmailDetailResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.email = this.convertValues(source["email"], EmailDetail);
	        this.error = source["error"];
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
	export class EmailMessage {
	    id: string;
	    from: string;
	    subject: string;
	    snippet: string;
	    date: string;
	    isRead: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EmailMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.from = source["from"];
	        this.subject = source["subject"];
	        this.snippet = source["snippet"];
	        this.date = source["date"];
	        this.isRead = source["isRead"];
	    }
	}
	export class EmailResult {
	    emails: EmailMessage[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new EmailResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.emails = this.convertValues(source["emails"], EmailMessage);
	        this.error = source["error"];
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
	export class GoogleAuthStatus {
	    authenticated: boolean;
	    email: string;
	    expiresAt: string;
	
	    static createFrom(source: any = {}) {
	        return new GoogleAuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authenticated = source["authenticated"];
	        this.email = source["email"];
	        this.expiresAt = source["expiresAt"];
	    }
	}
	export class GoogleUserInfo {
	    name: string;
	    email: string;
	    picture: string;
	
	    static createFrom(source: any = {}) {
	        return new GoogleUserInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.email = source["email"];
	        this.picture = source["picture"];
	    }
	}
	export class HandshakeStatus {
	    connected: boolean;
	    lastHandshake: string;
	
	    static createFrom(source: any = {}) {
	        return new HandshakeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.lastHandshake = source["lastHandshake"];
	    }
	}
	export class ServiceStatus {
	    name: string;
	    status: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}
	export class HealthReport {
	    overall: string;
	    services: ServiceStatus[];
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.overall = source["overall"];
	        this.services = this.convertValues(source["services"], ServiceStatus);
	        this.timestamp = source["timestamp"];
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
	export class Idea {
	    id: string;
	    title: string;
	    status: string;
	    priority: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Idea(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class Project {
	    id: string;
	    clientId: string;
	    businessUnitId: string;
	    name: string;
	    description: string;
	    status: string;
	    stack: string;
	    repoUrl: string;
	    priority: string;
	    startDate: string;
	    dueDate: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.clientId = source["clientId"];
	        this.businessUnitId = source["businessUnitId"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.stack = source["stack"];
	        this.repoUrl = source["repoUrl"];
	        this.priority = source["priority"];
	        this.startDate = source["startDate"];
	        this.dueDate = source["dueDate"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ProjectStats {
	    todo: number;
	    inProgress: number;
	    inReview: number;
	    done: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new ProjectStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.todo = source["todo"];
	        this.inProgress = source["inProgress"];
	        this.inReview = source["inReview"];
	        this.done = source["done"];
	        this.total = source["total"];
	    }
	}
	export class SendEmailResult {
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SendEmailResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	
	export class Task {
	    id: string;
	    projectId: string;
	    title: string;
	    description: string;
	    status: string;
	    priority: string;
	    assignedTo: string;
	    estimatedHours: number;
	    tags: string[];
	    createdAt: string;
	    updatedAt: string;
	    completedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	        this.assignedTo = source["assignedTo"];
	        this.estimatedHours = source["estimatedHours"];
	        this.tags = source["tags"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.completedAt = source["completedAt"];
	    }
	}

}

