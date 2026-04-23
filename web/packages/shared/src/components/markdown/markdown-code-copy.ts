const COPY_RESET_DELAY_MS = 2000;

function setIconVisibility(icon: Element | null, visible: boolean) {
	if (!(icon instanceof SVGElement)) {
		return;
	}

	icon.style.display = visible ? 'block' : 'none';
	icon.setAttribute('aria-hidden', visible ? 'false' : 'true');
}

function setCopyState(button: HTMLButtonElement, state: 'copy' | 'check', statusText: string) {
	const copyIcon = button.querySelector('[data-copy-icon="copy"]');
	const checkIcon = button.querySelector('[data-copy-icon="check"]');
	const status = button.querySelector<HTMLElement>('[data-copy-status]');

	setIconVisibility(copyIcon, state === 'copy');
	setIconVisibility(checkIcon, state === 'check');

	if (status) {
		status.textContent = statusText;
	}
}

/** Handles copy button clicks inside rendered markdown code blocks. */
export function markdownCodeCopy(node: HTMLElement) {
	const resetTimeouts = new WeakMap<HTMLButtonElement, ReturnType<typeof setTimeout>>();

	node.querySelectorAll<HTMLButtonElement>('[data-docs-copy-button]').forEach((button) => {
		setCopyState(button, 'copy', '');
	});

	function scheduleReset(button: HTMLButtonElement) {
		const existing = resetTimeouts.get(button);
		if (existing) {
			clearTimeout(existing);
		}

		const timeout = setTimeout(() => {
			setCopyState(button, 'copy', '');
			resetTimeouts.delete(button);
		}, COPY_RESET_DELAY_MS);

		resetTimeouts.set(button, timeout);
	}

	async function copyFromEvent(event: MouseEvent) {
		const target = event.target as HTMLElement | null;
		const button = target?.closest('[data-docs-copy-button]');

		if (!(button instanceof HTMLButtonElement) || !node.contains(button)) {
			return;
		}

		const code = button.dataset.docsCopyCode;
		if (!code) {
			return;
		}

		try {
			await navigator.clipboard.writeText(code);
			setCopyState(button, 'check', 'Code copied to clipboard');
			scheduleReset(button);
		} catch (error) {
			console.error('Failed to copy markdown code block', error);
			setCopyState(button, 'copy', 'Copy failed');
			scheduleReset(button);
		}
	}

	const handleClick = (event: MouseEvent) => {
		void copyFromEvent(event);
	};

	node.addEventListener('click', handleClick);

	return {
		destroy() {
			node.removeEventListener('click', handleClick);
		}
	};
}
