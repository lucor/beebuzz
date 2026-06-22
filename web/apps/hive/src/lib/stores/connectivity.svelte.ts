import { browser } from '$app/environment';

function createConnectivityStore() {
	let online = $state(true);

	function handleOnline() {
		online = true;
	}

	function handleOffline() {
		online = false;
	}

	function init() {
		if (!browser) return;
		online = navigator.onLine;
		window.addEventListener('online', handleOnline);
		window.addEventListener('offline', handleOffline);
	}

	function destroy() {
		if (!browser) return;
		window.removeEventListener('online', handleOnline);
		window.removeEventListener('offline', handleOffline);
	}

	const tone = $derived(online ? ('online' as const) : ('offline' as const));
	const label = $derived(online ? 'Online' : 'Offline');

	function _resetForTest(value: boolean) {
		online = value;
	}

	return {
		get online() {
			return online;
		},
		get tone() {
			return tone;
		},
		get label() {
			return label;
		},
		init,
		destroy,
		_resetForTest
	};
}

export const connectivity = createConnectivityStore();
