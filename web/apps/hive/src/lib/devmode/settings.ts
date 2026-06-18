import { writable } from 'svelte/store';

export type HiveDeveloperSettings = {
	enabled: boolean;
};

const DEFAULT_SETTINGS: HiveDeveloperSettings = {
	enabled: false
};

function createSettingsStore() {
	const store = writable<HiveDeveloperSettings>(DEFAULT_SETTINGS);

	return {
		subscribe: store.subscribe,
		set: store.set
	};
}

export const developerSettings = createSettingsStore();
