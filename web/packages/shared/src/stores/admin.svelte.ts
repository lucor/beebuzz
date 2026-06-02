// Admin store - uses httpOnly cookies for authentication
// Token is managed by the server and not stored in localStorage

function createAdminStore() {
	return {
		// Note: Authentication is now cookie-based (httpOnly)
		// Frontend no longer needs to manage token state
		logout() {
			// Cookies are cleared server-side on logout
		}
	};
}

export const adminStore = createAdminStore();
