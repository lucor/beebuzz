import type { HiveErrorSnapshot, HiveDebugReport } from './types';

export function buildHiveDebugReport(snapshot: HiveErrorSnapshot): HiveDebugReport {
	return {
		schema_version: 1,
		source: 'hive',
		report_type: 'manual_error_report',
		created_at: new Date().toISOString(),
		snapshot
	};
}
