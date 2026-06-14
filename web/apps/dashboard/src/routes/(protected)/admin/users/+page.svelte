<script lang="ts">
	import { toast } from '@beebuzz/shared/stores';
	import { onMount } from 'svelte';
	import { Users, Search, Loader, Check, X, RotateCcw } from '@lucide/svelte';
	import {
		adminApi,
		userStatusLabel,
		userStatusBadgeClass,
		userActionInfo,
		userTargetStatusForAction,
		type AdminUser
	} from '@beebuzz/shared/api';
	import { ApiError } from '@beebuzz/shared/errors';

	let users: AdminUser[] = $state([]);
	let loading = $state(true);
	let searchQuery = $state('');

	let selectedUser: AdminUser | null = $state(null);
	let modalAction: 'approve' | 'block' | 'reactivate' | null = $state(null);
	let actionLoading = $state(false);

	onMount(async () => {
		await loadUsers();
	});

	async function loadUsers() {
		try {
			users = await adminApi.listUsers();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Failed to load users');
		} finally {
			loading = false;
		}
	}

	let filteredUsers = $derived(
		users.filter((user) => user.email.toLowerCase().includes(searchQuery.toLowerCase()))
	);

	function formatTrialEndDate(trialStartedAt: string): string {
		const start = new Date(trialStartedAt);
		const end = new Date(start.getTime() + 14 * 24 * 60 * 60 * 1000);
		return end.toLocaleDateString();
	}

	function openModal(user: AdminUser, action: 'approve' | 'block' | 'reactivate') {
		selectedUser = user;
		modalAction = action;
	}

	function closeModal() {
		selectedUser = null;
		modalAction = null;
	}

	async function confirmAction() {
		if (!selectedUser || !modalAction) return;

		actionLoading = true;
		const targetStatus = userTargetStatusForAction(modalAction);

		try {
			await adminApi.updateUserStatus(selectedUser.id, targetStatus);
			toast.success(
				`User ${modalAction === 'approve' ? 'approved' : modalAction === 'block' ? 'blocked' : 'reactivated'} successfully`
			);
			await loadUsers();
			closeModal();
		} catch (err) {
			if (err instanceof ApiError && err.status === 409) {
				toast.error("This user's status was changed by another admin. Please refresh the page.");
			} else {
				toast.error(err instanceof ApiError ? err.userMessage : 'Failed to update user status');
			}
		} finally {
			actionLoading = false;
		}
	}
</script>

<div>
	<div class="flex items-center justify-between mb-6">
		<div>
			<h2 class="text-2xl font-bold text-base-content">Users Management</h2>
			<p class="text-sm text-base-content/70 mt-1">Manage registered users</p>
		</div>
		<div class="badge badge-lg font-semibold gap-1 bg-primary/20 text-primary border-0">
			<Users size={16} />
			{users.length}
			{users.length === 1 ? 'user' : 'users'}
		</div>
	</div>

	<!-- Search Bar -->
	<div class="mb-6 flex gap-2">
		<div class="join flex-1">
			<span class="join-item bg-base-200 px-4 flex items-center">
				<Search size={20} class="text-base-content/50" />
			</span>
			<input
				type="text"
				placeholder="Search by email..."
				class="input input-bordered join-item flex-1"
				bind:value={searchQuery}
			/>
		</div>
	</div>

	<!-- Users Table -->
	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="text-center">
				<Loader size={32} class="animate-spin text-primary mx-auto mb-2" />
				<p class="text-base-content/70">Loading users...</p>
			</div>
		</div>
	{:else if filteredUsers.length === 0}
		<div class="card bg-base-200 border border-base-300">
			<div class="card-body items-center text-center">
				<Users size={48} class="text-base-content/30 mb-4" />
				<h3 class="text-lg font-semibold text-base-content">No users found</h3>
				<p class="text-sm text-base-content/70">
					{searchQuery ? 'Try adjusting your search' : 'No users registered yet'}
				</p>
			</div>
		</div>
	{:else}
		<div class="overflow-x-auto border border-base-300 rounded-lg shadow">
			<table class="table w-full">
				<thead class="bg-base-200 border-b border-base-300">
					<tr>
						<th class="text-base-content font-semibold">Email</th>
						<th class="text-base-content font-semibold">Type</th>
						<th class="text-base-content font-semibold">Status</th>
						<th class="text-base-content font-semibold">Reason</th>
						<th class="text-base-content font-semibold">Trial</th>
						<th class="text-base-content font-semibold text-right">Joined</th>
						<th class="text-base-content font-semibold text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-base-300">
					{#each filteredUsers as user (user.id)}
						<tr class="hover:bg-base-200/50 transition-colors">
							<td class="text-base-content font-medium">{user.email}</td>
							<td>
								<span
									class={`badge badge-sm font-semibold ${user.is_admin ? 'badge-info' : 'badge-success'}`}
								>
									{user.is_admin ? 'Admin' : 'User'}
								</span>
							</td>
							<td>
								<span class="badge badge-sm {userStatusBadgeClass(user)}">
									{userStatusLabel(user)}
								</span>
							</td>
							<td class="text-base-content/70 text-sm">
								{user.signup_reason ? user.signup_reason : '-'}
							</td>
							<td class="text-base-content/70 text-sm">
								{#if user.trial_started_at}
									Trial ends: {formatTrialEndDate(user.trial_started_at)}
								{:else}
									-
								{/if}
							</td>
							<td class="text-base-content/70 text-sm text-right">
								{user.created_at ? new Date(user.created_at).toLocaleDateString() : '-'}
							</td>
							<td class="text-right">
								{#if !user.is_admin}
									{@const actionInfo = userActionInfo(user)}
									{#if actionInfo.action}
										<button
											class={`btn btn-sm ${actionInfo.class}`}
											onclick={() =>
												openModal(user, actionInfo.action as 'approve' | 'block' | 'reactivate')}
										>
											{#if actionInfo.action === 'approve'}
												<Check size={14} />
											{:else if actionInfo.action === 'block'}
												<X size={14} />
											{:else}
												<RotateCcw size={14} />
											{/if}
											{actionInfo.label}
										</button>
									{/if}
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<!-- Pagination Info -->
		<div class="mt-4 text-sm text-base-content/70 text-center">
			Showing {filteredUsers.length} of {users.length}
			{users.length === 1 ? 'user' : 'users'}
		</div>
	{/if}
</div>

<!-- Confirmation Modal -->
{#if selectedUser && modalAction}
	<div class="modal modal-open">
		<div class="modal-box">
			<h3 class="font-bold text-lg">
				{modalAction === 'approve'
					? 'Approve User'
					: modalAction === 'block'
						? 'Block User'
						: 'Reactivate User'}
			</h3>
			<p class="py-4">
				{#if modalAction === 'approve'}
					Are you sure you want to approve <strong>{selectedUser.email}</strong>? They will be able
					to sign in.
				{:else if modalAction === 'block'}
					Are you sure you want to block <strong>{selectedUser.email}</strong>? They will lose
					access immediately.
				{:else}
					Are you sure you want to reactivate <strong>{selectedUser.email}</strong>? They will
					regain access.
				{/if}
			</p>
			<div class="modal-action flex flex-col gap-2 sm:flex-row sm:justify-end">
				<button type="button" class="btn btn-outline" onclick={closeModal} disabled={actionLoading}>
					Cancel
				</button>
				<button
					type="button"
					class={`btn ${modalAction === 'approve' ? 'btn-success' : modalAction === 'block' ? 'btn-error' : 'btn-warning'}`}
					onclick={confirmAction}
					disabled={actionLoading}
				>
					{#if actionLoading}
						<Loader size={16} class="animate-spin" />
					{/if}
					{modalAction === 'approve' ? 'Approve' : modalAction === 'block' ? 'Block' : 'Reactivate'}
				</button>
			</div>
		</div>
		<button class="modal-backdrop" type="button" aria-label="Close" onclick={closeModal}></button>
	</div>
{/if}
