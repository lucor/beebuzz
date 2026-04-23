<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { toast } from '@beebuzz/shared/stores';
	import { verifyOtp } from '@beebuzz/shared/services/auth';
	import { ApiError, isInlineError } from '@beebuzz/shared/errors';
	import { STORAGE_KEY_STATE, STORAGE_KEY_EMAIL, OTP_LENGTH } from '@beebuzz/shared/constants/auth';
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { ArrowLeft } from '@lucide/svelte';
	import { isSaasMode } from '$lib/config/deployment';

	type ViewState = 'loading' | 'otp';

	let view = $state<ViewState>('loading');
	let email = $state('');
	let verificationCode = $state('');
	let isLoading = $state(false);
	let error = $state<string | undefined>(undefined);
	let otpInput = $state<HTMLInputElement | null>(null);

	/** Formats OTP input to digits only. */
	const formatOtp = (value: string): string => {
		return value.replace(/\D/g, '').slice(0, OTP_LENGTH);
	};

	/** Handles OTP input filtering. */
	const handleOtpInput = (event: Event) => {
		error = undefined;
		const input = event.target as HTMLInputElement;
		input.value = formatOtp(input.value);
		verificationCode = input.value;
	};

	/** Submits OTP code for verification. */
	const handleVerifyOtp = async (event: SubmitEvent) => {
		event.preventDefault();

		if (verificationCode.length !== OTP_LENGTH) {
			error = 'Please enter all 6 digits';
			return;
		}

		const storedState = localStorage.getItem(STORAGE_KEY_STATE);
		if (!storedState) {
			error = 'Login session expired. Please start over.';
			return;
		}

		isLoading = true;
		error = undefined;

		try {
			await verifyOtp(verificationCode, storedState);
			toast.success('Authentication successful');
			await goto(resolve('/account'));
		} catch (err) {
			if (err instanceof ApiError && isInlineError(err.code)) {
				error = err.userMessage;
			} else {
				error = undefined;
				toast.error(err instanceof ApiError ? err.userMessage : 'Verification failed');
			}
		} finally {
			isLoading = false;
		}
	};

	/** Navigates back to login. */
	const handleBack = async () => {
		localStorage.removeItem(STORAGE_KEY_STATE);
		sessionStorage.removeItem(STORAGE_KEY_EMAIL);
		await goto(resolve('/login'));
	};

	onMount(async () => {
		const storedEmail = sessionStorage.getItem(STORAGE_KEY_EMAIL);

		if (!storedEmail) {
			void goto(resolve('/login'));
			return;
		}

		email = storedEmail;
		view = 'otp';
		await tick();
		otpInput?.focus();
	});
</script>

<svelte:head>
	<title>BeeBuzz | Verify Code</title>
</svelte:head>

<div class="bb-page">
	<div class="w-full max-w-sm">
		<!-- Header -->
		<div class="text-center mb-8">
			<div class="flex flex-col items-center justify-center">
				<BeeBuzzLogo variant="img" class="w-16 h-16 mb-2" />
				<BeeBuzzLogo variant="text" class="w-48 h-12" />
			</div>
			<h1 class="text-xl font-bold text-base-content mt-4">Check your email</h1>
			{#if email}
				{#if isSaasMode}
					<p class="text-sm text-base-content/70 mt-2">
						For privacy, we can't confirm approval status. If approved, we sent a 6-digit code to <strong
							>{email}</strong
						>.
					</p>
				{:else}
					<p class="text-sm text-base-content/70 mt-2">
						We sent a 6-digit code to <strong>{email}</strong>.
					</p>
				{/if}
			{/if}
		</div>

		{#if view === 'loading'}
			<div
				class="card bg-base-100 shadow-md border border-base-300 p-12 flex flex-col items-center justify-center"
			>
				<span class="loading loading-spinner loading-lg text-primary"></span>
				<p class="mt-4 text-sm text-base-content/70">Processing...</p>
			</div>
		{:else if view === 'otp'}
			<div class="card bg-base-100 shadow-md border border-base-300 p-6">
				{#if error !== undefined}
					<div class="alert alert-error mb-4" role="alert">
						<span>{error}</span>
					</div>
				{/if}

				<form class="space-y-4" onsubmit={handleVerifyOtp}>
					<div>
						<label for="code" class="block text-sm font-semibold text-base-content mb-3">
							Enter the code
						</label>
						<div class="relative">
							<input
								type="text"
								id="code"
								bind:this={otpInput}
								pattern="[0-9]*"
								maxlength={OTP_LENGTH}
								autocomplete="one-time-code"
								inputmode="numeric"
								enterkeyhint="done"
								class="absolute inset-0 h-full w-full cursor-text opacity-0"
								bind:value={verificationCode}
								oninput={handleOtpInput}
								disabled={isLoading}
								aria-label="One-time code"
								required
							/>

							<div class="grid grid-cols-6 gap-2" aria-hidden="true">
								{#each Array.from({ length: OTP_LENGTH }, (_, index) => index) as index (index)}
									<div
										class={[
											'flex aspect-square items-center justify-center rounded-xl border bg-base-200 text-2xl font-bold tabular-nums transition',
											index === verificationCode.length && verificationCode.length < OTP_LENGTH
												? 'border-primary ring-2 ring-primary/20'
												: 'border-base-300',
											verificationCode[index]
												? 'text-base-content shadow-sm'
												: 'text-base-content/30'
										].join(' ')}
									>
										{verificationCode[index] ?? ''}
									</div>
								{/each}
							</div>
						</div>
					</div>

					<button
						type="submit"
						class="btn btn-primary mt-6 w-full"
						disabled={isLoading || verificationCode.length !== OTP_LENGTH}
					>
						{#if isLoading}
							<span class="loading loading-spinner loading-sm"></span>
							Verifying...
						{:else}
							Verify Code
						{/if}
					</button>
				</form>
			</div>

			<div class="mt-4 p-4 bg-base-200 rounded-lg">
				<p class="text-sm font-semibold text-base-content mb-2">No code yet?</p>
				<ul class="text-xs text-base-content/70 space-y-1 list-disc list-inside">
					<li>Codes usually arrive within 1–2 minutes.</li>
					<li>Check your spam or promotions folder.</li>
					{#if isSaasMode}
						<li>During private beta, codes are only sent to approved emails.</li>
						<li>
							If nothing arrives, your access may still be pending — you'll receive an email when
							approved.
						</li>
					{/if}
				</ul>
			</div>

			<button
				type="button"
				class="w-full mt-4 text-sm text-base-content/70 hover:text-base-content transition-colors flex items-center justify-center gap-2"
				disabled={isLoading}
				onclick={handleBack}
			>
				<ArrowLeft size={16} />
				Use a different email
			</button>
		{/if}
	</div>
</div>
