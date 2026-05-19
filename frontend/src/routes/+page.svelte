<script>
	import api from '$lib/api';
	import { onMount } from 'svelte';

	let projects = $state([]);
	let birthdays = $state([]);
	let activity = $state([]);
	let overdueTasks = $state([]);
	let loading = $state(true);

	async function load() {
		loading = true;
		try {
			const [pData, bData, aData] = await Promise.all([
				api.get('/api/v1/projects'),
				api.get('/api/v1/birthdays/upcoming?days=7'),
				api.get('/api/v1/activity?limit=10')
			]);

			projects = pData.projects || [];
			birthdays = bData.birthdays || [];
			activity = aData.activity || [];

			const overdueList = [];
			for (const p of projects) {
				try {
					const board = await api.get(`/api/v1/projects/${p.id}/board`);
					for (const bw of board.buckets || []) {
						for (const task of bw.tasks || []) {
							if (task.due_date && !task.done && task.due_date < new Date().toISOString().slice(0, 10)) {
								overdueList.push({ ...task, project_name: p.name, project_id: p.id });
							}
						}
					}
				} catch (e) {
					console.warn(`Failed to load board for project ${p.name}:`, e);
				}
			}
			overdueTasks = overdueList;
		} catch {}
		finally {
			loading = false;
		}
	}

	onMount(load);

	function formatDaysUntil(d) {
		if (d === 0) return 'Today!';
		if (d === 1) return 'Tomorrow';
		return `In ${d} days`;
	}

	function parseDetails(details) {
		try { return JSON.parse(details); } catch { return {}; }
	}

	function actionLabel(action) {
		const labels = {
			task_created: 'created task',
			task_updated: 'updated task',
			task_deleted: 'deleted task',
			task_moved: 'moved task',
			bucket_created: 'created bucket',
			bucket_updated: 'updated bucket',
			bucket_deleted: 'deleted bucket',
			label_created: 'created label',
			label_deleted: 'deleted label',
			comment_created: 'added comment',
			sprint_archived: 'archived sprint'
		};
		return labels[action] || action;
	}
</script>

<h2 class="text-2xl font-bold mb-6">Dashboard</h2>

{#if loading}
	<p class="text-gray-400">Loading overview...</p>
{:else}
	<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
		<div>
			<h3 class="text-lg font-semibold mb-3">Upcoming Birthdays</h3>
			{#if birthdays.length === 0}
				<p class="text-gray-500 text-sm">No birthdays in the next 7 days.</p>
			{:else}
				<div class="space-y-2">
					{#each birthdays as b (b.id)}
						<div class="bg-gray-800 rounded-lg p-3 flex items-center justify-between">
							<div class="flex items-center gap-2">
								<span class="text-xl">&#127874;</span>
								<span class="font-medium text-sm">{b.name}</span>
								{#if b.age_turning != null}
									<span class="text-xs text-purple-400">turning {b.age_turning}</span>
								{/if}
							</div>
							<span class="text-sm {b.days_until === 0 ? 'text-green-400 font-bold' : 'text-yellow-400'}">{formatDaysUntil(b.days_until)}</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<div>
			<h3 class="text-lg font-semibold mb-3">Overdue Tasks</h3>
			{#if overdueTasks.length === 0}
				<p class="text-gray-500 text-sm">No overdue tasks. Great job!</p>
			{:else}
				<div class="space-y-2">
					{#each overdueTasks.slice(0, 10) as task (task.id)}
						<a href="/projects/{task.project_id}" class="block bg-gray-800 rounded-lg p-3 hover:bg-gray-750">
							<div class="flex items-center justify-between">
								<span class="text-sm font-medium">{task.title}</span>
								<span class="text-xs text-red-400">{task.due_date}</span>
							</div>
							<span class="text-xs text-gray-500">{task.project_name}</span>
						</a>
					{/each}
				</div>
			{/if}
		</div>

		<div class="lg:col-span-2">
			<h3 class="text-lg font-semibold mb-3">Recent Activity</h3>
			{#if activity.length === 0}
				<p class="text-gray-500 text-sm">No recent activity.</p>
			{:else}
				<div class="bg-gray-800 rounded-lg divide-y divide-gray-700">
					{#each activity as entry (entry.id)}
						<div class="p-3 flex items-center gap-3">
							<span class="text-xs text-gray-500 w-20 shrink-0">{(entry.created_at || '').slice(0, 10)}</span>
							<span class="text-sm">
								{#if entry.project_name}
									<a href="/projects/{entry.project_id}" class="text-blue-400 hover:underline">{entry.project_name}</a>
									&middot;
								{/if}
								<span class="text-gray-300">{actionLabel(entry.action)}</span>
								{#if parseDetails(entry.details).title}
									<span class="text-gray-400">: {parseDetails(entry.details).title}</span>
								{/if}
							</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<div class="lg:col-span-2">
			<h3 class="text-lg font-semibold mb-3">Projects</h3>
			{#if projects.length === 0}
				<p class="text-gray-500 text-sm">No projects yet. <a href="/projects/new" class="text-blue-400 hover:underline">Create one</a>.</p>
			{:else}
				<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
					{#each projects as p (p.id)}
						<a href="/projects/{p.id}" class="block bg-gray-800 rounded-lg p-4 hover:bg-gray-750 border border-gray-700 hover:border-gray-600 transition-colors">
							<h4 class="font-semibold">{p.name}</h4>
							{#if p.description}
								<p class="text-sm text-gray-400 mt-1 line-clamp-2">{p.description}</p>
							{/if}
						</a>
					{/each}
				</div>
			{/if}
		</div>
	</div>
{/if}
