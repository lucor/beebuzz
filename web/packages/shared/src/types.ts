export interface Attachment {
	data?: string;
	mime?: string;
	token?: string;
	filename?: string;
}

export type NotificationPriority = 'high' | 'normal';

export interface Notification {
	id: string;
	title: string;
	body: string;
	topicId?: string | null;
	topic: string | null;
	sentAt: Date;
	priority?: NotificationPriority;
	attachment?: Attachment;
}

export interface AuthState {
	userId: string | null;
	topics: string[];
	isAuthenticated: boolean;
	email: string | null;
	isAdmin: boolean;
}

export type PushMessage =
	| {
			type: 'PUSH_RECEIVED';
			id?: string;
			deviceId?: string;
			title: string;
			body: string;
			topicId?: string | null;
			topic: string | null;
			attachment?: Attachment;
			sentAt: string;
			priority?: string;
	  }
	| {
			type: 'SUBSCRIPTION_CHANGED';
	  }
	| {
			type: 'NOTIFICATION_CLICKED';
			notification?: {
				id?: string;
				deviceId?: string;
				title?: string;
				body?: string;
				topicId?: string | null;
				topic?: string | null;
				sentAt?: string;
				priority?: string;
				attachment?: Attachment;
			};
	  };

export type ToastType = 'info' | 'success' | 'error';

export interface ToastMessage {
	message: string;
	type: ToastType;
	id: number;
}

export interface User {
	userId: string;
	email: string;
	isAdmin: boolean;
	createdAt?: string;
	reason?: string;
}
