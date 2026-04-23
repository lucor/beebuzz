export type DashboardTimeRange = 0 | 1 | 7 | 30 | 90;

export type DashboardChartGranularity = 'daily' | 'weekly' | 'monthly';

export interface DashboardChartPoint {
	date: string;
	notifications_created: number;
	deliveries_succeeded: number;
	deliveries_failed: number;
	notifications_server_trusted: number;
	notifications_e2e: number;
}

export interface DashboardChartBar {
	key: string;
	label: string;
	tooltipLabel: string;
	notificationsCreated: number;
	deliveriesSucceeded: number;
	deliveriesFailed: number;
	notificationsServerTrusted: number;
	notificationsE2e: number;
}

export interface DashboardChartSeries {
	bars: DashboardChartBar[];
	granularity: DashboardChartGranularity;
}

interface MutableBucket {
	key: string;
	label: string;
	tooltipLabel: string;
	notificationsCreated: number;
	deliveriesSucceeded: number;
	deliveriesFailed: number;
	notificationsServerTrusted: number;
	notificationsE2e: number;
}

const DAILY_RANGE_THRESHOLD = 30;
const WEEK_START_DAY = 1;
const QUARTER_MONTH_SPAN = 3;

/**
 * Buckets dashboard points so long ranges stay readable.
 */
export const buildDashboardChartSeries = (
	points: DashboardChartPoint[],
	range: DashboardTimeRange
): DashboardChartSeries => {
	if (points.length === 0) {
		return { bars: [], granularity: 'daily' };
	}

	const granularity = getChartGranularity(points, range);
	const buckets = new Map<string, MutableBucket>();

	for (const point of points) {
		const bucket = getBucketDefinition(point.date, granularity);
		const existing = buckets.get(bucket.key);
		if (existing) {
			existing.notificationsCreated += point.notifications_created;
			existing.deliveriesSucceeded += point.deliveries_succeeded;
			existing.deliveriesFailed += point.deliveries_failed;
			existing.notificationsServerTrusted += point.notifications_server_trusted;
			existing.notificationsE2e += point.notifications_e2e;
			continue;
		}

		buckets.set(bucket.key, {
			...bucket,
			notificationsCreated: point.notifications_created,
			deliveriesSucceeded: point.deliveries_succeeded,
			deliveriesFailed: point.deliveries_failed,
			notificationsServerTrusted: point.notifications_server_trusted,
			notificationsE2e: point.notifications_e2e
		});
	}

	return {
		granularity,
		bars: Array.from(buckets.values())
	};
};

/**
 * Returns a compact label step based on the number of visible bars.
 */
export const getDashboardChartLabelStep = (barCount: number): number => {
	if (barCount <= 8) return 1;
	if (barCount <= 16) return 2;
	if (barCount <= 24) return 3;
	if (barCount <= 40) return 4;
	return 6;
};

/**
 * Returns a chart title suffix for the current granularity.
 */
export const getDashboardChartTitle = (granularity: DashboardChartGranularity): string => {
	switch (granularity) {
		case 'weekly':
			return 'Weekly Delivery Trend';
		case 'monthly':
			return 'Monthly Delivery Trend';
		default:
			return 'Daily Delivery Trend';
	}
};

const getChartGranularity = (
	points: DashboardChartPoint[],
	range: DashboardTimeRange
): DashboardChartGranularity => {
	if (range === 90) {
		return 'weekly';
	}

	if (range === 0) {
		const monthCount = countDistinctMonths(points);
		if (monthCount > QUARTER_MONTH_SPAN) {
			return 'monthly';
		}
		return 'weekly';
	}

	if (range > DAILY_RANGE_THRESHOLD) {
		return 'weekly';
	}

	return 'daily';
};

const getBucketDefinition = (
	isoDate: string,
	granularity: DashboardChartGranularity
): Pick<MutableBucket, 'key' | 'label' | 'tooltipLabel'> => {
	const date = parseUtcDate(isoDate);

	switch (granularity) {
		case 'weekly': {
			const weekStart = startOfUtcWeek(date);
			const weekEnd = addUtcDays(weekStart, 6);
			return {
				key: formatIsoDate(weekStart),
				label: formatMonthDay(weekStart),
				tooltipLabel: `${formatMonthDay(weekStart)} - ${formatMonthDay(weekEnd)}`
			};
		}
		case 'monthly': {
			const monthKey = formatMonthKey(date);
			return {
				key: monthKey,
				label: formatMonthYear(date),
				tooltipLabel: formatMonthYearLong(date)
			};
		}
		default:
			return {
				key: isoDate,
				label: formatMonthDay(date),
				tooltipLabel: formatMonthDayLong(date)
			};
	}
};

const countDistinctMonths = (points: DashboardChartPoint[]): number => {
	const months = new Set(points.map((point) => point.date.slice(0, 7)));
	return months.size;
};

const parseUtcDate = (isoDate: string): Date => {
	return new Date(`${isoDate}T00:00:00Z`);
};

const startOfUtcWeek = (date: Date): Date => {
	const day = date.getUTCDay();
	const offset = day === 0 ? 6 : day - WEEK_START_DAY;
	return addUtcDays(date, -offset);
};

const addUtcDays = (date: Date, days: number): Date => {
	const copy = new Date(date);
	copy.setUTCDate(copy.getUTCDate() + days);
	return copy;
};

const formatIsoDate = (date: Date): string => {
	return date.toISOString().slice(0, 10);
};

const formatMonthKey = (date: Date): string => {
	return formatIsoDate(date).slice(0, 7);
};

const monthDayFormatter = new Intl.DateTimeFormat('en-US', {
	month: 'short',
	day: 'numeric',
	timeZone: 'UTC'
});

const monthDayLongFormatter = new Intl.DateTimeFormat('en-US', {
	month: 'short',
	day: 'numeric',
	year: 'numeric',
	timeZone: 'UTC'
});

const monthYearFormatter = new Intl.DateTimeFormat('en-US', {
	month: 'short',
	year: '2-digit',
	timeZone: 'UTC'
});

const monthYearLongFormatter = new Intl.DateTimeFormat('en-US', {
	month: 'long',
	year: 'numeric',
	timeZone: 'UTC'
});

const formatMonthDay = (date: Date): string => {
	return monthDayFormatter.format(date);
};

const formatMonthDayLong = (date: Date): string => {
	return monthDayLongFormatter.format(date);
};

const formatMonthYear = (date: Date): string => {
	return monthYearFormatter.format(date);
};

const formatMonthYearLong = (date: Date): string => {
	return monthYearLongFormatter.format(date);
};
