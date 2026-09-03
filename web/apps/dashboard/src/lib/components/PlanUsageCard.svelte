<script lang="ts">
	import { resolve } from '$app/paths';
	import type { PlanLimitUsage, PlanUsage } from '@beebuzz/shared/api';
	import { CreditCard } from '@lucide/svelte';

	type Props = {
		planUsage: PlanUsage;
		showUpgrade?: boolean;
		embedded?: boolean;
	};

	let { planUsage, showUpgrade = false, embedded = false }: Props = $props();

	const BILLING_URL = resolve('/account/billing');
	const PLAN_WARNING_THRESHOLD = 0.8;

	function formatQuotaNumber(n: number): string {
		return new Intl.NumberFormat('en-US').format(n);
	}

	function quotaPercent(quota: PlanLimitUsage): number {
		if (quota.limit <= 0) return 0;
		return Math.min(quota.used / quota.limit, 1);
	}

	function quotaPercentLabel(quota: PlanLimitUsage): string {
		return Math.round(quotaPercent(quota) * 100) + '%';
	}

	function quotaRemaining(quota: PlanLimitUsage): number {
		return Math.max(quota.limit - quota.used, 0);
	}

	function quotaProgressClass(quota: PlanLimitUsage): string {
		const percent = quotaPercent(quota);
		if (percent >= 0.95) return 'bg-error';
		if (percent >= PLAN_WARNING_THRESHOLD) return 'bg-warning';
		return 'bg-primary';
	}

	function quotaFillWidth(quota: PlanLimitUsage): string {
		const percent = quotaPercent(quota) * 100;
		if (percent <= 0) return '0%';
		return `${Math.max(percent, 2)}%`;
	}

	function quotaStatusLabel(label: string, quota: PlanLimitUsage): string {
		const percent = quotaPercent(quota);
		if (percent >= 1) return `${label} limit reached`;
		if (percent >= 0.95) return `Almost at ${label.toLowerCase()} limit`;
		if (percent >= PLAN_WARNING_THRESHOLD) return `Approaching ${label.toLowerCase()} limit`;
		return label;
	}

	function monthlyStatusLabel(plan: PlanUsage['plan'], quota: PlanLimitUsage): string {
		if (plan !== 'hosted') {
			return quotaStatusLabel('This month', quota);
		}

		const percent = quotaPercent(quota);
		if (percent >= 1) return 'Fair-use threshold reached';
		if (percent >= 0.95) return 'Almost at fair-use threshold';
		if (percent >= PLAN_WARNING_THRESHOLD) return 'Approaching fair-use threshold';
		return 'Fair use this calendar month';
	}

	function monthlyUsedLabel(quota: PlanLimitUsage): string {
		return `${formatQuotaNumber(quota.used)} / ${formatQuotaNumber(quota.limit)} messages sent`;
	}

	function formatDailyReset(value: string): string {
		const reset = new Date(value).getTime();
		const diffMs = reset - Date.now();
		if (diffMs <= 0) return 'soon';

		const totalMinutes = Math.ceil(diffMs / 60_000);
		const hours = Math.floor(totalMinutes / 60);
		const minutes = totalMinutes % 60;
		if (hours <= 0) return `in ${minutes}m`;
		if (minutes === 0) return `in ${hours}h`;
		return `in ${hours}h ${minutes}m`;
	}

	function formatMonthlyReset(value: string): string {
		return new Intl.DateTimeFormat('en-GB', {
			day: 'numeric',
			month: 'short',
			year: 'numeric',
			timeZone: 'UTC'
		}).format(new Date(value));
	}

	function monthlyResetLabel(plan: PlanUsage['plan'], value: string): string {
		const date = formatMonthlyReset(value);
		return plan === 'hosted' ? `Fair use resets on ${date}` : `Resets on ${date}`;
	}
</script>

{#snippet usageContent()}
	<div class={embedded ? 'mt-6 border-t border-base-300 pt-6' : ''}>
		<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<div>
				{#if embedded}
					<h3 class="text-base font-semibold text-base-content">Included usage</h3>
				{:else}
					<h3 class="text-lg font-semibold text-base-content">Plan usage</h3>
					<p class="text-sm text-base-content/60">
						{planUsage.plan === 'hosted' ? 'Hosted plan' : 'Free plan'}
					</p>
				{/if}
			</div>
			{#if showUpgrade}
				<a href={BILLING_URL} class="btn btn-primary btn-sm self-start sm:self-auto">
					<CreditCard size={16} />
					Manage plan
				</a>
			{/if}
		</div>

		<div class="mt-5 grid gap-5 {planUsage.daily ? 'lg:grid-cols-2' : embedded ? 'max-w-xl' : ''}">
			{#if planUsage.daily}
				{@const remaining = quotaRemaining(planUsage.daily)}
				<div class="space-y-3">
					<div>
						<p class="font-medium text-base-content">
							{quotaStatusLabel('Today', planUsage.daily)}
						</p>
						<p class="mt-2 text-2xl font-bold text-base-content">
							{formatQuotaNumber(remaining)} remaining
						</p>
						<p class="mt-1 text-sm text-base-content/65">
							{formatQuotaNumber(planUsage.daily.used)} / {formatQuotaNumber(planUsage.daily.limit)}
							messages sent
						</p>
					</div>
					<div
						class="h-2 overflow-hidden rounded-full bg-base-300"
						role="progressbar"
						aria-label="Daily plan usage"
						aria-valuemin="0"
						aria-valuemax="100"
						aria-valuenow={Math.round(quotaPercent(planUsage.daily) * 100)}
					>
						<div
							class="h-full rounded-full {quotaProgressClass(planUsage.daily)}"
							style={`width: ${quotaFillWidth(planUsage.daily)}`}
						></div>
					</div>
					<p class="text-xs text-base-content/60">
						{quotaPercentLabel(planUsage.daily)} used. Resets {formatDailyReset(
							planUsage.daily.resets_at
						)}.
					</p>
				</div>
			{/if}

			{#if planUsage.monthly}
				{@const remaining = quotaRemaining(planUsage.monthly)}
				<div class="space-y-3">
					<div>
						<p class="font-medium text-base-content">
							{monthlyStatusLabel(planUsage.plan, planUsage.monthly)}
						</p>
						{#if planUsage.plan === 'free'}
							<p class="mt-2 text-2xl font-bold text-base-content">
								{formatQuotaNumber(remaining)} remaining
							</p>
						{/if}
						<p class="mt-1 text-sm text-base-content/65">
							{monthlyUsedLabel(planUsage.monthly)}
						</p>
					</div>
					<div
						class="h-2 overflow-hidden rounded-full bg-base-300"
						role="progressbar"
						aria-label="Monthly plan usage"
						aria-valuemin="0"
						aria-valuemax="100"
						aria-valuenow={Math.round(quotaPercent(planUsage.monthly) * 100)}
					>
						<div
							class="h-full rounded-full {quotaProgressClass(planUsage.monthly)}"
							style={`width: ${quotaFillWidth(planUsage.monthly)}`}
						></div>
					</div>
					<p class="text-xs text-base-content/60">
						{quotaPercentLabel(planUsage.monthly)} used. {monthlyResetLabel(
							planUsage.plan,
							planUsage.monthly.resets_at
						)}.
					</p>
				</div>
			{/if}
		</div>
	</div>
{/snippet}

{#if embedded}
	{@render usageContent()}
{:else}
	<div class="card border border-base-300 bg-base-200">
		<div class="card-body p-4 sm:p-5">
			{@render usageContent()}
		</div>
	</div>
{/if}
