<script>
	import api from '$lib/api';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';

	let id = $derived($page.params.id);
	let project = $state(null);
	let sprints = $state([]);
	let loading = $state(true);
	let error = $state('');

	async function load() {
		loading = true;
		error = '';
		try {
			const [pData, sData] = await Promise.all([
				api.get(`/api/v1/projects/${id}`),
				api.get(`/api/v1/projects/${id}/sprints`)
			]);
			project = pData;
			sprints = sData.sprints || [];
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(load);

	function formatDate(d) {
		if (!d) return '';
		return d.slice(0, 10);
	}
</script>

<div class="mb-4 flex items-center justify-between">
	<div class="flex items-center gap-3">
		<a href="/projects/{id}" class="text-gray-400 hover:text-white text-sm">&larr; Board</a>
		<h2 class="text-2xl font-bold">Sprints</h2>
		{#if project}
			<span class="text-gray-400 text-sm">— {project.name}</span>
		{/if}
	</div>
</div>

{#if loading}
	<p class="text-gray-400">Loading sprints...</p>
{:else if error}
	<p class="text-red-400">{error}</p>
{:else if sprints.length === 0}
	<p class="text-gray-500">No sprints archived yet. End a sprint from the board to see it here.</p>
{:else}
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
		{#each sprints as sprint (sprint.id)}
			<a href="/projects/{id}/sprints/{sprint.id}" class="block bg-gray-800 rounded-lg p-4 hover:bg-gray-750 hover:border-gray-600 border border-gray-700 transition-colors">
				<div class="flex items-center justify-between mb-2">
					<h3 class="font-semibold">{sprint.name}</h3>
					<span class="text-xs px-2 py-0.5 bg-gray-700 rounded">{sprint.task_count} task{#if sprint.task_count !== 1}s{/if}</span>
				</div>
				{#if sprint.summary}
					<p class="text-sm text-gray-400 mb-2 line-clamp-2">{sprint.summary}</p>
				{/if}
				<div class="text-xs text-gray-500">Ended {formatDate(sprint.ended_at)}</div>
			</a>
		{/each}
	</div>
{/if}
