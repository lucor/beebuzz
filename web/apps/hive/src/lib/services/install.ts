// PWA install lifecycle helpers.

/** Chromium-specific BeforeInstallPromptEvent. Not in lib.dom.d.ts. */
interface BeforeInstallPromptEvent extends Event {
	readonly platforms: string[];
	readonly userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>;
	prompt(): Promise<void>;
}

const INSTALLED_KEY = 'bb_installed';

let deferredPrompt: BeforeInstallPromptEvent | null = null;
let appInstalledFired = false;
let listenersRegistered = false;

/** Registers the beforeinstallprompt and appinstalled listeners. Idempotent — safe to call multiple times. */
export const initInstallListener = (
	onPromptAvailable?: () => void,
	onInstalled?: () => void
): void => {
	if (listenersRegistered) return;
	listenersRegistered = true;

	window.addEventListener('beforeinstallprompt', (e: Event) => {
		e.preventDefault();
		deferredPrompt = e as BeforeInstallPromptEvent;
		onPromptAvailable?.();
	});

	window.addEventListener('appinstalled', () => {
		appInstalledFired = true;
		deferredPrompt = null;
		localStorage.setItem(INSTALLED_KEY, 'true');
		onInstalled?.();
	});
};

/** Returns true if the browser has a deferred install prompt (Chromium only). */
export const canInstallNatively = (): boolean => {
	return deferredPrompt !== null;
};

/** Returns true if the appinstalled event has fired this session. */
export const wasInstalled = (): boolean => {
	return appInstalledFired;
};

/** Triggers the native install prompt. Returns true if the user accepted. */
export const triggerInstall = async (): Promise<boolean> => {
	if (!deferredPrompt) return false;

	await deferredPrompt.prompt();
	const { outcome } = await deferredPrompt.userChoice;
	deferredPrompt = null;

	if (outcome === 'accepted') {
		localStorage.setItem(INSTALLED_KEY, 'true');
	}

	return outcome === 'accepted';
};

/** Returns true if the app was installed in a previous session. */
export const wasInstalledPreviously = (): boolean => {
	return localStorage.getItem(INSTALLED_KEY) === 'true';
};
