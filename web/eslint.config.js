import prettier from 'eslint-config-prettier';
import path from 'node:path';
import { includeIgnoreFile } from '@eslint/compat';
import js from '@eslint/js';
import svelte from 'eslint-plugin-svelte';
import { defineConfig } from 'eslint/config';
import globals from 'globals';
import ts from 'typescript-eslint';

const gitignorePath = path.resolve(import.meta.dirname, '.gitignore');

export default defineConfig(
	includeIgnoreFile(gitignorePath),
	js.configs.recommended,
	...ts.configs.recommendedTypeChecked,
	...svelte.configs.recommended,

	// Ignore build artifacts
	{
		ignores: ['static/sw.js']
	},

	// Type-aware parsing for all TS and Svelte files
	{
		files: ['**/*.{ts,tsx,cts,mts}', '**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
		languageOptions: {
			parserOptions: {
				projectService: true,
				tsconfigRootDir: import.meta.dirname,
				extraFileExtensions: ['.svelte'],
				parser: ts.parser
			}
		},
		rules: {
			// typescript-eslint strongly recommend that you do not use the no-undef lint rule on TypeScript projects.
			// see: https://typescript-eslint.io/troubleshooting/faqs/eslint/#i-get-errors-from-the-no-undef-rule-about-global-variables-not-being-defined-even-though-there-are-no-typescript-errors
			'no-undef': 'off',
			// Disable resolve requirement for href - the resolve() is used but ESLint can't track it through variables
			'svelte/no-navigation-without-resolve': 'off',
			// Flag deprecated APIs (stdlib, third-party, and project-internal) at lint time.
			'@typescript-eslint/no-deprecated': 'warn'
		}
	},

	// Browser globals for Svelte components
	{
		files: ['**/*.svelte'],
		languageOptions: {
			globals: globals.browser
		}
	},

	// Frontend naming convention: app-internal code uses camelCase.
	// Keep boundary snake_case confined to DTO/raw payload files.
	{
		files: ['apps/**/*.{ts,svelte}', 'packages/shared/src/**/*.{ts,svelte}'],
		ignores: ['packages/shared/src/api/**/*.ts', 'apps/hive/src/sw.ts', '**/*.{test,spec}.ts'],
		rules: {
			'@typescript-eslint/naming-convention': [
				'error',
				{
					selector: 'variable',
					format: ['camelCase', 'UPPER_CASE', 'PascalCase'],
					leadingUnderscore: 'allow'
				},
				{
					selector: 'import',
					format: ['camelCase', 'PascalCase']
				},
				{
					selector: 'parameter',
					format: ['camelCase'],
					leadingUnderscore: 'allow'
				},
				{
					selector: 'typeLike',
					format: ['PascalCase']
				}
			]
		}
	},

	// Node globals for config files and server-side code
	{
		files: [
			'**/*.config.{js,ts,mjs,cjs}',
			'**/+server.*',
			'**/+page.server.*',
			'**/+layout.server.*',
			'**/hooks.server.*'
		],
		languageOptions: {
			globals: globals.node
		}
	},

	// Vitest globals for test files
	{
		files: ['**/*.{test,spec}.{js,ts}'],
		languageOptions: {
			globals: globals.vitest
		}
	},

	// Disable type-checked rules on files not covered by any tsconfig
	{
		files: [
			'**/*.{js,cjs,mjs}',
			'**/vite.config*.ts',
			'**/vitest.config*.ts',
			'**/playwright.config*.ts',
			'**/svelte.config*.js',
			'**/postcss.config*.js',
			'tests/**/*.ts'
		],
		...ts.configs.disableTypeChecked
	},

	// Service worker globals
	{
		files: ['apps/hive/src/sw.ts'],
		languageOptions: {
			globals: globals.serviceworker
		}
	},

	...svelte.configs.prettier,
	prettier
);
