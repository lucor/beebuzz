import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { marked } from 'marked';
import { copySharedAssets } from '@beebuzz/shared/vite-plugin-copy-assets';
import type { Plugin } from 'vite';
import { codeToTokens } from 'shiki';

function escapeHtml(value: string): string {
	return value
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;')
		.replace(/'/g, '&#39;');
}

function parseFrontmatter(content: string): { frontmatter: Record<string, unknown>; body: string } {
	const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---/);
	if (!match) {
		return { frontmatter: {}, body: content };
	}

	const frontmatterBlock = match[1];
	const frontmatter: Record<string, unknown> = {};

	for (const line of frontmatterBlock.split('\n')) {
		const colonIndex = line.indexOf(':');
		if (colonIndex === -1) continue;

		const key = line.slice(0, colonIndex).trim();
		let value: unknown = line.slice(colonIndex + 1).trim();

		if (value === '') {
			// empty value, skip
		} else if (typeof value === 'string' && (value.startsWith("'") || value.startsWith('"'))) {
			value = value.slice(1, -1);
		} else if (!isNaN(Number(value))) {
			value = Number(value);
		} else if (value === 'true') {
			value = true;
		} else if (value === 'false') {
			value = false;
		}

		frontmatter[key] = value;
	}

	const body = content.slice(match[0].length).trim();
	return { frontmatter, body };
}

function normalizeLanguage(lang: string | undefined): string {
	const resolved = (lang || 'text').toLowerCase();

	if (resolved === 'shell') {
		return 'shellscript';
	}

	return resolved;
}

function isShellLikeLanguage(lang: string): boolean {
	return lang === 'bash' || lang === 'sh' || lang === 'zsh' || lang === 'shellscript';
}

/** Returns per-line prefixes for shell blocks: '$' for commands, '>' for continuations. */
function getShellLinePrefixes(rawLines: string[]): Array<string | undefined> {
	let continuation = false;

	return rawLines.map((line) => {
		if (line.trim() === '') {
			continuation = false;
			return undefined;
		}

		const prefix = continuation ? '>' : '$';
		continuation = line.trimEnd().endsWith('\\');
		return prefix;
	});
}

function renderTokenSpan(content: string, color: string | undefined, fontStyle: number): string {
	const styles: string[] = [];

	if (color) {
		styles.push(`color:${color}`);
	}

	if (fontStyle & 1) {
		styles.push('font-style:italic');
	}

	if (fontStyle & 2) {
		styles.push('font-weight:600');
	}

	if (fontStyle & 4) {
		styles.push('text-decoration:underline');
	}

	const styleAttribute = styles.length > 0 ? ` style="${styles.join(';')}"` : '';
	return `<span${styleAttribute}>${escapeHtml(content)}</span>`;
}

async function renderTerminalCodeBlock(code: string, lang: string | undefined): Promise<string> {
	const resolvedLanguage = normalizeLanguage(lang);
	const highlightedLanguage = isShellLikeLanguage(resolvedLanguage)
		? 'shellscript'
		: resolvedLanguage;
	const codeToRender = code.replace(/\n$/, '');

	let lines;
	try {
		const tokens = await codeToTokens(codeToRender, {
			lang: highlightedLanguage as never,
			theme: 'github-dark'
		});
		lines = tokens.tokens;
	} catch {
		const fallback = codeToRender.split('\n').map((line) => [
			{
				content: line,
				color: undefined,
				fontStyle: 0
			}
		]);
		lines = fallback;
	}

	if (lines.length === 0) {
		lines = [[]];
	}

	const shellLike = isShellLikeLanguage(highlightedLanguage);
	const rawLines = codeToRender.split('\n');
	const prefixes = shellLike ? getShellLinePrefixes(rawLines) : [];

	const lineHtml = lines
		.map((lineTokens, lineIndex) => {
			const prefix = prefixes[lineIndex];
			const prefixAttribute = prefix ? ` data-prefix="${prefix}"` : '';
			const renderedLine = lineTokens
				.map((token) => renderTokenSpan(token.content, token.color, token.fontStyle ?? 0))
				.join('');

			return `<pre${prefixAttribute}><code>${renderedLine || '&nbsp;'}</code></pre>`;
		})
		.join('\n');

	const languageLabel = isShellLikeLanguage(highlightedLanguage) ? 'shell' : resolvedLanguage;
	const copiedCode = escapeHtml(codeToRender);

	return `<div class="not-prose docs-terminal" data-code-language="${escapeHtml(languageLabel)}">
	<div class="docs-terminal-bar">
		<div class="flex items-center gap-2">
			<div class="docs-terminal-dots" aria-hidden="true">
				<span class="docs-terminal-dot bg-error"></span>
				<span class="docs-terminal-dot bg-warning"></span>
				<span class="docs-terminal-dot bg-success"></span>
			</div>
		</div>
		<button
			type="button"
			class="docs-terminal-copy btn btn-ghost btn-xs btn-square"
			data-docs-copy-button
			data-docs-copy-code="${copiedCode}"
			aria-label="Copy code block"
		>
			<svg data-copy-icon="copy" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>
			<svg data-copy-icon="check" class="hidden" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>
			<span class="sr-only" data-copy-status aria-live="polite"></span>
		</button>
	</div>
	<div class="docs-terminal-body">${lineHtml}</div>
</div>`;
}

function renderTable(header: string, body: string): string {
	return `<div class="not-prose my-6 overflow-x-auto"><table class="table table-zebra w-full">${header}${body}</table></div>`;
}

async function transformMarkdown(body: string): Promise<string> {
	const codeBlockPattern = /```([\w-]+)?\n([\s\S]*?)\n```/g;
	let transformedBody = '';
	let lastIndex = 0;
	let match: RegExpExecArray | null;

	while ((match = codeBlockPattern.exec(body)) !== null) {
		transformedBody += body.slice(lastIndex, match.index);
		transformedBody += await renderTerminalCodeBlock(match[2], match[1]);
		lastIndex = match.index + match[0].length;
	}

	transformedBody += body.slice(lastIndex);
	return transformedBody;
}

const markdownPlugin: Plugin = {
	name: 'vite-plugin-markdown-to-svelte',
	async transform(code: string, id: string) {
		if (!id.endsWith('.md')) {
			return null;
		}

		const { frontmatter, body } = parseFrontmatter(code);
		const renderer = new marked.Renderer();

		renderer.table = function (token) {
			let header = '';
			for (const cell of token.header) {
				header += this.tablecell(cell);
			}

			let body = '';
			for (const row of token.rows) {
				let text = '';
				for (const cell of row) {
					text += this.tablecell(cell);
				}

				body += this.tablerow({ text });
			}

			return renderTable(header, body);
		};

		const transformedBody = await transformMarkdown(body);
		const html = await marked.parse(transformedBody, { renderer });

		const svelteCode = `<script lang="ts">
	export const frontmatter = ${JSON.stringify(frontmatter)};
</script>

{@html \`${html.replace(/`/g, '\\`').replace(/\$/g, '\\$')}\`}
`;

		return {
			code: svelteCode,
			map: null
		};
	}
};

export default defineConfig({
	plugins: [copySharedAssets(import.meta.dirname), markdownPlugin, sveltekit()],
	define: {
		'import.meta.env.VITE_BEEBUZZ_DEBUG': JSON.stringify(process.env.VITE_BEEBUZZ_DEBUG === 'true')
	},
	server: {
		port: 5173,
		allowedHosts: [process.env.BEEBUZZ_DOMAIN ?? 'localhost']
	},
	ssr: {
		noExternal: ['@lucide/svelte']
	},
	build: {
		sourcemap: false,
		minify: 'esbuild'
	}
});
