<script>
	import api from '$lib/api';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { renderMarkdown } from '$lib/sanitize';

	let id = $derived($page.params.id);
	let project = $state(null);
	let wikiTree = $state([]);
	let docsTree = $state([]);
	let loading = $state(true);
	let error = $state('');

	let selectedPath = $state('');
	let selectedSource = $state('');
	let pageContent = $state(null);
	let pageLoading = $state(false);
	let pageError = $state('');

	let renderedHtml = $derived(pageContent ? renderMarkdown(pageContent.content) : '');

	async function load() {
		loading = true;
		error = '';
		try {
			const pData = await api.get(`/api/v1/projects/${id}`);
			project = pData;

			const results = await Promise.allSettled([
				api.get(`/api/v1/projects/${id}/wiki`),
				api.get(`/api/v1/projects/${id}/wiki/docs`)
			]);

			if (results[0].status === 'fulfilled') {
				wikiTree = results[0].value.pages || [];
			}
			if (results[1].status === 'fulfilled') {
				docsTree = results[1].value.pages || [];
			}
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(load);

	async function selectPage(path, source) {
		selectedPath = path;
		selectedSource = source;
		pageLoading = true;
		pageError = '';
		try {
			const prefix = source === 'docs' ? 'docs/' : '';
			pageContent = await api.get(`/api/v1/projects/${id}/wiki/${prefix}${path}`);
		} catch (e) {
			pageError = e.message;
			pageContent = null;
		} finally {
			pageLoading = false;
		}
	}
</script>

<svelte:head>
	<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css">
</svelte:head>

<div class="mb-4 flex items-center gap-3">
	<a href="/projects/{id}" class="text-gray-400 hover:text-white text-sm">&larr; Board</a>
	<h2 class="text-2xl font-bold">Wiki & Docs</h2>
	{#if project}
		<span class="text-gray-400 text-sm">— {project.name}</span>
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
		<div class="w-64 shrink-0 bg-gray-800 rounded-lg p-3 overflow-y-auto">
			{#if wikiTree.length > 0}
				<h3 class="text-sm font-semibold text-gray-400 uppercase mb-2">Wiki</h3>
				<div class="space-y-0.5 mb-4">
					{#each wikiTree as node (node.path)}
						{@render fileNode(node, 'wiki')}
					{/each}
				</div>
				{#if docsTree.length > 0}
					<hr class="border-gray-700 my-3" />
				{/if}
			{/if}
			{#if docsTree.length > 0}
				<h3 class="text-sm font-semibold text-gray-400 uppercase mb-2">Docs</h3>
				<div class="space-y-0.5">
					{#each docsTree as node (node.path)}
						{@render fileNode(node, 'docs')}
					{/each}
				</div>
			{/if}
			{#if wikiTree.length === 0 && docsTree.length === 0}
				<p class="text-sm text-gray-500">No wiki or docs found.</p>
				<p class="text-xs text-gray-600 mt-1">Add <code class="bg-gray-700 px-1 rounded">.waypoint/wiki/</code> or configure a docs path to get started.</p>
			{/if}
		</div>

		<div class="flex-1 bg-gray-800 rounded-lg p-6 overflow-y-auto">
			{#if pageLoading}
				<p class="text-gray-400">Loading page...</p>
			{:else if pageError}
				<p class="text-red-400">{pageError}</p>
			{:else if pageContent}
				<h2 class="text-xl font-bold mb-4">{pageContent.title}</h2>
				<div class="prose prose-invert max-w-none">
					{@html renderedHtml}
				</div>
			{:else}
				<p class="text-gray-500">Select a page from the sidebar to view it.</p>
			{/if}
		</div>
	</div>
{/if}

{#snippet fileNode(node, source, depth = 0)}
	{#if node.is_dir}
		<div>
			<div class="text-sm font-medium text-gray-400 py-1 px-2" style="padding-left: {depth * 16 + 8}px">
				&#128193; {node.name}
			</div>
			{#if node.children}
				{#each node.children as child (child.path)}
					{@render fileNode(child, source, depth + 1)}
				{/each}
			{/if}
		</div>
	{:else}
		<button
			onclick={() => selectPage(node.path, source)}
			class="w-full text-left text-sm py-1 px-2 rounded hover:bg-gray-700 {selectedPath === node.path && selectedSource === source ? 'bg-gray-700 text-white' : 'text-gray-300'}"
			style="padding-left: {depth * 16 + 8}px">
			&#128196; {node.name}
		</button>
	{/if}
{/snippet}
