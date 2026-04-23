<script lang="ts">
	import {
		getDashboardChartLabelStep,
		getDashboardChartTitle,
		type DashboardChartBar,
		type DashboardChartGranularity
	} from '../utils/dashboard-chart';

	interface Props {
		bars: DashboardChartBar[];
		granularity: DashboardChartGranularity;
	}

	const { bars, granularity }: Props = $props();

	const CHART_HEIGHT = 200;
	const CHART_WIDTH = 900;
	const PADDING_LEFT = 44;
	const PADDING_RIGHT = 12;
	const PADDING_TOP = 12;
	const PADDING_BOTTOM = 36;
	const GRID_LINE_COUNT = 4;

	let chartTitle = $derived(getDashboardChartTitle(granularity));
	let labelStep = $derived(getDashboardChartLabelStep(bars.length));
	let chartWidth = $derived(CHART_WIDTH);
	let plotHeight = $derived(CHART_HEIGHT - PADDING_TOP - PADDING_BOTTOM);
	let plotWidth = $derived(CHART_WIDTH - PADDING_LEFT - PADDING_RIGHT);
	let baselineY = $derived(CHART_HEIGHT - PADDING_BOTTOM);

	let maxValue = $derived.by(() => {
		if (bars.length === 0) return 1;
		return Math.max(...bars.map((bar) => bar.deliveriesSucceeded + bar.deliveriesFailed), 1);
	});

	let barPositions = $derived.by(() => {
		if (bars.length === 0) return [];

		const slotWidth = plotWidth / bars.length;
		const barWidth = Math.max(10, Math.min(slotWidth * 0.56, 32));

		return bars.map((bar, index) => {
			const deliveredHeight = (bar.deliveriesSucceeded / maxValue) * plotHeight;
			const failedHeight = (bar.deliveriesFailed / maxValue) * plotHeight;
			const cx = PADDING_LEFT + slotWidth * index + slotWidth / 2;

			return {
				...bar,
				cx,
				x: cx - barWidth / 2,
				width: barWidth,
				deliveredY: baselineY - failedHeight - deliveredHeight,
				deliveredHeight,
				failedY: baselineY - failedHeight,
				failedHeight
			};
		});
	});

	let gridLines = $derived.by(() => {
		return Array.from({ length: GRID_LINE_COUNT }, (_, index) => {
			const ratio = index / (GRID_LINE_COUNT - 1);
			return PADDING_TOP + plotHeight * ratio;
		});
	});

	let yAxisTicks = $derived.by(() => {
		return gridLines.map((y, index) => {
			const ratio = 1 - index / (GRID_LINE_COUNT - 1);
			const value = Math.round(maxValue * ratio);
			return { y, value: formatAxisValue(value) };
		});
	});

	const formatAxisValue = (value: number): string => {
		if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
		if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
		return value.toString();
	};
</script>

<div class="card bg-base-200 border border-base-300 mb-6 overflow-hidden">
	<div class="card-body p-4">
		<div class="flex items-center justify-between mb-4">
			<h3 class="text-lg font-semibold text-base-content">{chartTitle}</h3>
			<div class="flex items-center gap-4 text-xs text-base-content/70">
				<span class="flex items-center gap-1.5">
					<span class="inline-block w-3 h-3 rounded-sm bg-success"></span>
					Delivered
				</span>
				<span class="flex items-center gap-1.5">
					<span class="inline-block w-3 h-3 rounded-sm bg-error"></span>
					Failed
				</span>
			</div>
		</div>
		<div>
			<svg
				class="block w-full h-auto"
				width={chartWidth}
				height={CHART_HEIGHT}
				viewBox={`0 0 ${chartWidth} ${CHART_HEIGHT}`}
				role="img"
				aria-label={chartTitle}
			>
				<line
					x1={PADDING_LEFT}
					y1={PADDING_TOP}
					x2={PADDING_LEFT}
					y2={baselineY}
					class="stroke-base-300"
					stroke-width="1"
				/>

				{#each gridLines as y (y)}
					<line
						x1={PADDING_LEFT}
						y1={y}
						x2={chartWidth}
						y2={y}
						class="stroke-base-300"
						stroke-width="1"
					/>
				{/each}

				{#each yAxisTicks as tick (tick.y)}
					<text
						x={PADDING_LEFT - 8}
						y={tick.y + 4}
						text-anchor="end"
						class="fill-base-content/50 text-[10px]"
					>
						{tick.value}
					</text>
				{/each}

				{#each barPositions as bar, index (bar.key)}
					<g>
						{#if bar.deliveredHeight > 0}
							<rect
								x={bar.x}
								y={bar.deliveredY}
								width={bar.width}
								height={bar.deliveredHeight}
								rx="2"
								class="fill-success"
							/>
						{/if}
						{#if bar.failedHeight > 0}
							<rect
								x={bar.x}
								y={bar.failedY}
								width={bar.width}
								height={bar.failedHeight}
								rx="2"
								class="fill-error"
							/>
						{/if}

						<rect
							x={bar.x}
							y={bar.deliveredY}
							width={bar.width}
							height={bar.deliveredHeight + bar.failedHeight}
							fill="transparent"
						>
							<title>
								{`${bar.tooltipLabel}
Created: ${bar.notificationsCreated}
Delivered: ${bar.deliveriesSucceeded}
Failed: ${bar.deliveriesFailed}`}
							</title>
						</rect>

						{#if index % labelStep === 0 || index === barPositions.length - 1}
							<text
								x={bar.cx}
								y={CHART_HEIGHT - 10}
								text-anchor="middle"
								class="fill-base-content/50 text-[10px]"
							>
								{bar.label}
							</text>
						{/if}
					</g>
				{/each}
			</svg>
		</div>
	</div>
</div>
