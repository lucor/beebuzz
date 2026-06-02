/** Minimal OpenAPI 3.1 subset used by the API reference renderer. */

export type HttpMethod = 'get' | 'post' | 'put' | 'patch' | 'delete' | 'head' | 'options';

export const HTTP_METHODS: HttpMethod[] = [
	'get',
	'post',
	'put',
	'patch',
	'delete',
	'head',
	'options'
];

export interface ApiSchema {
	type?: string | string[];
	format?: string;
	description?: string;
	enum?: unknown[];
	default?: unknown;
	example?: unknown;
	examples?: unknown[];
	nullable?: boolean;
	readOnly?: boolean;
	writeOnly?: boolean;
	deprecated?: boolean;
	required?: string[];
	properties?: Record<string, ApiSchema>;
	additionalProperties?: boolean | ApiSchema;
	items?: ApiSchema;
	allOf?: ApiSchema[];
	oneOf?: ApiSchema[];
	anyOf?: ApiSchema[];
	minimum?: number;
	maximum?: number;
	minLength?: number;
	maxLength?: number;
	pattern?: string;
	title?: string;
	$refName?: string;
}

export interface ApiParameter {
	name: string;
	in: 'query' | 'header' | 'path' | 'cookie';
	description?: string;
	required?: boolean;
	deprecated?: boolean;
	schema?: ApiSchema;
	example?: unknown;
}

export interface ApiMediaType {
	mediaType: string;
	schema?: ApiSchema;
	example?: unknown;
}

export interface ApiRequestBody {
	description?: string;
	required?: boolean;
	content: ApiMediaType[];
}

export interface ApiResponse {
	status: string;
	description?: string;
	content: ApiMediaType[];
	headers?: Record<string, { description?: string; schema?: ApiSchema }>;
}

/** Per-requirement map: scheme name → required scopes (empty for non-OAuth). */
export type ApiSecurityRequirement = Record<string, string[]>;

export interface ApiSecurityScheme {
	name: string;
	type: string;
	scheme?: string;
	bearerFormat?: string;
	in?: 'header' | 'query' | 'cookie';
	headerName?: string;
	description?: string;
}

export interface ApiOperation {
	method: HttpMethod;
	path: string;
	operationId?: string;
	summary?: string;
	description?: string;
	tags: string[];
	deprecated?: boolean;
	audience?: string[];
	stability?: string;
	parameters: ApiParameter[];
	requestBody?: ApiRequestBody;
	responses: ApiResponse[];
	security: ApiSecurityRequirement[];
}

export interface ApiTag {
	name: string;
	description?: string;
	operations: ApiOperation[];
}

export interface ApiSpec {
	title: string;
	version: string;
	description?: string;
	tags: ApiTag[];
	operationsByTag: Record<string, ApiOperation[]>;
	securitySchemes: Record<string, ApiSecurityScheme>;
}
