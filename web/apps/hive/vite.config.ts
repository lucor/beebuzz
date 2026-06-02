import { readFileSync } from 'fs';
import { join } from 'path';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { copySharedAssets } from '@beebuzz/shared/vite-plugin-copy-assets';
import type { Plugin } from 'vite';

/** In dev mode, serves dev PWA assets (manifest, favicon, apple-touch-icon) at their canonical paths. */
const devPwa = (appRoot: string, sharedStaticRoot: string): Plugin => ({
	name: 'beebuzz-dev-pwa',
	configureServer(server) {
		server.middlewares.use('/manifest.json', (_req, res) => {
			const content = readFileSync(join(appRoot, 'pwa', 'manifest.dev.json'), 'utf-8');
			res.setHeader('Content-Type', 'application/manifest+json');
			res.end(content);
		});

		const serveFile = (urlPath: string, filePath: string, contentType: string): void => {
			server.middlewares.use(urlPath, (_req, res) => {
				const content = readFileSync(filePath);
				res.setHeader('Content-Type', contentType);
				res.end(content);
			});
		};

		const devAssets = join(sharedStaticRoot, 'assets', 'dev');
		serveFile('/favicon.ico', join(devAssets, 'favicon-196.png'), 'image/png');
		serveFile('/apple-touch-icon.png', join(devAssets, 'apple-icon-180.png'), 'image/png');
		serveFile(
			'/apple-touch-icon-precomposed.png',
			join(devAssets, 'apple-icon-180.png'),
			'image/png'
		);
	}
});

export default defineConfig({
	plugins: [
		copySharedAssets(import.meta.dirname),
		devPwa(import.meta.dirname, join(import.meta.dirname, '../../packages/shared/static')),
		sveltekit()
	],
	define: {
		'import.meta.env.VITE_BEEBUZZ_DEBUG': JSON.stringify(process.env.VITE_BEEBUZZ_DEBUG === 'true')
	},
	server: {
		port: 5174,
		allowedHosts: [`hive.${process.env.BEEBUZZ_DOMAIN ?? 'localhost'}`]
	},
	ssr: {
		noExternal: ['@lucide/svelte']
	},
	build: {
		sourcemap: false,
		minify: 'esbuild'
	}
});
