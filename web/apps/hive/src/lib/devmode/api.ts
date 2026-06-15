import type { HiveDebugReport, HiveDebugReportResponse } from './types';
import { captureHiveError } from './error-capture';
import { API_URL } from '@beebuzz/shared/config';

const API_BASE = `${API_URL}/v1/hive`;

async function getDeviceToken(): Promise<string | null> {
	try {
		const { deviceKeysRepository } = await import('$lib/services/device-keys-repository');
		const credentials = await deviceKeysRepository.getDeviceCredentials();
		return credentials?.deviceToken || null;
	} catch {
		return null;
	}
}

export async function submitDebugReport(
	report: HiveDebugReport
): Promise<HiveDebugReportResponse | null> {
	const token = await getDeviceToken();
	if (!token) {
		await captureHiveError({
			scope: 'network',
			event: 'debug_report.missing_device_token',
			message: 'Debug report submission unavailable because this device is not paired',
			severity: 'warn'
		});
		return null;
	}

	try {
		const response = await fetch(`${API_BASE}/debug-reports`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				Authorization: `Bearer ${token}`
			},
			body: JSON.stringify(report)
		});

		if (!response.ok) {
			await captureHiveError({
				scope: 'network',
				event: 'debug_report.submit_failed',
				message: 'Debug report submission failed',
				error: new Error(`HTTP ${response.status}`),
				severity: 'error'
			});
			return null;
		}

		return (await response.json()) as HiveDebugReportResponse;
	} catch (error) {
		await captureHiveError({
			scope: 'network',
			event: 'debug_report.submit_failed',
			message: 'Debug report submission failed',
			error,
			severity: 'error'
		});
		return null;
	}
}
