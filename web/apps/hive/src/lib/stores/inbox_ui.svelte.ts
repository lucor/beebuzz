import { SvelteSet } from 'svelte/reactivity';

function createInboxUiStore() {
	let selectionMode = $state(false);
	const selectedIds = new SvelteSet<string>();

	function enterSelection(initialId?: string) {
		selectionMode = true;
		if (initialId) {
			selectedIds.add(initialId);
		}
	}

	function exitSelection() {
		selectionMode = false;
		selectedIds.clear();
	}

	function toggleSelection(id: string) {
		selectionMode = true;
		if (selectedIds.has(id)) {
			selectedIds.delete(id);
			return;
		}
		selectedIds.add(id);
	}

	/** Selects multiple IDs at once. When replace is true, clears existing selection first. */
	function selectMany(ids: string[], replace = false) {
		selectionMode = true;
		if (replace) selectedIds.clear();
		for (const id of ids) selectedIds.add(id);
	}

	/** Deselects multiple IDs at once. */
	function deselectMany(ids: string[]) {
		for (const id of ids) selectedIds.delete(id);
	}

	/** Toggles all provided IDs: selects all if not fully selected, else deselects all. */
	function toggleAll(ids: string[]) {
		selectionMode = true;
		const allSelected = ids.length > 0 && ids.every((id) => selectedIds.has(id));
		if (allSelected) {
			deselectMany(ids);
		} else {
			selectMany(ids);
		}
	}

	/** Keeps only selected IDs that still exist in the feed. */
	function pruneSelection(validIds: string[]) {
		if (selectedIds.size === 0) return;

		const validIdSet = new Set(validIds);
		for (const id of selectedIds) {
			if (!validIdSet.has(id)) {
				selectedIds.delete(id);
			}
		}
	}

	return {
		get selectionMode() {
			return selectionMode;
		},
		get selectedIds() {
			return selectedIds;
		},
		get selectedCount() {
			return selectedIds.size;
		},
		enterSelection,
		exitSelection,
		toggleSelection,
		selectMany,
		deselectMany,
		toggleAll,
		pruneSelection
	};
}

export const inboxUiStore = createInboxUiStore();
