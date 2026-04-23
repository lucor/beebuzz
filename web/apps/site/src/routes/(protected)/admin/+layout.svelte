<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { toast } from '@beebuzz/shared/stores';
	import { logout } from '@beebuzz/shared/services/auth';
	import { ApiError } from '@beebuzz/shared/errors';
	import { BeeBuzzLogo } from '@beebuzz/shared/components';
	import { Menu, X, LogOut, Users, ShieldAlert, LayoutDashboard, type Icon } from '@lucide/svelte';

	let { children }: { children: import('svelte').Snippet } = $props();

	type NavItem = {
		label: string;
		href: string;
		icon: typeof Icon;
	};

	let sidebarOpen = $state(false);

	let currentPath = $derived(page.url.pathname);

	const navItems: NavItem[] = [
		{ label: 'Dashboard', href: '/admin', icon: LayoutDashboard },
		{ label: 'Users', href: '/admin/users', icon: Users }
	];

	/** Handles admin logout. */
	const handleLogout = async () => {
		try {
			await logout();
			toast.success('Logged out successfully');
			await goto(resolve('/login'));
		} catch (err) {
			toast.error(err instanceof ApiError ? err.userMessage : 'Logout failed');
			await goto(resolve('/login'));
		}
	};
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

			<!-- Admin Badge -->
			<div class="ml-auto lg:ml-0 flex items-center gap-2 badge badge-warning gap-1">
				<ShieldAlert size={16} />
				<span class="hidden sm:inline">Admin</span>
			</div>
		</div>

		<!-- Center: Logo (mobile only) -->
		<div class="navbar-center sm:hidden">
			<a href={resolve('/')} class="flex items-center gap-2">
				<BeeBuzzLogo variant="img" class="w-10 h-10" />
			</a>
		</div>

		<div class="navbar-end"></div>
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
				</ul>

				<div class="mt-auto border-t border-base-300 p-4">
					<button
						type="button"
						class="btn btn-ghost btn-sm w-full justify-start text-error"
						onclick={handleLogout}
					>
						<LogOut size={16} />
						Logout
					</button>
				</div>
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
