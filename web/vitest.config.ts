import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import path from 'node:path';

export default defineConfig({
	plugins: [svelte({ hot: false })],
	resolve: {
		alias: {
			$lib: path.resolve(import.meta.dirname, 'apps/hive/src/lib'),
			'@beebuzz/shared': path.resolve(import.meta.dirname, 'packages/shared/src')
		}
	},
	test: {
		include: ['{apps,packages}/*/src/**/*.{test,spec}.{js,ts}'],
		environment: 'jsdom',
		globals: true
	}
});
