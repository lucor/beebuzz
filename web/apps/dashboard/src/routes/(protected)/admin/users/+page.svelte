<script lang="ts">
	import { toast } from '@beebuzz/shared/stores';
	import { onMount } from 'svelte';
	import { Users, Search, Loader, X, RotateCcw } from '@lucide/svelte';
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
	let accountStatusFilter = $state<'all' | AdminUser['account_status']>('all');
	let planFilter = $state<'all' | AdminUser['plan']>('all');
	let subscriptionFilter = $state<'all' | 'none' | NonNullable<AdminUser['subscription_status']>>(
		'all'
	);
	let createdFrom = $state('');
	let createdTo = $state('');

	let selectedUser: AdminUser | null = $state(null);
	let modalAction: 'suspend' | 'reactivate' | null = $state(null);
	let actionLoading = $state(false);
	const numberFormatter = new Intl.NumberFormat();

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

	let hasActiveFilters = $derived(
		searchQuery.trim() !== '' ||
			accountStatusFilter !== 'all' ||
			planFilter !== 'all' ||
			subscriptionFilter !== 'all' ||
			createdFrom !== '' ||
			createdTo !== ''
	);

	let filteredUsers = $derived(
		users.filter((user) => {
			const emailMatches = user.email.toLowerCase().includes(searchQuery.trim().toLowerCase());
			const accountStatusMatches =
				accountStatusFilter === 'all' || user.account_status === accountStatusFilter;
			const planMatches = planFilter === 'all' || user.plan === planFilter;
			const subscriptionMatches =
				subscriptionFilter === 'all' ||
				(subscriptionFilter === 'none' && !user.subscription_status) ||
				user.subscription_status === subscriptionFilter;
			const createdAt = new Date(user.created_at).getTime();
			const createdFromMatches = createdFrom === '' || createdAt >= startOfDay(createdFrom);
			const createdToMatches = createdTo === '' || createdAt <= endOfDay(createdTo);

			return (
				emailMatches &&
				accountStatusMatches &&
				planMatches &&
				subscriptionMatches &&
				createdFromMatches &&
				createdToMatches
			);
		})
	);

	function planLabel(plan: AdminUser['plan']): string {
		return plan === 'hosted' ? 'Hosted' : 'Free';
	}

	function planBadgeClass(plan: AdminUser['plan']): string {
		return plan === 'hosted' ? 'badge-primary' : 'badge-ghost';
	}

	function subscriptionLabel(status: AdminUser['subscription_status']): string {
		switch (status) {
			case 'active':
				return 'Active';
			case 'scheduled_cancel':
				return 'Scheduled cancel';
			case 'past_due':
				return 'Past due';
			case 'canceled':
				return 'Canceled';
			case 'expired':
				return 'Expired';
			case 'incomplete':
				return 'Incomplete';
			default:
				return '-';
		}
	}

	function subscriptionBadgeClass(status: AdminUser['subscription_status']): string {
		switch (status) {
			case 'active':
			case 'scheduled_cancel':
				return 'badge-success';
			case 'past_due':
				return 'badge-warning';
			case 'canceled':
			case 'expired':
				return 'badge-error';
			case 'incomplete':
				return 'badge-ghost';
			default:
				return 'badge-ghost';
		}
	}

	function formatDate(value: string | null | undefined): string {
		if (!value) return '-';
		return new Intl.DateTimeFormat('en-GB', {
			day: '2-digit',
			month: 'short',
			year: 'numeric'
		}).format(new Date(value));
	}

	function formatNumber(value: number): string {
		return numberFormatter.format(value);
	}

	function startOfDay(value: string): number {
		const date = new Date(`${value}T00:00:00`);
		return date.getTime();
	}

	function endOfDay(value: string): number {
		const date = new Date(`${value}T23:59:59.999`);
		return date.getTime();
	}

	function resetFilters() {
		searchQuery = '';
		accountStatusFilter = 'all';
		planFilter = 'all';
		subscriptionFilter = 'all';
		createdFrom = '';
		createdTo = '';
	}

	function openModal(user: AdminUser, action: 'suspend' | 'reactivate') {
		selectedUser = user;
		modalAction = action;
	}

	function openActionModal(user: AdminUser, action: 'suspend' | 'reactivate' | null) {
		if (!action) return;
		openModal(user, action);
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
			toast.success(`User ${modalAction === 'suspend' ? 'suspended' : 'reactivated'} successfully`);
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

	<div class="mb-6 space-y-3">
		<div class="flex gap-2">
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
			<button
				type="button"
				class="btn btn-outline"
				disabled={!hasActiveFilters}
				onclick={resetFilters}
			>
				<RotateCcw size={16} />
				Reset
			</button>
		</div>

		<div class="grid gap-3 md:grid-cols-5">
			<label class="form-control">
				<span class="label pb-1">
					<span class="label-text text-xs font-semibold text-base-content/70">Account status</span>
				</span>
				<select class="select select-bordered select-sm" bind:value={accountStatusFilter}>
					<option value="all">All statuses</option>
					<option value="active">Active</option>
					<option value="blocked">Suspended</option>
				</select>
			</label>

			<label class="form-control">
				<span class="label pb-1">
					<span class="label-text text-xs font-semibold text-base-content/70">Plan</span>
				</span>
				<select class="select select-bordered select-sm" bind:value={planFilter}>
					<option value="all">All plans</option>
					<option value="free">Free</option>
					<option value="hosted">Hosted</option>
				</select>
			</label>

			<label class="form-control">
				<span class="label pb-1">
					<span class="label-text text-xs font-semibold text-base-content/70">Subscription</span>
				</span>
				<select class="select select-bordered select-sm" bind:value={subscriptionFilter}>
					<option value="all">All subscriptions</option>
					<option value="none">None</option>
					<option value="incomplete">Incomplete</option>
					<option value="active">Active</option>
					<option value="scheduled_cancel">Scheduled cancel</option>
					<option value="past_due">Past due</option>
					<option value="canceled">Canceled</option>
					<option value="expired">Expired</option>
				</select>
			</label>

			<label class="form-control">
				<span class="label pb-1">
					<span class="label-text text-xs font-semibold text-base-content/70">Created from</span>
				</span>
				<input type="date" class="input input-bordered input-sm" bind:value={createdFrom} />
			</label>

			<label class="form-control">
				<span class="label pb-1">
					<span class="label-text text-xs font-semibold text-base-content/70">Created to</span>
				</span>
				<input type="date" class="input input-bordered input-sm" bind:value={createdTo} />
			</label>
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
					{hasActiveFilters ? 'Try adjusting your filters' : 'No users registered yet'}
				</p>
			</div>
		</div>
	{:else}
		<div class="overflow-x-auto border border-base-300 rounded-lg shadow">
			<table class="table w-full">
				<thead class="bg-base-200 border-b border-base-300">
					<tr>
						<th class="text-base-content font-semibold">Email</th>
						<th class="text-base-content font-semibold">Status</th>
						<th class="text-base-content font-semibold">Plan</th>
						<th class="text-base-content font-semibold">Subscription</th>
						<th class="text-base-content font-semibold text-right">Usage this month</th>
						<th class="text-base-content font-semibold">Plan expires</th>
						<th class="text-base-content font-semibold text-right">Joined</th>
						<th class="text-base-content font-semibold text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-base-300">
					{#each filteredUsers as user (user.id)}
						<tr class="hover:bg-base-200/50 transition-colors">
							<td class="text-base-content font-medium">
								<div class="flex items-center gap-2">
									<span class="break-all">{user.email}</span>
									{#if user.is_admin}
										<span class="badge badge-info badge-sm shrink-0 font-semibold">Admin</span>
									{/if}
								</div>
							</td>
							<td>
								<span class="badge badge-sm {userStatusBadgeClass(user)}">
									{userStatusLabel(user)}
								</span>
							</td>
							<td>
								<span class="badge badge-sm {planBadgeClass(user.plan)}">
									{planLabel(user.plan)}
								</span>
							</td>
							<td>
								<span
									class="badge badge-sm whitespace-nowrap {subscriptionBadgeClass(
										user.subscription_status
									)}"
								>
									{subscriptionLabel(user.subscription_status)}
								</span>
							</td>
							<td class="text-base-content/70 text-sm text-right">
								{formatNumber(user.usage_this_month)}
							</td>
							<td class="text-base-content/70 text-sm">
								{formatDate(user.plan_expires_at)}
							</td>
							<td class="text-base-content/70 text-sm text-right">
								{formatDate(user.created_at)}
							</td>
							<td class="text-right">
								{#if !user.is_admin}
									{@const actionInfo = userActionInfo(user)}
									{#if actionInfo.action}
										<button
											class={`btn btn-sm ${actionInfo.class}`}
											onclick={() => openActionModal(user, actionInfo.action)}
										>
											{#if actionInfo.action === 'suspend'}
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
				{modalAction === 'suspend' ? 'Suspend Account' : 'Reactivate Account'}
			</h3>
			<p class="py-4">
				{#if modalAction === 'suspend'}
					Suspend <strong>{selectedUser.email}</strong>? The user will be signed out and lose access
					to the API, CLI, and Hive. Their plan and subscription are unaffected.
				{:else}
					Reactivate <strong>{selectedUser.email}</strong>? The user will regain access with their
					existing plan.
				{/if}
			</p>
			<div class="modal-action flex flex-col gap-2 sm:flex-row sm:justify-end">
				<button type="button" class="btn btn-outline" onclick={closeModal} disabled={actionLoading}>
					Cancel
				</button>
				<button
					type="button"
					class={`btn ${modalAction === 'suspend' ? 'btn-error' : 'btn-warning'}`}
					onclick={confirmAction}
					disabled={actionLoading}
				>
					{#if actionLoading}
						<Loader size={16} class="animate-spin" />
					{/if}
					{modalAction === 'suspend' ? 'Suspend account' : 'Reactivate'}
				</button>
			</div>
		</div>
		<button class="modal-backdrop" type="button" aria-label="Close" onclick={closeModal}></button>
	</div>
{/if}
