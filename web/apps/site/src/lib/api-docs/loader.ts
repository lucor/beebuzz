import yaml from 'js-yaml';
import {
	HTTP_METHODS,
	type ApiMediaType,
	type ApiOperation,
	type ApiParameter,
	type ApiRequestBody,
	type ApiResponse,
	type ApiSchema,
	type ApiSecurityRequirement,
	type ApiSecurityScheme,
	type ApiSpec,
	type ApiTag,
	type HttpMethod
} from './types';

type RawDoc = Record<string, unknown>;

export interface ParseOptions {
	/** Keep only operations whose `x-audience` intersects this set. Defaults to ['public']. */
	audiences?: string[];
}

/** Parses a raw YAML string into a normalized ApiSpec, resolving local $refs eagerly. */
export function parseOpenApi(raw: string, options: ParseOptions = {}): ApiSpec {
	const doc = yaml.load(raw) as RawDoc;
	if (!doc || typeof doc !== 'object') {
		throw new Error('Invalid OpenAPI document');
	}

	const audienceAllow = new Set(options.audiences ?? ['public']);

	const info = (doc.info as RawDoc) ?? {};
	const components = (doc.components as RawDoc) ?? {};
	const resolver = createResolver(components);
	const securitySchemes = normalizeSecuritySchemes(
		components.securitySchemes as RawDoc | undefined
	);
	const globalSecurity = normalizeSecurity(doc.security);

	const tagDefs = Array.isArray(doc.tags)
		? (doc.tags as Array<{ name: string; description?: string }>)
		: [];
	const tagOrder: string[] = tagDefs.map((tag) => tag.name);
	const tagDescriptions = new Map<string, string | undefined>();
	for (const tag of tagDefs) {
		tagDescriptions.set(tag.name, tag.description);
	}

	const operations = collectOperations(doc.paths as RawDoc, resolver, globalSecurity).filter((op) =>
		operationMatchesAudience(op, audienceAllow)
	);

	const operationsByTag: Record<string, ApiOperation[]> = {};
	for (const op of operations) {
		const tag = op.tags[0] ?? 'Other';
		if (!operationsByTag[tag]) {
			operationsByTag[tag] = [];
			if (!tagOrder.includes(tag)) {
				tagOrder.push(tag);
			}
		}
		operationsByTag[tag].push(op);
	}

	const tags: ApiTag[] = tagOrder
		.filter((name) => operationsByTag[name]?.length)
		.map((name) => ({
			name,
			description: tagDescriptions.get(name),
			operations: operationsByTag[name]
		}));

	return {
		title: typeof info.title === 'string' ? info.title : 'API',
		version: typeof info.version === 'string' ? info.version : '',
		description: typeof info.description === 'string' ? info.description : undefined,
		tags,
		operationsByTag,
		securitySchemes
	};
}

function operationMatchesAudience(op: ApiOperation, allow: Set<string>): boolean {
	if (allow.size === 0) return true;
	if (!op.audience || op.audience.length === 0) return false;
	return op.audience.some((aud) => allow.has(aud));
}

function normalizeSecuritySchemes(raw: RawDoc | undefined): Record<string, ApiSecurityScheme> {
	if (!raw) return {};
	const result: Record<string, ApiSecurityScheme> = {};
	for (const [name, entry] of Object.entries(raw)) {
		const obj = (entry as RawDoc) ?? {};
		const scheme: ApiSecurityScheme = {
			name,
			type: typeof obj.type === 'string' ? obj.type : 'http',
			scheme: typeof obj.scheme === 'string' ? obj.scheme : undefined,
			bearerFormat: typeof obj.bearerFormat === 'string' ? obj.bearerFormat : undefined,
			in: obj.in === 'header' || obj.in === 'query' || obj.in === 'cookie' ? obj.in : undefined,
			headerName: typeof obj.name === 'string' ? obj.name : undefined,
			description: typeof obj.description === 'string' ? obj.description : undefined
		};
		result[name] = scheme;
	}
	return result;
}

function normalizeSecurity(raw: unknown): ApiSecurityRequirement[] {
	if (!Array.isArray(raw)) return [];
	const result: ApiSecurityRequirement[] = [];
	for (const entry of raw) {
		if (!entry || typeof entry !== 'object') continue;
		const req: ApiSecurityRequirement = {};
		for (const [scheme, scopes] of Object.entries(entry as RawDoc)) {
			req[scheme] = Array.isArray(scopes)
				? scopes.filter((s): s is string => typeof s === 'string')
				: [];
		}
		result.push(req);
	}
	return result;
}

/** Resolves local $ref pointers against the components section, with cycle protection. */
function createResolver(components: RawDoc) {
	const cache = new Map<string, unknown>();

	function resolveRef(ref: string, seen: Set<string>): unknown {
		if (seen.has(ref)) {
			return { $refName: refName(ref) };
		}
		if (cache.has(ref)) {
			return cache.get(ref);
		}

		const segments = ref.replace(/^#\//, '').split('/');
		let node: unknown = { components, ...components };
		for (const seg of segments) {
			if (node && typeof node === 'object' && seg in (node as RawDoc)) {
				node = (node as RawDoc)[seg];
			} else {
				return undefined;
			}
		}

		const nextSeen = new Set(seen).add(ref);
		const resolved = walk(node, nextSeen);
		if (resolved && typeof resolved === 'object' && !Array.isArray(resolved)) {
			(resolved as RawDoc).$refName = refName(ref);
		}
		cache.set(ref, resolved);
		return resolved;
	}

	function walk(node: unknown, seen: Set<string>): unknown {
		if (Array.isArray(node)) {
			return node.map((entry) => walk(entry, seen));
		}
		if (node && typeof node === 'object') {
			const obj = node as RawDoc;
			if (typeof obj.$ref === 'string') {
				return resolveRef(obj.$ref, seen);
			}
			const result: RawDoc = {};
			for (const [key, value] of Object.entries(obj)) {
				result[key] = walk(value, seen);
			}
			return result;
		}
		return node;
	}

	return {
		resolve<T = unknown>(value: unknown): T {
			return walk(value, new Set()) as T;
		}
	};
}

function refName(ref: string): string {
	const parts = ref.split('/');
	return parts[parts.length - 1];
}

function collectOperations(
	paths: RawDoc | undefined,
	resolver: ReturnType<typeof createResolver>,
	globalSecurity: ApiSecurityRequirement[]
): ApiOperation[] {
	if (!paths) return [];

	const ops: ApiOperation[] = [];
	for (const [path, pathItemRaw] of Object.entries(paths)) {
		const pathItem = resolver.resolve<RawDoc>(pathItemRaw) ?? {};
		const sharedParameters = Array.isArray(pathItem.parameters)
			? (pathItem.parameters as RawDoc[]).map((p) => normalizeParameter(p))
			: [];

		for (const method of HTTP_METHODS) {
			const opRaw = pathItem[method];
			if (!opRaw || typeof opRaw !== 'object') continue;
			ops.push(
				normalizeOperation(
					method,
					path,
					opRaw as RawDoc,
					sharedParameters,
					resolver,
					globalSecurity
				)
			);
		}
	}
	return ops;
}

function normalizeOperation(
	method: HttpMethod,
	path: string,
	op: RawDoc,
	sharedParameters: ApiParameter[],
	resolver: ReturnType<typeof createResolver>,
	globalSecurity: ApiSecurityRequirement[]
): ApiOperation {
	const opParams = Array.isArray(op.parameters)
		? (op.parameters as RawDoc[]).map((p) => normalizeParameter(resolver.resolve<RawDoc>(p)))
		: [];

	// op-level parameters override path-level ones with same (name, in)
	const seen = new Set(opParams.map((p) => `${p.in}:${p.name}`));
	const parameters = [
		...opParams,
		...sharedParameters.filter((p) => !seen.has(`${p.in}:${p.name}`))
	];

	const tags = Array.isArray(op.tags)
		? (op.tags as unknown[]).filter((t): t is string => typeof t === 'string')
		: [];

	const xAudience = op['x-audience'];
	const audience = Array.isArray(xAudience)
		? (xAudience as unknown[]).filter((a): a is string => typeof a === 'string')
		: undefined;

	// OpenAPI: an op-level `security` overrides the global one (including `[]` to disable).
	const security = 'security' in op ? normalizeSecurity(op.security) : globalSecurity;

	return {
		method,
		path,
		operationId: typeof op.operationId === 'string' ? op.operationId : undefined,
		summary: typeof op.summary === 'string' ? op.summary : undefined,
		description: typeof op.description === 'string' ? op.description : undefined,
		tags: tags.length > 0 ? tags : ['Other'],
		deprecated: op.deprecated === true,
		audience,
		stability: typeof op['x-stability'] === 'string' ? op['x-stability'] : undefined,
		parameters,
		requestBody: normalizeRequestBody(resolver.resolve<RawDoc>(op.requestBody)),
		responses: normalizeResponses(resolver.resolve<RawDoc>(op.responses)),
		security
	};
}

function normalizeParameter(p: RawDoc | undefined): ApiParameter {
	const safe = p ?? {};
	const location = safe.in;
	const validIn: ApiParameter['in'] =
		location === 'query' || location === 'header' || location === 'path' || location === 'cookie'
			? location
			: 'query';

	return {
		name: typeof safe.name === 'string' ? safe.name : '',
		in: validIn,
		description: typeof safe.description === 'string' ? safe.description : undefined,
		required: safe.required === true,
		deprecated: safe.deprecated === true,
		schema: safe.schema as ApiSchema | undefined,
		example: safe.example
	};
}

function normalizeRequestBody(rb: RawDoc | undefined): ApiRequestBody | undefined {
	if (!rb) return undefined;
	const content = normalizeContent(rb.content as RawDoc | undefined);
	if (content.length === 0 && !rb.description) return undefined;
	return {
		description: typeof rb.description === 'string' ? rb.description : undefined,
		required: rb.required === true,
		content
	};
}

function normalizeResponses(responses: RawDoc | undefined): ApiResponse[] {
	if (!responses) return [];

	const result: ApiResponse[] = [];
	for (const [status, valueRaw] of Object.entries(responses)) {
		const value = (valueRaw as RawDoc) ?? {};
		const headers: ApiResponse['headers'] = {};
		if (value.headers && typeof value.headers === 'object') {
			for (const [name, h] of Object.entries(value.headers as RawDoc)) {
				const hr = (h as RawDoc) ?? {};
				headers[name] = {
					description: typeof hr.description === 'string' ? hr.description : undefined,
					schema: hr.schema as ApiSchema | undefined
				};
			}
		}
		result.push({
			status,
			description: typeof value.description === 'string' ? value.description : undefined,
			content: normalizeContent(value.content as RawDoc | undefined),
			headers: Object.keys(headers).length > 0 ? headers : undefined
		});
	}

	// stable order: success first, then by numeric status
	result.sort((a, b) => statusSort(a.status) - statusSort(b.status));
	return result;
}

function statusSort(status: string): number {
	const n = parseInt(status, 10);
	if (Number.isFinite(n)) return n;
	if (status === 'default') return 9999;
	return 10000;
}

function normalizeContent(content: RawDoc | undefined): ApiMediaType[] {
	if (!content) return [];
	return Object.entries(content).map(([mediaType, mt]) => {
		const obj = (mt as RawDoc) ?? {};
		return {
			mediaType,
			schema: obj.schema as ApiSchema | undefined,
			example: obj.example
		};
	});
}
