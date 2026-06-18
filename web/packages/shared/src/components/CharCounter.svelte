<script lang="ts">
	import { MAX_DISPLAY_NAME_LEN, MAX_DISPLAY_NAME_SOFT_LEN } from '../constants/validation';

	let { value }: { value: number } = $props();

	function tone(length: number): string {
		if (length > MAX_DISPLAY_NAME_LEN) return 'text-error';
		if (length > MAX_DISPLAY_NAME_SOFT_LEN) return 'text-warning';
		return 'text-base-content/70';
	}
</script>

<p class="text-xs tabular-nums whitespace-nowrap {tone(value)}">
	({value}/{MAX_DISPLAY_NAME_LEN})
</p>
