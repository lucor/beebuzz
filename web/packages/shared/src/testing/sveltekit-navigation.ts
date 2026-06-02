// Mock for $app/navigation in Vitest.
export const goto = (
	url: string | URL,
	opts?: { replaceState?: boolean; noscroll?: boolean; keepfocus?: boolean; state?: unknown }
): Promise<void> => {
	void url;
	void opts;
	return Promise.resolve();
};
