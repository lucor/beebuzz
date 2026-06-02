import { copyFileSync, mkdirSync, readdirSync, existsSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import type { Plugin } from 'vite';

const SHARED_STATIC = join(dirname(fileURLToPath(import.meta.url)), '../static');

/** Directories excluded from production builds (only served via Vite dev middleware). */
const DEV_ONLY_DIRS = ['dev'];

/** Recursively copies src directory into dest directory, optionally skipping dev-only subdirectories. */
const syncDir = (src: string, dest: string, excludeDirs: string[] = []): void => {
	if (!existsSync(src)) return;
	mkdirSync(dest, { recursive: true });
	for (const entry of readdirSync(src, { withFileTypes: true })) {
		const srcPath = join(src, entry.name);
		const destPath = join(dest, entry.name);
		if (entry.isDirectory()) {
			if (excludeDirs.includes(entry.name)) continue;
			syncDir(srcPath, destPath, excludeDirs);
		} else {
			copyFileSync(srcPath, destPath);
		}
	}
};

/** Copies shared static assets from packages/shared/static/ into the app's static/ dir. Excludes dev-only assets in production builds. */
export const copySharedAssets = (appRoot: string): Plugin => ({
	name: 'beebuzz-copy-shared-assets',
	buildStart() {
		syncDir(SHARED_STATIC, join(appRoot, 'static'), DEV_ONLY_DIRS);
	},
	configureServer() {
		syncDir(SHARED_STATIC, join(appRoot, 'static'));
	}
});
