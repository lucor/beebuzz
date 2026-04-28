<script lang="ts">
	interface Props {
		open: boolean;
		title: string;
		description?: string;
		size?: 'sm' | 'md' | 'lg' | 'xl';
		showClose?: boolean;
		children: import('svelte').Snippet;
		actions?: import('svelte').Snippet;
		onClose?: () => void;
	}

	let {
		open,
		title,
		description,
		size = 'md',
		showClose = true,
		children,
		actions,
		onClose
	}: Props = $props();

	const sizeClasses: Record<NonNullable<Props['size']>, string> = {
		sm: 'max-w-sm',
		md: 'max-w-md',
		lg: 'max-w-lg',
		xl: 'max-w-2xl'
	};

	const handleClose = () => {
		onClose?.();
	};

	// Stable id for aria-labelledby so the dialog has an accessible name.
	const titleId = `settings-modal-title-${Math.random().toString(36).slice(2, 10)}`;
</script>

{#if open}
	<dialog
		class="modal"
		open
		aria-modal="true"
		aria-labelledby={titleId}
		oncancel={(e) => {
			e.preventDefault();
			handleClose();
		}}
	>
		<div
			class={`modal-box w-full ${sizeClasses[size]} bg-base-100 text-base-content border border-base-300`}
		>
			<div class="flex items-start justify-between gap-4">
				<div>
					<h3 id={titleId} class="text-lg font-bold text-base-content">{title}</h3>
					{#if description}
						<p class="text-sm text-base-content/70 mt-1">{description}</p>
					{/if}
				</div>
				{#if showClose}
					<button
						type="button"
						class="btn btn-sm btn-ghost"
						onclick={handleClose}
						aria-label="Close"
					>
						✕
					</button>
				{/if}
			</div>

			<div class="mt-4">
				{@render children()}
			</div>

			{#if actions}
				<div
					class="mt-6 flex flex-col items-stretch gap-2 sm:flex-row sm:justify-end [&>*]:w-full sm:[&>*]:w-auto"
				>
					{@render actions()}
				</div>
			{/if}
		</div>
		<button type="button" class="modal-backdrop" aria-label="Close modal" onclick={handleClose}
		></button>
	</dialog>
{/if}
