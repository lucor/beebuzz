import { beforeEach, describe, expect, it, vi } from 'vitest';

const loadAccountApi = async () => {
	vi.resetModules();
	vi.stubEnv('VITE_BEEBUZZ_DOMAIN', 'example.test');
	return import('./account');
};

const stubFetch = () => {
	const fetchMock = vi.fn(() =>
		Promise.resolve(
			new Response(JSON.stringify({ id: 'hook-1', token: 'token-1', name: 'hook' }), {
				status: 200
			})
		)
	);
	vi.stubGlobal('fetch', fetchMock);
	return fetchMock;
};

const requestBody = (fetchMock: ReturnType<typeof stubFetch>) => {
	const [, init] = fetchMock.mock.calls[0];
	return JSON.parse(String(init?.body)) as Record<string, unknown>;
};

describe('accountApi webhook payloads', () => {
	beforeEach(() => {
		vi.unstubAllEnvs();
		vi.restoreAllMocks();
		vi.unstubAllGlobals();
	});

	it('omits path title fields when creating a custom webhook with a static title', async () => {
		const { accountApi } = await loadAccountApi();
		const fetchMock = stubFetch();

		await accountApi.createWebhook(
			'hook',
			'',
			'custom',
			'data.title',
			'data.body',
			'normal',
			'static',
			'Fixed title',
			['topic-1']
		);

		expect(requestBody(fetchMock)).toEqual({
			name: 'hook',
			description: '',
			payload_type: 'custom',
			topics: ['topic-1'],
			priority: 'normal',
			title_source: 'static',
			body_path: 'data.body',
			title_value: 'Fixed title'
		});
	});

	it('omits custom mapping fields when creating a beebuzz webhook', async () => {
		const { accountApi } = await loadAccountApi();
		const fetchMock = stubFetch();

		await accountApi.createWebhook(
			'hook',
			'',
			'beebuzz',
			'data.title',
			'data.body',
			'normal',
			'static',
			'Fixed title',
			['topic-1']
		);

		expect(requestBody(fetchMock)).toEqual({
			name: 'hook',
			description: '',
			payload_type: 'beebuzz',
			topics: ['topic-1'],
			priority: 'normal'
		});
	});

	it('omits static title fields when updating a custom webhook with a path title', async () => {
		const { accountApi } = await loadAccountApi();
		const fetchMock = stubFetch();

		await accountApi.updateWebhook(
			'hook-1',
			'hook',
			'',
			'custom',
			'data.title',
			'data.body',
			'high',
			'path',
			'Stale fixed title',
			['topic-1']
		);

		expect(requestBody(fetchMock)).toEqual({
			name: 'hook',
			description: '',
			payload_type: 'custom',
			topics: ['topic-1'],
			priority: 'high',
			title_source: 'path',
			body_path: 'data.body',
			title_path: 'data.title'
		});
	});

	it('finalizes inspect sessions with only the active title source fields', async () => {
		const { accountApi } = await loadAccountApi();
		const fetchMock = stubFetch();

		await accountApi.finalizeInspect('data.title', 'data.body', 'static', 'Fixed title');

		expect(requestBody(fetchMock)).toEqual({
			title_source: 'static',
			body_path: 'data.body',
			title_value: 'Fixed title'
		});
	});
});
