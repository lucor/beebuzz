import type { HiveErrorSnapshot, HiveDebugReport, HiveConsoleDiagnosticEntry } from './types';

export function buildHiveDebugReport(
	snapshot: HiveErrorSnapshot,
	consoleDiagnostics?: HiveConsoleDiagnosticEntry[]
): HiveDebugReport {
	const report: HiveDebugReport = {
		schema_version: 1,
		source: 'hive',
		report_type: 'manual_error_report',
		created_at: new Date().toISOString(),
		snapshot
	};
	if (consoleDiagnostics && consoleDiagnostics.length > 0) {
		report.console_diagnostics = consoleDiagnostics;
	}
	return report;
}
