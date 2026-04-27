<script lang="ts">
	import { resolve } from '$app/paths';
	import {
		ArrowRight,
		BellRing,
		CodeXml,
		ExternalLink,
		Globe,
		LockKeyhole,
		Server,
		ShieldCheck,
		Zap
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

	type ProofPoint = {
		label: string;
		value: string;
		description: string;
	};

	const useCases: UseCase[] = [
		{
			title: 'Homelab and personal infrastructure',
			description:
				'Get private alerts from your server, router, backups, or cron jobs without relying on chat apps.',
			icon: Server
		},
		{
			title: 'CI, deployments, and automation',
			description:
				'Send deploy results, pipeline failures, and workflow notifications directly to the devices you control.',
			icon: CodeXml
		},
		{
			title: 'Monitoring that reaches you fast',
			description:
				'Deliver operational messages with less noise than email and more control than general-purpose messengers.',
			icon: BellRing
		}
	];

	const proofPoints: ProofPoint[] = [
		{
			label: 'Delivery model',
			value: 'Trusted or end-to-end encrypted',
			description:
				'Start fast in trusted mode or use paired-device encryption when privacy is the priority.'
		},
		{
			label: 'Client footprint',
			value: 'One PWA across desktop and mobile',
			description: 'Hive runs from a single install path and pairs devices for encrypted delivery.'
		},
		{
			label: 'Operational posture',
			value: 'Open source and auditable',
			description:
				'Inspect the code, self-host your own instance, or evaluate the hosted beta before committing.'
		}
	];
</script>

<svelte:head>
	<title>BeeBuzz | Encrypted Push For Servers And Apps</title>
	<meta
		name="description"
		content="BeeBuzz delivers private push notifications from your apps and servers to your devices, with open-source infrastructure and optional end-to-end encryption."
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
				class="grid gap-10 py-10 lg:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)] lg:items-center lg:py-16"
			>
				<div>
					<p class="text-sm font-semibold uppercase tracking-[0.22em] text-primary">
						Private push for servers and apps
					</p>
					<h1
						class="mt-4 max-w-4xl text-4xl font-bold leading-tight text-base-content md:text-5xl lg:text-6xl"
					>
						Simple, private push notifications with real end-to-end encryption.
					</h1>
					<p class="mt-6 max-w-3xl text-lg leading-8 text-base-content/75 md:text-xl">
						BeeBuzz delivers alerts over Web Push with a small, auditable stack. In E2E mode, the
						server stores ciphertext instead of plaintext.
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
							Hosted access is free during beta. After beta, the hosted service will move to a
							single paid plan, priced to keep the project sustainable. Self-hosting remains free
							and open source.
						</p>
						<p class="mt-2 text-sm leading-6 text-base-content/65">
							If your email is already approved, you can sign in now. Otherwise the same flow
							submits your access request.
						</p>
					{/if}

					<div class="mt-10 grid gap-4 sm:grid-cols-3">
						<div class="rounded-2xl border border-base-300 bg-base-200/45 p-4">
							<div class="mb-3 flex items-center gap-2 text-base-content">
								<ShieldCheck class="h-5 w-5 text-primary" />
								<p class="font-semibold">Optional end-to-end delivery</p>
							</div>
							<p class="text-sm leading-6 text-base-content/70">
								In E2E mode, pair devices so BeeBuzz stores ciphertext instead of plaintext message
								bodies.
							</p>
						</div>

						<div class="rounded-2xl border border-base-300 bg-base-200/45 p-4">
							<div class="mb-3 flex items-center gap-2 text-base-content">
								<Zap class="h-5 w-5 text-primary" />
								<p class="font-semibold">One job, done simply</p>
							</div>
							<p class="text-sm leading-6 text-base-content/70">
								BeeBuzz is not a general messaging platform. It is a focused delivery path for the
								alerts you actually care about.
							</p>
						</div>

						<div class="rounded-2xl border border-base-300 bg-base-200/45 p-4">
							<div class="mb-3 flex items-center gap-2 text-base-content">
								<Globe class="h-5 w-5 text-primary" />
								<p class="font-semibold">Minimal client model</p>
							</div>
							<p class="text-sm leading-6 text-base-content/70">
								Use the same Hive PWA across desktop and mobile without extra apps or extra moving
								parts.
							</p>
						</div>
					</div>
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
								How it works
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
									Choose the integration path that matches your system, from API calls to outbound
									event hooks.
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
									Start with the fastest setup path, then refine delivery mode and privacy controls
									as needed.
								</p>
							</div>
						</li>
					</ul>

					{#if isSaasMode}
						<div class="mt-6 rounded-2xl border border-primary/20 bg-primary/8 p-4">
							<p class="text-sm font-semibold text-base-content">Already approved?</p>
							<p class="mt-1 text-sm leading-6 text-base-content/70">
								Continue to your account, then pair a device in Hive.
							</p>
							<div class="mt-3 flex flex-wrap gap-3">
								<a href={resolve('/login')} class="link link-primary text-sm font-medium">Sign in</a
								>
								<a href={resolve(QUICKSTART_PATH)} class="link link-primary text-sm font-medium">
									Setup guide
								</a>
							</div>
						</div>
					{/if}
				</div>
			</section>

			<section class="py-10">
				<div class="mb-8">
					<p class="text-sm font-semibold uppercase tracking-[0.18em] text-primary">
						Who it is for
					</p>
					<h2 class="mt-2 text-3xl font-bold text-base-content">
						BeeBuzz is strongest when the alert matters and the audience is small.
					</h2>
					<p class="mt-3 text-base leading-7 text-base-content/70">
						This is not another team chat surface. It is a delivery path for messages you want to
						send directly from machines to people, with tighter control over privacy and attention.
					</p>
				</div>

				<div class="grid gap-5 lg:grid-cols-3">
					{#each useCases as useCase (useCase.title)}
						<div class="rounded-3xl border border-base-300 bg-base-100 p-6 shadow-sm">
							<div
								class="flex h-12 w-12 items-center justify-center rounded-2xl bg-warning/15 text-warning"
							>
								<svelte:component this={useCase.icon} class="h-6 w-6" />
							</div>
							<h3 class="mt-5 text-xl font-semibold text-base-content">{useCase.title}</h3>
							<p class="mt-3 text-sm leading-7 text-base-content/70">{useCase.description}</p>
						</div>
					{/each}
				</div>
			</section>

			<section class="py-10">
				<div class="mb-8 max-w-3xl">
					<p class="text-sm font-semibold uppercase tracking-[0.18em] text-primary">
						Trust and evaluation
					</p>
					<h2 class="mt-2 text-3xl font-bold text-base-content">
						Strong product claims need a fast way to verify them.
					</h2>
				</div>

				<div class="grid gap-5 lg:grid-cols-3">
					{#each proofPoints as point (point.label)}
						<div class="rounded-3xl border border-base-300 bg-base-200/40 p-6">
							<p class="text-sm font-semibold uppercase tracking-[0.16em] text-base-content/45">
								{point.label}
							</p>
							<h3 class="mt-3 text-xl font-semibold text-base-content">{point.value}</h3>
							<p class="mt-3 text-sm leading-7 text-base-content/70">{point.description}</p>
						</div>
					{/each}
				</div>

				<div
					class="mt-8 flex flex-col gap-3 rounded-3xl border border-base-300 bg-base-100 p-6 sm:flex-row sm:items-center sm:justify-between"
				>
					<div>
						<h3 class="text-xl font-semibold text-base-content">
							Evaluate BeeBuzz through the path that fits your risk profile.
						</h3>
						<p class="mt-2 text-sm leading-6 text-base-content/70">
							Read the setup guide, inspect the code, or test the hosted beta if you want the
							shortest path to first delivery.
						</p>
					</div>
					<div class="flex flex-wrap gap-3">
						{#if isSaasMode}
							<a href={resolve(QUICKSTART_PATH)} class="btn btn-outline">Quickstart</a>
						{/if}
						<a
							href={GITHUB_REPO_URL}
							target="_blank"
							rel="noopener noreferrer"
							class="btn btn-primary"
						>
							View Source
						</a>
					</div>
				</div>
			</section>
		</main>

		<PublicFooter />
	</div>
</div>
