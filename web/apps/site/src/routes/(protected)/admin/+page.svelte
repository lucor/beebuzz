<script lang="ts">
	import { toast } from '@beebuzz/shared/stores';
	import { onMount } from 'svelte';
	import { DashboardChart } from '@beebuzz/shared/components';
	import {
		buildDashboardChartSeries,
		type DashboardTimeRange
	} from '@beebuzz/shared/utils/dashboard-chart';
	import {
		BellRing,
		CircleAlert,
		CircleCheckBig,
		CodeXml,
		Gauge,
		Loader,
		LockKeyhole,
		Send,
		Terminal,
		Users,
		Webhook,
		WifiOff
	} from '@lucide/svelte';
	import { adminApi, type PlatformDashboard } from '@beebuzz/shared/api';
	import { ApiError } from '@beebuzz/shared/errors';

	type TimeRange = DashboardTimeRange;

	let dashboard: PlatformDashboard | null = $state(null);
	let loading = $state(true);
	let selectedRange: TimeRange = $state(7);

	const ranges: { value: TimeRange; label: string }[] = [
		{ value: 7, label: '7d' },
		{ value: 30, label: '30d' },
		{ value: 90, label: '90d' },
		{ value: 0, label: 'All time' }
	];

	onMount(async () => {
		await loadDashboard();
	});

	/** Fetches platform dashboard data for the selected time range. */
	async function loadDashboard() {
		loading = true;
		try {
			dashboard = await adminApi.getDashboard(selectedRange);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load dashboard');
		} finally {
			loading = false;
		}
	}

	/** Handles time range button click. */
	async function handleRangeChange(range: TimeRange) {
		selectedRange = range;
		await loadDashboard();
	}

	/** Formats large numbers with K/M suffix. */
	function formatNumber(n: number): string {
		if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
		if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
		return n.toString();
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

	let totalSources = $derived.by(() => {
		if (!dashboard) return 0;
		return (
			dashboard.sources_cli +
			dashboard.sources_webhook +
			dashboard.sources_api +
			dashboard.sources_internal
		);
	});

	let trustedShare = $derived.by(() => {
		if (!dashboard || dashboard.notifications_created === 0) return 0;
		return dashboard.notifications_server_trusted / dashboard.notifications_created;
	});

	let e2eShare = $derived.by(() => {
		if (!dashboard || dashboard.notifications_created === 0) return 0;
		return dashboard.notifications_e2e / dashboard.notifications_created;
	});

	let reliabilityTone = $derived.by(() => {
		if (!dashboard || dashboard.delivery_attempts === 0) {
			return 'text-base-content/70';
		}

		if (dashboard.delivery_success_rate >= 0.98) {
			return 'text-success';
		}

		if (dashboard.delivery_success_rate >= 0.9) {
			return 'text-warning';
		}

		return 'text-error';
	});

	let chartSeries = $derived.by(() => {
		if (!dashboard) {
			return buildDashboardChartSeries([], selectedRange);
		}

		return buildDashboardChartSeries(dashboard.daily_breakdown, selectedRange);
	});

	let sourceBreakdown = $derived.by(() => {
		if (!dashboard) {
			return [];
		}

		return [
			{
				key: 'cli',
				label: 'CLI',
				value: dashboard.sources_cli,
				share: totalSources === 0 ? 0 : dashboard.sources_cli / totalSources,
				icon: Terminal,
				barClass: 'bg-sky-500/75'
			},
			{
				key: 'webhook',
				label: 'Webhook',
				value: dashboard.sources_webhook,
				share: totalSources === 0 ? 0 : dashboard.sources_webhook / totalSources,
				icon: Webhook,
				barClass: 'bg-cyan-600/70'
			},
			{
				key: 'api',
				label: 'API',
				value: dashboard.sources_api,
				share: totalSources === 0 ? 0 : dashboard.sources_api / totalSources,
				icon: CodeXml,
				barClass: 'bg-indigo-500/70'
			},
			{
				key: 'internal',
				label: 'Internal',
				value: dashboard.sources_internal,
				share: totalSources === 0 ? 0 : dashboard.sources_internal / totalSources,
				icon: BellRing,
				barClass: 'bg-amber-500/75'
			}
		];
	});
</script>

<div class="space-y-6">
	<div>
		<h2 class="text-2xl font-bold text-base-content">Platform Dashboard</h2>
		<p class="mt-1 text-sm text-base-content/70">
			Platform-wide notification activity over the last 7 days by default.
		</p>
	</div>

	<div class="space-y-6">
		<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<div>
				<h3 class="text-xl font-bold text-base-content">Usage</h3>
				<p class="text-sm text-base-content/60">Daily buckets, adapted to the selected range.</p>
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
					<p class="text-base-content/70">Loading dashboard...</p>
				</div>
			</div>
		{:else if dashboard}
			<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
				<div class="card border border-base-300 bg-base-200">
					<div class="card-body p-4">
						<div class="flex items-center gap-2 text-sm text-base-content/70">
							<Users size={16} />
							<span>Active users</span>
						</div>
						<p class="mt-1 text-2xl font-bold text-base-content">
							{formatNumber(dashboard.active_users)}
						</p>
						<p class="mt-1 text-xs text-base-content/60">Accounts active in this range</p>
					</div>
				</div>

				<div class="card border border-base-300 bg-base-200">
					<div class="card-body p-4">
						<div class="flex items-center gap-2 text-sm text-base-content/70">
							<Send size={16} />
							<span>Sent</span>
						</div>
						<p class="mt-1 text-2xl font-bold text-base-content">
							{formatNumber(dashboard.notifications_created)}
						</p>
						<p class="mt-1 text-xs text-base-content/60">Notifications created in this range</p>
					</div>
				</div>

				<div class="card border border-base-300 bg-base-200">
					<div class="card-body p-4">
						<div class="flex items-center gap-2 text-sm text-success">
							<CircleCheckBig size={16} />
							<span>Delivered</span>
						</div>
						<p class="mt-1 text-2xl font-bold text-base-content">
							{formatNumber(dashboard.deliveries_succeeded)}
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
							{formatNumber(dashboard.deliveries_failed)}
						</p>
						<p class="mt-1 text-xs text-base-content/60">
							{formatNumber(dashboard.devices_lost)} devices lost
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
							{formatPercent(dashboard.delivery_success_rate)}
						</p>
						<p class="mt-1 text-xs text-base-content/60">
							{formatNumber(dashboard.deliveries_succeeded)} of {formatNumber(
								dashboard.delivery_attempts
							)} attempts succeeded
						</p>
					</div>
				</div>
			</div>

			<DashboardChart bars={chartSeries.bars} granularity={chartSeries.granularity} />

			<div class="grid gap-4 xl:grid-cols-3">
				<div class="card border border-base-300 bg-base-200">
					<div class="card-body p-4">
						<div class="mb-4">
							<h3 class="text-lg font-semibold text-base-content">Source breakdown</h3>
							<p class="text-sm text-base-content/60">How notifications were triggered.</p>
						</div>
						{#if sourceBreakdown.length > 0}
							<div class="space-y-4">
								{#each sourceBreakdown as source (source.key)}
									<div class="space-y-2">
										<div class="flex items-center gap-3">
											<source.icon size={18} class="shrink-0 text-base-content/65" />
											<div class="flex-1">
												<div class="flex items-center justify-between gap-4">
													<span class="text-sm font-medium text-base-content">{source.label}</span>
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
								No notifications sent in this range yet.
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
										{formatNumber(dashboard.notifications_server_trusted)} ({formatPercent(
											trustedShare
										)})
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
										{formatNumber(dashboard.notifications_e2e)} ({formatPercent(e2eShare)})
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
							<h3 class="text-lg font-semibold text-base-content">Usage details</h3>
							<p class="text-sm text-base-content/60">Secondary metrics for the selected range.</p>
						</div>
						<div class="space-y-4">
							<div class="flex items-center justify-between gap-4">
								<span class="text-sm text-base-content/70">Delivery attempts</span>
								<span class="text-sm font-semibold text-base-content">
									{formatNumber(dashboard.delivery_attempts)}
								</span>
							</div>
							<div class="flex items-center justify-between gap-4">
								<span class="text-sm text-base-content/70">Attachment data</span>
								<span class="text-sm font-semibold text-base-content">
									{formatBytes(dashboard.attachment_bytes_total)}
								</span>
							</div>
							<div class="flex items-center justify-between gap-4">
								<span class="text-sm text-base-content/70">With attachment</span>
								<span class="text-sm font-semibold text-base-content">
									{formatNumber(dashboard.notifications_with_attachment)}
								</span>
							</div>
							<div class="flex items-center justify-between gap-4">
								<div class="flex items-center gap-2 text-sm text-base-content/70">
									<WifiOff size={16} />
									<span>Devices lost</span>
								</div>
								<span class="text-sm font-semibold text-base-content">
									{formatNumber(dashboard.devices_lost)}
								</span>
							</div>
						</div>
					</div>
				</div>
			</div>
		{:else}
			<div class="card border border-base-300 bg-base-200">
				<div class="card-body items-center text-center">
					<p class="text-base-content/70">No dashboard data available</p>
				</div>
			</div>
		{/if}
	</div>
</div>
