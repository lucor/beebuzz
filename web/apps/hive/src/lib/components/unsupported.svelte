<script lang="ts">
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { CircleAlert, Globe } from '@lucide/svelte';
	import { onboarding } from '$lib/onboarding.svelte';
	import type { CapabilityResult } from '$lib/services/capability';

	interface Props {
		capabilities?: CapabilityResult | null;
	}

	const { capabilities = onboarding.capabilities }: Props = $props();

	const CAPABILITY_LABELS: Record<string, string> = {
		secure: 'Secure context (HTTPS)',
		serviceWorker: 'Service Worker',
		pushManager: 'Push API',
		notification: 'Notification API',
		encryption: 'Modern encryption (X25519)'
	};

	const REQUIRED_KEYS = ['secure', 'serviceWorker', 'pushManager', 'notification', 'encryption'];

	const failedCapabilities = $derived.by(() => {
		if (!capabilities) return [];
		return REQUIRED_KEYS.filter((k) => !capabilities[k as keyof CapabilityResult]);
	});
</script>

<main class="flex items-center justify-center min-h-dvh bg-[#FDF7ED] px-4">
	<div class="w-full max-w-md">
		<!-- Header -->
		<div class="text-center mb-8">
			<div class="flex flex-col items-center justify-center mb-4">
				<BeeBuzzLogo variant="img" class="w-16 h-16 mb-2" />
			</div>
		</div>

		<!-- Card -->
		<div class="bg-white rounded-lg shadow-md border border-[#E6E1D8] p-8">
			<div class="flex items-center justify-center mb-4">
				<div class="p-3 bg-red-100 rounded-full">
					<CircleAlert size={32} class="text-red-500" />
				</div>
			</div>

			<h2 class="text-lg font-bold text-[#2D3748] text-center mb-2">Browser Not Supported</h2>

			<p class="text-[#6B7280] text-sm text-center mb-6">
				Your browser does not support the features required by BeeBuzz.
			</p>

			{#if failedCapabilities.length > 0}
				<div class="bg-red-50 rounded-lg p-3 mb-6">
					<p class="text-xs font-semibold text-red-700 mb-2">Missing capabilities</p>
					<ul class="text-xs text-red-600 space-y-1">
						{#each failedCapabilities as key (key)}
							<li>{CAPABILITY_LABELS[key] ?? key}</li>
						{/each}
					</ul>
				</div>
			{/if}

			<div class="bg-[#FDF7ED] rounded-lg p-4">
				<p class="text-sm font-semibold text-[#2D3748] mb-3">Supported browsers</p>
				<ul class="text-sm text-[#6B7280] space-y-2">
					<li class="flex items-center gap-2">
						<Globe size={16} class="flex-shrink-0" />
						<span>Chrome / Edge (desktop and Android)</span>
					</li>
					<li class="flex items-center gap-2">
						<Globe size={16} class="flex-shrink-0" />
						<span>Safari (macOS Sonoma+ and iOS 16.4+)</span>
					</li>
					<li class="flex items-center gap-2">
						<Globe size={16} class="flex-shrink-0" />
						<span>Firefox (desktop)</span>
					</li>
				</ul>
			</div>
		</div>
	</div>
</main>
