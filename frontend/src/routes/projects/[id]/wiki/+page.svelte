<script>
	import api from '$lib/api';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { renderMarkdown } from '$lib/sanitize';

	let id = $derived($page.params.id);
	let project = $state(null);
	let wikiPages = $state([]);
	let loading = $state(true);
	let error = $state('');

	let selectedSlug = $state('');
	let pageContent = $state(null);
	let pageLoading = $state(false);
	let pageError = $state('');

	let editing = $state(false);
	let editContent = $state('');

	let renderedHtml = $derived(pageContent ? renderMarkdown(pageContent.content) : '');

	async function load() {
		loading = true;
		error = '';
		try {
			const [pData, wData] = await Promise.all([
				api.get(`/api/v1/projects/${id}`),
				api.get(`/api/v1/projects/${id}/wiki`)
			]);
			project = pData;
			wikiPages = wData.pages || [];
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(load);

	async function selectPage(slug) {
		selectedSlug = slug;
		editing = false;
		pageLoading = true;
		pageError = '';
		try {
			pageContent = await api.get(`/api/v1/projects/${id}/wiki/${slug}`);
		} catch (e) {
			pageError = e.message;
			pageContent = null;
		} finally {
			pageLoading = false;
		}
	}

	function startEdit() {
		if (!pageContent) return;
		editContent = pageContent.content || '';
		editing = true;
	}

	async function savePage() {
		if (!selectedSlug) return;
		try {
			await api.put(`/api/v1/projects/${id}/wiki/${selectedSlug}`, { content: editContent });
			editing = false;
			await selectPage(selectedSlug);
		} catch (e) {
			alert(e.message);
		}
	}

	function cancelEdit() {
		editing = false;
		editContent = '';
	}

	async function deletePage(slug) {
		if (!confirm(`Delete page "${slug}"?`)) return;
		try {
			await api.del(`/api/v1/projects/${id}/wiki/${slug}`);
			if (selectedSlug === slug) {
				selectedSlug = '';
				pageContent = null;
			}
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function newPage() {
		let name = prompt('Page name (e.g. meeting-notes):');
		if (!name || !name.trim()) return;
		const slug = name.trim().toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '');
		if (!slug) return;
		try {
			await api.put(`/api/v1/projects/${id}/wiki/${slug}`, { content: `# ${name.trim()}\n\n` });
			await load();
			await selectPage(slug);
		} catch (e) {
			alert(e.message);
		}
	}
</script>

<svelte:head>
	<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css">
</svelte:head>

<div class="mb-4 flex items-center gap-3">
	<a href="/projects/{id}" class="text-gray-400 hover:text-white text-sm">&larr; Board</a>
	<h2 class="text-2xl font-bold">Wiki</h2>
	{#if project}
		<span class="text-gray-400 text-sm">&mdash; {project.name}</span>
	{/if}
</div>

{#if loading}
	<p class="text-gray-400">Loading...</p>
{:else if error}
	<p class="text-red-400">{error}</p>
{:else}
	{#if project?.description}
		<p class="text-gray-400 text-sm mb-4">{project.description}</p>
	{/if}

	<div class="flex gap-4 h-[calc(100vh-200px)]">
		<div class="w-64 shrink-0 bg-gray-800 rounded-lg p-3 overflow-y-auto flex flex-col">
			{#if wikiPages.length > 0}
				<h3 class="text-sm font-semibold text-gray-400 uppercase mb-2">Pages</h3>
				<div class="space-y-0.5 flex-1">
					{#each wikiPages as p (p.slug || p.path || p.name)}
						<div class="flex items-center group">
							<button
								onclick={() => selectPage(p.slug || p.path || p.name)}
								class="flex-1 text-left text-sm py-1 px-2 rounded hover:bg-gray-700 {selectedSlug === (p.slug || p.path || p.name) ? 'bg-gray-700 text-white' : 'text-gray-300'}">
								&#128196; {p.title || p.slug || p.name}
							</button>
							<button
								onclick={() => deletePage(p.slug || p.path || p.name)}
								class="text-gray-600 hover:text-red-400 text-xs px-1 opacity-0 group-hover:opacity-100"
								title="Delete page">
								✕
							</button>
						</div>
					{/each}
				</div>
			{:else}
				<p class="text-sm text-gray-500 mb-2">No wiki pages yet.</p>
			{/if}
			<button onclick={newPage} class="mt-2 w-full text-left text-sm text-gray-500 hover:text-gray-300 px-2 py-1 hover:bg-gray-700 rounded">+ New page</button>
		</div>

		<div class="flex-1 bg-gray-800 rounded-lg p-6 overflow-y-auto">
			{#if pageLoading}
				<p class="text-gray-400">Loading page...</p>
			{:else if pageError}
				<p class="text-red-400">{pageError}</p>
			{:else if pageContent}
				<div class="flex items-center justify-between mb-4">
					<h2 class="text-xl font-bold">{pageContent.title || selectedSlug}</h2>
					{#if editing}
						<div class="flex gap-2">
							<button onclick={cancelEdit} class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">Cancel</button>
							<button onclick={savePage} class="px-3 py-1 bg-blue-600 hover:bg-blue-700 rounded text-sm">Save</button>
						</div>
					{:else}
						<button onclick={startEdit} class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">Edit</button>
					{/if}
				</div>
				{#if editing}
					<textarea bind:value={editContent} class="w-full h-[calc(100%-60px)] p-4 bg-gray-900 border border-gray-600 rounded font-mono text-sm resize-none" placeholder="Write markdown here..."></textarea>
				{:else}
					<div class="prose prose-invert max-w-none">
						{@html renderedHtml}
					</div>
				{/if}
			{:else}
				<p class="text-gray-500">Select a page from the sidebar or create a new one.</p>
			{/if}
		</div>
	</div>
{/if}
