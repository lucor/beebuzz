// Base API client with centralized fetch logic.
import { goto } from '$app/navigation';
import { ApiError } from '../errors';
import { auth } from '../stores';
import { API_URL, SITE_URL } from '../config';

const API_VERSION = '/v1';
const API_BASE = `${API_URL}${API_VERSION}`;
const STATUS_UNAUTHORIZED = 401;
const STATUS_FORBIDDEN = 403;
const STATUS_NO_CONTENT = 204;

export interface RequestOptions extends RequestInit {
	credentials?: RequestCredentials;
}

/**
 * Wrapper for fetch with centralized API base URL and structured error handling.
 * Throws ApiError for both backend errors and network failures.
 */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const { credentials = 'include', ...fetch_options } = options;

	let response: Response;
	try {
		response = await fetch(`${API_BASE}${path}`, {
			...fetch_options,
			credentials,
			headers: {
				'Content-Type': 'application/json',
				...fetch_options.headers
			}
		});
	} catch {
		throw new ApiError('NETWORK_ERROR', 0, 'Network error');
	}

	if (!response.ok) {
		if (response.status === STATUS_UNAUTHORIZED) {
			auth.clear();
			await goto(new URL('/login', SITE_URL).toString(), { replaceState: true });
		}
		if (response.status === STATUS_FORBIDDEN) {
			await goto(new URL('/account', SITE_URL).toString(), { replaceState: true });
		}

		const body = (await response.json().catch(() => null)) as {
			code?: string;
			message?: string;
		} | null;
		if (body?.code) {
			throw new ApiError(body.code, response.status, body.message ?? '');
		}
		throw new ApiError('UNKNOWN_ERROR', response.status, response.statusText);
	}

	if (response.status === STATUS_NO_CONTENT) {
		return undefined as T;
	}

	return response.json() as Promise<T>;
}

/**
 * Fetch binary data from the API. Uses API_BASE and credentials: include.
 * Use this for non-JSON endpoints (e.g. image/attachment downloads).
 */
export async function fetchBlob(path: string): Promise<Blob> {
	let response: Response;
	try {
		response = await fetch(`${API_BASE}${path}`, { credentials: 'include' });
	} catch {
		throw new ApiError('NETWORK_ERROR', 0, 'Network error');
	}

	if (!response.ok) {
		throw new ApiError('FETCH_ERROR', response.status, response.statusText);
	}

	return response.blob();
}

// API client with typed methods.
export const api = {
	/**
	 * GET request.
	 */
	get: <T>(path: string) => request<T>(path, { method: 'GET' }),

	/**
	 * POST request.
	 */
	post: <T>(path: string, data?: unknown) =>
		request<T>(path, {
			method: 'POST',
			body: data ? JSON.stringify(data) : undefined
		}),

	/**
	 * PATCH request.
	 */
	patch: <T>(path: string, data: unknown) =>
		request<T>(path, {
			method: 'PATCH',
			body: JSON.stringify(data)
		}),

	/**
	 * DELETE request.
	 */
	delete: <T>(path: string, data?: unknown) =>
		request<T>(path, {
			method: 'DELETE',
			body: data ? JSON.stringify(data) : undefined
		})
};
