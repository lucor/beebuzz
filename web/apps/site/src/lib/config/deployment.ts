export type DeploymentMode = 'self_hosted' | 'saas';

const DEFAULT_DEPLOYMENT_MODE: DeploymentMode = 'self_hosted';

/** Normalizes the public site deployment mode from the Vite env. */
export function parseDeploymentMode(value: string | undefined): DeploymentMode {
	return value === 'saas' ? 'saas' : DEFAULT_DEPLOYMENT_MODE;
}

export const deploymentMode = parseDeploymentMode(import.meta.env.VITE_BEEBUZZ_DEPLOYMENT_MODE);
export const isSaasMode = deploymentMode === 'saas';
export const isSelfHostedMode = deploymentMode === 'self_hosted';
