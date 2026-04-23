/// <reference types="svelte-kit" />
/// <reference types="vite/client" />

interface ImportMetaEnv {
	readonly VITE_BEEBUZZ_DOMAIN: string;
	readonly VITE_BEEBUZZ_DEBUG: boolean;
	readonly VITE_BEEBUZZ_DEPLOYMENT_MODE?: 'self_hosted' | 'saas';
}

interface ImportMeta {
	readonly env: ImportMetaEnv;
}

declare module '$app/paths' {
	export function resolve(route: string): string;
}

declare module '$app/navigation' {
	export function goto(href: string, opts?: { replaceState?: boolean }): Promise<void>;
}

declare module '$app/state' {
	import type { Readonly } from 'svelte';
	export const page: Readonly<{
		url: URL;
		data: unknown;
		state: unknown;
	}>;
}
