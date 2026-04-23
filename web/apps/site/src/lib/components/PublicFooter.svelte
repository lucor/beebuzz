<script lang="ts">
	import { resolve } from '$app/paths';
	import { isSaasMode } from '$lib/config/deployment';

	const CREATOR_URL = 'https://lucor.dev';
	const GITHUB_REPO_URL = 'https://github.com/lucor/beebuzz';
	const STATUS_URL = 'https://status.beebuzz.app';
</script>

<footer class="border-t border-base-300 px-6 py-6 text-sm text-base-content/60">
	<div
		class="mx-auto flex max-w-6xl flex-col items-center gap-3 text-center md:flex-row md:items-center md:justify-between md:gap-6 md:text-left"
	>
		<p class="leading-6">
			BeeBuzz is designed, built{#if isSaasMode}, and operated{/if} by
			<a
				href={CREATOR_URL}
				target="_blank"
				rel="noopener noreferrer"
				class="underline transition-colors hover:text-base-content"
			>
				Luca Corbo
			</a>
		</p>

		{#if isSaasMode}
			<nav
				aria-label="Footer"
				class="flex flex-wrap items-center justify-center gap-x-4 gap-y-2 md:justify-end"
			>
				<a href={resolve('/docs')} class="transition-colors hover:text-base-content">Docs</a>
				<a href={resolve('/contact')} class="transition-colors hover:text-base-content">Contact</a>
				<a
					href={STATUS_URL}
					target="_blank"
					rel="noopener noreferrer"
					class="transition-colors hover:text-base-content"
				>
					Status
				</a>
				<a href={resolve('/policies')} class="transition-colors hover:text-base-content">Policies</a
				>
				<a
					href={GITHUB_REPO_URL}
					target="_blank"
					rel="noopener noreferrer"
					class="transition-colors hover:text-base-content"
				>
					GitHub
				</a>
				<span>© Luca Corbo</span>
			</nav>
		{:else}
			<nav
				aria-label="Footer"
				class="flex flex-wrap items-center justify-center gap-x-4 gap-y-2 md:justify-end"
			>
				<a
					href={GITHUB_REPO_URL}
					target="_blank"
					rel="noopener noreferrer"
					class="transition-colors hover:text-base-content"
				>
					GitHub
				</a>
				<span>© Luca Corbo</span>
			</nav>
		{/if}
	</div>
</footer>
