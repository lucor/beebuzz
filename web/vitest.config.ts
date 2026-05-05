import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import path from 'node:path';

export default defineConfig({
	plugins: [svelte({ hot: false })],
	resolve: {
		alias: {
			// Vitest cannot resolve SvelteKit runtime modules out of the box.
			// Hive is a client-side PWA: all unit tests run in jsdom and every
			// module under test expects a browser context, so we alias
			// $app/environment to a static shim with browser=true unconditionally.
			'$app/environment': path.resolve(
				import.meta.dirname,
				'packages/shared/src/testing/sveltekit-environment.ts'
			),
			$lib: path.resolve(import.meta.dirname, 'apps/hive/src/lib'),
			'@beebuzz/shared': path.resolve(import.meta.dirname, 'packages/shared/src')
		}
	},
	test: {
		include: ['{apps,packages}/*/src/**/*.{test,spec}.{js,ts}'],
		setupFiles: ['packages/shared/src/testing/setup.ts'],
		environment: 'jsdom',
		globals: true
	}
});
