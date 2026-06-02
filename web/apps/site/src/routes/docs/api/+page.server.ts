import { error } from '@sveltejs/kit';
import openapiYaml from '../../../../../../../docs/openapi.yaml?raw';
import { isSaasMode } from '$lib/config/deployment';
import { parseOpenApi } from '$lib/api-docs/loader';

export const prerender = isSaasMode;

export function load() {
	if (!isSaasMode) {
		error(404, 'Not found');
	}

	return {
		spec: parseOpenApi(openapiYaml)
	};
}
