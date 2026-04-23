<script lang="ts">
	import { toast } from '../stores';

	const typeClasses: Record<string, string> = {
		success: 'alert-success',
		error: 'alert-error',
		info: 'alert-info'
	};
</script>

{#if toast.current}
	<div class="toast toast-bottom toast-center z-50">
		<div class="alert {typeClasses[toast.current.type] ?? 'alert-info'} shadow-lg">
			<span>{toast.current.message}</span>
		</div>
	</div>
{/if}
