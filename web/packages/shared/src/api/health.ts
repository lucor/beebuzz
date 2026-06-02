// Health check and system status API endpoints.
import { api } from './client';

export interface HealthCheckStatus {
	status: string;
	version: string;
}

/**
 * Health check API namespace.
 */
export const healthApi = {
	/**
	 * Check server health status.
	 */
	checkHealth: () => api.get<HealthCheckStatus>('/health')
};
