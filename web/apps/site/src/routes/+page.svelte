<script lang="ts">
	import { resolve } from '$app/paths';
	import {
		ArrowRight,
		BellRing,
		Cloud,
		CodeXml,
		Eye,
		ExternalLink,
		LockKeyhole,
		Send,
		Server,
		ShieldCheck,
		Smartphone
	} from '@lucide/svelte';
	import { BeeBuzzLogo, isLoggedIn } from '@beebuzz/shared';
	import { onMount } from 'svelte';

	import { isSaasMode } from '$lib/config/deployment';
	import PublicFooter from '$lib/components/PublicFooter.svelte';

	let loggedIn = $state(false);

	onMount(() => {
		loggedIn = isLoggedIn();
	});

	const GITHUB_REPO_URL = 'https://github.com/lucor/beebuzz';
	const QUICKSTART_PATH = '/docs/quickstart';

	type UseCase = {
		title: string;
		description: string;
		icon: typeof Server;
	};

	const useCases: UseCase[] = [
		{
			title: 'Homelab and self-hosted services',
			description: 'Notifications from servers, routers, backups, cron jobs, and homelab services.',
			icon: Server
		},
		{
			title: 'Scripts, cron, and CI/CD',
			description: 'Notifications from scripts, deployments, pipelines, and scheduled jobs.',
			icon: CodeXml
		},
		{
			title: 'Private delivery to your devices',
			description: 'Direct delivery to your own devices, or to a small set of paired devices.',
			icon: BellRing
		}
	];

	type FlowNode = {
		label: string;
		description: string;
		icon: typeof Send;
	};

	const e2eNodes: FlowNode[] = [
		{ label: 'Sender', description: 'Encrypts locally', icon: Send },
		{ label: 'BeeBuzz', description: 'Stores ciphertext only', icon: Server },
		{ label: 'Web Push', description: 'Delivers encrypted envelope', icon: Cloud },
		{ label: 'Hive', description: 'Fetches and decrypts', icon: Smartphone }
	];

	const trustedNodes: FlowNode[] = [
		{ label: 'Sender', description: 'Sends the message', icon: Send },
		{ label: 'BeeBuzz', description: 'Reads to prepare delivery', icon: Server },
		{ label: 'Web Push', description: 'Delivers', icon: Cloud },
		{ label: 'Hive', description: 'Shows the message', icon: Smartphone }
	];
</script>

<svelte:head>
	<title>BeeBuzz | Your tools. Your notifications. Your keys.</title>
	<meta
		name="description"
		content="BeeBuzz is a focused push delivery system for private machine-to-person notifications from servers, automations, scripts, apps, and webhooks. End-to-end encrypted delivery first, trusted delivery when needed."
	/>
</svelte:head>

<div class="bg-base-100">
	<div class="mx-auto flex min-h-dvh max-w-7xl flex-col px-4 sm:px-6 lg:px-8">
		<header class="py-5">
			<div
				class="flex flex-col gap-4 rounded-3xl border border-base-300/70 bg-base-100/95 px-4 py-4 shadow-sm md:flex-row md:items-center md:justify-between md:px-6"
			>
				<a href={resolve('/')} class="inline-flex items-center gap-3">
					<BeeBuzzLogo variant="img" class="h-10 w-10" />
					<BeeBuzzLogo variant="text" class="h-8 w-24" />
				</a>

				<nav
					aria-label="Primary"
					class="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm font-medium text-base-content/70"
				>
					{#if isSaasMode}
						<a href={resolve(QUICKSTART_PATH)} class="transition-colors hover:text-base-content"
							>Quickstart</a
						>
					{/if}
					<a
						href={GITHUB_REPO_URL}
						target="_blank"
						rel="noopener noreferrer"
						class="transition-colors hover:text-base-content"
					>
						GitHub
					</a>
					{#if isSaasMode}
						<a
							href={resolve(loggedIn ? '/account/overview' : '/login')}
							class="transition-colors hover:text-base-content"
						>
							{loggedIn ? 'Dashboard' : 'Sign in'}
						</a>
					{/if}
				</nav>
			</div>
		</header>

		<main class="flex-1 pb-16">
			<section
				class="grid gap-10 pb-12 pt-6 md:pt-8 lg:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)] lg:items-center lg:pb-16 lg:pt-12"
			>
				<div>
					<h1
						class="max-w-4xl text-3xl font-bold leading-snug text-base-content md:text-4xl lg:text-5xl"
					>
						Your tools.<br />Your notifications.<br />Your keys.
					</h1>
					<p class="mt-6 max-w-3xl text-lg leading-8 text-base-content/75 md:text-xl">
						BeeBuzz is a push delivery system for private machine-to-person notifications from
						servers, automations, scripts, apps, and webhooks. Use end-to-end encrypted delivery
						when the sender can encrypt; use trusted delivery when it can't.
					</p>

					<div class="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
						{#if isSaasMode}
							<a
								href={resolve('/login')}
								class="btn btn-primary btn-lg gap-2 text-base font-semibold"
							>
								Request beta access
								<ArrowRight class="h-5 w-5" />
							</a>
							<a href={resolve(QUICKSTART_PATH)} class="btn btn-outline btn-lg">
								Read Quickstart
							</a>
						{/if}
						<a
							href={GITHUB_REPO_URL}
							target="_blank"
							rel="noopener noreferrer"
							class="btn btn-ghost btn-lg gap-2"
						>
							<ExternalLink class="h-5 w-5" />
							Self-host on GitHub
						</a>
					</div>

					{#if isSaasMode}
						<p class="mt-4 text-sm leading-6 text-base-content/65">
							Hosted access is currently in beta. Self-hosting is open source under AGPL.
						</p>
					{/if}
				</div>

				<div
					class="rounded-[2rem] border border-base-300 bg-gradient-to-br from-base-200 via-base-100 to-warning/10 p-6 shadow-sm sm:p-8"
				>
					<div class="flex items-center gap-3">
						<div
							class="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/12 text-primary"
						>
							<LockKeyhole class="h-6 w-6" />
						</div>
						<div>
							<p class="text-sm font-semibold uppercase tracking-[0.18em] text-primary">
								Quickstart
							</p>
							<h2 class="text-2xl font-bold text-base-content">From sign-in to first message</h2>
						</div>
					</div>

					<ul class="timeline timeline-snap-icon timeline-compact timeline-vertical mt-8">
						<li>
							<div class="timeline-middle">
								<div
									class="flex h-8 w-8 items-center justify-center rounded-full border border-primary/20 bg-primary text-sm font-semibold text-primary-content"
								>
									1
								</div>
							</div>
							<div class="timeline-end mb-8 ml-4 pt-1 text-left">
								<h3 class="text-lg font-semibold text-base-content">Sign in to BeeBuzz</h3>
								<p class="mt-2 max-w-md text-sm leading-6 text-base-content/70">
									Use the hosted access flow to sign in with your approved email or request beta
									access.
								</p>
							</div>
							<hr class="bg-primary/20" />
						</li>
						<li>
							<hr class="bg-primary/20" />
							<div class="timeline-middle">
								<div
									class="flex h-8 w-8 items-center justify-center rounded-full border border-primary/20 bg-primary text-sm font-semibold text-primary-content"
								>
									2
								</div>
							</div>
							<div class="timeline-end mb-8 ml-4 pt-1 text-left">
								<h3 class="text-lg font-semibold text-base-content">Pair one device</h3>
								<p class="mt-2 max-w-md text-sm leading-6 text-base-content/70">
									Install Hive as a PWA and connect the device that should receive notifications.
								</p>
							</div>
							<hr class="bg-primary/20" />
						</li>
						<li>
							<hr class="bg-primary/20" />
							<div class="timeline-middle">
								<div
									class="flex h-8 w-8 items-center justify-center rounded-full border border-primary/20 bg-primary text-sm font-semibold text-primary-content"
								>
									3
								</div>
							</div>
							<div class="timeline-end mb-8 ml-4 pt-1 text-left">
								<h3 class="text-lg font-semibold text-base-content">
									Create one API token or webhook
								</h3>
								<p class="mt-2 max-w-md text-sm leading-6 text-base-content/70">
									Choose the integration path that matches your sender.
								</p>
							</div>
							<hr class="bg-primary/20" />
						</li>
						<li>
							<hr class="bg-primary/20" />
							<div class="timeline-middle">
								<div
									class="flex h-8 w-8 items-center justify-center rounded-full border border-primary/20 bg-primary text-sm font-semibold text-primary-content"
								>
									4
								</div>
							</div>
							<div class="timeline-end ml-4 pt-1 text-left">
								<h3 class="text-lg font-semibold text-base-content">Send your first message</h3>
								<p class="mt-2 max-w-md text-sm leading-6 text-base-content/70">
									Use encrypted delivery when possible, or trusted delivery when the sender cannot
									encrypt.
								</p>
							</div>
						</li>
					</ul>
				</div>
			</section>

			<section class="py-10">
				<div class="mb-8 max-w-3xl">
					<h2 class="text-3xl font-bold text-base-content">
						Built for private machine-to-person notifications
					</h2>
					<p class="mt-3 text-base leading-7 text-base-content/70">
						For developers, homelabbers, and small teams sending notifications from systems they
						control. Not chat, not a team inbox, not a general messaging platform.
					</p>
				</div>

				<div class="grid gap-5 lg:grid-cols-3">
					{#each useCases as useCase (useCase.title)}
						<div class="rounded-3xl border border-base-300 bg-base-100 p-6 shadow-sm">
							<div class="flex items-center gap-3">
								<div
									class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-warning/15 text-warning"
								>
									<useCase.icon class="h-6 w-6" />
								</div>
								<h3 class="text-xl font-semibold text-base-content">{useCase.title}</h3>
							</div>
							<p class="mt-3 text-sm leading-7 text-base-content/70">{useCase.description}</p>
						</div>
					{/each}
				</div>
			</section>

			<section class="py-10">
				<div class="mb-8 max-w-3xl">
					<h2 class="text-3xl font-bold text-base-content">How delivery works</h2>
					<p class="mt-3 text-base leading-7 text-base-content/70">
						Two delivery modes, chosen by one question: can the sender encrypt before it talks to
						BeeBuzz?
					</p>
				</div>

				<div class="grid gap-5 lg:grid-cols-2">
					<!-- E2E Encrypted -->
					<div
						class="relative h-full overflow-hidden rounded-3xl border-2 border-primary/30 bg-base-100 shadow-sm"
					>
						<div class="flex h-full flex-col p-6">
							<div class="mb-5 flex items-center gap-3">
								<div
									class="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/15 text-primary"
								>
									<LockKeyhole class="h-6 w-6" />
								</div>
								<div class="flex-1">
									<div class="flex flex-wrap items-center gap-2">
										<h3 class="text-xl font-semibold text-base-content">
											End-to-end encrypted delivery
										</h3>
										<span class="badge badge-primary badge-sm text-primary-content"
											>Recommended</span
										>
									</div>
								</div>
							</div>

							<div
								class="flex flex-1 flex-col items-stretch gap-3 md:flex-row md:items-stretch md:gap-2"
							>
								{#each e2eNodes as node, index (node.label)}
									{@const isPivot = node.label === 'BeeBuzz'}
									<div
										class={[
											'flex min-h-36 flex-1 flex-col items-center rounded-2xl p-4 text-center',
											isPivot ? 'border-2 border-primary/40 bg-primary/10' : 'bg-base-200/60'
										].join(' ')}
									>
										<div
											class={[
												'flex h-10 w-10 items-center justify-center rounded-xl',
												isPivot
													? 'bg-primary/20 text-primary'
													: 'bg-base-300/60 text-base-content/70'
											].join(' ')}
										>
											<node.icon class="h-5 w-5" />
										</div>
										<p class="mt-2 text-sm font-semibold text-base-content">
											{node.label}
										</p>
										<p class="mt-1 text-xs leading-4 text-base-content/70">
											{node.description}
										</p>
									</div>

									{#if index < e2eNodes.length - 1}
										<div
											class="flex shrink-0 items-center justify-center text-primary md:w-6"
											aria-hidden="true"
										>
											<ArrowRight class="h-5 w-5 rotate-90 md:rotate-0" />
										</div>
									{/if}
								{/each}
							</div>

							<div
								class="mt-4 flex items-center gap-2 rounded-xl border border-primary/20 bg-primary/5 px-4 py-3"
							>
								<ShieldCheck class="h-4 w-4 shrink-0 text-primary" />
								<p class="text-sm font-medium text-base-content/85">
									BeeBuzz can't read the message.
								</p>
							</div>

							<div class="mt-4">
								<p class="mb-2 text-xs font-medium uppercase tracking-wider text-base-content/60">
									Example senders
								</p>
								<div class="flex flex-wrap gap-1.5">
									<span class="badge badge-neutral badge-soft badge-sm">BeeBuzz CLI</span>
									<span class="badge badge-neutral badge-soft badge-sm">BeeBuzz libraries</span>
									<span class="badge badge-neutral badge-soft badge-sm">Home Assistant plugin</span>
								</div>
							</div>
						</div>
					</div>

					<!-- Trusted -->
					<div
						class="relative h-full overflow-hidden rounded-3xl border border-base-300 bg-base-100 shadow-sm"
					>
						<div class="flex h-full flex-col p-6">
							<div class="mb-5 flex items-center gap-3">
								<div
									class="flex h-12 w-12 items-center justify-center rounded-2xl bg-base-200 text-base-content/70"
								>
									<Server class="h-6 w-6" />
								</div>
								<div class="flex-1">
									<div class="flex flex-wrap items-center gap-2">
										<h3 class="text-xl font-semibold text-base-content">Trusted delivery</h3>
										<span class="badge badge-neutral badge-soft badge-sm">When needed</span>
									</div>
								</div>
							</div>

							<div
								class="flex flex-1 flex-col items-stretch gap-3 md:flex-row md:items-stretch md:gap-2"
							>
								{#each trustedNodes as node, index (node.label)}
									{@const isPivot = node.label === 'BeeBuzz'}
									<div
										class={[
											'flex min-h-36 flex-1 flex-col items-center rounded-2xl p-4 text-center',
											isPivot ? 'border-2 border-warning/40 bg-warning/10' : 'bg-base-200/60'
										].join(' ')}
									>
										<div
											class={[
												'flex h-10 w-10 items-center justify-center rounded-xl',
												isPivot
													? 'bg-warning/20 text-warning'
													: 'bg-base-300/60 text-base-content/60'
											].join(' ')}
										>
											<node.icon class="h-5 w-5" />
										</div>
										<p class="mt-2 text-sm font-semibold text-base-content">
											{node.label}
										</p>
										<p class="mt-1 text-xs leading-4 text-base-content/70">
											{node.description}
										</p>
									</div>

									{#if index < trustedNodes.length - 1}
										<div
											class="flex shrink-0 items-center justify-center text-base-content/40 md:w-6"
											aria-hidden="true"
										>
											<ArrowRight class="h-5 w-5 rotate-90 md:rotate-0" />
										</div>
									{/if}
								{/each}
							</div>

							<div
								class="mt-4 flex items-center gap-2 rounded-xl border border-warning/30 bg-warning/10 px-4 py-3"
							>
								<Eye class="h-4 w-4 shrink-0 text-warning" />
								<p class="text-sm font-medium text-base-content/85">
									BeeBuzz can read the message to prepare delivery.
								</p>
							</div>

							<div class="mt-4">
								<p class="mb-2 text-xs font-medium uppercase tracking-wider text-base-content/60">
									Example senders
								</p>
								<div class="flex flex-wrap gap-1.5">
									<span class="badge badge-neutral badge-soft badge-sm">Webhooks</span>
									<span class="badge badge-neutral badge-soft badge-sm">cURL</span>
									<span class="badge badge-neutral badge-soft badge-sm">CI/CD</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			</section>
		</main>

		<PublicFooter />
	</div>
</div>
