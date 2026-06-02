import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { resolve } from 'path';
import { fileURLToPath } from 'url';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

/** @type {import('@sveltejs/kit').Config} */
const config = {
	extensions: ['.svelte', '.md'],
	preprocess: vitePreprocess({ script: true }),
	kit: {
		adapter: adapter({
			pages: '../../build/site',
			assets: '../../build/site',
			fallback: 'fallback.html',
			strict: false
		}),
		paths: {
			base: '',
			relative: false
		},
		alias: {
			'@beebuzz/shared': resolve(__dirname, '../../packages/shared/src')
		},
		prerender: {
			handleHttpError: 'warn',
			handleUnseenRoutes: 'ignore'
		}
	}
};

export default config;
