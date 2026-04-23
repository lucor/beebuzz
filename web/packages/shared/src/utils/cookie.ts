// Cookie utility functions for client-side cookie access.

/**
 * Get a cookie value by name.
 */
export const getCookie = (name: string): string | null => {
	if (typeof document === 'undefined') return null;

	const cookies = document.cookie.split(';');
	for (const cookie of cookies) {
		const [key, value] = cookie.trim().split('=');
		if (key === name) {
			return decodeURIComponent(value);
		}
	}
	return null;
};

/**
 * Check if user is logged in by reading beebuzz_logged_in cookie.
 */
export const isLoggedIn = (): boolean => {
	return getCookie('beebuzz_logged_in') === '1';
};
