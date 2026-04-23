<script lang="ts">
	import { resolve } from '$app/paths';

	const LAST_UPDATED = 'April 23, 2026';
	const CONTACT_EMAIL = 'support@beebuzz.app';
	const GITHUB_REPO_URL = 'https://github.com/lucor/beebuzz';
</script>

<svelte:head>
	<title>Policies – BeeBuzz</title>
	<meta
		name="description"
		content="Privacy Policy, Terms of Service, and Cookie Policy for BeeBuzz at beebuzz.app."
	/>
</svelte:head>

<div class="not-prose mb-8 border-b border-base-300 pb-8">
	<h1 class="mt-2 text-4xl font-bold text-base-content">Policies</h1>
	<p class="mt-2 text-w-4xl text-base text-base-content/70">
		This page applies only to BeeBuzz, the service we operate at
		<a href="https://beebuzz.app" class="underline hover:text-base-content">beebuzz.app</a>. It
		covers how BeeBuzz handles personal data, account access, device registration, push delivery,
		and the limited cookies required to run the service.
	</p>
	<div class="mt-4 flex flex-col gap-1 text-sm text-base-content/60">
		<p>Last updated: {LAST_UPDATED}</p>
		<p>
			Questions?
			<a href={resolve('/contact')} class="underline hover:text-base-content">Contact us</a>
			or email
			<a href={`mailto:${CONTACT_EMAIL}`} class="underline hover:text-base-content">
				{CONTACT_EMAIL}
			</a>
		</p>
	</div>
</div>

<section id="privacy" class="scroll-mt-24">
	<h2>Privacy Policy</h2>

	<h3>Who we are</h3>
	<p>
		BeeBuzz is operated by Luca Corbo, an individual developer based in the EU. For data protection
		purposes, Luca Corbo is the controller of personal data processed through the BeeBuzz service
		operated at beebuzz.app. In this page, "BeeBuzz", "we", "us", and "our" refer to Luca Corbo as
		the operator of that service.
	</p>
	<p>
		You can reach us at
		<a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a>
		for any privacy or data protection questions.
	</p>

	<h3>Our approach</h3>
	<p>
		BeeBuzz is built to keep data collection narrow. We do not sell personal data. We do not use
		advertising trackers on BeeBuzz. We only keep the account, device, and delivery data needed to
		run the service.
	</p>

	<h3>Information we collect</h3>
	<p>Depending on how you use BeeBuzz, we may store:</p>
	<ul>
		<li>your email address for sign-in, account access, and support</li>
		<li>
			basic account records such as your user ID, account status, signup reason (when provided), and
			paired devices
		</li>
		<li>session records and essential cookies used to keep you signed in securely</li>
		<li>device names you choose and Web Push subscription details used to deliver notifications</li>
		<li>public encryption keys used for end-to-end delivery on paired devices</li>
		<li>
			delivery metadata needed to route messages, remove invalid subscriptions, and enforce limits
		</li>
		<li>
			limited operational logs and IP-based security data used for abuse prevention and debugging
		</li>
	</ul>

	<h3>Why we process your data</h3>
	<p>We process personal data for the following purposes and legal bases:</p>
	<ul>
		<li>
			<strong>Contract / service delivery:</strong> account creation, sign-in, device pairing, push delivery,
			and related service functionality. This processing is necessary to provide BeeBuzz to you.
		</li>
		<li>
			<strong>Legitimate interests:</strong> abuse prevention, rate limiting, security logging, and debugging.
			These activities protect the hosted service and its users.
		</li>
	</ul>
	<p>
		We do not currently rely on consent as a legal basis. If that changes in the future (for
		example, if we add optional analytics or marketing emails), we will update this policy and
		request your consent before processing.
	</p>

	<h3>Messages and delivery</h3>
	<p>
		BeeBuzz supports two delivery models. In <strong>server-trusted delivery</strong>, BeeBuzz can
		process notification payloads and attachments in order to send them to your paired devices. In
		<strong>end-to-end encrypted delivery</strong>, BeeBuzz stores and forwards only ciphertext, not
		plaintext message bodies.
	</p>
	<p>
		Even in end-to-end encrypted mode, BeeBuzz still processes metadata needed to operate the
		service, such as account records, device registrations, public encryption keys, delivery state,
		and temporary attachment or envelope records.
	</p>

	<h3>Third-party services</h3>
	<p>
		BeeBuzz uses third-party services where needed to operate. The following third parties may
		process personal data on our behalf:
	</p>
	<ul>
		<li>
			<strong>Web Push providers</strong> operated by browser and platform vendors (such as Google, Mozilla,
			and Apple) handle the technical delivery step required to transport push notifications to your devices.
		</li>
		<li>
			<strong>GlitchTip</strong> is used for error monitoring and debugging of unexpected service failures.
			It may process limited technical metadata related to requests and server errors. BeeBuzz is configured
			not to intentionally send message contents, email addresses, tokens, or other sensitive data to
			this service.
		</li>
		<li>
			<strong>Resend</strong> is used to send account emails such as sign-in or access-related messages.
			Your email address and basic email delivery metadata are processed by Resend when those messages
			are sent.
		</li>
	</ul>

	<h3>International transfers</h3>
	<p>
		Some third-party services used by BeeBuzz, such as Resend and browser push providers, may
		process data outside the European Economic Area. Where applicable, these transfers rely on the
		safeguards provided by those services, such as Standard Contractual Clauses or adequacy
		decisions.
	</p>

	<h3>Data retention</h3>
	<p>
		We keep data for as long as it is needed to operate BeeBuzz. Account and device data are kept
		while your account is active. We also keep per-account daily usage summaries to provide usage
		history and operate the service. Session data, temporary delivery records, raw event records,
		and operational logs are retained for shorter periods and removed as they expire or are no
		longer needed. If you request account deletion, we will delete the personal data associated with
		your account unless we must retain limited information for legal compliance, security, abuse
		prevention, or the establishment, exercise, or defense of legal claims.
	</p>

	<h3>Your rights</h3>
	<p>Under applicable data protection law, you have the right to:</p>
	<ul>
		<li>access the personal data we hold about you</li>
		<li>request correction of inaccurate data</li>
		<li>request deletion of your account and associated data</li>
		<li>request restriction of processing</li>
		<li>request data portability</li>
		<li>object to processing based on legitimate interests</li>
		<li>lodge a complaint with your local data protection supervisory authority</li>
	</ul>
	<p>
		To exercise any of these rights, visit our
		<a href={resolve('/contact')}>contact page</a>. BeeBuzz does not currently provide a
		self-service account deletion button. We may need to verify your identity before acting on a
		request, and we will respond within the time required by applicable law.
	</p>

	<h3>Required data</h3>
	<p>
		A valid email address is required to create and access a BeeBuzz account. Without it, we cannot
		provide the hosted service.
	</p>

	<h3>Automated decision-making</h3>
	<p>
		BeeBuzz does not use automated decision-making or profiling with legal or similarly significant
		effects.
	</p>

	<h3>Your choices</h3>
	<p>Beyond formal data rights, you also have practical options:</p>
	<ul>
		<li>you can remove paired devices from your account</li>
		<li>
			you can <a href={resolve('/contact')}>contact us</a> to request deletion of your account and associated
			data
		</li>
		<li>
			you can
			<a href={GITHUB_REPO_URL} target="_blank" rel="noopener noreferrer">self-host BeeBuzz</a>
			if you want full control over storage and retention
		</li>
	</ul>

	<h3>Changes to this policy</h3>
	<p>
		We may update this Privacy Policy as BeeBuzz evolves. Changes will be posted on this page with
		an updated "Last updated" date.
	</p>
</section>

<section id="terms" class="scroll-mt-24">
	<h2>Terms of Service</h2>

	<h3>Scope</h3>
	<p>
		These terms apply only to the BeeBuzz service we operate at beebuzz.app. The BeeBuzz source code
		is separately licensed under
		<a href={`${GITHUB_REPO_URL}/blob/main/LICENSE`} target="_blank" rel="noopener noreferrer"
			>AGPL-3.0-only</a
		> for self-hosted use.
	</p>

	<h3>Acceptance</h3>
	<p>
		These terms apply to BeeBuzz as served by us through the public BeeBuzz site and Hive app at
		beebuzz.app. By requesting access, signing in, or using the service, you agree to these terms.
	</p>

	<h3>Your responsibilities</h3>
	<ul>
		<li>
			You must provide a valid email address that you control in order to access your account.
		</li>
		<li>
			You are responsible for activity performed with your account, API tokens, and paired devices.
		</li>
		<li>
			You may not use BeeBuzz for abusive traffic, unlawful content, credential abuse, or attempts
			to disrupt the service or other users.
		</li>
	</ul>

	<h3>Our rights</h3>
	<ul>
		<li>
			We may rate-limit, suspend, or block access when needed to protect the service, enforce beta
			access rules, or respond to misuse.
		</li>
		<li>We may change, limit, or discontinue beta features while the product is still evolving.</li>
	</ul>

	<h3>Termination</h3>
	<p>
		You may stop using BeeBuzz at any time. If you want your account deleted,
		<a href={resolve('/contact')}>contact us</a> to request deletion. We may suspend or terminate your
		access if you violate these terms or if continued access would harm the service or other users.
	</p>

	<h3>Disclaimers</h3>
	<p>
		BeeBuzz is provided on an as-available basis during its beta period with no uptime guarantee.
		Delivery depends on third-party browser and push infrastructure, so uninterrupted or error-free
		operation cannot be guaranteed.
	</p>
	<p>
		BeeBuzz is not intended for emergency, life-safety, medical, or other time-critical
		notifications.
	</p>
	<p>
		You remain responsible for the notification content you send through the service and for
		complying with the laws and policies that apply to that content.
	</p>

	<h3>Liability</h3>
	<p>
		To the extent permitted by applicable law, we are not liable for indirect, incidental, or
		consequential damages arising from your use of BeeBuzz.
	</p>

	<h3>Governing law and disputes</h3>
	<p>
		These terms are governed by the laws of Italy. Any dispute arising from or relating to these
		terms or your use of BeeBuzz shall be submitted to the exclusive jurisdiction of the courts of
		Italy, unless mandatory consumer protection rules in your country of residence provide
		otherwise.
	</p>
</section>

<section id="cookies" class="scroll-mt-24">
	<h2>Cookie Policy</h2>

	<h3>Essential cookies</h3>
	<p>
		BeeBuzz uses only essential cookies required for authentication and account access across
		BeeBuzz subdomains.
	</p>
	<ul>
		<li>
			<code>beebuzz_session</code>: the main server-side session cookie, used to authenticate
			requests and maintain your signed-in state. Scoped to the <code>.beebuzz.app</code> domain.
			Set with <code>HttpOnly</code>, <code>Secure</code>, and <code>SameSite=Lax</code>. Expires
			when the server-side session ends.
		</li>
		<li>
			<code>beebuzz_logged_in</code>: a lightweight client-side cookie used by the frontend to
			detect whether a session exists, avoiding unnecessary redirects. Scoped to the
			<code>.beebuzz.app</code> domain. Set with <code>Secure</code> and
			<code>SameSite=Lax</code>. Not set as <code>HttpOnly</code> because it is read by the frontend.
			Expires with the session.
		</li>
	</ul>
	<p>
		These cookies are required for BeeBuzz to function. They are not used for advertising or
		cross-site tracking. Blocking them will prevent sign-in and authenticated access.
	</p>

	<h3>Other storage</h3>
	<p>
		BeeBuzz also uses limited browser storage during login and device onboarding flows, such as
		temporary sign-in state in session storage and local device state in the Hive app. That storage
		supports core product functionality rather than cross-site tracking.
	</p>

	<h3>Optional cookies</h3>
	<p>
		BeeBuzz does not currently set optional advertising, personalization, or analytics cookies. If
		that changes, this page will be updated before those tools are enabled.
	</p>
</section>
