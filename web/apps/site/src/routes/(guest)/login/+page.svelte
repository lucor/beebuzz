<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { toast } from '@beebuzz/shared/stores';
	import { login } from '@beebuzz/shared/services/auth';
	import { ApiError, isInlineError } from '@beebuzz/shared/errors';
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { Info } from '@lucide/svelte';
	import { isSaasMode } from '$lib/config/deployment';

	let email = $state('');
	let reason = $state('');
	let isLoading = $state(false);
	let error = $state<string | undefined>(undefined);

	/** Handles login form submission. */
	const handleRequest = async (event: SubmitEvent) => {
		event.preventDefault();
		isLoading = true;
		error = undefined;

		try {
			await login(email, reason.trim() || undefined);
			await goto(resolve('/verify'));
		} catch (err) {
			if (err instanceof ApiError && isInlineError(err.code)) {
				error = err.userMessage;
			} else {
				error = undefined;
				toast.error(err instanceof ApiError ? err.userMessage : 'Request failed');
			}
		} finally {
			isLoading = false;
		}
	};
</script>

<svelte:head>
	<title>BeeBuzz | Sign In</title>
</svelte:head>

<div class="bb-page">
	<div class="w-full max-w-sm">
		<!-- Header -->
		<div class="text-center mb-8">
			<a href={resolve('/')} class="inline-flex flex-col items-center justify-center">
				<BeeBuzzLogo variant="img" class="w-16 h-16 mb-2" />
				<BeeBuzzLogo variant="text" class="w-48 h-12" />
			</a>
			<h1 class="text-xl font-bold text-base-content mt-4">Continue with your email</h1>
		</div>

		<!-- Login Card -->
		<div class="card bg-base-100 shadow-md border border-base-300 p-6">
			{#if error !== undefined}
				<div class="alert alert-error mb-4" role="alert">
					<span>{error}</span>
				</div>
			{/if}

			{#if isSaasMode}
				<div class="mb-6 p-4 bg-info/10 border border-info/30 rounded-lg">
					<div class="flex items-center gap-2 mb-2">
						<Info class="w-4 h-4 text-info" />
						<p class="text-sm text-base-content font-semibold">BeeBuzz is in private beta</p>
					</div>
					<p class="text-sm text-base-content/70">
						If your email is approved, we'll send a 6-digit sign-in code. Otherwise, this serves as
						your beta access request and you'll receive an email when approved.
					</p>
				</div>
			{/if}

			<form class="space-y-4" onsubmit={handleRequest}>
				<div>
					<label for="email" class="block text-sm font-semibold text-base-content mb-2">
						Email Address
					</label>
					<input
						type="email"
						id="email"
						placeholder="you@example.com"
						class="input input-bordered w-full"
						bind:value={email}
						required
						disabled={isLoading}
						aria-label="Email address"
					/>
					<p class="text-xs text-base-content/50 mt-1">
						We only use your email for sign-in and beta updates.
					</p>
				</div>

				<div>
					<label for="reason" class="block text-sm font-semibold text-base-content mb-2">
						Requesting beta access? Tell us what you'd use BeeBuzz for
						<span class="font-normal text-base-content/50">(optional)</span>
					</label>
					<textarea
						id="reason"
						placeholder="e.g. I want to get alerts from my home server, monitor my CI pipelines..."
						class="textarea textarea-bordered w-full"
						rows="3"
						bind:value={reason}
						disabled={isLoading}
						aria-label="Reason for wanting to use BeeBuzz"
					></textarea>
				</div>

				<button type="submit" class="btn btn-primary w-full" disabled={isLoading || !email.trim()}>
					{#if isLoading}
						<span class="loading loading-spinner loading-sm"></span>
						Continuing...
					{:else}
						Continue
					{/if}
				</button>

				{#if isSaasMode}
					<p class="text-xs text-base-content/60">
						By continuing, you agree to the
						<a href={`${resolve('/policies')}#terms`} class="underline hover:text-base-content">
							Terms of Service
						</a>
						and confirm that you read the
						<a href={`${resolve('/policies')}#privacy`} class="underline hover:text-base-content">
							Privacy Policy
						</a>
						.
					</p>
				{/if}
			</form>
		</div>
	</div>
</div>
