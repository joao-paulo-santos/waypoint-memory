<script>
	import api from '$lib/api';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';

	let id = $derived($page.params.id);
	let labels = $state([]);
	let loading = $state(true);
	let error = $state('');

	let newTitle = $state('');
	let newColor = $state('#6366f1');
	let newDesc = $state('');
	let showCreate = $state(false);

	const palette = [
		'#ef4444', '#f97316', '#f59e0b', '#84cc16', '#22c55e',
		'#14b8a6', '#06b6d4', '#3b82f6', '#6366f1', '#8b5cf6',
		'#a855f7', '#d946ef', '#ec4899', '#f43f5e', '#64748b'
	];

	async function load() {
		loading = true;
		error = '';
		try {
			const data = await api.get(`/api/v1/projects/${id}/labels`);
			labels = data.labels || [];
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(load);

	async function createLabel() {
		if (!newTitle.trim()) return;
		try {
			await api.post(`/api/v1/projects/${id}/labels`, {
				title: newTitle,
				description: newDesc,
				hex_color: newColor
			});
			newTitle = '';
			newDesc = '';
			newColor = '#6366f1';
			showCreate = false;
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function deleteLabel(lid) {
		if (!confirm('Delete this label?')) return;
		try {
			await api.del(`/api/v1/projects/${id}/labels/${lid}`);
			await load();
		} catch (e) {
			alert(e.message);
		}
	}
</script>

<div class="mb-4 flex items-center justify-between">
	<div class="flex items-center gap-3">
		<a href="/projects/{id}" class="text-gray-400 hover:text-white text-sm">&larr; Board</a>
		<h2 class="text-2xl font-bold">Labels</h2>
	</div>
	<button onclick={() => (showCreate = !showCreate)} class="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 rounded text-sm">
		{showCreate ? 'Cancel' : '+ New Label'}
	</button>
</div>

{#if showCreate}
	<div class="bg-gray-800 rounded-lg p-4 mb-4">
		<div class="flex gap-3 items-end">
			<div class="flex-1">
				<label for="label-title" class="block text-sm text-gray-400 mb-1">Title</label>
				<input id="label-title" bind:value={newTitle} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" placeholder="Label title" />
			</div>
			<div class="w-48">
				<label for="label-color" class="block text-sm text-gray-400 mb-1">Color</label>
				<div class="flex items-center gap-2">
					<input id="label-color" type="color" bind:value={newColor} class="w-8 h-8 rounded cursor-pointer border-0 bg-transparent" />
					<input bind:value={newColor} class="flex-1 p-2 bg-gray-700 border border-gray-600 rounded text-sm font-mono" />
				</div>
			</div>
		</div>
		<div class="mt-2">
			<label for="label-desc" class="block text-sm text-gray-400 mb-1">Description (optional)</label>
			<input id="label-desc" bind:value={newDesc} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" placeholder="Optional description" />
		</div>
		<div class="mt-2 flex gap-1.5 flex-wrap">
			{#each palette as color}
				<button
					onclick={() => (newColor = color)}
					class="w-6 h-6 rounded-full border-2 {newColor === color ? 'border-white' : 'border-transparent'}"
					style="background: {color}"
					title={color}></button>
			{/each}
		</div>
		<div class="mt-3">
			<button onclick={createLabel} class="px-4 py-1.5 bg-blue-600 hover:bg-blue-700 rounded text-sm">Create Label</button>
		</div>
	</div>
{/if}

{#if loading}
	<p class="text-gray-400">Loading labels...</p>
{:else if error}
	<p class="text-red-400">{error}</p>
{:else if labels.length === 0}
	<p class="text-gray-500">No labels yet. Create one to get started.</p>
{:else}
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
		{#each labels as label (label.id)}
			<div class="bg-gray-800 rounded-lg p-4 flex items-center justify-between">
				<div class="flex items-center gap-3">
					<span class="w-5 h-5 rounded-full shrink-0" style="background: {label.hex_color || '#555'}"></span>
					<div>
						<span class="font-medium">{label.title}</span>
						{#if label.description}
							<p class="text-xs text-gray-500 mt-0.5">{label.description}</p>
						{/if}
					</div>
				</div>
				<button onclick={() => deleteLabel(label.id)} class="text-gray-500 hover:text-red-400 text-sm" title="Delete label">&#128465;</button>
			</div>
		{/each}
	</div>
{/if}
