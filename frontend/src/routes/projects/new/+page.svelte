<script>
	import api from '$lib/api';

	let tab = $state('create');
	let name = $state('');
	let path = $state('');
	let description = $state('');
	let message = $state('');
	let error = $state('');

	async function createProject() {
		error = '';
		message = '';
		try {
			const result = await api.post('/api/v1/projects/create', { name, description });
			message = `Created "${result.name}" — redirecting...`;
			setTimeout(() => window.location.href = `/projects/${result.id}`, 1000);
		} catch (e) {
			error = e.message;
		}
	}

	async function initProject() {
		error = '';
		message = '';
		try {
			const result = await api.post('/api/v1/projects/init', { path, name, description });
			message = `Initialized "${result.name}" — redirecting...`;
			setTimeout(() => window.location.href = `/projects/${result.id}`, 1000);
		} catch (e) {
			error = e.message;
		}
	}

	async function addProject() {
		error = '';
		message = '';
		try {
			const result = await api.post('/api/v1/projects/add', { path, name });
			message = `Added "${result.name}" — redirecting...`;
			setTimeout(() => window.location.href = `/projects/${result.id}`, 1000);
		} catch (e) {
			error = e.message;
		}
	}
</script>

<h2 class="text-2xl font-bold mb-6">New Project</h2>

<div class="flex gap-2 mb-6">
	<button class="px-4 py-2 rounded {tab === 'create' ? 'bg-blue-600 text-white' : 'bg-gray-700 text-gray-300'}" onclick={() => (tab = 'create')}>Create</button>
	<button class="px-4 py-2 rounded {tab === 'init' ? 'bg-blue-600 text-white' : 'bg-gray-700 text-gray-300'}" onclick={() => (tab = 'init')}>Initialize</button>
	<button class="px-4 py-2 rounded {tab === 'add' ? 'bg-blue-600 text-white' : 'bg-gray-700 text-gray-300'}" onclick={() => (tab = 'add')}>Add Existing</button>
</div>

{#if error}
	<p class="text-red-400 mb-4">{error}</p>
{/if}
{#if message}
	<p class="text-green-400 mb-4">{message}</p>
{/if}

{#if tab === 'create'}
	<div class="max-w-md space-y-4">
		<div><label class="block text-sm text-gray-400 mb-1">Name</label><input bind:value={name} class="w-full p-2 bg-gray-800 border border-gray-600 rounded" /></div>
		<div><label class="block text-sm text-gray-400 mb-1">Description</label><input bind:value={description} class="w-full p-2 bg-gray-800 border border-gray-600 rounded" /></div>
		<button onclick={createProject} class="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded text-white">Create Project</button>
	</div>
{:else if tab === 'init'}
	<div class="max-w-md space-y-4">
		<div><label class="block text-sm text-gray-400 mb-1">Directory Path</label><input bind:value={path} class="w-full p-2 bg-gray-800 border border-gray-600 rounded" placeholder="/path/to/project" /></div>
		<div><label class="block text-sm text-gray-400 mb-1">Name (optional)</label><input bind:value={name} class="w-full p-2 bg-gray-800 border border-gray-600 rounded" /></div>
		<div><label class="block text-sm text-gray-400 mb-1">Description</label><input bind:value={description} class="w-full p-2 bg-gray-800 border border-gray-600 rounded" /></div>
		<button onclick={initProject} class="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded text-white">Initialize</button>
	</div>
{:else}
	<div class="max-w-md space-y-4">
		<div><label class="block text-sm text-gray-400 mb-1">Directory Path</label><input bind:value={path} class="w-full p-2 bg-gray-800 border border-gray-600 rounded" placeholder="/path/to/existing/project" /></div>
		<div><label class="block text-sm text-gray-400 mb-1">Name (optional)</label><input bind:value={name} class="w-full p-2 bg-gray-800 border border-gray-600 rounded" /></div>
		<button onclick={addProject} class="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded text-white">Add Project</button>
	</div>
{/if}
