<script lang="ts">
	import { resolve } from '$app/paths';
	import { auth, toast } from '@beebuzz/shared/stores';
	import { DashboardChart } from '@beebuzz/shared/components';
	import {
		accountApi,
		type AccountUsage,
		type ApiToken,
		type Device,
		type PlanLimitUsage,
		type PlanUsage
	} from '@beebuzz/shared/api';
	import PlanUsageCard from '$lib/components/PlanUsageCard.svelte';
	import {
		buildDashboardChartSeries,
		type DashboardTimeRange
	} from '@beebuzz/shared/utils/dashboard-chart';
	import { ApiError } from '@beebuzz/shared/errors';
	import { onMount } from 'svelte';
	import {
		BellRing,
		CircleAlert,
		Check,
		CircleCheckBig,
		CodeXml,
		CreditCard,
		Gauge,
		Key,
		Loader,
		LockKeyhole,
		Paperclip,
		Plus,
		Send,
		Terminal,
		Webhook,
		WifiOff
	} from '@lucide/svelte';

	const DOCS_QUICKSTART_URL = 'https://docs.beebuzz.app/quickstart/';
	const BILLING_URL = resolve('/account/billing');
	const PLAN_WARNING_THRESHOLD = 0.8;

	type TimeRange = DashboardTimeRange;
	type PlanAlert = {
		title: string;
		body: string;
		tone: string;
		showUpgrade: boolean;
	};

	let usage: AccountUsage | null = $state(null);
	let loading = $state(true);
	let planUsage = $state<PlanUsage | null>(null);
	let selectedRange: TimeRange = $state(7);
	let devices = $state<Device[]>([]);
	let tokens = $state<ApiToken[]>([]);
	let onboardingLoading = $state(true);
	let isAdmin = $derived(auth.user?.is_admin === true);

	const ranges: { value: TimeRange; label: string }[] = [
		{ value: 7, label: '7d' },
		{ value: 30, label: '30d' },
		{ value: 90, label: '90d' },
		{ value: 0, label: 'All time' }
	];

	onMount(async () => {
		await Promise.all([loadUsage(), loadPlanUsage(), loadDevices(), loadTokens()]);
		onboardingLoading = false;
	});

	/** Fetches account usage data for the selected time range. */
	async function loadUsage() {
		loading = true;
		try {
			usage = await accountApi.getUsage(selectedRange);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load usage data');
		} finally {
			loading = false;
		}
	}

	/** Fetches current plan quota state for operational visibility. */
	async function loadPlanUsage() {
		if (isAdmin) {
			planUsage = null;
			return;
		}

		try {
			planUsage = await accountApi.getPlanUsage();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load plan usage');
		}
	}

	/** Fetches paired devices so onboarding hints can be shown accurately. */
	async function loadDevices() {
		try {
			devices = await accountApi.listDevices();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load devices');
		}
	}

	/** Fetches API tokens so onboarding hints can be shown accurately. */
	async function loadTokens() {
		try {
			tokens = await accountApi.listApiTokens();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load API tokens');
		}
	}

	/** Handles time range button click. */
	async function handleRangeChange(range: TimeRange) {
		selectedRange = range;
		await loadUsage();
	}

	/** Formats large numbers with K/M suffix. */
	function formatNumber(n: number): string {
		if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
		if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
		return n.toString();
	}

	/** Formats exact quota numbers. */
	function formatQuotaNumber(n: number): string {
		return new Intl.NumberFormat('en-US').format(n);
	}

	/** Formats byte values with human-readable units. */
	function formatBytes(n: number): string {
		if (n >= 1_000_000_000) return (n / 1_000_000_000).toFixed(1) + ' GB';
		if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + ' MB';
		if (n >= 1_000) return (n / 1_000).toFixed(1) + ' KB';
		return n + ' B';
	}

	/** Formats ratios as percentages. */
	function formatPercent(n: number): string {
		return (n * 100).toFixed(1) + '%';
	}

	function quotaPercent(quota: PlanLimitUsage): number {
		if (quota.limit <= 0) return 0;
		return Math.min(quota.used / quota.limit, 1);
	}

	function quotaRemaining(quota: PlanLimitUsage): number {
		return Math.max(quota.limit - quota.used, 0);
	}

	function compactMonthlyResetLabel(value: string): string {
		return `resets on ${formatMonthlyReset(value)}`;
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

	function planAlert(currentPlanUsage: PlanUsage | null): PlanAlert | null {
		if (!currentPlanUsage) {
			return null;
		}

		if (currentPlanUsage.daily && quotaPercent(currentPlanUsage.daily) >= PLAN_WARNING_THRESHOLD) {
			const remaining = quotaRemaining(currentPlanUsage.daily);
			const reached = remaining === 0;
			return {
				title: reached ? 'Daily limit reached' : 'You are close to your daily limit',
				body: reached
					? `You can send messages again ${formatDailyReset(currentPlanUsage.daily.resets_at)}.`
					: `${formatQuotaNumber(currentPlanUsage.daily.used)} / ${formatQuotaNumber(
							currentPlanUsage.daily.limit
							)} messages sent | ${formatQuotaNumber(remaining)} remaining | resets ${formatDailyReset(
							currentPlanUsage.daily.resets_at
						)}`,
				tone: reached ? 'alert-error' : 'alert-warning',
				showUpgrade: true
			};
		}

		if (
			!currentPlanUsage.monthly ||
			quotaPercent(currentPlanUsage.monthly) < PLAN_WARNING_THRESHOLD
		) {
			return null;
		}

		const remaining = quotaRemaining(currentPlanUsage.monthly);
		if (currentPlanUsage.plan === 'hosted') {
			const reached = remaining === 0;
			return {
				title: reached ? 'Fair-use threshold reached' : 'Approaching fair-use threshold',
				body: `${formatQuotaNumber(currentPlanUsage.monthly.used)} / ${formatQuotaNumber(
					currentPlanUsage.monthly.limit
				)} messages sent | ${compactMonthlyResetLabel(currentPlanUsage.monthly.resets_at)}`,
				tone: reached ? 'alert-warning' : 'alert-info',
				showUpgrade: false
			};
		}

		const reached = remaining === 0;
		return {
			title: reached ? 'Monthly limit reached' : 'You are close to your monthly limit',
			body: `${formatQuotaNumber(currentPlanUsage.monthly.used)} / ${formatQuotaNumber(
				currentPlanUsage.monthly.limit
			)} messages sent | ${formatQuotaNumber(remaining)} remaining | ${compactMonthlyResetLabel(
				currentPlanUsage.monthly.resets_at
			)}`,
			tone: reached ? 'alert-error' : 'alert-warning',
			showUpgrade: true
		};
	}

	let totals = $derived.by(() => {
		if (!usage) {
			return {
				created: 0,
				attempts: 0,
				delivered: 0,
				failed: 0,
				devicesLost: 0,
				attachments: 0,
				attachmentsBytes: 0,
				serverTrusted: 0,
				e2e: 0,
				sourcesCli: 0,
				sourcesWebhook: 0,
				sourcesApi: 0,
				sourcesInternal: 0
			};
		}

		return usage.data.reduce(
			(acc, day) => ({
				created: acc.created + day.notifications_created,
				attempts: acc.attempts + day.delivery_attempts,
				delivered: acc.delivered + day.deliveries_succeeded,
				failed: acc.failed + day.deliveries_failed,
				devicesLost: acc.devicesLost + day.devices_lost,
				attachments: acc.attachments + day.notifications_with_attachment,
				attachmentsBytes: acc.attachmentsBytes + day.attachment_bytes_total,
				serverTrusted: acc.serverTrusted + day.notifications_server_trusted,
				e2e: acc.e2e + day.notifications_e2e,
				sourcesCli: acc.sourcesCli + day.sources_cli,
				sourcesWebhook: acc.sourcesWebhook + day.sources_webhook,
				sourcesApi: acc.sourcesApi + day.sources_api,
				sourcesInternal: acc.sourcesInternal + day.sources_internal
			}),
			{
				created: 0,
				attempts: 0,
				delivered: 0,
				failed: 0,
				devicesLost: 0,
				attachments: 0,
				attachmentsBytes: 0,
				serverTrusted: 0,
				e2e: 0,
				sourcesCli: 0,
				sourcesWebhook: 0,
				sourcesApi: 0,
				sourcesInternal: 0
			}
		);
	});

	let chartSeries = $derived.by(() => {
		if (!usage) {
			return buildDashboardChartSeries([], selectedRange);
		}

		return buildDashboardChartSeries(usage.data, selectedRange);
	});

	let deliverySuccessRate = $derived.by(() => {
		if (totals.attempts === 0) return 0;
		return totals.delivered / totals.attempts;
	});

	let attachmentShare = $derived.by(() => {
		if (totals.created === 0) return 0;
		return totals.attachments / totals.created;
	});

	let totalSources = $derived(
		totals.sourcesCli + totals.sourcesWebhook + totals.sourcesApi + totals.sourcesInternal
	);
	let trustedShare = $derived.by(() => {
		if (totals.created === 0) return 0;
		return totals.serverTrusted / totals.created;
	});
	let e2eShare = $derived.by(() => {
		if (totals.created === 0) return 0;
		return totals.e2e / totals.created;
	});

	let reliabilityTone = $derived.by(() => {
		if (totals.attempts === 0) {
			return 'text-base-content/70';
		}

		if (deliverySuccessRate >= 0.98) {
			return 'text-success';
		}

		if (deliverySuccessRate >= 0.9) {
			return 'text-warning';
		}

		return 'text-error';
	});

	let sourceBreakdown = $derived.by(() => {
		return [
			{
				key: 'cli',
				label: 'CLI',
				value: totals.sourcesCli,
				share: totalSources === 0 ? 0 : totals.sourcesCli / totalSources,
				icon: Terminal,
				barClass: 'bg-sky-500/75'
			},
			{
				key: 'webhook',
				label: 'Webhook',
				value: totals.sourcesWebhook,
				share: totalSources === 0 ? 0 : totals.sourcesWebhook / totalSources,
				icon: Webhook,
				barClass: 'bg-cyan-600/70'
			},
			{
				key: 'api',
				label: 'API',
				value: totals.sourcesApi,
				share: totalSources === 0 ? 0 : totals.sourcesApi / totalSources,
				icon: CodeXml,
				barClass: 'bg-indigo-500/70'
			},
			{
				key: 'internal',
				label: 'Internal',
				value: totals.sourcesInternal,
				share: totalSources === 0 ? 0 : totals.sourcesInternal / totalSources,
				icon: BellRing,
				barClass: 'bg-amber-500/75'
			}
		];
	});

	let hasUsageData = $derived.by(() => {
		if (!usage) {
			return false;
		}

		return usage.data.length > 0;
	});

	let isOnboardingIncomplete = $derived(
		!onboardingLoading && (devices.length === 0 || tokens.length === 0)
	);
	let shouldShowUsage = $derived(!isOnboardingIncomplete || hasUsageData);
	let shouldShowOnboardingHero = $derived(isOnboardingIncomplete && !hasUsageData);
	let shouldShowPlanUsage = $derived.by(() => {
		if (isAdmin || !planUsage) {
			return false;
		}
		return planUsage.daily !== null || planUsage.monthly !== null;
	});
	let shouldShowPrimaryPlanUsage = $derived(shouldShowPlanUsage && planUsage?.plan === 'free');
	let currentPlanAlert = $derived.by(() => (isAdmin ? null : planAlert(planUsage)));
</script>

<div class="space-y-6">
	<div>
		{#if shouldShowOnboardingHero}
			<h1 class="text-3xl font-bold text-base-content">Welcome to BeeBuzz</h1>
			<p class="mt-1 text-base-content/70">
				Set up your first device and API token to start sending encrypted messages.
			</p>
		{:else}
			<h1 class="text-3xl font-bold text-base-content">Overview</h1>
			<p class="mt-1 text-base-content/70">Welcome back!</p>
		{/if}
	</div>

	{#if isOnboardingIncomplete}
		<div class="card bg-primary/5 border border-primary/20 p-6">
			<h2 class="mb-4 text-lg font-bold text-base-content">Getting started</h2>
			<div class="space-y-3">
				<a
					href="/account/devices"
					class="flex items-center gap-3 rounded-lg bg-base-100 p-3 transition-colors hover:bg-base-200"
				>
					<span
						class="flex h-8 w-8 items-center justify-center rounded-full {devices.length > 0
							? 'bg-success text-success-content'
							: 'bg-primary text-primary-content'} text-sm font-bold"
					>
						{#if devices.length > 0}
							<Check size={16} />
						{:else}
							1
						{/if}
					</span>
					<div class="flex-1">
						<p class="font-semibold text-base-content">Pair your first device</p>
						<p class="text-sm text-base-content/70">Generate a pairing code for Hive</p>
					</div>
					{#if devices.length === 0}
						<Plus size={18} class="text-primary" />
					{/if}
				</a>
				<a
					href="/account/api-tokens"
					class="flex items-center gap-3 rounded-lg bg-base-100 p-3 transition-colors hover:bg-base-200"
				>
					<span
						class="flex h-8 w-8 items-center justify-center rounded-full {tokens.length > 0
							? 'bg-success text-success-content'
							: 'bg-primary text-primary-content'} text-sm font-bold"
					>
						{#if tokens.length > 0}
							<Check size={16} />
						{:else}
							2
						{/if}
					</span>
					<div class="flex-1">
						<p class="font-semibold text-base-content">Create an API token</p>
						<p class="text-sm text-base-content/70">Get your token for the CLI</p>
					</div>
					{#if tokens.length === 0}
						<Key size={18} class="text-primary" />
					{/if}
				</a>
				<a
					href={DOCS_QUICKSTART_URL}
					class="flex items-center gap-3 rounded-lg bg-base-100 p-3 transition-colors hover:bg-base-200"
					target="_blank"
				>
					<span
						class="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-sm font-bold text-primary-content"
					>
						3
					</span>
					<div class="flex-1">
						<p class="font-semibold text-base-content">Send your first message</p>
						<p class="text-sm text-base-content/70">Install CLI and send a test message</p>
					</div>
					<Send size={18} class="text-primary" />
				</a>
			</div>
		</div>
	{/if}

	{#if currentPlanAlert}
		<div class="alert {currentPlanAlert.tone}">
			<CircleAlert size={20} />
			<div class="flex flex-1 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div>
					<div class="font-semibold">{currentPlanAlert.title}</div>
					<div class="text-sm">{currentPlanAlert.body}</div>
				</div>
				{#if currentPlanAlert.showUpgrade}
					<a href={BILLING_URL} class="btn btn-sm">
						<CreditCard size={16} />
						Manage plan
					</a>
				{/if}
			</div>
		</div>
	{/if}

	{#if shouldShowPrimaryPlanUsage}
		{#if planUsage}
			<PlanUsageCard {planUsage} showUpgrade />
		{/if}
	{/if}

	{#if shouldShowUsage}
		<div class="space-y-6">
			<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div>
					<h2 class="text-xl font-bold text-base-content">Activity</h2>
					<p class="text-sm text-base-content/60">
						Your message activity over the selected time range.
					</p>
				</div>
				<div class="join self-start sm:self-auto">
					{#each ranges as range (range.value)}
						<button
							class="join-item btn btn-sm {selectedRange === range.value
								? 'btn-primary'
								: 'btn-ghost'}"
							onclick={() => handleRangeChange(range.value)}
						>
							{range.label}
						</button>
					{/each}
				</div>
			</div>

			{#if loading}
				<div class="flex items-center justify-center py-12">
					<div class="text-center">
						<Loader size={32} class="mx-auto mb-2 animate-spin text-primary" />
						<p class="text-base-content/70">Loading usage data...</p>
					</div>
				</div>
			{:else}
				{#if usage && usage.data.length > 0}
					<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
						<div class="card border border-base-300 bg-base-200">
							<div class="card-body p-4">
								<div class="flex items-center gap-2 text-sm text-base-content/70">
									<Send size={16} />
									<span>Sent</span>
								</div>
								<p class="mt-1 text-2xl font-bold text-base-content">
									{formatNumber(totals.created)}
								</p>
								<p class="mt-1 text-xs text-base-content/60">Messages sent in this range</p>
							</div>
						</div>

						<div class="card border border-base-300 bg-base-200">
							<div class="card-body p-4">
								<div class="flex items-center gap-2 text-sm text-success">
									<CircleCheckBig size={16} />
									<span>Delivered</span>
								</div>
								<p class="mt-1 text-2xl font-bold text-base-content">
									{formatNumber(totals.delivered)}
								</p>
								<p class="mt-1 text-xs text-base-content/60">Successful device deliveries</p>
							</div>
						</div>

						<div class="card border border-base-300 bg-base-200">
							<div class="card-body p-4">
								<div class="flex items-center gap-2 text-sm text-error">
									<CircleAlert size={16} />
									<span>Failed</span>
								</div>
								<p class="mt-1 text-2xl font-bold text-base-content">
									{formatNumber(totals.failed)}
								</p>
								<p class="mt-1 text-xs text-base-content/60">
									{formatNumber(totals.devicesLost)} devices lost
								</p>
							</div>
						</div>

						<div class="card border border-base-300 bg-base-200">
							<div class="card-body p-4">
								<div class="flex items-center gap-2 text-sm {reliabilityTone}">
									<Gauge size={16} />
									<span>Delivery reliability</span>
								</div>
								<p class="mt-1 text-2xl font-bold text-base-content">
									{formatPercent(deliverySuccessRate)}
								</p>
								<p class="mt-1 text-xs text-base-content/60">
									{formatNumber(totals.delivered)} of {formatNumber(totals.attempts)} attempts succeeded
								</p>
							</div>
						</div>
					</div>
				{/if}

				{#if usage && usage.data.length > 0}
					<DashboardChart bars={chartSeries.bars} granularity={chartSeries.granularity} />

					<div class="grid gap-4 xl:grid-cols-3">
						<div class="card border border-base-300 bg-base-200">
							<div class="card-body p-4">
								<div class="mb-4">
									<h3 class="text-lg font-semibold text-base-content">Source breakdown</h3>
									<p class="text-sm text-base-content/60">How messages were sent.</p>
								</div>
								{#if totalSources > 0}
									<div class="space-y-4">
										{#each sourceBreakdown as source (source.key)}
											<div class="space-y-2">
												<div class="flex items-center gap-3">
													<source.icon size={18} class="shrink-0 text-base-content/65" />
													<div class="flex-1">
														<div class="flex items-center justify-between gap-4">
															<span class="text-sm font-medium text-base-content"
																>{source.label}</span
															>
															<span class="text-sm text-base-content/70">
																{formatNumber(source.value)} ({formatPercent(source.share)})
															</span>
														</div>
													</div>
												</div>
												<div class="h-2 overflow-hidden rounded-full bg-base-300">
													<div
														class="h-full rounded-full transition-all {source.barClass}"
														style={`width: ${source.share * 100}%`}
													></div>
												</div>
											</div>
										{/each}
									</div>
								{:else}
									<p class="py-6 text-center text-sm text-base-content/50">
										No messages sent in this range yet.
									</p>
								{/if}
							</div>
						</div>

						<div class="card border border-base-300 bg-base-200">
							<div class="card-body p-4">
								<div class="mb-4">
									<h3 class="text-lg font-semibold text-base-content">Delivery modes</h3>
									<p class="text-sm text-base-content/60">Trusted and end-to-end mix.</p>
								</div>
								<div class="space-y-4">
									<div class="space-y-2">
										<div class="flex items-center justify-between gap-4">
											<div class="flex items-center gap-2 text-sm text-base-content">
												<LockKeyhole size={16} class="text-base-content/65" />
												<span>Trusted</span>
											</div>
											<span class="text-sm text-base-content/70">
												{formatNumber(totals.serverTrusted)} ({formatPercent(trustedShare)})
											</span>
										</div>
										<div class="h-2 overflow-hidden rounded-full bg-base-300">
											<div
												class="h-full rounded-full bg-slate-500/75 transition-all"
												style={`width: ${trustedShare * 100}%`}
											></div>
										</div>
									</div>
									<div class="space-y-2">
										<div class="flex items-center justify-between gap-4">
											<div class="flex items-center gap-2 text-sm text-base-content">
												<LockKeyhole size={16} class="text-base-content/65" />
												<span>E2E</span>
											</div>
											<span class="text-sm text-base-content/70">
												{formatNumber(totals.e2e)} ({formatPercent(e2eShare)})
											</span>
										</div>
										<div class="h-2 overflow-hidden rounded-full bg-base-300">
											<div
												class="h-full rounded-full bg-violet-500/70 transition-all"
												style={`width: ${e2eShare * 100}%`}
											></div>
										</div>
									</div>
								</div>
							</div>
						</div>

						<div class="card border border-base-300 bg-base-200">
							<div class="card-body p-4">
								<div class="mb-4">
									<h3 class="text-lg font-semibold text-base-content">Activity details</h3>
									<p class="text-sm text-base-content/60">
										Secondary activity metrics for the selected range.
									</p>
								</div>
								<div class="space-y-4">
									<div class="flex items-center justify-between gap-4">
										<span class="text-sm text-base-content/70">Delivery attempts</span>
										<span class="text-sm font-semibold text-base-content">
											{formatNumber(totals.attempts)}
										</span>
									</div>
									<div class="flex items-center justify-between gap-4">
										<div class="flex items-center gap-2 text-sm text-base-content/70">
											<Paperclip size={16} />
											<span>Attachments</span>
										</div>
										<div class="text-right">
											<p class="text-sm font-semibold text-base-content">
												{formatNumber(totals.attachments)}
											</p>
											<p class="text-xs text-base-content/60">
												{formatPercent(attachmentShare)} of sent
											</p>
										</div>
									</div>
									<div class="flex items-center justify-between gap-4">
										<span class="text-sm text-base-content/70">Attachment data</span>
										<span class="text-sm font-semibold text-base-content">
											{formatBytes(totals.attachmentsBytes)}
										</span>
									</div>
									<div class="flex items-center justify-between gap-4">
										<div class="flex items-center gap-2 text-sm text-base-content/70">
											<WifiOff size={16} />
											<span>Devices lost</span>
										</div>
										<span class="text-sm font-semibold text-base-content">
											{formatNumber(totals.devicesLost)}
										</span>
									</div>
								</div>
							</div>
						</div>
					</div>
				{:else if usage}
					<div class="card border border-base-300 bg-base-200">
						<div class="card-body items-center text-center">
							<p class="text-base-content/70">No usage data for this period</p>
						</div>
					</div>
				{/if}
			{/if}
		</div>
	{/if}
</div>
