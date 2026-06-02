<script>
	import api from '$lib/api';
	import { onMount } from 'svelte';

	let popupRef;
	let btnRef;

	let projects = $state([]);
	let loading = $state(true);
	let error = $state('');
	let showNew = $state(false);
	let name = $state('');
	let description = $state('');
	let color = $state('');
	let newMessage = $state('');
	let newError = $state('');

	let presetColors = ['#3B82F6', '#8B5CF6', '#EC4899', '#EF4444', '#F59E0B', '#10B981', '#6366F1', '#6B7280'];

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

	function toggleNew() {
		showNew = !showNew;
		if (!showNew) {
			name = '';
			description = '';
			color = '';
			newMessage = '';
			newError = '';
		}
	}

	function onWindowClick(e) {
		if (showNew && popupRef && !popupRef.contains(e.target) && !btnRef.contains(e.target)) {
			showNew = false;
			name = '';
			description = '';
			color = '';
			newMessage = '';
			newError = '';
		}
	}

	async function submitProject() {
		newError = '';
		newMessage = '';
		if (!name.trim()) {
			newError = 'Name is required';
			return;
		}
		try {
			const body = { name };
			if (description) body.description = description;
			if (color) body.color = color;
			const result = await api.post('/api/v1/projects', body);
			newMessage = `Created "${result.name}" — redirecting...`;
			setTimeout(() => window.location.href = `/projects/${result.id}`, 1000);
		} catch (e) {
			newError = e.message;
		}
	}

	async function deleteProject(project, e) {
		e.preventDefault();
		e.stopPropagation();
		if (!confirm(`Delete "${project.name}" and all its data? This cannot be undone.`)) return;
		try {
			await api.del(`/api/v1/projects/${project.id}`);
			projects = projects.filter(p => p.id !== project.id);
		} catch (err) {
			alert(err.message);
		}
	}
</script>

<svelte:window onclick={onWindowClick} />

<div class="flex items-center justify-between mb-6 relative">
	<h2 class="text-2xl font-bold">Projects</h2>
	<div class="relative" bind:this={btnRef}>
		<button onclick={toggleNew} class="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded text-white text-sm">
			New Project
		</button>
		{#if showNew}
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div bind:this={popupRef} class="absolute right-0 top-full mt-2 w-80 bg-gray-800 border border-gray-600 rounded-lg shadow-xl z-50 p-4 space-y-3">
				<div>
					<label for="proj-name" class="block text-sm text-gray-400 mb-1">Name</label>
					<input id="proj-name" bind:value={name} class="w-full p-2 bg-gray-900 border border-gray-600 rounded text-sm" required />
				</div>
				<div>
					<label for="proj-desc" class="block text-sm text-gray-400 mb-1">Description</label>
					<input id="proj-desc" bind:value={description} class="w-full p-2 bg-gray-900 border border-gray-600 rounded text-sm" />
				</div>
				<div>
					<span class="block text-sm text-gray-400 mb-1">Color</span>
					<div class="flex gap-2 flex-wrap">
						{#each presetColors as c}
							<button
								onclick={() => color = color === c ? '' : c}
								aria-label="Select color {c}"
								class="w-7 h-7 rounded-full border-2 {color === c ? 'border-white' : 'border-transparent'} transition-colors"
								style="background: {c}">
							</button>
						{/each}
					</div>
				</div>
				{#if newError}
					<p class="text-red-400 text-sm">{newError}</p>
				{/if}
				{#if newMessage}
					<p class="text-green-400 text-sm">{newMessage}</p>
				{/if}
				<button onclick={submitProject} class="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded text-white text-sm">
					Create Project
				</button>
			</div>
		{/if}
	</div>
</div>

{#if loading}
	<p class="text-gray-400">Loading...</p>
{:else if error}
	<p class="text-red-400">{error}</p>
{:else if projects.length === 0}
	<p class="text-gray-400">No projects yet. Create one to get started.</p>
{:else}
	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
		{#each projects as project (project.id)}
			<a href="/projects/{project.id}" class="block p-4 bg-gray-800 rounded-lg border border-gray-700 hover:border-gray-600 transition-colors group relative">
				<button onclick={(e) => deleteProject(project, e)} class="absolute top-2 right-2 text-gray-500 hover:text-red-400 opacity-0 group-hover:opacity-100 transition-opacity" title="Delete project">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
				</button>
				<div class="flex items-center gap-2 mb-1">
					{#if project.color}
						<span class="w-3 h-3 rounded-full shrink-0" style="background: {project.color}"></span>
					{/if}
					<h3 class="text-lg font-semibold">{project.name}</h3>
				</div>
				{#if project.description}
					<p class="text-gray-400 text-sm">{project.description}</p>
				{/if}
			</a>
		{/each}
	</div>
{/if}
