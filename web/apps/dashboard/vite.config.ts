import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { copySharedAssets } from '@beebuzz/shared/vite-plugin-copy-assets';

const BEEBUZZ_DOMAIN = process.env.BEEBUZZ_DOMAIN;

if (!BEEBUZZ_DOMAIN) {
	throw new Error('BEEBUZZ_DOMAIN is required to build the dashboard app.');
}

export default defineConfig({
	plugins: [copySharedAssets(import.meta.dirname), sveltekit()],
	define: {
		'import.meta.env.VITE_BEEBUZZ_DOMAIN': JSON.stringify(BEEBUZZ_DOMAIN),
		'import.meta.env.VITE_BEEBUZZ_DEBUG': JSON.stringify(process.env.VITE_BEEBUZZ_DEBUG === 'true')
	},
	server: {
		port: 5173,
		allowedHosts: [`dashboard.${BEEBUZZ_DOMAIN}`, 'localhost']
	},
	ssr: {
		noExternal: ['@lucide/svelte']
	},
	build: {
		sourcemap: false,
		minify: 'esbuild'
	}
});
