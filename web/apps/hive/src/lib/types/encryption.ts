export type EncryptionProbeStatus = 'idle' | 'running' | 'passed' | 'failed';

export interface EncryptionProbeStepResult {
	id: string;
	label: string;
	ok: boolean;
	detail: string;
}

export interface EncryptionProbeScenarioResult {
	id: string;
	label: string;
	ok: boolean;
	recipient: string | null;
	steps: EncryptionProbeStepResult[];
}

export interface KeyPersistenceProbeResult {
	ok: boolean;
	structuredClone: EncryptionProbeStepResult;
	scenarios: EncryptionProbeScenarioResult[];
}

export interface WrappingKeyProbeResult {
	ok: boolean;
	steps: EncryptionProbeStepResult[];
}

export interface EncryptionProbeResult {
	runAt: string;
	userAgent: string;
	keyPersistence: KeyPersistenceProbeResult;
	wrappingKey: WrappingKeyProbeResult;
}

export interface PushDebugSnapshot {
	userAgent: string;
	controllerScriptURL: string | null;
	controllerState: string | null;
	registrationScope: string | null;
	registrationInstallingState: string | null;
	registrationWaitingState: string | null;
	registrationActiveState: string | null;
	subscriptionEndpointHost: string | null;
	subscriptionP256dhLength: number;
	subscriptionAuthLength: number;
}
