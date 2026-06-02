import type { AuthUser } from '../api';

const state = $state<{ user: AuthUser | null }>({ user: null });

export const auth = {
	get user() {
		return state.user;
	},
	set: (user: AuthUser) => {
		state.user = user;
	},
	clear: () => {
		state.user = null;
	}
};
