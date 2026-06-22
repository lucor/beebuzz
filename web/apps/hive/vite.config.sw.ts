import { defineConfig } from 'vite';

export default defineConfig({
	define: {
		'import.meta.env.VITE_BEEBUZZ_VERSION': JSON.stringify(
			process.env.VITE_BEEBUZZ_VERSION || 'dev'
		),
		'import.meta.env.VITE_BEEBUZZ_DOMAIN': JSON.stringify(process.env.BEEBUZZ_DOMAIN)
	},
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
