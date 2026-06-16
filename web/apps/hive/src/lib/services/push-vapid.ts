/** Returns true when a subscription was created with the current VAPID public key. */
export const subscriptionUsesVapidKey = (
	subscription: PushSubscription,
	vapidKey: string
): boolean => {
	const existingKey = subscription.options?.applicationServerKey;
	if (!existingKey) return false;

	const newKey = urlBase64ToUint8Array(vapidKey);
	const existingBytes = new Uint8Array(existingKey);
	return (
		existingBytes.length === newKey.length &&
		existingBytes.every((value, index) => value === newKey[index])
	);
};

/** Converts a URL-safe base64 string to a Uint8Array. */
export const urlBase64ToUint8Array = (base64String: string): Uint8Array => {
	const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
	const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
	const rawData = window.atob(base64);
	const outputArray = new Uint8Array(rawData.length);

	for (let i = 0; i < rawData.length; ++i) {
		outputArray[i] = rawData.charCodeAt(i);
	}
	return outputArray;
};
