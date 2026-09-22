import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { copySharedAssets } from '@beebuzz/shared/vite-plugin-copy-assets';
import type { Plugin } from 'vite';

const BEEBUZZ_DOMAIN = process.env.BEEBUZZ_DOMAIN;

const runtimeConfig: Plugin = {
	name: 'beebuzz-runtime-config',
	configureServer(server) {
		if (!BEEBUZZ_DOMAIN) {
			throw new Error('BEEBUZZ_DOMAIN is required to run the dashboard dev server.');
		}
		server.middlewares.use('/config.js', (_req, res) => {
			res.setHeader('Content-Type', 'application/javascript');
			res.end(`window.__BEEBUZZ_CONFIG__ = { domain: '${BEEBUZZ_DOMAIN}' };`);
		});
	}
};

export default defineConfig({
	plugins: [runtimeConfig, copySharedAssets(import.meta.dirname), sveltekit()],
	define: {
		'import.meta.env.VITE_BEEBUZZ_DEBUG': JSON.stringify(process.env.VITE_BEEBUZZ_DEBUG === 'true')
	},
	server: {
		port: 5173,
		allowedHosts: BEEBUZZ_DOMAIN ? [`dashboard.${BEEBUZZ_DOMAIN}`] : []
	},
	ssr: {
		noExternal: ['@lucide/svelte']
	},
	build: {
		sourcemap: false,
		minify: 'esbuild'
	}
});
