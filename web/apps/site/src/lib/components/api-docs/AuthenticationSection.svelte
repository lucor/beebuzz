<script lang="ts">
	import type { ApiSecurityScheme } from '$lib/api-docs/types';
	import { Lock } from '@lucide/svelte';

	let {
		schemes,
		preferredScheme = 'bearerAuth'
	}: {
		schemes: Record<string, ApiSecurityScheme>;
		preferredScheme?: string;
	} = $props();

	const scheme = $derived(schemes[preferredScheme]);
</script>

{#if scheme && scheme.type === 'http' && scheme.scheme === 'bearer'}
	<section id="authentication" class="not-prose scroll-mt-20">
		<div class="card bg-base-100 border border-base-300 shadow-sm">
			<div class="card-body p-4 md:p-6 gap-3">
				<div class="flex items-center gap-2">
					<Lock size={18} aria-hidden="true" />
					<h2 class="text-xl font-bold m-0">Authentication</h2>
				</div>
				<p class="text-sm text-base-content/80 m-0">
					Public BeeBuzz API endpoints authenticate with an <strong>API token</strong> sent as a
					<code class="font-mono">Bearer</code> credential in the
					<code class="font-mono">Authorization</code> header.
				</p>
				<div class="overflow-hidden rounded-box border border-base-300 bg-base-200">
					<div
						class="border-b border-base-300 bg-base-300/40 px-3 py-2 text-xs font-semibold uppercase tracking-wide text-base-content/70"
					>
						Authorization header
					</div>
					<pre class="px-3 py-3 text-sm font-mono whitespace-pre-wrap break-all"><code
							>Authorization: Bearer YOUR_API_TOKEN</code
						></pre>
				</div>
				<p class="text-sm text-base-content/70 m-0">
					API tokens are created from <strong>Account → Tokens</strong> in the BeeBuzz web app. Treat
					them like passwords: store them securely and rotate if leaked.
				</p>
			</div>
		</div>
	</section>
{/if}
