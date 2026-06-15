const HIVE_ROUTES = new Set(['/', '/device', '/developer', '/pair']);

export function normalizeHiveRoute(url: string): string {
	try {
		const parsed = new URL(url, window.location.origin);
		const path = parsed.pathname.replace(/\/$/, '') || '/';

		if (HIVE_ROUTES.has(path)) return path;

		return path;
	} catch {
		return '/';
	}
}

export type NetworkRecord = {
	method: string;
	route: string;
	status: number;
	duration_ms: number;
	ok: boolean;
	error_code?: string;
	retry_count: number;
};
