// Structured API error with backend code and UI message mapping.

/** Maps backend error codes to user-facing messages. */
const ERROR_MESSAGES: Record<string, string> = {
	// Auth
	invalid_session: 'Your session has expired. Please log in again.',
	invalid_otp: 'Invalid or expired code. Please try again.',
	too_many_requests_otp: 'Too many attempts. Please request a new code.',

	// API keys & tokens
	invalid_api_key: 'Invalid or unauthorized API key.',
	invalid_token: 'Invalid or expired token.',
	invalid_webhook_token: 'Invalid or revoked webhook token.',

	// Validation
	validation_error: 'Please check your input and try again.',
	invalid_json: 'Invalid request format.',
	missing_param: 'A required parameter is missing.',
	missing_auth: 'Authentication is required.',
	missing_fields: 'Required fields are missing.',

	// Authorization
	forbidden: 'You do not have permission to perform this action.',

	// Resources
	not_found: 'The requested resource was not found.',

	// Payload
	invalid_content_type: 'Unsupported content type.',
	invalid_payload: 'Invalid payload format.',
	payload_too_large: 'The request payload is too large.',
	attachment_too_large: 'The attachment exceeds the maximum allowed size.',
	missing_attachment: 'An attachment is required.',
	invalid_multipart: 'Invalid multipart form data.',

	// Server
	internal_error: 'Something went wrong. Please try again later.',

	// Frontend-only
	NETWORK_ERROR: 'Connection error. Check your internet and try again.',
	UNKNOWN_ERROR: 'An unexpected error occurred.'
};

const DEFAULT_MESSAGE = 'An unexpected error occurred.';

/** Error codes that should be displayed inline near the relevant form field. */
const INLINE_ERROR_CODES = new Set(['invalid_otp', 'too_many_requests_otp', 'validation_error']);

/** Returns a user-facing message for the given error code. */
export const mapErrorCode = (code: string): string => {
	return ERROR_MESSAGES[code] ?? DEFAULT_MESSAGE;
};

/** Returns true if the error should be shown inline near the form field. */
export const isInlineError = (code: string): boolean => {
	return INLINE_ERROR_CODES.has(code);
};

/** Represents an error returned by the backend API or a network failure. */
export class ApiError extends Error {
	readonly code: string;
	readonly status: number;

	constructor(code: string, status: number, message: string) {
		super(message);
		this.code = code;
		this.status = status;
		this.name = 'ApiError';
	}

	/** Returns the user-facing message mapped from the error code. */
	get userMessage(): string {
		return mapErrorCode(this.code);
	}
}
