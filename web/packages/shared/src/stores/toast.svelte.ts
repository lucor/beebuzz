import type { ToastMessage, ToastType } from '../types';

const TOAST_DURATION = 3000;

function createToastStore() {
	let current = $state<ToastMessage | null>(null);
	let timeoutId: ReturnType<typeof setTimeout> | null = null;

	function show(message: string, type: ToastType = 'info') {
		if (timeoutId) {
			clearTimeout(timeoutId);
		}

		current = { message, type, id: Date.now() };

		timeoutId = setTimeout(() => {
			current = null;
			timeoutId = null;
		}, TOAST_DURATION);
	}

	function dismiss() {
		if (timeoutId) {
			clearTimeout(timeoutId);
			timeoutId = null;
		}
		current = null;
	}

	return {
		get current() {
			return current;
		},
		show,
		info: (message: string) => show(message, 'info'),
		success: (message: string) => show(message, 'success'),
		error: (message: string) => show(message, 'error'),
		dismiss
	};
}

export const toast = createToastStore();
