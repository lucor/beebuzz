import { defineConfig } from 'vite';

export default defineConfig({
	build: {
		lib: {
			entry: 'src/sw.ts',
			formats: ['iife'],
			name: 'sw',
			fileName: () => 'sw.js'
		},
		outDir: 'static',
		emptyOutDir: false,
		sourcemap: false,
		minify: true
	}
});
