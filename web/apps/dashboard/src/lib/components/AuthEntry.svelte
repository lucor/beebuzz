<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { LOGIN_HONEYPOT_FIELD_NAME } from '@beebuzz/shared/constants/auth';
	import { toast } from '@beebuzz/shared/stores';
	import { login } from '@beebuzz/shared/services/auth';
	import { ApiError, isInlineError } from '@beebuzz/shared/errors';
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { PUBLIC_SITE_URL } from '@beebuzz/shared/config';

	interface Props {
		redirectAfterSubmit?: '/auth/verify' | '/account';
	}

	let { redirectAfterSubmit = '/auth/verify' }: Props = $props();

	let email = $state('');
	let referralCode = $state('');
	let isLoading = $state(false);
	let error = $state<string | undefined>(undefined);

	/** Handles login form submission. */
	const handleRequest = async (event: SubmitEvent) => {
		event.preventDefault();
		isLoading = true;
		error = undefined;

		try {
			await login(email, referralCode || undefined);
			await goto(resolve(redirectAfterSubmit));
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
				<BeeBuzzLogo variant="full" class="h-16 w-auto mb-2" />
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

			<form class="space-y-4" onsubmit={handleRequest}>
				<!-- Honeypot field to reduce bot signups -->
				<div class="contents" aria-hidden="true">
					<input
						type="text"
						id={LOGIN_HONEYPOT_FIELD_NAME}
						name={LOGIN_HONEYPOT_FIELD_NAME}
						tabindex="-1"
						autocomplete="off"
						class="absolute -left-[10000px] w-px h-px overflow-hidden opacity-0 pointer-events-none"
						bind:value={referralCode}
					/>
				</div>

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
						We only use your email for sign-in and account notifications.
					</p>
				</div>

				<button type="submit" class="btn btn-primary w-full" disabled={isLoading || !email.trim()}>
					{#if isLoading}
						<span class="loading loading-spinner loading-sm"></span>
						Continuing...
					{:else}
						Continue
					{/if}
				</button>

				<p class="text-xs text-base-content/60">
					By continuing, you agree to the
					<a
						href={`${PUBLIC_SITE_URL}/policies#terms`}
						target="_blank"
						rel="noopener noreferrer"
						class="underline hover:text-base-content"
					>
						Terms of Service
					</a>
					and confirm that you read the
					<a
						href={`${PUBLIC_SITE_URL}/policies#privacy`}
						target="_blank"
						rel="noopener noreferrer"
						class="underline hover:text-base-content"
					>
						Privacy Policy
					</a>
					.
				</p>
			</form>
		</div>
	</div>
</div>
