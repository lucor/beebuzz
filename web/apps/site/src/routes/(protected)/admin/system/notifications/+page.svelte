<script lang="ts">
	import { onMount } from 'svelte';
	import { BellRing, Loader, Save } from '@lucide/svelte';
	import { adminApi, topicsApi, type Topic } from '@beebuzz/shared/api';
	import { toast } from '@beebuzz/shared/stores';
	import { ApiError } from '@beebuzz/shared/errors';

	let loading = $state(true);
	let saving = $state(false);
	let topics: Topic[] = $state([]);
	let enabled = $state(false);
	let signupCreatedEnabled = $state(false);
	let selectedTopicID = $state('');
	let recipientUserID = $state('');

	onMount(async () => {
		await loadSettings();
	});

	/** Loads admin-owned topics and current system notification settings. */
	async function loadSettings() {
		loading = true;
		try {
			const [settings, loadedTopics] = await Promise.all([
				adminApi.getSystemNotificationSettings(),
				topicsApi.listTopics()
			]);
			topics = loadedTopics;
			enabled = settings.enabled;
			signupCreatedEnabled = settings.signup_created_enabled;
			selectedTopicID = settings.topic_id ?? '';
			recipientUserID = settings.recipient_user_id ?? '';
		} catch (err) {
			toast.error(
				err instanceof ApiError ? err.userMessage : 'Failed to load system notifications'
			);
		} finally {
			loading = false;
		}
	}

	let canSave = $derived(!saving && (!enabled || selectedTopicID !== ''));

	/** Persists system notification settings for the current admin. */
	async function saveSettings() {
		if (!canSave) return;

		saving = true;
		try {
			const settings = await adminApi.updateSystemNotificationSettings({
				enabled,
				topic_id: selectedTopicID,
				signup_created_enabled: signupCreatedEnabled
			});
			recipientUserID = settings.recipient_user_id ?? '';
			toast.success('System notification settings saved');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to save settings');
		} finally {
			saving = false;
		}
	}
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between gap-4">
		<div>
			<h2 class="text-2xl font-bold text-base-content">System Notifications</h2>
			<p class="mt-1 text-sm text-base-content/70">
				Send BeeBuzz platform events to your paired admin devices.
			</p>
		</div>
		<div class="hidden rounded-2xl bg-warning/15 p-4 text-warning sm:block">
			<BellRing size={28} />
		</div>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="text-center">
				<Loader size={32} class="mx-auto mb-2 animate-spin text-primary" />
				<p class="text-base-content/70">Loading system notifications...</p>
			</div>
		</div>
	{:else}
		<div class="card border border-base-300 bg-base-100 shadow">
			<div class="card-body space-y-6">
				<div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
					<div>
						<h3 class="text-lg font-semibold text-base-content">Delivery</h3>
						<p class="mt-1 text-sm text-base-content/70">
							Settings are global for this BeeBuzz instance. The recipient is the admin account that
							saves them.
						</p>
						{#if recipientUserID}
							<p class="mt-2 font-mono text-xs text-base-content/50">
								Current recipient: {recipientUserID}
							</p>
						{/if}
					</div>
					<label class="label cursor-pointer justify-start gap-3">
						<span class="label-text font-medium">Enabled</span>
						<input type="checkbox" class="toggle toggle-primary" bind:checked={enabled} />
					</label>
				</div>

				<label class="form-control w-full">
					<span class="label">
						<span class="label-text font-medium">Destination topic</span>
					</span>
					<select class="select select-bordered w-full" bind:value={selectedTopicID}>
						<option value="">Select a topic</option>
						{#each topics as topic (topic.id)}
							<option value={topic.id}>{topic.name}</option>
						{/each}
					</select>
					<span class="label">
						<span class="label-text-alt text-base-content/60">
							Only topics owned by your admin account are available.
						</span>
					</span>
				</label>

				<div class="rounded-box border border-base-300 bg-base-200/60 p-4">
					<label class="label cursor-pointer justify-start gap-3">
						<input
							type="checkbox"
							class="checkbox checkbox-primary"
							bind:checked={signupCreatedEnabled}
						/>
						<span>
							<span class="block font-medium text-base-content">New signup</span>
							<span class="block text-sm text-base-content/70">
								Send a notification when BeeBuzz creates a new account.
							</span>
						</span>
					</label>
				</div>

				{#if enabled && selectedTopicID === ''}
					<div class="alert alert-warning">
						<span>Select a topic before enabling system notifications.</span>
					</div>
				{/if}

				<div class="card-actions justify-end">
					<button type="button" class="btn btn-primary" disabled={!canSave} onclick={saveSettings}>
						{#if saving}
							<Loader size={18} class="animate-spin" />
							Saving...
						{:else}
							<Save size={18} />
							Save settings
						{/if}
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>
