<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { isLoggedIn } from '@beebuzz/shared/utils/cookie';
	import { isSaasMode, isSelfHostedMode } from '$lib/config/deployment';
	import LandingPage from '$lib/components/LandingPage.svelte';
	import AuthEntry from '$lib/components/AuthEntry.svelte';
	import PublicFooter from '$lib/components/PublicFooter.svelte';

	onMount(() => {
		if (isSelfHostedMode && isLoggedIn()) {
			void goto(resolve('/account'));
		}
	});
</script>

{#if isSaasMode}
	<LandingPage />
{:else}
	<div class="min-h-dvh flex flex-col">
		<main class="flex-1 bb-page-wrapper">
			<AuthEntry redirectAfterSubmit="/auth/verify" />
		</main>
		<PublicFooter />
	</div>
{/if}
