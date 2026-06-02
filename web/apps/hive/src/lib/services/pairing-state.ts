export interface PairedStateDeps {
	notificationPermission: NotificationPermission;
	getRegistration: () => Promise<ServiceWorkerRegistration | undefined>;
	hasIdentity: () => Promise<boolean>;
}

/** Checks whether the current browser still has the local state required for a paired device. */
export const checkPairedState = async ({
	notificationPermission,
	getRegistration,
	hasIdentity
}: PairedStateDeps): Promise<boolean> => {
	if (notificationPermission !== 'granted') {
		return false;
	}

	const registration = await getRegistration();
	if (!registration) {
		return false;
	}

	const subscription = await registration.pushManager.getSubscription();
	if (!subscription) {
		return false;
	}

	return hasIdentity();
};
