<script lang="ts">
	import { resolve } from '$app/paths';
	import {
		ArrowRight,
		Bell,
		EllipsisVertical,
		Eye,
		House,
		Image,
		ListChecks,
		LockKeyhole,
		Menu,
		Send,
		Server,
		ShieldCheck,
		Webhook
	} from '@lucide/svelte';
	import { BeeBuzzLogo, isLoggedIn } from '@beebuzz/shared';
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';

	import { isSaasMode } from '$lib/config/deployment';
	import PublicFooter from '$lib/components/PublicFooter.svelte';

	let loggedIn = $state(false);
	let demoSection = $state<HTMLElement | undefined>(undefined);

	// Demo phases:
	// 0 = typing CLI command (both inboxes empty)
	// 1 = CLI success line shown (still no msg)
	// 2 = msg1 arrived on phone
	// 3 = msg1 arrived on browser
	// 4 = typing curl command
	// 5 = curl success line shown
	// 6 = msg2 arrived on phone
	// 7 = msg2 arrived on browser
	// 8 = hold final state
	let phase = $state(0);
	let typed1 = $state('');
	let typed2 = $state('');

	const cmd1 = 'beebuzz send -a photo.jpg "Hello 🐝" "End-to-end encrypted"';
	const cmd2 =
		'curl https://push.beebuzz.app/alerts \\\n  -H "Authorization: Bearer $TOKEN" \\\n  -F title="CPU > 90%" \\\n  -F body="Trusted mode test"';

	const typing1Active = $derived(phase === 0);
	const typing2Active = $derived(phase === 4);
	const result1Visible = $derived(phase >= 1);
	const cmd2Visible = $derived(phase >= 4);
	const result2Visible = $derived(phase >= 5);
	const phoneMessageCount = $derived(phase >= 6 ? 2 : phase >= 2 ? 1 : 0);
	const browserMessageCount = $derived(phase >= 7 ? 2 : phase >= 3 ? 1 : 0);

	onMount(() => {
		loggedIn = isLoggedIn();

		const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

		if (prefersReducedMotion) {
			typed1 = cmd1;
			typed2 = cmd2;
			phase = 8;
			return;
		}

		const timers: ReturnType<typeof setTimeout>[] = [];
		const queue = (callback: () => void, delay: number) => {
			const timer = setTimeout(callback, delay);
			timers.push(timer);
		};

		const clearTimers = () => {
			for (const timer of timers) {
				clearTimeout(timer);
			}
			timers.length = 0;
		};

		const typeText = (text: string, setter: (s: string) => void, onDone: () => void) => {
			const chars = Array.from(text);
			let i = 0;
			const step = () => {
				setter(chars.slice(0, i).join(''));
				if (i <= chars.length) {
					i += 1;
					queue(step, 40);
					return;
				}
				onDone();
			};
			step();
		};

		const runDemo = () => {
			typed1 = '';
			typed2 = '';
			phase = 0;

			typeText(
				cmd1,
				(s) => (typed1 = s),
				() => {
					queue(() => (phase = 1), 350);
					queue(() => (phase = 2), 650);
					queue(() => (phase = 3), 900);
					// Pause then start typing curl command
					queue(() => {
						phase = 4;
						typeText(
							cmd2,
							(s) => (typed2 = s),
							() => {
								queue(() => (phase = 5), 350);
								queue(() => (phase = 6), 650);
								queue(() => (phase = 7), 900);
								queue(() => (phase = 8), 1100);
								// Hold final state, then loop
								queue(runDemo, 4500);
							}
						);
					}, 1900);
				}
			);
		};

		// Only animate when the section is in view. Pauses off-screen.
		let isRunning = false;
		const observer = new IntersectionObserver(
			(entries) => {
				const entry = entries[0];
				if (!entry) return;
				if (entry.isIntersecting && !isRunning) {
					isRunning = true;
					runDemo();
				} else if (!entry.isIntersecting && isRunning) {
					isRunning = false;
					clearTimers();
				}
			},
			{ threshold: 0.2 }
		);

		if (demoSection) {
			observer.observe(demoSection);
		} else {
			runDemo();
		}

		return () => {
			observer.disconnect();
			clearTimers();
		};
	});

	const QUICKSTART_PATH = '/docs/quickstart';
</script>

<svelte:head>
	<title>BeeBuzz | Minimalist, privacy-first push notifications.</title>
	<meta
		name="description"
		content="BeeBuzz sends notifications from your tools to your devices. End-to-end encryption where supported. Trusted delivery when needed."
	/>
</svelte:head>

<div class="bg-base-100">
	<div class="mx-auto flex min-h-dvh max-w-7xl flex-col px-4 sm:px-6 lg:px-8">
		<header class="py-5">
			<div
				class="flex flex-col gap-4 rounded-3xl border border-base-300/70 bg-base-100/95 px-4 py-4 shadow-sm md:flex-row md:items-center md:justify-between md:px-6"
			>
				<a href={resolve('/')}>
					<BeeBuzzLogo variant="full" class="h-10 w-auto" />
				</a>

				<nav
					aria-label="Primary"
					class="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm font-medium text-base-content/70"
				>
					<a href={resolve(QUICKSTART_PATH)} class="transition-colors hover:text-base-content"
						>Quickstart</a
					>
					<a
						href={resolve(loggedIn ? '/account/overview' : '/auth')}
						class="transition-colors hover:text-base-content"
					>
						{loggedIn ? 'Dashboard' : 'Sign in / Sign up'}
					</a>
				</nav>
			</div>
		</header>

		<main class="flex-1 pb-16">
			<section class="py-12 md:py-16 lg:py-20">
				<div>
					<h1 class="text-3xl font-bold leading-snug text-base-content md:text-4xl lg:text-5xl">
						Minimalist, privacy-first push notifications.
					</h1>
					<p class="mt-6 max-w-4xl text-lg leading-8 text-base-content/75 md:text-xl">
						BeeBuzz sends notifications from your tools to your devices.
					</p>
					<p class="mt-3 max-w-4xl text-lg leading-8 text-base-content/75 md:text-xl">
						No app stores, no native apps: just the Hive <a
							href={resolve('/docs/browser-support')}
							class="font-medium text-primary underline decoration-primary/30 underline-offset-4 transition-colors hover:text-primary-focus hover:decoration-primary"
							>PWA</a
						> in your browser.
					</p>
					<p class="mt-3 max-w-4xl text-lg leading-8 text-base-content/75 md:text-xl">
						End-to-end encryption where supported. Trusted delivery when needed.
					</p>
					<p class="mt-4 text-base font-semibold text-base-content/80">
						Your tools. Your notifications. Your keys.
					</p>

					<div class="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
						{#if isSaasMode}
							<a
								href={resolve('/auth')}
								class="btn btn-primary btn-lg gap-2 text-base font-semibold"
							>
								Request beta access
								<ArrowRight class="h-5 w-5" />
							</a>
							<a href={resolve(QUICKSTART_PATH)} class="btn btn-outline btn-lg">
								Read Quickstart
							</a>
						{:else}
							<a
								href={resolve(QUICKSTART_PATH)}
								class="btn btn-primary btn-lg gap-2 text-base font-semibold"
							>
								Read Quickstart
								<ArrowRight class="h-5 w-5" />
							</a>
						{/if}
					</div>

					<p class="mt-4 text-sm text-base-content/60">
						<a
							href="https://codeberg.org/beebuzz/beebuzz"
							target="_blank"
							rel="noopener noreferrer"
							class="underline hover:text-primary"
						>
							Open source
						</a>. Self-host or use hosted beta access.
					</p>
				</div>
			</section>

			<section class="py-10" bind:this={demoSection}>
				<div class="rounded-[2rem] border border-base-300 bg-base-100 p-4 shadow-sm sm:p-6">
					<div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
						<div>
							<p class="text-sm font-semibold uppercase tracking-[0.18em] text-primary">
								Send to Hive
							</p>
							<h2 class="mt-1 text-3xl font-bold text-base-content">From tools to devices</h2>
							<p class="mt-3 max-w-none text-base leading-7 text-base-content/70">
								Send from command-line workflows or HTTP clients and see the same message arrive on
								your devices in Hive.
							</p>
						</div>
						<div
							class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-primary/12 text-primary"
						>
							<Send class="h-6 w-6" />
						</div>
					</div>

					<!-- Step 1: terminal on top -->
					<div class="mt-8">
						<p class="mb-3 text-xs font-semibold uppercase tracking-[0.18em] text-base-content/50">
							1. Send from your tools
						</p>
						<div class="mx-auto max-w-2xl">
							<div
								class="mockup-code min-h-52 w-full overflow-x-auto bg-neutral text-neutral-content"
							>
								<pre data-prefix="$"><code
										>{typed1}<span
											class={[
												'ml-0.5 inline-block h-4 w-2 translate-y-0.5 bg-primary',
												typing1Active ? 'opacity-100' : 'opacity-0'
											].join(' ')}
											aria-hidden="true"></span></code
									></pre>
								<pre
									data-prefix=">"
									class={[
										'text-success transition-opacity duration-300',
										result1Visible ? 'opacity-100' : 'opacity-0'
									].join(' ')}><code>encrypted and sent to paired devices</code></pre>
								{#each typed2.split('\n') as line, i (i)}
									{@const isLastLine = i === typed2.split('\n').length - 1}
									<pre
										data-prefix={i === 0 ? '$' : ' '}
										class={[
											'transition-opacity duration-300',
											cmd2Visible ? 'opacity-100' : 'opacity-0'
										].join(' ')}><code
											>{line}{#if isLastLine}<span
													class={[
														'ml-0.5 inline-block h-4 w-2 translate-y-0.5 bg-primary',
														typing2Active ? 'opacity-100' : 'opacity-0'
													].join(' ')}
													aria-hidden="true"></span>{/if}</code
										></pre>
								{/each}
								<pre
									data-prefix=">"
									class={[
										'text-success transition-opacity duration-300',
										result2Visible ? 'opacity-100' : 'opacity-0'
									].join(' ')}><code>delivered to your paired devices</code></pre>
							</div>
						</div>
					</div>

					<!-- Connector arrow -->
					<div class="my-3 flex justify-center text-primary/60" aria-hidden="true">
						<ArrowRight class="h-6 w-6 rotate-90" />
					</div>

					<!-- Step 2: faithful Hive UI mockups (empty → arrival) -->
					<div>
						<p class="mb-4 text-xs font-semibold uppercase tracking-[0.18em] text-base-content/50">
							2. Arrives on your devices
						</p>
						<div
							class="grid items-start justify-items-center gap-8 md:grid-cols-[auto_minmax(0,1fr)] md:gap-6"
						>
							<!-- Mobile — faithful slice of Hive inbox -->
							<div class="flex flex-col items-center gap-3">
								<div class="mockup-phone border-primary">
									<div class="mockup-phone-camera"></div>
									<div class="mockup-phone-display bg-base-100">
										<div class="flex h-96 w-[15rem] flex-col bg-base-100 sm:h-[30rem]">
											<!-- Status bar (empty top area, simulates safe area) -->
											<div class="h-6 shrink-0 bg-base-100" aria-hidden="true"></div>
											<!-- Hive top navbar: hamburger | logo | Healthy -->
											<div
												class="grid grid-cols-3 items-center border-b border-base-300 bg-base-100 px-2 py-2"
											>
												<div class="flex justify-start">
													<Menu class="h-5 w-5 text-base-content/70" />
												</div>
												<div class="flex justify-center">
													<BeeBuzzLogo variant="img" class="h-7 w-7" />
												</div>
												<div class="flex justify-end">
													<span
														class="inline-flex items-center gap-1 rounded-full border border-base-300 bg-base-100 px-1.5 py-0.5 text-[10px] font-medium text-base-content/80"
													>
														<span class="h-1.5 w-1.5 rounded-full bg-success"></span>
														Healthy
													</span>
												</div>
											</div>
											<!-- Inbox content -->
											<div class="flex-1 overflow-hidden p-3">
												<div class="mb-3 flex items-center justify-between">
													<h3 class="text-base font-semibold text-base-content">Inbox</h3>
													<ListChecks class="h-3.5 w-3.5 text-base-content/60" />
												</div>
												{#if phoneMessageCount === 0}
													<div class="card border border-base-300 bg-base-100 p-5 text-center">
														<div class="mb-2 flex justify-center">
															<Bell class="h-10 w-10 text-base-content/40" />
														</div>
														<p class="text-sm font-bold text-base-content">No notifications yet</p>
														<p class="mt-1 text-xs leading-4 text-base-content/60">
															You'll see new messages here when they arrive
														</p>
													</div>
												{:else}
													<div class="space-y-2">
														<h4
															class="text-[10px] font-semibold uppercase tracking-wide text-base-content/60"
														>
															Today
														</h4>
														{#if phoneMessageCount >= 2}
															<div
																class="rounded-box border border-base-300 bg-base-100 p-3"
																in:fly={{ y: 8, duration: 250 }}
															>
																<div class="flex items-start justify-between gap-2">
																	<div class="flex min-w-0 flex-1 items-start gap-1.5">
																		<span
																			class="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-primary"
																			aria-label="Unread"
																		></span>
																		<p
																			class="min-w-0 flex-1 text-sm font-bold leading-tight text-base-content"
																		>
																			CPU &gt; 90%
																		</p>
																	</div>
																	<EllipsisVertical class="h-4 w-4 shrink-0 text-base-content/50" />
																</div>
																<p class="mt-1.5 text-xs leading-4 text-base-content/70">
																	Trusted mode test
																</p>
																<div
																	class="mt-2 flex flex-wrap items-center gap-x-1.5 gap-y-1 text-[10px] text-base-content/70"
																>
																	<span
																		class="badge badge-xs badge-ghost border-base-300 font-medium"
																	>
																		#alerts
																	</span>
																	<span aria-hidden="true">·</span>
																	<span>now</span>
																</div>
															</div>
														{/if}
														<div
															class="rounded-box border border-base-300 bg-base-100 p-3"
															in:fly={{ y: 8, duration: 250 }}
														>
															<div class="flex items-start justify-between gap-2">
																<div class="flex min-w-0 flex-1 items-start gap-1.5">
																	<span
																		class={[
																			'mt-1.5 h-2 w-2 shrink-0 rounded-full',
																			phoneMessageCount >= 2 ? 'bg-base-300' : 'bg-primary'
																		].join(' ')}
																		aria-label={phoneMessageCount >= 2 ? 'Read' : 'Unread'}
																	></span>
																	<p
																		class={[
																			'min-w-0 flex-1 text-sm font-bold leading-tight',
																			phoneMessageCount >= 2
																				? 'text-base-content/70'
																				: 'text-base-content'
																		].join(' ')}
																	>
																		Hello 🐝
																	</p>
																</div>
																<EllipsisVertical class="h-4 w-4 shrink-0 text-base-content/50" />
															</div>
															<p
																class={[
																	'mt-1.5 text-xs leading-4',
																	phoneMessageCount >= 2
																		? 'text-base-content/55'
																		: 'text-base-content/70'
																].join(' ')}
															>
																End-to-end encrypted
															</p>
															<div
																class="mt-2 flex flex-wrap items-center gap-x-1.5 gap-y-1 text-[10px] text-base-content/70"
															>
																<span
																	class="badge badge-xs badge-ghost border-base-300 font-medium"
																>
																	#general
																</span>
																<span aria-hidden="true">·</span>
																<span>now</span>
															</div>
															<div class="mt-1.5">
																<span
																	class="inline-flex items-center gap-1 rounded px-1 py-0.5 text-[10px] text-base-content/60"
																>
																	<Image class="h-3 w-3" />
																	photo.jpg
																</span>
															</div>
														</div>
													</div>
												{/if}
											</div>
										</div>
									</div>
								</div>
								<p class="text-xs font-medium text-base-content/60">Mobile</p>
							</div>

							<!-- Desktop — faithful slice of Hive inbox -->
							<div class="flex w-full flex-col items-center gap-3">
								<div
									class="mockup-browser w-full border border-base-300 border-t-primary border-t-2 bg-base-100"
								>
									<div class="mockup-browser-toolbar">
										<div class="input input-sm flex-1 border-base-300 text-xs text-base-content/60">
											hive.beebuzz.app
										</div>
									</div>
									<div class="border-t border-base-300 bg-base-100">
										<!-- Hive top navbar -->
										<div
											class="flex items-center justify-between border-b border-base-300 bg-base-100 px-4 py-2.5"
										>
											<BeeBuzzLogo variant="full" class="h-7 w-auto" />
											<span
												class="inline-flex items-center gap-1.5 rounded-full border border-base-300 bg-base-100 px-2.5 py-1 text-xs font-medium text-base-content/80"
											>
												<span class="h-2 w-2 rounded-full bg-success"></span>
												<span class="hidden sm:inline">Device status</span>
												Healthy
											</span>
										</div>
										<!-- Inbox content -->
										<div class="min-h-96 p-4 sm:min-h-[30rem]">
											<div class="mb-4 flex items-center justify-between">
												<div>
													<h3 class="text-lg font-semibold text-base-content">Inbox</h3>
													<p class="hidden text-xs text-base-content/60 sm:block">
														Your notifications, grouped by day.
													</p>
												</div>
												<span class="inline-flex items-center gap-1.5 text-xs text-base-content/70">
													<ListChecks class="h-4 w-4" />
													Select
												</span>
											</div>
											{#if browserMessageCount === 0}
												<div class="card border border-base-300 bg-base-100 p-8 text-center">
													<div class="mb-3 flex justify-center">
														<Bell class="h-12 w-12 text-base-content/40" />
													</div>
													<p class="text-base font-bold text-base-content">No notifications yet</p>
													<p class="mt-1 text-sm text-base-content/60">
														You'll see new messages here when they arrive
													</p>
												</div>
											{:else}
												<div class="space-y-2">
													<h4
														class="text-xs font-semibold uppercase tracking-wide text-base-content/60"
													>
														Today
													</h4>
													{#if browserMessageCount >= 2}
														<div
															class="rounded-box border border-base-300 bg-base-100 p-4"
															in:fly={{ y: 8, duration: 250 }}
														>
															<div class="flex items-start justify-between gap-3">
																<div class="flex min-w-0 flex-1 items-start gap-2">
																	<span
																		class="mt-2 h-2.5 w-2.5 shrink-0 rounded-full bg-primary"
																		aria-label="Unread"
																	></span>
																	<p class="min-w-0 flex-1 text-base font-bold text-base-content">
																		CPU &gt; 90%
																	</p>
																</div>
																<EllipsisVertical class="h-4 w-4 shrink-0 text-base-content/50" />
															</div>
															<p class="mt-1.5 text-sm leading-5 text-base-content/70">
																Trusted mode test
															</p>
															<div
																class="mt-3 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-base-content/70"
															>
																<span
																	class="badge badge-sm badge-ghost border-base-300 font-medium"
																>
																	#alerts
																</span>
																<span aria-hidden="true">·</span>
																<span>now</span>
															</div>
														</div>
													{/if}
													<div
														class="rounded-box border border-base-300 bg-base-100 p-4"
														in:fly={{ y: 8, duration: 250 }}
													>
														<div class="flex items-start justify-between gap-3">
															<div class="flex min-w-0 flex-1 items-start gap-2">
																<span
																	class={[
																		'mt-2 h-2.5 w-2.5 shrink-0 rounded-full',
																		browserMessageCount >= 2 ? 'bg-base-300' : 'bg-primary'
																	].join(' ')}
																	aria-label={browserMessageCount >= 2 ? 'Read' : 'Unread'}
																></span>
																<p
																	class={[
																		'min-w-0 flex-1 text-base font-bold',
																		browserMessageCount >= 2
																			? 'text-base-content/70'
																			: 'text-base-content'
																	].join(' ')}
																>
																	Hello 🐝
																</p>
															</div>
															<EllipsisVertical class="h-4 w-4 shrink-0 text-base-content/50" />
														</div>
														<p
															class={[
																'mt-1.5 text-sm leading-5',
																browserMessageCount >= 2
																	? 'text-base-content/55'
																	: 'text-base-content/70'
															].join(' ')}
														>
															End-to-end encrypted
														</p>
														<div
															class="mt-3 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-base-content/70"
														>
															<span class="badge badge-sm badge-ghost border-base-300 font-medium">
																#general
															</span>
															<span aria-hidden="true">·</span>
															<span>now</span>
														</div>
														<div class="mt-2">
															<span
																class="inline-flex items-center gap-1.5 rounded px-1.5 py-1 text-xs text-base-content/70"
															>
																<Image class="h-3.5 w-3.5" />
																photo.jpg
															</span>
														</div>
													</div>
												</div>
											{/if}
										</div>
									</div>
								</div>
								<p class="text-xs font-medium text-base-content/60">Desktop</p>
							</div>
						</div>
					</div>
				</div>
			</section>

			<section class="py-10">
				<div class="mb-8 max-w-none">
					<h2 class="text-3xl font-bold text-base-content">
						Built for tools that need to reach you
					</h2>
					<p class="mt-3 text-base leading-7 text-base-content/70">
						For developers, homelabbers, and small teams sending notifications from systems they
						control.
					</p>
				</div>

				<div class="grid gap-5 lg:grid-cols-3">
					<div class="rounded-3xl border border-base-300 bg-base-100 p-6 shadow-sm">
						<div class="flex items-center gap-3">
							<div
								class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-warning/15 text-warning"
							>
								<Server class="h-6 w-6" />
							</div>
							<h3 class="text-xl font-semibold text-base-content">Servers & homelabs</h3>
						</div>
						<p class="mt-3 text-sm leading-7 text-base-content/70">
							Notifications from servers, routers, backups, and cron jobs. Use the CLI or Go SDK.
						</p>
					</div>

					<div class="rounded-3xl border border-base-300 bg-base-100 p-6 shadow-sm">
						<div class="flex items-center gap-3">
							<div
								class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-secondary/15 text-secondary"
							>
								<Webhook class="h-6 w-6" />
							</div>
							<h3 class="text-xl font-semibold text-base-content">Scripts & CI/CD</h3>
						</div>
						<p class="mt-3 text-sm leading-7 text-base-content/70">
							Notifications from scripts, deployments, and pipelines. Use webhooks, cURL, or the
							HTTP API.
						</p>
					</div>

					<div class="rounded-3xl border border-base-300 bg-base-100 p-6 shadow-sm">
						<div class="flex items-center gap-3">
							<div
								class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-primary/15 text-primary"
							>
								<House class="h-6 w-6" />
							</div>
							<h3 class="text-xl font-semibold text-base-content">Home Assistant</h3>
						</div>
						<p class="mt-3 text-sm leading-7 text-base-content/70">
							Connect automations and alerts to BeeBuzz with a dedicated integration.
						</p>
					</div>
				</div>
			</section>

			<section class="py-10">
				<div class="mb-8 max-w-none">
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

							<p class="flex-1 text-sm leading-7 text-base-content/70">
								The sender encrypts locally. BeeBuzz stores ciphertext only and never sees the
								content.
							</p>

							<div
								class="mt-4 flex items-center gap-2 rounded-xl border border-primary/20 bg-primary/5 px-4 py-3"
							>
								<ShieldCheck class="h-4 w-4 shrink-0 text-primary" />
								<p class="text-sm font-medium text-base-content/85">
									BeeBuzz can't read the message.
								</p>
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

							<p class="flex-1 text-sm leading-7 text-base-content/70">
								The sender sends plaintext. BeeBuzz reads to route, then delivers to your devices.
							</p>

							<div
								class="mt-4 flex items-center gap-2 rounded-xl border border-warning/30 bg-warning/10 px-4 py-3"
							>
								<Eye class="h-4 w-4 shrink-0 text-warning" />
								<p class="text-sm font-medium text-base-content/85">
									BeeBuzz can read the message to prepare delivery.
								</p>
							</div>
						</div>
					</div>
				</div>
			</section>
		</main>

		<PublicFooter />
	</div>
</div>
