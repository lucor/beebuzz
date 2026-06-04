import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { resolve } from 'path';
import { fileURLToPath } from 'url';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess({ script: true }),
	kit: {
		adapter: adapter({
			pages: '../../build/hive',
			assets: '../../build/hive',
			fallback: 'index.html',
			strict: false
		}),
		alias: {
			'@beebuzz/shared': resolve(__dirname, '../../packages/shared/src')
		},
		paths: {
			base: '',
			relative: false
		},
		// Hive holds the user's E2E private key in memory, so a successful
		// XSS would let an attacker exfiltrate it. The static adapter
		// makes any inline scripts SvelteKit emits deterministic at build
		// time, but the edge header still needs to allow the bootstrap
		// inline script because the browser enforces both policies.
		csp: {
			mode: 'hash',
			directives: {
				'script-src': ['self']
			}
		}
	}
};

export default config;
