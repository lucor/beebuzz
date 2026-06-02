<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { toast } from '@beebuzz/shared/stores';
	import { logout } from '@beebuzz/shared/services/auth';
	import { ApiError } from '@beebuzz/shared/errors';
	import { BeeBuzzLogo, HealthStatus } from '@beebuzz/shared/components';
	import { auth } from '@beebuzz/shared/stores';
	import {
		Menu,
		X,
		LogOut,
		User,
		Tag,
		Smartphone,
		Key,
		Webhook,
		Shield,
		LayoutDashboard
	} from '@lucide/svelte';
	import type { Icon } from '@lucide/svelte';

	let { children }: { children: import('svelte').Snippet } = $props();

	type NavItem = {
		label: string;
		href: string;
		icon: typeof Icon;
	};
	let sidebarOpen = $state(false);

	let currentPath = $derived(page.url.pathname);

	const navItems: NavItem[] = [
		{ label: 'Overview', href: resolve('/account/overview'), icon: LayoutDashboard },
		{ label: 'Topics', href: resolve('/account/topics'), icon: Tag },
		{ label: 'Devices', href: resolve('/account/devices'), icon: Smartphone },
		{ label: 'API Tokens', href: resolve('/account/api-tokens'), icon: Key },
		{ label: 'Webhooks', href: resolve('/account/webhooks'), icon: Webhook }
	];

	const profileHref = resolve('/account/profile');
	let accountMenuOpen = $state(false);

	async function handleLogout() {
		try {
			await logout();
			toast.success('Logged out successfully');
			await goto(resolve('/login'));
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Logout failed');
			await goto(resolve('/login'));
		}
	}
</script>

<div class="flex flex-col h-screen">
	<!-- Fixed Navbar -->
	<nav class="navbar bg-base-100 shadow-sm fixed top-0 left-0 right-0 z-50 px-4 md:px-8">
		<!-- Left: Hamburger + Logo -->
		<div class="flex-1 flex items-center gap-4">
			<button
				aria-label="Toggle sidebar"
				class="btn btn-square btn-ghost lg:hidden"
				onclick={() => (sidebarOpen = !sidebarOpen)}
			>
				{#if sidebarOpen}
					<X size={24} />
				{:else}
					<Menu size={24} />
				{/if}
			</button>

			<a href={resolve('/')} class="flex items-center gap-2 hidden sm:flex">
				<BeeBuzzLogo variant="img" class="w-10 h-10" />
				<BeeBuzzLogo variant="text" class="w-24 h-8 hidden md:block" />
			</a>
		</div>

		<!-- Center: Logo (mobile only) -->
		<div class="navbar-center sm:hidden">
			<a href={resolve('/')} class="flex items-center gap-2">
				<BeeBuzzLogo variant="img" class="w-10 h-10" />
			</a>
		</div>

		<!-- Right: System status + account actions -->
		<div class="navbar-end gap-2">
			<div
				class="hidden sm:flex items-center rounded-full border border-base-300 bg-base-100 px-3 py-1.5 text-sm text-base-content/70"
			>
				<HealthStatus />
			</div>

			<div class="dropdown dropdown-end">
				<button
					type="button"
					tabindex="0"
					class="btn btn-ghost btn-circle border border-base-300"
					aria-label="Open account menu"
					aria-expanded={accountMenuOpen}
					onclick={() => (accountMenuOpen = !accountMenuOpen)}
					onblur={() => (accountMenuOpen = false)}
				>
					<User size={20} />
				</button>

				<ul
					class="dropdown-content menu z-50 mt-3 w-64 rounded-xl border border-base-300 bg-base-100 p-2 shadow-xl"
				>
					<li class="menu-title px-3 py-2">
						<span class="truncate text-xs font-medium text-base-content/75">
							{auth.user?.email ?? 'Account'}
						</span>
					</li>
					<li>
						<a href={profileHref} onclick={() => (accountMenuOpen = false)}>
							<User size={16} aria-hidden="true" />
							Profile
						</a>
					</li>
					<li aria-hidden="true" class="pointer-events-none my-1 border-t border-base-300"></li>
					<li>
						<button type="button" onclick={handleLogout}>
							<LogOut size={16} aria-hidden="true" />
							Logout
						</button>
					</li>
				</ul>
			</div>
		</div>
	</nav>

	<!-- Main Layout with Sidebar -->
	<div class="flex flex-1 pt-16 overflow-hidden">
		<!-- Sidebar Overlay (mobile) -->
		{#if sidebarOpen}
			<button
				class="fixed inset-0 bg-black/50 z-30 lg:hidden"
				onclick={() => (sidebarOpen = false)}
				aria-label="Close sidebar"
				type="button"
			></button>
		{/if}

		<!-- Sidebar -->
		<aside
			class="fixed left-0 top-16 bottom-0 w-64 bg-base-200 border-r border-base-300 shadow-lg transition-transform duration-300 z-40 lg:relative lg:top-0 overflow-y-auto
				{sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}"
		>
			<div class="flex min-h-full flex-col">
				<ul class="menu w-full flex-1 gap-2 p-4 md:p-8 text-base-content">
					{#each navItems as item (item.href)}
						<li>
							<a
								href={item.href}
								class={`rounded-lg transition-colors ${
									currentPath === item.href
										? 'bg-primary text-primary-content font-semibold'
										: 'hover:bg-base-300'
								}`}
								onclick={() => (sidebarOpen = false)}
							>
								<item.icon size={20} aria-hidden="true" />
								<span>{item.label}</span>
							</a>
						</li>
					{/each}

					{#if auth.user?.is_admin}
						<li class="mt-4">
							<a
								href={resolve('/admin')}
								class="rounded-lg transition-colors hover:bg-base-300"
								onclick={() => (sidebarOpen = false)}
							>
								<Shield size={20} aria-hidden="true" />
								<span>Admin</span>
							</a>
						</li>
					{/if}
				</ul>
			</div>
		</aside>

		<!-- Main Content -->
		<main class="flex-1 overflow-y-auto bg-base-100">
			<div class="p-4 md:p-8 max-w-6xl mx-auto">
				{@render children()}
			</div>
		</main>
	</div>
</div>
