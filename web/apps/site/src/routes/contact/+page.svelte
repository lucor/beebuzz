<script lang="ts">
	import { resolve } from '$app/paths';
	import { Shield, MessageSquare, ScrollText } from '@lucide/svelte';

	const CONTACT_EMAIL = 'support@beebuzz.app';
	const GITHUB_ISSUES_URL = 'https://github.com/lucor/beebuzz/issues';
	const GITHUB_DISCUSSIONS_URL = 'https://github.com/lucor/beebuzz/discussions';
	const MAINTENANCE_POLICY_URL = 'https://lucor.dev/projects/#maintenance-policy';
	const SECURITY_POLICY_URL = 'https://lucor.dev/projects/#security';
</script>

<svelte:head>
	<title>Contact – BeeBuzz</title>
	<meta
		name="description"
		content="How to contact BeeBuzz for support, privacy requests, and bug reports."
	/>
</svelte:head>

<div class="not-prose mb-8 border-b border-base-300 pb-8">
	<p class="text-sm font-semibold uppercase tracking-[0.18em] text-primary">BeeBuzz</p>
	<h1 class="mt-2 text-4xl font-bold text-base-content">Contact</h1>
	<p class="mt-3 max-w-4xl text-base text-base-content/70">
		Support is simple and public. Use GitHub for bugs, features, and questions. Use email for
		privacy and security matters.
	</p>
</div>

<div class="grid gap-6 md:grid-cols-3">
	<section id="security" class="scroll-mt-24 rounded-2xl border border-base-300 bg-base-100 p-6">
		<h2 class="mt-0 flex items-center gap-2 text-xl font-bold text-base-content">
			<Shield class="size-5 shrink-0 text-base-content/60" aria-hidden="true" />
			<span>Security</span>
		</h2>
		<p class="mt-3 text-base leading-7 text-base-content/75">
			See the
			<a
				href={SECURITY_POLICY_URL}
				target="_blank"
				rel="noopener noreferrer"
				class="underline hover:text-base-content"
			>
				security policy
			</a>
			for reporting and disclosure details.
		</p>
	</section>

	<section id="support" class="scroll-mt-24 rounded-2xl border border-base-300 bg-base-100 p-6">
		<h2 class="mt-0 flex items-center gap-2 text-xl font-bold text-base-content">
			<MessageSquare class="size-5 shrink-0 text-base-content/60" aria-hidden="true" />
			<span>Support</span>
		</h2>
		<p class="mt-3 text-base leading-7 text-base-content/75">
			See the <a
				href={MAINTENANCE_POLICY_URL}
				target="_blank"
				rel="noopener noreferrer"
				class="underline hover:text-base-content">maintenance policy</a
			> for support and contribution expectations.
		</p>
		<p class="mt-4 text-base leading-7 text-base-content/75">
			Use GitHub <a
				href={GITHUB_ISSUES_URL}
				target="_blank"
				rel="noopener noreferrer"
				class="underline hover:text-base-content">issues</a
			> for bug reports and feature requests.
		</p>
		<p class="mt-4 text-base leading-7 text-base-content/75">
			Use GitHub <a
				href={GITHUB_DISCUSSIONS_URL}
				target="_blank"
				rel="noopener noreferrer"
				class="underline hover:text-base-content">discussions</a
			> for general questions and product feedback.
		</p>
	</section>

	<section id="privacy" class="scroll-mt-24 rounded-2xl border border-base-300 bg-base-100 p-6">
		<h2 class="mt-0 flex items-center gap-2 text-xl font-bold text-base-content">
			<ScrollText class="size-5 shrink-0 text-base-content/60" aria-hidden="true" />
			<span>Privacy</span>
		</h2>
		<p class="mt-3 text-base leading-7 text-base-content/75">
			Use email for privacy requests and data protection questions.
		</p>
		<p class="mt-4">
			<a href={`mailto:${CONTACT_EMAIL}`} class="link link-primary break-all text-base font-medium">
				{CONTACT_EMAIL}
			</a>
		</p>
		<p class="mt-4 text-base leading-7 text-base-content/75">
			Looking for our privacy policy or terms instead?
			<a href={resolve('/policies')} class="underline hover:text-base-content"
				>Open the policies page</a
			>.
		</p>
	</section>
</div>
