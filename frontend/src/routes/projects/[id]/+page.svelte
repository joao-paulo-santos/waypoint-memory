<script>
	import api from '$lib/api';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';

	let id = $derived($page.params.id);
	let project = $state(null);
	let board = $state(null);
	let labels = $state([]);
	let loading = $state(true);
	let error = $state('');

	let showTaskModal = $state(false);
	let editingTask = $state(null);
	let taskTitle = $state('');
	let taskDesc = $state('');
	let taskPriority = $state(0);
	let taskDueDate = $state('');
	let taskLabels = $state('');

	let showSprintDialog = $state(false);
	let sprintName = $state('');
	let sprintSummary = $state('');

	let newBucketTitle = $state('');
	let showNewBucket = $state(false);

	let editingBucketId = $state(null);
	let editingBucketTitle = $state('');

	async function load() {
		loading = true;
		error = '';
		try {
			const [pData, bData, lData] = await Promise.all([
				api.get(`/api/v1/projects/${id}`),
				api.get(`/api/v1/projects/${id}/board`),
				api.get(`/api/v1/projects/${id}/labels`)
			]);
			project = pData;
			board = bData;
			labels = (lData.labels || []);
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

	function isOverdue(task) {
		return task.due_date && !task.done && task.due_date < new Date().toISOString().slice(0, 10);
	}

	async function openNewTask(bucketId) {
		editingTask = null;
		taskTitle = '';
		taskDesc = '';
		taskPriority = 0;
		taskDueDate = '';
		taskLabels = '';
		showTaskModal = { bucketId };
	}

	async function openEditTask(task) {
		editingTask = task;
		taskTitle = task.title;
		taskDesc = task.description;
		taskPriority = task.priority;
		taskDueDate = task.due_date || '';
		taskLabels = (task.labels || []).map((l) => l.id).join(',');
		showTaskModal = { bucketId: task.bucket_id };
	}

	async function saveTask() {
		const { bucketId } = showTaskModal;
		try {
			const body = {
				bucket_id: bucketId,
				title: taskTitle,
				description: taskDesc,
				priority: taskPriority,
				labels: taskLabels
			};
			if (taskDueDate) body.due_date = taskDueDate;

			if (editingTask) {
				await api.put(`/api/v1/projects/${id}/board/tasks/${editingTask.id}`, body);
			} else {
				await api.post(`/api/v1/projects/${id}/board/tasks`, body);
			}
			showTaskModal = false;
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function deleteTask(taskId) {
		if (!confirm('Delete this task?')) return;
		try {
			await api.del(`/api/v1/projects/${id}/board/tasks/${taskId}`);
			showTaskModal = false;
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function toggleDone(task) {
		try {
			await api.put(`/api/v1/projects/${id}/board/tasks/${task.id}`, { done: !task.done });
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function moveTask(taskId, bucketId) {
		try {
			await api.post(`/api/v1/projects/${id}/board/tasks/${taskId}/move`, { bucket_id: bucketId });
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function startEditBucket(bucket) {
		editingBucketId = bucket.id;
		editingBucketTitle = bucket.title;
	}

	async function saveBucketTitle(bucketId) {
		try {
			await api.put(`/api/v1/projects/${id}/board/buckets/${bucketId}`, { title: editingBucketTitle });
			editingBucketId = null;
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function addBucket() {
		if (!newBucketTitle.trim()) return;
		try {
			await api.post(`/api/v1/projects/${id}/board/buckets`, { title: newBucketTitle });
			newBucketTitle = '';
			showNewBucket = false;
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function deleteBucket(bucketId) {
		if (!confirm('Delete this bucket?')) return;
		try {
			await api.del(`/api/v1/projects/${id}/board/buckets/${bucketId}`);
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function endSprint() {
		try {
			await api.post(`/api/v1/projects/${id}/sprints/end`, {
				sprint_name: sprintName,
				summary: sprintSummary
			});
			showSprintDialog = false;
			sprintName = '';
			sprintSummary = '';
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	function doneBucket() {
		return board?.buckets?.find((b) => b.bucket.is_done_bucket);
	}
</script>

{#if loading}
	<p class="text-gray-400">Loading board...</p>
{:else if error}
	<p class="text-red-400">{error}</p>
{:else if project && board}
	<div>
		<div class="flex items-center justify-between mb-4">
			<div class="flex items-center gap-3">
				<h2 class="text-2xl font-bold">{project.name}</h2>
				{#if project.description}
					<span class="text-gray-400 text-sm">— {project.description}</span>
				{/if}
			</div>
			<div class="flex gap-2">
				<a href="/projects/{id}/labels" class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">Labels</a>
				<a href="/projects/{id}/sprints" class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">Sprints</a>
				<a href="/projects/{id}/wiki" class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">Wiki</a>
			</div>
		</div>

		<div class="flex gap-4 overflow-x-auto pb-4">
			{#each board.buckets as bw (bw.bucket.id)}
				<div class="min-w-[280px] w-[280px] bg-gray-800 rounded-lg flex flex-col max-h-[calc(100vh-160px)]">
					<!-- Bucket Header -->
					<div class="p-3 border-b border-gray-700 flex items-center justify-between">
						{#if editingBucketId === bw.bucket.id}
							<input
								bind:value={editingBucketTitle}
								class="flex-1 p-1 bg-gray-700 border border-gray-600 rounded text-sm"
								onkeydown={(e) => e.key === 'Enter' && saveBucketTitle(bw.bucket.id)}
								onblur={() => saveBucketTitle(bw.bucket.id)}
							/>
						{:else}
							<div class="flex items-center gap-2 cursor-pointer" onclick={() => startEditBucket(bw.bucket)}>
								{#if bw.bucket.is_done_bucket}
									<span class="text-green-400">✓</span>
								{/if}
								<span class="font-semibold text-sm">{bw.bucket.title}</span>
								<span class="text-gray-500 text-xs">({bw.tasks.length})</span>
							</div>
						{/if}
						<div class="flex gap-1">
							{#if !bw.bucket.is_done_bucket}
								<button onclick={() => deleteBucket(bw.bucket.id)} class="text-gray-500 hover:text-red-400 text-xs" title="Delete">✕</button>
							{/if}
						</div>
					</div>

					<!-- Task List -->
					<div class="flex-1 overflow-y-auto p-2 space-y-2">
						{#each bw.tasks as task (task.id)}
							<div class="p-3 bg-gray-750 bg-gray-900/50 rounded border border-gray-700 hover:border-gray-600 cursor-pointer"
								 onclick={() => openEditTask(task)}>
								<div class="flex items-start gap-2">
									<button
										class="mt-0.5 text-gray-500 hover:text-green-400"
										onclick={(e) => { e.stopPropagation(); toggleDone(task); }}
										title={task.done ? 'Mark undone' : 'Mark done'}>
										{task.done ? '☑' : '☐'}
									</button>
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2">
											<span class="text-sm font-medium {task.done ? 'line-through text-gray-500' : ''}">{task.title}</span>
											{#if task.priority > 0}
												<span class="w-2 h-2 rounded-full {priorityDot(task.priority)}" title="Priority {task.priority}"></span>
											{/if}
										</div>
										{#if task.due_date}
											<span class="text-xs {isOverdue(task) ? 'text-red-400' : 'text-gray-500'}">{task.due_date}</span>
										{/if}
										{#if task.labels && task.labels.length > 0}
											<div class="flex gap-1 mt-1 flex-wrap">
												{#each task.labels as label}
														<span class="text-xs px-1.5 py-0.5 rounded" style="background: {label.hex_color || '#555'}">{label.title}</span>
												{/each}
											</div>
										{/if}
									</div>
								</div>
							</div>
						{/each}
					</div>

					<!-- Add Task -->
					<div class="p-2 border-t border-gray-700">
						<button onclick={() => openNewTask(bw.bucket.id)} class="w-full text-left text-sm text-gray-500 hover:text-gray-300 px-2 py-1">+ Add task</button>
					</div>
				</div>
			{/each}

			<!-- New Bucket -->
			<div class="min-w-[280px] w-[280px]">
				{#if showNewBucket}
					<div class="bg-gray-800 rounded-lg p-3">
						<input bind:value={newBucketTitle} class="w-full p-2 bg-gray-700 border border-gray-600 rounded text-sm mb-2" placeholder="Bucket title" />
						<div class="flex gap-2">
							<button onclick={addBucket} class="px-3 py-1 bg-blue-600 rounded text-sm">Add</button>
							<button onclick={() => (showNewBucket = false)} class="px-3 py-1 bg-gray-700 rounded text-sm">Cancel</button>
						</div>
					</div>
				{:else}
					<button onclick={() => (showNewBucket = true)} class="p-3 bg-gray-800/50 rounded-lg border-2 border-dashed border-gray-700 w-full text-gray-500 hover:text-gray-300 text-sm">+ Add bucket</button>
				{/if}
			</div>
		</div>

		<!-- End Sprint -->
		{#if doneBucket()}
			{@const db = doneBucket()}
			<div class="mt-4">
				<button
					onclick={() => (showSprintDialog = true)}
					disabled={db.tasks.length === 0}
					class="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-700 disabled:text-gray-500 rounded text-sm">
					End Sprint ({db.tasks.length} task{db.tasks.length !== 1 ? 's' : ''})
				</button>
			</div>
		{/if}
	</div>
{/if}

<!-- Task Modal -->
{#if showTaskModal}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => (showTaskModal = false)}>
		<div class="bg-gray-800 rounded-lg p-6 w-[500px] max-h-[80vh] overflow-y-auto" onclick={(e) => e.stopPropagation()}>
			<h3 class="text-lg font-bold mb-4">{editingTask ? 'Edit Task' : 'New Task'}</h3>
			<div class="space-y-3">
				<div><label class="block text-sm text-gray-400 mb-1">Title</label><input bind:value={taskTitle} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" /></div>
				<div><label class="block text-sm text-gray-400 mb-1">Description</label><textarea bind:value={taskDesc} rows="3" class="w-full p-2 bg-gray-700 border border-gray-600 rounded"></textarea></div>
				<div class="flex gap-3">
					<div class="flex-1"><label class="block text-sm text-gray-400 mb-1">Priority</label>
						<select bind:value={taskPriority} class="w-full p-2 bg-gray-700 border border-gray-600 rounded">
							<option value={0}>None</option><option value={1}>Urgent</option><option value={2}>High</option><option value={3}>Medium</option><option value={4}>Low</option>
						</select>
					</div>
					<div class="flex-1"><label class="block text-sm text-gray-400 mb-1">Due Date</label><input type="date" bind:value={taskDueDate} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" /></div>
				</div>
				{#if labels.length > 0}
					<div><label class="block text-sm text-gray-400 mb-1">Labels</label>
						<div class="flex gap-2 flex-wrap">
							{#each labels as label}
								<label class="flex items-center gap-1 text-sm cursor-pointer">
									<input type="checkbox" checked={taskLabels.split(',').includes(String(label.id))}
										onchange={(e) => {
											let ids = taskLabels ? taskLabels.split(',').filter(Boolean) : [];
											if (e.target.checked) ids.push(String(label.id)); else ids = ids.filter(x => x !== String(label.id));
											taskLabels = ids.join(',');
										}} />
									<span class="px-1.5 py-0.5 rounded text-xs" style="background: {label.hex_color || '#555'}">{label.title}</span>
								</label>
											{/each}
						</div>
					</div>
				{/if}
			</div>
			<div class="flex justify-between mt-6">
				<div>{#if editingTask}<button onclick={() => deleteTask(editingTask.id)} class="px-3 py-1 bg-red-600 hover:bg-red-700 rounded text-sm">Delete</button>{/if}</div>
				<div class="flex gap-2">
					<button onclick={() => (showTaskModal = false)} class="px-3 py-1 bg-gray-700 rounded text-sm">Cancel</button>
					<button onclick={saveTask} class="px-3 py-1 bg-blue-600 hover:bg-blue-700 rounded text-sm">{editingTask ? 'Save' : 'Create'}</button>
				</div>
			</div>
		</div>
	</div>
{/if}

<!-- Sprint Dialog -->
{#if showSprintDialog}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => (showSprintDialog = false)}>
		<div class="bg-gray-800 rounded-lg p-6 w-[400px]" onclick={(e) => e.stopPropagation()}>
			<h3 class="text-lg font-bold mb-4">End Sprint</h3>
			<div class="space-y-3">
				<div><label class="block text-sm text-gray-400 mb-1">Sprint Name (optional)</label><input bind:value={sprintName} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" /></div>
				<div><label class="block text-sm text-gray-400 mb-1">Summary (optional)</label><textarea bind:value={sprintSummary} rows="3" class="w-full p-2 bg-gray-700 border border-gray-600 rounded"></textarea></div>
			</div>
			<div class="flex justify-end gap-2 mt-6">
				<button onclick={() => (showSprintDialog = false)} class="px-3 py-1 bg-gray-700 rounded text-sm">Cancel</button>
				<button onclick={endSprint} class="px-3 py-1 bg-green-600 hover:bg-green-700 rounded text-sm">Archive</button>
			</div>
		</div>
	</div>
{/if}
