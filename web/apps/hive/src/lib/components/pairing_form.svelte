<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { onboarding } from '../onboarding.svelte';
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { OTP_LENGTH } from '@beebuzz/shared/constants/auth';
	import { Circle } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { SITE_URL } from '@beebuzz/shared/config';

	interface Props {
		showFirefoxHint?: boolean;
	}

	let { showFirefoxHint = false }: Props = $props();

	let pairingCode = $state('');
	let isPairing = $derived(onboarding.state === 'pairing');
	let pairingInput = $state<HTMLInputElement | null>(null);

	/** Formats the pairing code to digits only. */
	const formatPairingCode = (value: string): string => {
		return value.replace(/\D/g, '').slice(0, OTP_LENGTH);
	};

	/** Keeps the underlying input normalized for typing and paste. */
	const handlePairingInput = (event: Event) => {
		const input = event.target as HTMLInputElement;
		input.value = formatPairingCode(input.value);
		pairingCode = input.value;
	};

	onMount(async () => {
		await tick();
		pairingInput?.focus();
	});
</script>

<main class="flex items-center justify-center min-h-dvh bg-base-100 px-4">
	<div class="w-full max-w-md">
		<!-- Header -->
		<div class="text-center mb-8">
			<div class="flex flex-col items-center justify-center mb-4">
				<BeeBuzzLogo variant="img" class="w-16 h-16 mb-2" />
				<BeeBuzzLogo variant="text" class="w-48 h-12" />
			</div>
			<p class="text-lg text-base-content/80 font-medium">
				Connect a device to start receiving notifications
			</p>
		</div>

		<!-- Firefox hint -->
		{#if showFirefoxHint}
			<div class="alert alert-warning border-warning/20 p-4 mb-6 text-sm">
				Keep Firefox running to receive notifications. Firefox does not support PWA installation.
			</div>
		{/if}

		<!-- Card -->
		<div class="bg-base-100 rounded-lg shadow-md border border-base-300 p-8 mb-6">
			<!-- Description -->
			<p class="text-base-content/60 text-sm mb-6 leading-relaxed">
				To use BeeBuzz on this device, you need a Pairing Code. Pairing Codes are generated from
				your BeeBuzz account and securely link this device.
			</p>

			<!-- Form -->
			<form
				onsubmit={(e) => {
					e.preventDefault();
					if (pairingCode.length === OTP_LENGTH && !isPairing) {
						void onboarding.startPairing(pairingCode);
					}
				}}
			>
				<div class="mb-5">
					<label for="pairing-code" class="block text-sm font-semibold text-base-content mb-2">
						Pairing Code
					</label>
					<div class="relative">
						<input
							id="pairing-code"
							type="text"
							bind:this={pairingInput}
							pattern="[0-9]*"
							maxlength={OTP_LENGTH}
							autocomplete="one-time-code"
							inputmode="numeric"
							enterkeyhint="done"
							class="absolute inset-0 h-full w-full cursor-text opacity-0"
							bind:value={pairingCode}
							oninput={handlePairingInput}
							disabled={isPairing}
							aria-label="Pairing code"
							required
						/>

						<div class="grid grid-cols-6 gap-2" aria-hidden="true">
							{#each Array.from({ length: OTP_LENGTH }, (_, index) => index) as index (index)}
								<div
									class={[
										'flex aspect-square items-center justify-center rounded-xl border bg-base-200 text-2xl font-bold tabular-nums transition',
										index === pairingCode.length && pairingCode.length < OTP_LENGTH
											? 'border-primary ring-2 ring-primary/20'
											: 'border-base-300',
										pairingCode[index] ? 'text-base-content shadow-sm' : 'text-base-content/30'
									].join(' ')}
								>
									{pairingCode[index] ?? ''}
								</div>
							{/each}
						</div>
					</div>
				</div>

				<!-- Loading state -->
				{#if isPairing}
					<div class="flex items-center justify-center py-3 mb-4">
						<span class="loading loading-spinner loading-sm mr-2"></span>
						<span class="text-sm text-base-content">Pairing device...</span>
					</div>
				{/if}

				<!-- Submit button -->
				<button
					type="submit"
					class="btn btn-block btn-lg btn-primary font-semibold mb-4"
					disabled={isPairing || pairingCode.length !== OTP_LENGTH}
				>
					Connect Device
				</button>
			</form>

			<!-- Secondary action -->
			<div class="text-center text-sm text-base-content/60">
				Don't have a Pairing Code?
				<a
					href={SITE_URL + resolve('/login' as '/')}
					class="text-primary font-semibold hover:underline"
				>
					Sign in to generate one.
				</a>
			</div>
		</div>

		<!-- Notes -->
		<div class="text-xs text-base-content/60 space-y-2">
			<div class="flex items-start gap-2">
				<Circle size={12} class="flex-shrink-0 mt-0.5" />
				<span>Pairing Codes are single-use and time-limited</span>
			</div>
		</div>
	</div>
</main>
