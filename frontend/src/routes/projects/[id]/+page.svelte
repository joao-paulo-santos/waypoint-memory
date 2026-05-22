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

	let draggedTaskId = $state(null);
	let draggedBucketIdx = $state(null);
	let dropTargetBucketId = $state(null);
	let dropTargetTaskId = $state(null);
	let dropTargetBucketIdx = $state(null);

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

	function onTaskDragStart(e, taskId) {
		draggedTaskId = taskId;
		draggedBucketIdx = null;
		e.dataTransfer.effectAllowed = 'move';
		e.dataTransfer.setData('text/plain', String(taskId));
		e.currentTarget.style.opacity = '0.4';
	}

	function onTaskDragEnd(e) {
		draggedTaskId = null;
		dropTargetBucketId = null;
		dropTargetTaskId = null;
		e.currentTarget.style.opacity = '1';
	}

	function onBucketDragOver(e, bucketId) {
		if (draggedTaskId === null) return;
		e.preventDefault();
		e.dataTransfer.dropEffect = 'move';
		dropTargetBucketId = bucketId;
	}

	function onBucketDragLeave(e, bucketId) {
		if (!e.currentTarget.contains(e.relatedTarget)) {
			if (dropTargetBucketId === bucketId) dropTargetBucketId = null;
		}
	}

	async function onBucketDrop(e, bucketId) {
		e.preventDefault();
		dropTargetBucketId = null;
		if (draggedTaskId === null) return;
		try {
			await api.post(`/api/v1/projects/${id}/board/tasks/${draggedTaskId}/move`, { bucket_id: bucketId });
			await load();
		} catch (err) {
			alert(err.message);
		} finally {
			draggedTaskId = null;
		}
	}

	function onBucketHeaderDragStart(e, idx) {
		draggedBucketIdx = idx;
		draggedTaskId = null;
		e.dataTransfer.effectAllowed = 'move';
		e.dataTransfer.setData('text/plain', String(idx));
		const col = e.currentTarget.closest('[data-bucket-col]');
		if (col) col.style.opacity = '0.4';
	}

	function onBucketHeaderDragEnd(e) {
		draggedBucketIdx = null;
		dropTargetBucketIdx = null;
		const col = e.currentTarget.closest('[data-bucket-col]');
		if (col) col.style.opacity = '1';
	}

	function onBucketListDragOver(e, targetIdx) {
		if (draggedBucketIdx === null) return;
		e.preventDefault();
		e.dataTransfer.dropEffect = 'move';
		dropTargetBucketIdx = targetIdx;
	}

	function onBucketListDragLeave(e, targetIdx) {
		if (!e.currentTarget.contains(e.relatedTarget)) {
			if (dropTargetBucketIdx === targetIdx) dropTargetBucketIdx = null;
		}
	}

	async function onBucketListDrop(e, targetIdx) {
		e.preventDefault();
		dropTargetBucketIdx = null;
		if (draggedBucketIdx === null || draggedBucketIdx === targetIdx) return;
		const buckets = board.buckets || [];
		const reordered = [...buckets];
		const [moved] = reordered.splice(draggedBucketIdx, 1);
		reordered.splice(targetIdx, 0, moved);
		const bucketIds = reordered.map(bw => bw.bucket.id);
		try {
			await api.patch(`/api/v1/projects/${id}/board/buckets/reorder`, { bucket_ids: bucketIds });
			await load();
		} catch (err) {
			await load();
		} finally {
			draggedBucketIdx = null;
		}
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
				{#if project.color}
					<span class="w-4 h-4 rounded-full shrink-0" style="background: {project.color}"></span>
				{/if}
				<h2 class="text-2xl font-bold">{project.name}</h2>
				{#if project.description}
					<span class="text-gray-400 text-sm">&mdash; {project.description}</span>
				{/if}
			</div>
			<div class="flex gap-2">
				<a href="/projects/{id}/labels" class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">Labels</a>
				<a href="/projects/{id}/sprints" class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">Sprints</a>
				<a href="/projects/{id}/wiki" class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">Wiki</a>
			</div>
		</div>

		<div class="flex gap-4 overflow-x-auto pb-4">
			{#each board.buckets || [] as bw, bucketIdx (bw.bucket.id)}
				<div class="min-w-[280px] w-[280px] bg-gray-800 rounded-lg flex flex-col max-h-[calc(100vh-160px)] transition-all {draggedBucketIdx !== null && draggedBucketIdx !== bucketIdx && dropTargetBucketIdx === bucketIdx ? 'ring-2 ring-blue-500' : ''} {draggedBucketIdx === bucketIdx ? 'opacity-40' : ''}"
					data-bucket-col
					ondragover={(e) => { onBucketListDragOver(e, bucketIdx); onBucketDragOver(e, bw.bucket.id); }}
					ondragleave={(e) => { onBucketListDragLeave(e, bucketIdx); onBucketDragLeave(e, bw.bucket.id); }}
					ondrop={(e) => { onBucketListDrop(e, bucketIdx); onBucketDrop(e, bw.bucket.id); }}>
					<div class="p-3 border-b border-gray-700 flex items-center justify-between">
						<div class="flex items-center gap-2 flex-1 min-w-0">
							<div class="cursor-grab active:cursor-grabbing text-gray-600 hover:text-gray-400 select-none pr-1"
								role="button" tabindex="0" title="Drag to reorder"
								draggable="true"
								ondragstart={(e) => onBucketHeaderDragStart(e, bucketIdx)}
								ondragend={onBucketHeaderDragEnd}
								onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); } }}>
								<svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor"><circle cx="9" cy="6" r="1.5"/><circle cx="15" cy="6" r="1.5"/><circle cx="9" cy="12" r="1.5"/><circle cx="15" cy="12" r="1.5"/><circle cx="9" cy="18" r="1.5"/><circle cx="15" cy="18" r="1.5"/></svg>
							</div>
							{#if editingBucketId === bw.bucket.id}
								<input
									id="bucket-title-{bw.bucket.id}"
									bind:value={editingBucketTitle}
									class="flex-1 p-1 bg-gray-700 border border-gray-600 rounded text-sm"
									onkeydown={(e) => e.key === 'Enter' && saveBucketTitle(bw.bucket.id)}
									onblur={() => saveBucketTitle(bw.bucket.id)}
								/>
							{:else}
								<div class="flex items-center gap-2 cursor-pointer min-w-0" role="button" tabindex="0"
									onclick={() => startEditBucket(bw.bucket)}
									onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); startEditBucket(bw.bucket); } }}>
									{#if bw.bucket.is_done_bucket}
										<span class="text-green-400">✓</span>
									{/if}
									<span class="font-semibold text-sm truncate">{bw.bucket.title}</span>
									<span class="text-gray-500 text-xs shrink-0">({(bw.tasks || []).length})</span>
								</div>
							{/if}
						</div>
						<div class="flex gap-1 shrink-0">
							{#if !bw.bucket.is_done_bucket}
								<button onclick={() => deleteBucket(bw.bucket.id)} class="text-gray-500 hover:text-red-400 text-xs" title="Delete">✕</button>
							{/if}
						</div>
					</div>

					<div class="flex-1 overflow-y-auto p-2 space-y-2 {draggedTaskId !== null && dropTargetBucketId === bw.bucket.id ? 'bg-blue-900/20 ring-1 ring-blue-500/50 rounded mx-1' : ''}">
						{#each bw.tasks || [] as task (task.id)}
							<div class="p-3 bg-gray-900/50 rounded border border-gray-700 hover:border-gray-600 cursor-grab active:cursor-grabbing {draggedTaskId === task.id ? 'opacity-40' : ''} {dropTargetTaskId === task.id ? 'border-t-2 border-t-blue-500' : ''}"
								 role="button" tabindex="0"
								 draggable="true"
								 ondragstart={(e) => onTaskDragStart(e, task.id)}
								 ondragend={onTaskDragEnd}
								 onclick={() => openEditTask(task)}
								 onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openEditTask(task); } }}>
								<div class="min-w-0">
									<div class="flex items-center gap-2">
										<span class="text-sm font-medium {task.done ? 'line-through text-gray-500' : ''}">{task.title}</span>
										{#if task.priority > 0}
											<span class="w-2 h-2 rounded-full shrink-0 {priorityDot(task.priority)}" title="Priority {task.priority}"></span>
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
						{/each}
					</div>

					<div class="p-2 border-t border-gray-700">
						<button onclick={() => openNewTask(bw.bucket.id)} class="w-full text-left text-sm text-gray-500 hover:text-gray-300 px-2 py-1">+ Add task</button>
					</div>
				</div>
			{/each}

			<div class="min-w-[280px] w-[280px]">
				{#if showNewBucket}
					<div class="bg-gray-800 rounded-lg p-3">
						<label for="new-bucket" class="block text-sm text-gray-400 mb-1">Bucket Title</label>
						<input id="new-bucket" bind:value={newBucketTitle} class="w-full p-2 bg-gray-700 border border-gray-600 rounded text-sm mb-2" placeholder="Bucket title" />
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

		{#if doneBucket()}
			{@const db = doneBucket()}
			<div class="mt-4">
				<button
					onclick={() => (showSprintDialog = true)}
					disabled={(db.tasks || []).length === 0}
					class="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-700 disabled:text-gray-500 rounded text-sm">
					End Sprint ({(db.tasks || []).length} task{(db.tasks || []).length !== 1 ? 's' : ''})
				</button>
			</div>
		{/if}
	</div>
{/if}

{#if showTaskModal}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => (showTaskModal = false)} onkeydown={(e) => { if (e.key === 'Escape') showTaskModal = false; }}>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="bg-gray-800 rounded-lg p-6 w-[500px] max-h-[80vh] overflow-y-auto" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
			<h3 class="text-lg font-bold mb-4">{editingTask ? 'Edit Task' : 'New Task'}</h3>
			<div class="space-y-3">
				<div><label for="task-title" class="block text-sm text-gray-400 mb-1">Title</label><input id="task-title" bind:value={taskTitle} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" /></div>
				<div><label for="task-desc" class="block text-sm text-gray-400 mb-1">Description</label><textarea id="task-desc" bind:value={taskDesc} rows="3" class="w-full p-2 bg-gray-700 border border-gray-600 rounded"></textarea></div>
				<div class="flex gap-3">
					<div class="flex-1"><label for="task-priority" class="block text-sm text-gray-400 mb-1">Priority</label>
						<select id="task-priority" bind:value={taskPriority} class="w-full p-2 bg-gray-700 border border-gray-600 rounded">
							<option value={0}>None</option><option value={1}>Urgent</option><option value={2}>High</option><option value={3}>Medium</option><option value={4}>Low</option>
						</select>
					</div>
					<div class="flex-1"><label for="task-due" class="block text-sm text-gray-400 mb-1">Due Date</label><input id="task-due" type="date" bind:value={taskDueDate} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" /></div>
				</div>
				{#if labels.length > 0}
					<div><span class="block text-sm text-gray-400 mb-1">Labels</span>
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
				{#if editingTask}
					<div class="flex items-center gap-2 pt-1 border-t border-gray-700">
						<input id="task-done" type="checkbox" checked={editingTask.done}
							onchange={() => toggleDone(editingTask)} />
						<label for="task-done" class="text-sm text-gray-400">{editingTask.done ? 'Done' : 'Mark as done'}</label>
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

{#if showSprintDialog}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => (showSprintDialog = false)} onkeydown={(e) => { if (e.key === 'Escape') showSprintDialog = false; }}>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="bg-gray-800 rounded-lg p-6 w-[400px]" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
			<h3 class="text-lg font-bold mb-4">End Sprint</h3>
			<div class="space-y-3">
				<div><label for="sprint-name" class="block text-sm text-gray-400 mb-1">Sprint Name (optional)</label><input id="sprint-name" bind:value={sprintName} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" /></div>
				<div><label for="sprint-summary" class="block text-sm text-gray-400 mb-1">Summary (optional)</label><textarea id="sprint-summary" bind:value={sprintSummary} rows="3" class="w-full p-2 bg-gray-700 border border-gray-600 rounded"></textarea></div>
			</div>
			<div class="flex justify-end gap-2 mt-6">
				<button onclick={() => (showSprintDialog = false)} class="px-3 py-1 bg-gray-700 rounded text-sm">Cancel</button>
				<button onclick={endSprint} class="px-3 py-1 bg-green-600 hover:bg-green-700 rounded text-sm">Archive</button>
			</div>
		</div>
	</div>
{/if}
