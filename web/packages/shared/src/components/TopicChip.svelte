<script lang="ts">
	interface Props {
		name: string;
		muted?: boolean;
	}

	let { name, muted = false }: Props = $props();
</script>

<span
	class={`badge whitespace-nowrap gap-1 border-base-300 ${muted ? 'badge-ghost text-base-content/70' : 'badge-ghost text-base-content'}`}
>
	#{name}
</span>
