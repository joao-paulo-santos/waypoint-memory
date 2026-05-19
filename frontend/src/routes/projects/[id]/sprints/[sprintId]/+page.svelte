<script>
	import api from '$lib/api';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';

	let id = $derived($page.params.id);
	let sprintId = $derived($page.params.sprintId);
	let detail = $state(null);
	let loading = $state(true);
	let error = $state('');

	let expandedComments = $state({});

	async function load() {
		loading = true;
		error = '';
		try {
			detail = await api.get(`/api/v1/projects/${id}/sprints/${sprintId}`);
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(load);

	function priorityDot(p) {
		const colors = { 1: 'bg-red-500', 2: 'bg-orange-500', 3: 'bg-yellow-500', 4: 'bg-blue-500' };
		return colors[p] || '';
	}

	function formatDate(d) {
		if (!d) return '';
		return d.slice(0, 10);
	}

	function parseLabels(json) {
		try { return JSON.parse(json); } catch { return []; }
	}

	function parseComments(json) {
		try { return JSON.parse(json); } catch { return []; }
	}

	function toggleComments(taskId) {
		expandedComments[taskId] = !expandedComments[taskId];
		expandedComments = expandedComments;
	}
</script>

<div class="mb-4 flex items-center gap-3">
	<a href="/projects/{id}/sprints" class="text-gray-400 hover:text-white text-sm">&larr; Sprints</a>
	<h2 class="text-2xl font-bold">Sprint Detail</h2>
</div>

{#if loading}
	<p class="text-gray-400">Loading sprint...</p>
{:else if error}
	<p class="text-red-400">{error}</p>
{:else if detail}
	<div class="mb-6 bg-gray-800 rounded-lg p-4">
		<div class="flex items-center justify-between mb-2">
			<h3 class="text-xl font-bold">{detail.sprint.name}</h3>
			<span class="text-sm text-gray-400">{formatDate(detail.sprint.ended_at)}</span>
		</div>
		{#if detail.sprint.summary}
			<p class="text-gray-300 mb-2">{detail.sprint.summary}</p>
		{/if}
		<span class="text-xs px-2 py-0.5 bg-gray-700 rounded">{detail.sprint.task_count} task{#if detail.sprint.task_count !== 1}s{/if} archived</span>
	</div>

	{#if detail.tasks && detail.tasks.length > 0}
		<div class="space-y-3">
			{#each detail.tasks as task (task.id)}
				{@const taskLabels = parseLabels(task.labels)}
				{@const taskComments = parseComments(task.comments)}
				<div class="bg-gray-800 rounded-lg p-4 border border-gray-700">
					<div class="flex items-start gap-3">
						<span class="text-green-400 mt-0.5">&#10003;</span>
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-2 mb-1">
								<span class="font-medium">{task.title}</span>
								{#if task.priority > 0}
									<span class="w-2 h-2 rounded-full {priorityDot(task.priority)}" title="Priority {task.priority}"></span>
								{/if}
								<span class="text-xs text-gray-500 bg-gray-700 px-1.5 py-0.5 rounded">{task.bucket_title}</span>
							</div>
							{#if task.description}
								<p class="text-sm text-gray-400 mb-2">{task.description}</p>
							{/if}
							<div class="flex items-center gap-3 text-xs text-gray-500">
								{#if task.due_date}
									<span>Due: {task.due_date}</span>
								{/if}
								{#if task.created_by}
									<span>By: {task.created_by}</span>
								{/if}
								<span>Created: {formatDate(task.original_created)}</span>
							</div>
							{#if taskLabels.length > 0}
								<div class="flex gap-1 mt-2 flex-wrap">
									{#each taskLabels as label}
										<span class="text-xs px-1.5 py-0.5 rounded" style="background: {label.hex_color || '#555'}">{label.title}</span>
									{/each}
								</div>
							{/if}
							{#if taskComments.length > 0}
								<div class="mt-2">
									<button
										onclick={() => toggleComments(task.id)}
										class="text-xs text-gray-400 hover:text-gray-300">
										{expandedComments[task.id] ? '&#9660;' : '&#9654;'} {taskComments.length} comment{#if taskComments.length !== 1}s{/if}
									</button>
									{#if expandedComments[task.id]}
										<div class="mt-2 space-y-2 pl-3 border-l-2 border-gray-700">
											{#each taskComments as comment}
												<div class="text-sm">
													<span class="text-gray-400 font-medium">{comment.author || 'Anonymous'}</span>
													<span class="text-gray-600 text-xs ml-2">{formatDate(comment.created_at)}</span>
													<p class="text-gray-300">{comment.body}</p>
												</div>
											{/each}
										</div>
									{/if}
								</div>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else}
		<p class="text-gray-500">No archived tasks in this sprint.</p>
	{/if}
{/if}
