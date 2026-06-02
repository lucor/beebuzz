import type { ApiSchema } from './types';

/** Builds a deterministic JSON example value from an ApiSchema. */
export function buildExample(schema: ApiSchema | undefined, depth = 0): unknown {
	if (!schema || depth > 6) return null;

	if (schema.example !== undefined) return schema.example;
	if (Array.isArray(schema.examples) && schema.examples.length > 0) return schema.examples[0];
	if (schema.default !== undefined) return schema.default;
	if (Array.isArray(schema.enum) && schema.enum.length > 0) return schema.enum[0];

	if (schema.allOf?.length) {
		const merged: Record<string, unknown> = {};
		for (const part of schema.allOf) {
			const example = buildExample(part, depth + 1);
			if (example && typeof example === 'object' && !Array.isArray(example)) {
				Object.assign(merged, example);
			}
		}
		return merged;
	}

	if (schema.oneOf?.length) return buildExample(schema.oneOf[0], depth + 1);
	if (schema.anyOf?.length) return buildExample(schema.anyOf[0], depth + 1);

	const type = pickType(schema.type);
	if (type === 'object' || schema.properties) {
		const result: Record<string, unknown> = {};
		const properties = schema.properties ?? {};
		for (const [name, prop] of Object.entries(properties)) {
			result[name] = buildExample(prop, depth + 1);
		}
		return result;
	}

	if (type === 'array') {
		return [buildExample(schema.items, depth + 1)];
	}

	if (type === 'string') {
		if (schema.format === 'date-time') return '2024-01-15T10:30:00Z';
		if (schema.format === 'date') return '2024-01-15';
		if (schema.format === 'email') return 'user@example.com';
		if (schema.format === 'uri' || schema.format === 'url') return 'https://example.com';
		if (schema.format === 'uuid') return '00000000-0000-0000-0000-000000000000';
		if (schema.format === 'byte') return 'c3RyaW5n';
		return 'string';
	}

	if (type === 'integer') return 0;
	if (type === 'number') return 0;
	if (type === 'boolean') return false;
	if (type === 'null') return null;

	return null;
}

function pickType(t: ApiSchema['type']): string | undefined {
	if (Array.isArray(t)) return t.find((entry) => entry !== 'null');
	return t;
}

/** Renders a schema into a short human-readable type signature, e.g. "string", "array<User>", "object". */
export function typeLabel(schema: ApiSchema | undefined): string {
	if (!schema) return 'any';
	if (schema.$refName) return schema.$refName;

	if (schema.oneOf?.length) return schema.oneOf.map(typeLabel).join(' | ');
	if (schema.anyOf?.length) return schema.anyOf.map(typeLabel).join(' | ');
	if (schema.allOf?.length) return schema.allOf.map(typeLabel).join(' & ');

	const t = pickType(schema.type);
	if (t === 'array') return `array<${typeLabel(schema.items)}>`;
	if (t === 'object' || schema.properties) return 'object';
	if (Array.isArray(schema.enum) && schema.enum.length > 0) {
		return schema.enum.map((value) => JSON.stringify(value)).join(' | ');
	}
	if (schema.format) return `${t ?? 'string'}<${schema.format}>`;
	return t ?? 'any';
}
