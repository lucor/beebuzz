import { healthApi } from '../api';

export interface HealthState {
	status: 'unknown' | 'ok' | 'error';
	version: string | null;
	commit: string | null;
	loading: boolean;
}

function createHealthStore() {
	let state = $state<HealthState>({
		status: 'unknown',
		version: null,
		commit: null,
		loading: false
	});

	async function check() {
		state.loading = true;
		try {
			const health = await healthApi.checkHealth();
			state = {
				status: health.status === 'ok' ? 'ok' : 'error',
				version: health.version,
				commit: health.commit,
				loading: false
			};
		} catch {
			state = {
				status: 'error',
				version: null,
				commit: null,
				loading: false
			};
		}
	}

	return {
		get state() {
			return state;
		},
		get status() {
			return state.status;
		},
		get version() {
			return state.version;
		},
		get commit() {
			return state.commit;
		},
		get loading() {
			return state.loading;
		},
		check
	};
}

export const health = createHealthStore();
