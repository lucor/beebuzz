<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount, onDestroy } from 'svelte';
	import { toast } from '@beebuzz/shared/stores';
	import {
		accountApi,
		topicsApi,
		deviceIsPaired,
		deviceStatusLabel,
		deviceStatusBadgeClass,
		type CreatedDevice,
		type Device,
		type Topic,
		type ApiToken
	} from '@beebuzz/shared/api';
	import { ApiError } from '@beebuzz/shared/errors';
	import { SettingsModal, TopicChip } from '@beebuzz/shared/components';
	import {
		Copy,
		Check,
		Plus,
		RefreshCw,
		ExternalLink,
		EllipsisVertical,
		Pencil,
		Trash2,
		Key
	} from '@lucide/svelte';
	import {
		MAX_DISPLAY_NAME_LEN,
		MAX_DISPLAY_NAME_SOFT_LEN,
		MAX_DESCRIPTION_LEN
	} from '@beebuzz/shared';

	let devices = $state<Device[]>([]);
	let tokens = $state<ApiToken[]>([]);
	let topics = $state<Topic[]>([]);
	let isLoading = $state(false);
	let showCreateModal = $state(false);
	let newDeviceName = $state('');
	let newDescription = $state('');
	let selectedTopics = $state<string[]>([]);
	let createdDevice = $state<CreatedDevice | null>(null);
	let isCopiedCode = $state(false);
	let isCopiedUrl = $state(false);
	let secondsRemaining = $state(0);
	let countdownInterval: ReturnType<typeof setInterval> | null = null;
	let isRegenerating = $state(false);
	let revealedKeys = $state<Record<string, boolean>>({});
	let copiedKeyByDevice = $state<Record<string, boolean>>({});

	let minutes = $derived(Math.floor(secondsRemaining / 60));
	let seconds = $derived(secondsRemaining % 60);
	let isExpired = $derived(secondsRemaining <= 0);
	let isExpiringSoon = $derived(secondsRemaining > 0 && secondsRemaining < 60);
	let hasPairedDevice = $derived(devices.some(deviceIsPaired));
	let shouldShowTokenStep = $derived(hasPairedDevice && tokens.length === 0);

	let editingDeviceId = $state<string | null>(null);
	let editDeviceName = $state('');
	let editDescription = $state('');
	let editSelectedTopics = $state<string[]>([]);
	let devicePendingDelete = $state<Device | null>(null);
	let actionsMenuOpenDeviceId = $state<string | null>(null);
	let actionsMenuRef = $state<HTMLElement | undefined>(undefined);

	onMount(async () => {
		await Promise.all([loadDevices(), loadTokens(), loadTopics()]);
	});

	$effect(() => {
		if (!actionsMenuOpenDeviceId) return;

		// Close the actions menu when the click lands outside the current menu container.
		const handleClickOutside = (e: MouseEvent) => {
			if (actionsMenuRef && !actionsMenuRef.contains(e.target as Node)) {
				actionsMenuOpenDeviceId = null;
			}
		};

		document.addEventListener('click', handleClickOutside, true);
		return () => document.removeEventListener('click', handleClickOutside, true);
	});

	async function loadDevices() {
		isLoading = true;
		try {
			devices = await accountApi.listDevices();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load devices');
		} finally {
			isLoading = false;
		}
	}

	async function loadTopics() {
		try {
			topics = await topicsApi.listTopics();
		} catch {
			// Silently fail - topics are optional
		}
	}

	async function loadTokens() {
		try {
			tokens = await accountApi.listApiTokens();
		} catch {
			// Silently fail - token CTA is helpful but not critical
		}
	}

	async function handleCreate() {
		if (!newDeviceName.trim()) {
			toast.error('Device name is required');
			return;
		}

		if (newDeviceName.length > MAX_DISPLAY_NAME_LEN) {
			toast.error(`Device name must be ${MAX_DISPLAY_NAME_LEN} characters or less`);
			return;
		}

		if (newDescription.length > MAX_DESCRIPTION_LEN) {
			toast.error(`Description must be ${MAX_DESCRIPTION_LEN} characters or less`);
			return;
		}

		isLoading = true;
		try {
			const device = await accountApi.createDevice(newDeviceName, newDescription, selectedTopics);
			createdDevice = device;
			newDeviceName = '';
			newDescription = '';
			selectedTopics = [];
			showCreateModal = false;
			toast.success('Pairing code generated');
			await loadDevices();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to create device');
		} finally {
			isLoading = false;
		}
	}

	async function handleDelete() {
		if (!devicePendingDelete) return;

		try {
			await accountApi.deleteDevice(devicePendingDelete.id);
			toast.success('Device removed');
			devicePendingDelete = null;
			await loadDevices();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to delete device');
		}
	}

	// Countdown timer for pairing code expiration
	$effect(() => {
		const expiresAt = createdDevice?.expires_at;

		if (countdownInterval) {
			clearInterval(countdownInterval);
			countdownInterval = null;
		}

		if (!expiresAt) {
			secondsRemaining = 0;
			return;
		}

		const expiresMs = new Date(expiresAt).getTime();

		function updateRemaining() {
			const diff = Math.floor((expiresMs - Date.now()) / 1000);
			secondsRemaining = Math.max(0, diff);
			if (diff <= 0 && countdownInterval) {
				clearInterval(countdownInterval);
				countdownInterval = null;
			}
		}

		updateRemaining();
		countdownInterval = setInterval(updateRemaining, 1000);
	});

	onDestroy(() => {
		if (countdownInterval) {
			clearInterval(countdownInterval);
			countdownInterval = null;
		}
	});

	function copyToClipboard() {
		if (!createdDevice?.pairing_code) return;
		void navigator.clipboard.writeText(createdDevice.pairing_code);
		isCopiedCode = true;
		setTimeout(() => {
			isCopiedCode = false;
		}, 2000);
	}

	function copyUrlToClipboard() {
		if (!createdDevice?.pairing_url) return;
		void navigator.clipboard.writeText(createdDevice.pairing_url);
		isCopiedUrl = true;
		setTimeout(() => {
			isCopiedUrl = false;
		}, 2000);
	}

	async function handleRegenerate() {
		if (!createdDevice) return;

		isRegenerating = true;
		try {
			const response = await accountApi.regeneratePairingCode(createdDevice.device.id);
			createdDevice = {
				...createdDevice,
				pairing_code: response.pairing_code,
				pairing_url: response.pairing_url,
				qr_code: response.qr_code,
				expires_at: response.expires_at
			};
			toast.success('New pairing code generated');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to regenerate pairing code');
		} finally {
			isRegenerating = false;
		}
	}

	async function generateCodeForDevice(device: Device) {
		actionsMenuOpenDeviceId = null;
		isLoading = true;
		try {
			const response = await accountApi.regeneratePairingCode(device.id);
			createdDevice = {
				device: device,
				pairing_code: response.pairing_code,
				pairing_url: response.pairing_url,
				qr_code: response.qr_code,
				expires_at: response.expires_at
			};
			toast.success('Pairing code generated');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to generate pairing code');
		} finally {
			isLoading = false;
		}
	}

	function closeCreatedModal() {
		createdDevice = null;
		void loadDevices();
	}

	async function copyAgeRecipient(device: Device) {
		if (!device.age_recipient) {
			return;
		}

		await navigator.clipboard.writeText(device.age_recipient);
		copiedKeyByDevice = { ...copiedKeyByDevice, [device.id]: true };
		setTimeout(() => {
			copiedKeyByDevice = { ...copiedKeyByDevice, [device.id]: false };
		}, 2000);
	}

	function toggleRevealAgeRecipient(deviceId: string) {
		revealedKeys = { ...revealedKeys, [deviceId]: !revealedKeys[deviceId] };
	}

	function toggleTopic(topicId: string) {
		if (selectedTopics.includes(topicId)) {
			selectedTopics = selectedTopics.filter((t) => t !== topicId);
		} else {
			selectedTopics = [...selectedTopics, topicId];
		}
	}

	function startEdit(device: Device) {
		actionsMenuOpenDeviceId = null;
		editingDeviceId = device.id;
		editDeviceName = device.name || 'Device';
		editDescription = device.description || '';
		// device.topic_ids already contains topic IDs from the backend
		editSelectedTopics = device.topic_ids || [];
	}

	function cancelEdit() {
		editingDeviceId = null;
		editDeviceName = '';
		editDescription = '';
		editSelectedTopics = [];
	}

	function toggleDeviceActions(deviceId: string) {
		actionsMenuOpenDeviceId = actionsMenuOpenDeviceId === deviceId ? null : deviceId;
	}

	function openDeleteDialog(device: Device) {
		actionsMenuOpenDeviceId = null;
		devicePendingDelete = device;
	}

	async function handleEdit() {
		if (!editDeviceName.trim()) {
			toast.error('Device name is required');
			return;
		}

		if (editDeviceName.length > MAX_DISPLAY_NAME_LEN) {
			toast.error(`Device name must be ${MAX_DISPLAY_NAME_LEN} characters or less`);
			return;
		}

		if (editDescription.length > MAX_DESCRIPTION_LEN) {
			toast.error(`Description must be ${MAX_DESCRIPTION_LEN} characters or less`);
			return;
		}

		if (topics.length > 0 && editSelectedTopics.length === 0) {
			toast.error('At least one topic must be selected');
			return;
		}

		isLoading = true;
		try {
			await accountApi.updateDevice(
				editingDeviceId!,
				editDeviceName,
				editDescription,
				editSelectedTopics
			);
			toast.success('Device updated');
			await loadDevices();
			editingDeviceId = null;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to update device');
		} finally {
			isLoading = false;
		}
	}

	function toggleEditTopic(topicId: string) {
		if (editSelectedTopics.includes(topicId)) {
			editSelectedTopics = editSelectedTopics.filter((t) => t !== topicId);
		} else {
			editSelectedTopics = [...editSelectedTopics, topicId];
		}
	}

	// Auto-select the general topic when it's the only one (and thus read-only)
	$effect(() => {
		const singleGeneral = topics.length === 1 && topics[0].name === 'general';
		if (singleGeneral && selectedTopics.length === 0) {
			selectedTopics = [topics[0].id];
		}
	});

	$effect(() => {
		const singleGeneral = topics.length === 1 && topics[0].name === 'general';
		if (singleGeneral && editSelectedTopics.length === 0) {
			editSelectedTopics = [topics[0].id];
		}
	});

	function isCreateFormValid(): boolean {
		if (!newDeviceName.trim()) return false;
		if (newDeviceName.length > MAX_DISPLAY_NAME_LEN) return false;
		if (newDescription.length > MAX_DESCRIPTION_LEN) return false;
		if (topics.length > 0 && selectedTopics.length === 0) return false;
		return true;
	}

	function isEditFormValid(): boolean {
		if (!editDeviceName.trim()) return false;
		if (editDeviceName.length > MAX_DISPLAY_NAME_LEN) return false;
		if (editDescription.length > MAX_DESCRIPTION_LEN) return false;
		if (topics.length > 0 && editSelectedTopics.length === 0) return false;
		return true;
	}

	async function goToTokenStep() {
		await goto('/account/api-tokens?onboarding=send-first-message');
	}
</script>

<div class="p-6">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-base-content">Devices</h1>
		<button
			onclick={() => {
				showCreateModal = true;
			}}
			disabled={isLoading}
			class="btn btn-primary"
		>
			<Plus size={20} />
			Add Device
		</button>
	</div>

	<div class="divider my-4"></div>

	<p class="text-sm text-base-content/70 mb-6">Manage your paired devices for push notifications</p>

	{#if shouldShowTokenStep}
		<div class="card bg-primary/5 border border-primary/20 mb-6">
			<div class="card-body gap-3 md:flex-row md:items-center md:justify-between">
				<div>
					<h2 class="card-title text-base-content">Pairing done. Create your API token next.</h2>
					<p class="text-sm text-base-content/70">
						Your device is ready. The next onboarding step is creating a token so you can send a
						first test message.
					</p>
				</div>
				<button class="btn btn-primary gap-2" onclick={() => void goToTokenStep()}>
					<Key size={18} />
					Go to API Tokens
				</button>
			</div>
		</div>
	{/if}

	<SettingsModal
		open={Boolean(createdDevice)}
		title="Pairing Code Generated"
		onClose={closeCreatedModal}
	>
		{#if createdDevice}
			<div class="flex flex-col gap-4 w-full">
				<div class="flex gap-3">
					<div
						class="flex-shrink-0 w-6 h-6 rounded-full bg-primary text-primary-content flex items-center justify-center text-sm font-bold"
					>
						1
					</div>
					<div class="flex-1">
						<span class="text-sm font-medium">Open Hive App</span>
						<div class="mt-1">
							<div class="bg-base-200 border border-base-300 rounded-lg p-2 inline-block">
								<img src={createdDevice.qr_code} alt="QR Code" class="w-24 h-24" />
							</div>
						</div>
						<div class="text-xs text-base-content/60 mt-1">or use link below</div>
						<div class="flex gap-1 mt-1">
							<input
								type="text"
								readonly
								value={createdDevice.pairing_url}
								class="input input-bordered input-xs flex-1 bg-base-100 text-base-content text-xs font-mono"
							/>
							<button
								onclick={copyUrlToClipboard}
								class="btn btn-xs btn-outline btn-square"
								title="Copy URL"
							>
								{#if isCopiedUrl}
									<Check size={12} />
								{:else}
									<Copy size={12} />
								{/if}
							</button>
						</div>
					</div>
				</div>

				<div class="flex gap-3">
					<div
						class="flex-shrink-0 w-6 h-6 rounded-full bg-primary text-primary-content flex items-center justify-center text-sm font-bold"
					>
						2
					</div>
					<div class="flex-1">
						<span class="text-sm font-medium">Install Hive (if required)</span>
						<div class="mt-2 p-3 bg-base-200 rounded-lg text-sm">
							<p class="mb-2 text-base-content/80">
								Installation is <strong>required</strong> on iPhone/iPad and
								<strong>recommended</strong> on desktop for reliable notifications.
							</p>
							<a
								href="/docs/browser-support"
								class="link link-primary text-xs inline-flex items-center gap-1"
								target="_blank"
							>
								View browser requirements
								<ExternalLink size={12} />
							</a>
						</div>
					</div>
				</div>

				<div class="flex gap-3">
					<div
						class="flex-shrink-0 w-6 h-6 rounded-full bg-primary text-primary-content flex items-center justify-center text-sm font-bold"
					>
						3
					</div>
					<div class="flex-1">
						<span class="text-sm font-medium">Enter code</span>
						<div class="flex items-center justify-start gap-2 mt-1">
							<div class="bg-base-200 border border-base-300 rounded-lg px-4 py-2">
								<p class="font-mono text-2xl tracking-widest text-base-content font-bold">
									{createdDevice.pairing_code}
								</p>
							</div>
							<button
								onclick={copyToClipboard}
								class="btn btn-sm btn-outline btn-square"
								title="Copy code"
							>
								{#if isCopiedCode}
									<Check size={16} />
								{:else}
									<Copy size={16} />
								{/if}
							</button>
						</div>
						{#if isExpired}
							<p class="text-sm text-error font-semibold mt-1">Code expired</p>
						{:else}
							<p
								class="text-xs mt-1 {isExpiringSoon
									? 'text-warning font-semibold'
									: 'text-base-content/60'}"
							>
								Expires in <span class="font-mono {isExpiringSoon ? 'text-warning font-bold' : ''}">
									{minutes}:{seconds.toString().padStart(2, '0')}
								</span>
							</p>
						{/if}
					</div>
				</div>
			</div>
		{/if}
		{#snippet actions()}
			{#if isExpired}
				<button type="button" onclick={closeCreatedModal} class="btn btn-sm btn-outline"
					>Close</button
				>
				<button
					type="button"
					onclick={() => void handleRegenerate()}
					disabled={isRegenerating}
					class="btn btn-sm btn-primary gap-2"
				>
					{#if isRegenerating}
						<span class="loading loading-spinner loading-xs"></span>
						Generating...
					{:else}
						<RefreshCw size={16} />
						Generate New Code
					{/if}
				</button>
			{:else}
				<button type="button" onclick={closeCreatedModal} class="btn btn-sm btn-primary"
					>Done</button
				>
			{/if}
		{/snippet}
	</SettingsModal>

	<SettingsModal
		open={showCreateModal}
		title="Add Device"
		onClose={() => {
			showCreateModal = false;
		}}
	>
		<form
			id="create-device-form"
			onsubmit={(e) => {
				e.preventDefault();
				void handleCreate();
			}}
			class="space-y-4"
		>
			<div>
				<div class="mb-2 flex items-center justify-between gap-3">
					<label for="device-name" class="text-sm font-semibold text-base-content">
						Device Name
					</label>
					<p
						class="text-xs tabular-nums whitespace-nowrap"
						class:text-base-content-70={newDeviceName.length <= MAX_DISPLAY_NAME_SOFT_LEN}
						class:text-warning={newDeviceName.length > MAX_DISPLAY_NAME_SOFT_LEN &&
							newDeviceName.length <= MAX_DISPLAY_NAME_LEN}
						class:text-error={newDeviceName.length > MAX_DISPLAY_NAME_LEN}
					>
						({newDeviceName.length}/{MAX_DISPLAY_NAME_LEN})
					</p>
				</div>
				<input
					type="text"
					id="device-name"
					placeholder="e.g., Chrome on MacBook, Firefox on Phone"
					class="input input-bordered w-full bg-base-100 text-base-content"
					bind:value={newDeviceName}
					disabled={isLoading}
					maxlength={MAX_DISPLAY_NAME_LEN}
					required
				/>
			</div>

			<div>
				<div class="mb-2 flex items-center justify-between gap-3">
					<label for="device-description" class="text-sm font-semibold text-base-content">
						Description
					</label>
					<p class="text-xs text-base-content/70 tabular-nums whitespace-nowrap">
						({newDescription.length}/{MAX_DESCRIPTION_LEN})
					</p>
				</div>
				<textarea
					id="device-description"
					placeholder="Add a description for this device..."
					class="textarea textarea-bordered w-full bg-base-100 text-base-content"
					bind:value={newDescription}
					disabled={isLoading}
					maxlength={MAX_DESCRIPTION_LEN}
					rows="2"
				></textarea>
			</div>

			{#if topics.length > 0}
				<fieldset>
					<legend class="block text-sm font-semibold text-base-content mb-2">
						Topics
						<span class="text-xs font-normal text-base-content/70"
							>At least one topic must be selected</span
						>
					</legend>
					<div class="space-y-2">
						{#each topics as topic (topic.id)}
							{@const isReadOnly = topic.name === 'general' && topics.length === 1}
							<label
								class={`flex items-center gap-2 ${isReadOnly ? 'opacity-70' : 'cursor-pointer'}`}
							>
								<input
									type="checkbox"
									checked={selectedTopics.includes(topic.id)}
									onchange={() => !isReadOnly && toggleTopic(topic.id)}
									disabled={isLoading || isReadOnly}
									class="checkbox checkbox-sm"
								/>
								<TopicChip name={topic.name} muted={isReadOnly} />
								{#if topic.name === 'general'}
									<span class="text-xs text-base-content/70">(default)</span>
								{/if}
							</label>
						{/each}
					</div>
				</fieldset>
			{/if}
		</form>
		{#snippet actions()}
			<button
				type="button"
				class="btn btn-outline"
				onclick={() => {
					showCreateModal = false;
				}}
				disabled={isLoading}
			>
				Cancel
			</button>
			<button
				type="submit"
				class="btn btn-primary"
				disabled={isLoading || !isCreateFormValid()}
				form="create-device-form"
			>
				{#if isLoading}
					<span class="loading loading-spinner loading-sm"></span>
					Generating...
				{:else}
					Generate Pairing Code
				{/if}
			</button>
		{/snippet}
	</SettingsModal>

	<div class="space-y-4">
		{#if isLoading && devices.length === 0}
			<div class="flex justify-center py-8">
				<span class="loading loading-spinner loading-lg text-primary"></span>
			</div>
		{:else if devices.length === 0}
			<div class="alert alert-info bg-base-200 border border-base-300">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					class="stroke-current shrink-0 w-6 h-6"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					></path>
				</svg>
				<div class="flex-1">
					<h3 class="font-bold text-base-content">No devices yet</h3>
					<div class="text-sm text-base-content/70 mb-3">
						Add a device to generate a pairing code
					</div>
					<div class="flex gap-2">
						<button
							type="button"
							class="btn btn-primary btn-sm"
							onclick={() => (showCreateModal = true)}
						>
							<Plus size={16} />
							Add Your First Device
						</button>
						<a href="/docs/quickstart#2-pair-your-first-device" class="btn btn-ghost btn-sm">
							Learn how
						</a>
					</div>
				</div>
			</div>
		{:else}
			<div
				class="rounded-xl border border-base-300 bg-base-200/60 p-4 text-sm text-base-content/80"
			>
				BeeBuzz stores only public age recipients for paired devices. Compare these values when you
				expect a specific device or after re-pairing, so unexpected key changes are easier to spot.
			</div>
			{#each devices as device (device.id)}
				{#if editingDeviceId === device.id}
					<div class="card bg-base-100 shadow border border-base-300 p-4">
						<form
							onsubmit={(e) => {
								e.preventDefault();
								void handleEdit();
							}}
							class="space-y-4"
						>
							<div>
								<div class="mb-2 flex items-center justify-between gap-3">
									<label
										for="edit-device-name-{device.id}"
										class="text-sm font-semibold text-base-content"
									>
										Device Name
									</label>
									<p
										class="text-xs tabular-nums whitespace-nowrap"
										class:text-base-content-70={editDeviceName.length <= MAX_DISPLAY_NAME_SOFT_LEN}
										class:text-warning={editDeviceName.length > MAX_DISPLAY_NAME_SOFT_LEN &&
											editDeviceName.length <= MAX_DISPLAY_NAME_LEN}
										class:text-error={editDeviceName.length > MAX_DISPLAY_NAME_LEN}
									>
										({editDeviceName.length}/{MAX_DISPLAY_NAME_LEN})
									</p>
								</div>
								<input
									type="text"
									id="edit-device-name-{device.id}"
									placeholder="e.g., Chrome on MacBook"
									class="input input-bordered w-full bg-base-100 text-base-content"
									bind:value={editDeviceName}
									disabled={isLoading}
									maxlength={MAX_DISPLAY_NAME_LEN}
									required
								/>
							</div>

							<div>
								<div class="mb-2 flex items-center justify-between gap-3">
									<label
										for="edit-device-desc-{device.id}"
										class="text-sm font-semibold text-base-content"
									>
										Description
									</label>
									<p class="text-xs text-base-content/70 tabular-nums whitespace-nowrap">
										({editDescription.length}/{MAX_DESCRIPTION_LEN})
									</p>
								</div>
								<textarea
									id="edit-device-desc-{device.id}"
									placeholder="Add a description for this device..."
									class="textarea textarea-bordered w-full bg-base-100 text-base-content"
									bind:value={editDescription}
									disabled={isLoading}
									maxlength={MAX_DESCRIPTION_LEN}
									rows="2"
								></textarea>
							</div>

							{#if topics.length > 0}
								<fieldset>
									<legend class="block text-sm font-semibold text-base-content mb-2">
										Topics
										<span class="text-xs font-normal text-base-content/70"
											>At least one topic must be selected</span
										>
									</legend>
									<div class="space-y-2">
										{#each topics as topic (topic.id)}
											{@const isReadOnly = topic.name === 'general' && topics.length === 1}
											<label
												class={`flex items-center gap-2 ${isReadOnly ? 'opacity-70' : 'cursor-pointer'}`}
											>
												<input
													type="checkbox"
													checked={editSelectedTopics.includes(topic.id)}
													onchange={() => !isReadOnly && toggleEditTopic(topic.id)}
													disabled={isLoading || isReadOnly}
													class="checkbox checkbox-sm"
												/>
												<TopicChip name={topic.name} muted={isReadOnly} />
												{#if topic.name === 'general'}
													<span class="text-xs text-base-content/70">(default)</span>
												{/if}
											</label>
										{/each}
									</div>
								</fieldset>
							{/if}

							<div class="flex flex-col gap-2 pt-4 sm:flex-row sm:justify-end">
								<button
									type="button"
									class="btn btn-outline btn-sm"
									onclick={cancelEdit}
									disabled={isLoading}
								>
									Cancel
								</button>
								<button
									type="submit"
									class="btn btn-primary btn-sm"
									disabled={isLoading || !isEditFormValid()}
								>
									{#if isLoading}
										<span class="loading loading-spinner loading-sm"></span>
										Saving...
									{:else}
										Save Changes
									{/if}
								</button>
							</div>
						</form>
					</div>
				{:else}
					<div class="card bg-base-100 shadow border border-base-300 p-4">
						<div class="flex justify-between items-start">
							<div class="flex-1">
								<div class="flex items-center gap-2">
									<p class="font-semibold text-base-content">{device.name || 'Device'}</p>
									<span class="badge badge-sm {deviceStatusBadgeClass(device)}">
										{deviceStatusLabel(device)}
									</span>
								</div>
								<p class="text-xs text-base-content/70 mt-1">
									Created on {new Date(device.created_at).toLocaleDateString()}
								</p>
								{#if device.description}
									<p class="text-sm text-base-content mt-2">{device.description}</p>
								{/if}
								{#if device.topic_ids && device.topic_ids.length > 0}
									<div class="flex flex-wrap gap-2 mt-3">
										{#each device.topic_ids as topicId, idx (idx)}
											<TopicChip name={topics.find((t) => t.id === topicId)?.name ?? topicId} />
										{/each}
									</div>
								{:else if device.topic_ids !== undefined}
									<p class="text-xs text-base-content/70 mt-2">No topics</p>
								{/if}
								{#if deviceIsPaired(device) && device.age_recipient}
									<div class="mt-4 rounded-lg border border-base-300 bg-base-200/60 p-3">
										<div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
											<div class="min-w-0 flex-1">
												<p
													class="text-xs font-semibold uppercase tracking-wide text-base-content/60"
												>
													Public age key
												</p>
												<p class="mt-1 text-sm text-base-content/80">
													Fingerprint:
													<span class="font-mono text-base-content">
														{device.age_recipient_fingerprint}
													</span>
												</p>
												{#if revealedKeys[device.id]}
													<p class="mt-2 break-all font-mono text-xs text-base-content">
														{device.age_recipient}
													</p>
												{/if}
											</div>
											<div class="flex flex-wrap gap-2">
												<button
													type="button"
													class="btn btn-xs btn-outline"
													onclick={() => toggleRevealAgeRecipient(device.id)}
												>
													{revealedKeys[device.id] ? 'Hide Key' : 'Reveal Key'}
												</button>
												<button
													type="button"
													class="btn btn-xs btn-outline"
													onclick={() => void copyAgeRecipient(device)}
												>
													{#if copiedKeyByDevice[device.id]}
														<Check size={14} />
														Copied
													{:else}
														<Copy size={14} />
														Copy Key
													{/if}
												</button>
											</div>
										</div>
									</div>
								{/if}
							</div>
							<div class="relative">
								<button
									type="button"
									onclick={() => toggleDeviceActions(device.id)}
									disabled={isLoading}
									class="btn btn-ghost btn-circle btn-xs text-base-content/60"
									aria-label={`Open actions for ${device.name || 'device'}`}
									aria-expanded={actionsMenuOpenDeviceId === device.id}
								>
									<EllipsisVertical size={14} />
								</button>
								{#if actionsMenuOpenDeviceId === device.id}
									<ul
										role="menu"
										bind:this={actionsMenuRef}
										class="menu absolute right-0 z-20 mt-1 w-48 rounded-box border border-base-300 bg-base-100 p-2 shadow"
									>
										<li>
											<button type="button" onclick={() => generateCodeForDevice(device)}>
												<RefreshCw size={16} />
												{deviceIsPaired(device) ? 'Regenerate code' : 'Generate code'}
											</button>
										</li>
										<li>
											<button type="button" onclick={() => startEdit(device)}>
												<Pencil size={16} />
												Edit
											</button>
										</li>
										<li>
											<button
												type="button"
												class="text-error"
												onclick={() => openDeleteDialog(device)}
											>
												<Trash2 size={16} />
												Delete
											</button>
										</li>
									</ul>
								{/if}
							</div>
						</div>
					</div>
				{/if}
			{/each}
		{/if}
	</div>

	<SettingsModal
		open={Boolean(devicePendingDelete)}
		title="Remove Device"
		description={devicePendingDelete
			? `Remove "${devicePendingDelete.name || 'Device'}"? It will stop receiving notifications.`
			: undefined}
		onClose={() => {
			devicePendingDelete = null;
		}}
		size="sm"
	>
		<p class="text-sm text-base-content/70">
			The device will stop receiving notifications immediately.
		</p>
		{#snippet actions()}
			<button type="button" class="btn btn-outline" onclick={() => (devicePendingDelete = null)}>
				Cancel
			</button>
			<button type="button" class="btn btn-error" onclick={handleDelete}>Remove Device</button>
		{/snippet}
	</SettingsModal>
</div>
