/** Rejects when an async operation does not settle within the expected time budget. */
export const withTimeout = async <T>(
	promise: Promise<T>,
	timeoutMs: number,
	label: string
): Promise<T> => {
	let timeoutId: ReturnType<typeof setTimeout> | undefined;

	const timeoutPromise = new Promise<never>((_, reject) => {
		timeoutId = setTimeout(() => {
			reject(new Error(`${label} timed out after ${timeoutMs}ms`));
		}, timeoutMs);
	});

	try {
		return await Promise.race([promise, timeoutPromise]);
	} finally {
		if (timeoutId) {
			clearTimeout(timeoutId);
		}
	}
};
