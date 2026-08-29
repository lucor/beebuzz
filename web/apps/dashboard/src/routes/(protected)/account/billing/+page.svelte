<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { accountApi, type AuthUser, type PlanUsage } from '@beebuzz/shared/api';
	import { me } from '@beebuzz/shared/services/account';
	import { auth, toast } from '@beebuzz/shared/stores';
	import { ApiError } from '@beebuzz/shared/errors';
	import PlanUsageCard from '$lib/components/PlanUsageCard.svelte';
	import { CircleCheck, CreditCard, ExternalLink, Loader, RefreshCw } from '@lucide/svelte';

	const CHECKOUT_STATUS_PARAM = 'checkout';
	const CHECKOUT_SUCCESS_VALUE = 'success';
	const CONFIRMATION_POLL_INTERVAL_MS = 3000;
	const CONFIRMATION_MAX_ATTEMPTS = 20;

	let user = $state<AuthUser | null>(null);
	let planUsage = $state<PlanUsage | null>(null);
	let loading = $state(true);
	let actionLoading = $state<'checkout' | 'portal' | null>(null);
	let refreshLoading = $state(false);
	let confirmingCheckout = $state(false);
	let confirmationTimedOut = $state(false);

	let planLabel = $derived(user?.plan === 'hosted' ? 'Hosted' : 'Free');
	let planDescription = $derived(getPlanDescription(user));
	let periodLabel = $derived(getPeriodLabel(user));
	let billingActionLabel = $derived(
		user?.subscription_status === 'past_due' ? 'Update payment' : 'Manage billing'
	);
	let expiresAtLabel = $derived(formatDate(user?.plan_expires_at));
	let isCheckoutReturn = $derived(
		page.url.searchParams.get(CHECKOUT_STATUS_PARAM) === CHECKOUT_SUCCESS_VALUE
	);
	let shouldShowCheckoutSuccess = $derived(isCheckoutReturn && user?.plan === 'hosted');
	let shouldShowPastDueNotice = $derived(
		user?.plan === 'hosted' && user.subscription_status === 'past_due'
	);

	onMount(() => {
		if (auth.user?.is_admin) {
			void goto(resolve('/account/overview'), { replaceState: true });
			return;
		}

		let stopped = false;
		let timeout: ReturnType<typeof setTimeout> | undefined;

		const pollConfirmation = async (attempt: number) => {
			if (stopped) return;

			try {
				const [currentUser] = await Promise.all([loadBillingDetails(), loadPlanUsage()]);
				if (currentUser.plan === 'hosted') {
					confirmingCheckout = false;
					confirmationTimedOut = false;
					return;
				}
			} catch {
				// The next poll can still succeed if the provider webhook arrives late.
			}

			if (attempt >= CONFIRMATION_MAX_ATTEMPTS) {
				confirmingCheckout = false;
				confirmationTimedOut = true;
				return;
			}

			timeout = setTimeout(() => {
				void pollConfirmation(attempt + 1);
			}, CONFIRMATION_POLL_INTERVAL_MS);
		};

		const loadInitialBillingDetails = async () => {
			try {
				const [currentUser] = await Promise.all([loadBillingDetails(), loadPlanUsage()]);
				if (isCheckoutReturn && currentUser.plan !== 'hosted') {
					confirmingCheckout = true;
					void pollConfirmation(1);
				}
			} catch (err) {
				toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load billing details');
			} finally {
				loading = false;
			}
		};

		void loadInitialBillingDetails();

		return () => {
			stopped = true;
			if (timeout) clearTimeout(timeout);
		};
	});

	async function loadBillingDetails(): Promise<AuthUser> {
		const currentUser = await me();
		user = currentUser;
		return currentUser;
	}

	async function loadPlanUsage(): Promise<PlanUsage | null> {
		try {
			const currentPlanUsage = await accountApi.getPlanUsage();
			planUsage = currentPlanUsage;
			return currentPlanUsage;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load plan usage');
			return null;
		}
	}

	function formatDate(value: string | null | undefined): string | null {
		if (!value) return null;

		return new Intl.DateTimeFormat(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		}).format(new Date(value));
	}

	function getPlanDescription(currentUser: AuthUser | null): string {
		if (!currentUser || currentUser.plan === 'free') {
			if (
				currentUser?.subscription_status === 'canceled' ||
				currentUser?.subscription_status === 'expired'
			) {
				return 'Your Hosted plan has ended. Free includes 50 messages/day and 500 messages/month.';
			}
			return '50 messages/day and 500 messages/month.';
		}

		switch (currentUser.subscription_status) {
			case 'scheduled_cancel':
				return '100,000 messages/month fair use until the current billing period ends.';
			case 'past_due':
				return '100,000 messages/month fair use during the payment grace period.';
			default:
				return '100,000 messages/month fair use.';
		}
	}

	function getPeriodLabel(currentUser: AuthUser | null): string {
		if (!currentUser || currentUser.plan === 'free') return 'Plan date';

		switch (currentUser.subscription_status) {
			case 'scheduled_cancel':
				return 'Active until';
			case 'past_due':
				return 'Access until';
			default:
				return 'Renews or expires';
		}
	}

	async function handleRefreshStatus() {
		refreshLoading = true;
		try {
			await Promise.all([loadBillingDetails(), loadPlanUsage()]);
			confirmationTimedOut = false;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to refresh billing status');
		} finally {
			refreshLoading = false;
		}
	}

	async function handleUpgrade() {
		actionLoading = 'checkout';
		try {
			const checkout = await accountApi.createBillingCheckout();
			window.location.assign(checkout.checkout_url);
		} catch (err) {
			actionLoading = null;
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to start checkout');
		}
	}

	async function handleManageBilling() {
		try {
			const portal = await accountApi.createBillingPortal();
			window.open(portal.portal_url, '_blank', 'noopener,noreferrer');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to open billing portal');
		}
	}
</script>

<div class="space-y-6">
	<div class="max-w-3xl">
		<h1 class="text-3xl font-bold text-base-content">Plan & Billing</h1>
		<p class="mt-2 text-base-content/70">View your plan usage and manage hosted billing.</p>
	</div>

	{#if confirmingCheckout}
		<div class="alert alert-info">
			<Loader size={20} class="animate-spin" />
			<div>
				<div class="font-semibold">Confirming payment</div>
				<div class="text-sm">
					Checkout completed. Waiting for Creem to confirm your subscription.
				</div>
			</div>
		</div>
	{:else if confirmationTimedOut}
		<div class="alert alert-warning">
			<RefreshCw size={20} />
			<div class="flex flex-1 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div>
					<div class="font-semibold">Payment confirmation is still pending</div>
					<div class="text-sm">
						Refresh the billing status after the provider webhook is delivered.
					</div>
				</div>
				<button
					type="button"
					class="btn btn-sm"
					onclick={handleRefreshStatus}
					disabled={refreshLoading}
				>
					{#if refreshLoading}
						<Loader size={16} class="animate-spin" />
					{:else}
						<RefreshCw size={16} />
					{/if}
					Refresh status
				</button>
			</div>
		</div>
	{:else if shouldShowCheckoutSuccess}
		<div class="alert alert-success">
			<CircleCheck size={20} />
			<div>
				<div class="font-semibold">Hosted is active</div>
				<div class="text-sm">Your account plan has been updated.</div>
			</div>
		</div>
	{/if}

	{#if shouldShowPastDueNotice}
		<div class="alert alert-warning">
			<CreditCard size={20} />
			<div>
				<div class="font-semibold">Payment issue</div>
				<div class="text-sm">
					Your Hosted access continues during the grace period. Update payment details in the
					billing portal.
				</div>
			</div>
		</div>
	{/if}

	{#if loading}
		<div class="py-8 text-center">
			<Loader size={32} class="mx-auto mb-2 animate-spin text-primary" />
			<p class="text-base-content/70">Loading plan and billing...</p>
		</div>
	{:else if user}
		<div
			class={user.plan === 'free'
				? 'grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(20rem,1fr)]'
				: 'max-w-3xl'}
		>
			<div class="card border border-base-300 bg-base-200">
				<div class="card-body">
					<div class="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
						<div class="flex items-start gap-4">
							<div
								class="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10 text-primary"
							>
								<CreditCard size={22} />
							</div>
							<div>
								<h2 class="card-title text-lg text-base-content">Current plan</h2>
								<p class="mt-1 text-sm text-base-content/65">{planDescription}</p>
							</div>
						</div>

						<span class="badge badge-primary badge-lg font-semibold">{planLabel}</span>
					</div>

					{#if user.plan === 'hosted' && expiresAtLabel}
						<div class="mt-6 rounded-lg border border-base-300 bg-base-100 p-4">
							<div class="text-sm font-semibold text-base-content/70">{periodLabel}</div>
							<div class="mt-2 text-xl font-bold text-base-content">
								{expiresAtLabel}
							</div>
						</div>
					{/if}

					{#if planUsage}
						<PlanUsageCard {planUsage} embedded />
					{/if}

					{#if user.plan === 'hosted'}
						<div class="mt-6 flex flex-col gap-3 sm:flex-row">
							<button type="button" class="btn btn-primary" onclick={handleManageBilling}>
								<ExternalLink size={18} />
								{billingActionLabel}
							</button>
						</div>
					{/if}
				</div>
			</div>

			{#if user.plan === 'free'}
				<div class="card border border-primary/25 bg-primary/5">
					<div class="card-body">
						<div>
							<p class="text-sm font-semibold uppercase tracking-wide text-primary">Hosted</p>
							<h2 class="mt-2 text-2xl font-bold text-base-content">BeeBuzz, managed for you.</h2>
							<p class="mt-2 text-sm text-base-content/65">
								Everything included, without running your own server.
							</p>
						</div>

						<div class="mt-2">
							<span class="text-4xl font-bold text-base-content">29 EUR</span>
							<span class="text-sm text-base-content/60">/year</span>
						</div>

						<div class="space-y-3 text-sm text-base-content/75">
							<div class="flex gap-2">
								<CircleCheck size={18} class="mt-0.5 shrink-0 text-primary" />
								<span>No server to run</span>
							</div>
							<div class="flex gap-2">
								<CircleCheck size={18} class="mt-0.5 shrink-0 text-primary" />
								<span>Up to 100,000 messages sent per month</span>
							</div>
							<div class="flex gap-2">
								<CircleCheck size={18} class="mt-0.5 shrink-0 text-primary" />
								<span>No daily message limit</span>
							</div>
						</div>

						<button
							type="button"
							class="btn btn-primary mt-2 w-full"
							onclick={handleUpgrade}
							disabled={actionLoading !== null}
						>
							{#if actionLoading === 'checkout'}
								<Loader size={18} class="animate-spin" />
							{:else}
								<ExternalLink size={18} />
							{/if}
							Upgrade to Hosted
						</button>

						<div class="border-t border-primary/20 pt-4 text-xs text-base-content/55">
							Checkout and subscription management are securely handled by Creem. BeeBuzz does not
							store your billing email or payment details.
						</div>
					</div>
				</div>
			{/if}
		</div>
	{:else}
		<div class="py-8 text-center">
			<p class="text-base-content/70">Billing details are unavailable.</p>
		</div>
	{/if}
</div>
