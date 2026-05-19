<script>
	import api from '$lib/api';
	import { onMount } from 'svelte';

	let projects = $state([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			const data = await api.get('/api/v1/projects');
			projects = data.projects || [];
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	});
</script>

<div class="flex items-center justify-between mb-6">
	<h2 class="text-2xl font-bold">Projects</h2>
	<a href="/projects/new" class="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded text-white text-sm">New Project</a>
</div>

{#if loading}
	<p class="text-gray-400">Loading...</p>
{:else if error}
	<p class="text-red-400">{error}</p>
{:else if projects.length === 0}
	<p class="text-gray-400">No projects yet. Create one to get started.</p>
{:else}
	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
		{#each projects as project}
			<a href="/projects/{project.id}" class="block p-4 bg-gray-800 rounded-lg border border-gray-700 hover:border-gray-600 transition-colors">
				<h3 class="text-lg font-semibold mb-1">{project.name}</h3>
				{#if project.description}
					<p class="text-gray-400 text-sm">{project.description}</p>
				{/if}
			</a>
		{/each}
	</div>
{/if}
