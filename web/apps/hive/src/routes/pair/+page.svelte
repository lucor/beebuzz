<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { paired } from '$lib/stores/paired.svelte';
	import { onboarding } from '$lib/onboarding.svelte';

	import InstallScreen from '$lib/components/install_screen.svelte';
	import PairingForm from '$lib/components/pairing_form.svelte';
	import PermissionBlocked from '$lib/components/permission_blocked.svelte';
	import Unsupported from '$lib/components/unsupported.svelte';
	import OnboardingError from '$lib/components/onboarding_error.svelte';

	onMount(() => {
		const init = async () => {
			if (paired.isPaired) {
				await goto('/');
				return;
			}

			await onboarding.init();
		};

		void init();
	});

	// Redirect to app once pairing is complete
	$effect(() => {
		if (onboarding.state === 'paired') {
			void goto('/');
		}
	});

	let showFirefoxHint = $derived(onboarding.installPlatform === 'firefox');
</script>

{#if onboarding.state === 'checking'}
	<main class="flex items-center justify-center min-h-dvh bg-base-100">
		<span class="loading loading-spinner loading-lg text-primary"></span>
	</main>
{:else if onboarding.state === 'unsupported'}
	<Unsupported />
{:else if onboarding.state === 'not-installed'}
	<InstallScreen platform={onboarding.installPlatform} />
{:else if onboarding.state === 'ready' || onboarding.state === 'pairing' || onboarding.state === 'permission-prompt'}
	<PairingForm {showFirefoxHint} />
{:else if onboarding.state === 'permission-blocked'}
	<PermissionBlocked />
{:else if onboarding.state === 'error'}
	<OnboardingError />
{/if}
