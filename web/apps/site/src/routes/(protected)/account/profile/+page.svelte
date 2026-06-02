<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from '@beebuzz/shared/stores';
	import { me } from '@beebuzz/shared/services/account';
	import type { AuthUser } from '@beebuzz/shared/api';
	import { Copy, Mail, Shield, ShieldCheck } from '@lucide/svelte';

	let user = $state<AuthUser | null>(null);

	onMount(async () => {
		try {
			user = await me();
		} catch {
			user = null;
		}
	});

	async function handleCopyUserId() {
		if (!user?.id) return;

		try {
			await navigator.clipboard.writeText(user.id);
			toast.success('User ID copied');
		} catch {
			toast.error('Failed to copy user ID');
		}
	}
</script>

<div class="space-y-6">
	<div class="max-w-3xl">
		<h1 class="text-3xl font-bold text-base-content">Profile</h1>
		<p class="mt-2 text-base-content/70">View your account details and technical information.</p>
	</div>

	{#if user}
		<div class="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(18rem,1fr)]">
			<div class="card border border-base-300 bg-base-200">
				<div class="card-body">
					<div class="flex items-start gap-4">
						<div
							class="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/10 text-primary"
						>
							<Mail size={22} />
						</div>
						<div>
							<h2 class="card-title text-lg text-base-content">Account details</h2>
							<p class="mt-1 text-sm text-base-content/65">Email and access information.</p>
						</div>
					</div>

					<div class="mt-6 space-y-5">
						<div>
							<div class="text-sm font-semibold text-base-content/70">Email address</div>
							<div class="mt-2 flex flex-wrap items-center gap-2">
								<p class="text-base font-medium text-base-content">{user.email}</p>
								{#if user.is_admin}
									<span class="badge badge-info gap-1 font-semibold">
										<ShieldCheck size={14} />
										Admin
									</span>
								{/if}
							</div>
						</div>
					</div>
				</div>
			</div>

			<div class="card border border-base-300 bg-base-100">
				<div class="card-body">
					<div class="flex items-start gap-3">
						<div
							class="flex h-10 w-10 items-center justify-center rounded-xl bg-base-200 text-base-content/70"
						>
							<Shield size={18} />
						</div>
						<div>
							<h2 class="card-title text-base text-base-content">Technical details</h2>
							<p class="mt-1 text-sm text-base-content/65">Useful when contacting support.</p>
						</div>
					</div>

					<div class="mt-5">
						<div class="text-sm font-semibold text-base-content/70">User ID</div>
						<div
							class="mt-2 flex items-center gap-2 rounded-xl border border-base-300 bg-base-200 px-3 py-2"
						>
							<code class="min-w-0 flex-1 truncate text-xs text-base-content/80">{user.id}</code>
							<button
								type="button"
								class="btn btn-ghost btn-xs"
								onclick={handleCopyUserId}
								aria-label="Copy user ID"
							>
								<Copy size={14} />
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>
	{:else}
		<div class="py-8 text-center">
			<p class="text-base-content/70">Loading profile...</p>
		</div>
	{/if}
</div>
