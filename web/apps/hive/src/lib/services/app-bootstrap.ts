export interface AppBootstrapDeps<TRegistration> {
	registerServiceWorker: () => Promise<TRegistration>;
	checkPaired: () => Promise<boolean>;
	getDeviceId: () => Promise<string | null>;
	activateNotifications: (deviceId: string) => void;
	attachServiceWorkerListeners: () => void;
	migrateLegacyNotifications: (deviceId: string) => Promise<void>;
	loadPersistedNotifications: (phase: 'initial' | 'final') => Promise<void>;
	runPostPairingChecks: (registration: TRegistration) => Promise<void>;
}

export interface AppBootstrapResult<TRegistration> {
	registration: TRegistration;
	isPaired: boolean;
	deviceId: string | null;
}

/**
 * Boots the Hive app shell after resolving the current backend device identity.
 *
 * Invariants enforced by this function:
 * 1. Paired state and stored device credentials are checked before any local
 *    notification history is loaded.
 * 2. Notification history is activated for the current backend device ID before
 *    the service worker bridge is attached or IndexedDB is drained.
 * 3. Legacy IndexedDB records (created before per-device scoping) are stamped
 *    with the current deviceId or removed if they belong to a different device.
 * 4. A final drain runs after slower paired-device checks, closing the residual
 *    window before polling starts. If those checks reject, the final drain is
 *    skipped and the caller owns listener teardown.
 *
 * Early-return contracts:
 * - `checkPaired` resolves to `false`: no further work runs and the caller is
 *   expected to redirect to the pairing flow.
 * - `getDeviceId` resolves to `null` while paired: notification activation,
 *   listener attach, and IndexedDB drains are all skipped. The caller owns
 *   recovery (typically through `reconcilePushState`).
 */
export async function bootstrapAppShell<TRegistration>(
	deps: AppBootstrapDeps<TRegistration>
): Promise<AppBootstrapResult<TRegistration>> {
	const registration = await deps.registerServiceWorker();

	const isPaired = await deps.checkPaired();
	if (!isPaired) {
		return { registration, isPaired, deviceId: null };
	}

	const deviceId = await deps.getDeviceId();
	if (!deviceId) {
		return { registration, isPaired, deviceId: null };
	}

	deps.activateNotifications(deviceId);
	deps.attachServiceWorkerListeners();
	await deps.migrateLegacyNotifications(deviceId);
	await deps.loadPersistedNotifications('initial');
	await deps.runPostPairingChecks(registration);
	await deps.loadPersistedNotifications('final');

	return { registration, isPaired, deviceId };
}
