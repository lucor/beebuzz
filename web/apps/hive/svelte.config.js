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
		}
	}
};

export default config;
