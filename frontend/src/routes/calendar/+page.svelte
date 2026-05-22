<script>
	import api from '$lib/api';
	import { onMount } from 'svelte';

	let currentDate = $state(new Date());
	let events = $state([]);
	let loading = $state(true);
	let error = $state('');
	let selectedEvent = $state(null);

	let showTasks = $state(true);
	let showBirthdays = $state(true);
	let showEvents = $state(true);

	let showAddEvent = $state(false);
	let addEventDate = $state('');
	let eventForm = $state({ title: '', date: '', time: '', category: '', description: '', recurrence: '' });

	let year = $derived(currentDate.getFullYear());
	let month = $derived(currentDate.getMonth());
	let monthName = $derived(currentDate.toLocaleString('default', { month: 'long', year: 'numeric' }));

	let firstDay = $derived(new Date(year, month, 1).getDay());
	let daysInMonth = $derived(new Date(year, month + 1, 0).getDate());
	let today = $derived(new Date().toISOString().slice(0, 10));

	let from = $derived(`${year}-${String(month + 1).padStart(2, '0')}-01`);
	let to = $derived(`${year}-${String(month + 1).padStart(2, '0')}-${String(daysInMonth).padStart(2, '0')}`);

	let filteredEvents = $derived(events.filter(e => {
		if (e.type === 'task_due') return showTasks;
		if (e.type === 'birthday') return showBirthdays;
		if (e.type === 'event') return showEvents;
		return true;
	}));

	let eventsByDate = $derived.by(() => {
		const map = {};
		for (const e of filteredEvents) {
			if (!map[e.date]) map[e.date] = [];
			map[e.date].push(e);
		}
		return map;
	});

	function pad(n) { return String(n).padStart(2, '0'); }

	async function load() {
		loading = true;
		error = '';
		try {
			const data = await api.get(`/api/v1/calendar?from=${from}&to=${to}`);
			events = data.events || [];
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	$effect(() => { from; to; load(); });

	function prevMonth() {
		currentDate = new Date(year, month - 1, 1);
	}

	function nextMonth() {
		currentDate = new Date(year, month + 1, 1);
	}

	function goToday() {
		currentDate = new Date();
	}

	function eventColor(type, done) {
		switch (type) {
			case 'task_due': return done ? 'bg-green-600' : 'bg-blue-500';
			case 'birthday': return 'bg-purple-500';
			case 'event': return 'bg-orange-500';
			default: return 'bg-gray-500';
		}
	}

	function eventIcon(type, done) {
		switch (type) {
			case 'task_due': return done ? '✓' : '';
			case 'birthday': return '🎂';
			case 'event': return '📅';
			default: return '';
		}
	}

	function daysUntil(dateStr) {
		const diff = Math.ceil((new Date(dateStr) - new Date(today)) / 86400000);
		if (diff === 0) return 'Today';
		if (diff === 1) return 'Tomorrow';
		if (diff > 0) return `In ${diff} days`;
		return `${Math.abs(diff)} days ago`;
	}

	function openAddEvent(dateStr) {
		addEventDate = dateStr;
		eventForm = { title: '', date: dateStr, time: '', category: '', description: '', recurrence: '' };
		showAddEvent = true;
	}

	async function createEvent() {
		if (!eventForm.title.trim() || !eventForm.date) return;
		try {
			const body = { title: eventForm.title, date: eventForm.date };
			if (eventForm.time) body.time = eventForm.time;
			if (eventForm.category) body.category = eventForm.category;
			if (eventForm.description) body.description = eventForm.description;
			if (eventForm.recurrence) body.recurrence = eventForm.recurrence;
			await api.post('/api/v1/events', body);
			showAddEvent = false;
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function deleteEvent(id) {
		if (!confirm('Delete this event?')) return;
		try {
			await api.del(`/api/v1/events/${id}`);
			selectedEvent = null;
			await load();
		} catch (e) {
			alert(e.message);
		}
	}
</script>

<div class="mb-4">
	<h2 class="text-2xl font-bold mb-4">Calendar</h2>
</div>

<div class="flex items-center justify-between mb-4">
	<div class="flex items-center gap-2">
		<button onclick={prevMonth} class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">&larr; Prev</button>
		<span class="text-lg font-semibold min-w-[200px] text-center">{monthName}</span>
		<button onclick={nextMonth} class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">Next &rarr;</button>
	</div>
	<button onclick={goToday} class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">Today</button>
</div>

<div class="flex gap-3 mb-4 text-sm">
	<label class="flex items-center gap-1.5 cursor-pointer">
		<input type="checkbox" bind:checked={showTasks} class="accent-blue-500" />
		<span class="w-3 h-3 rounded bg-blue-500 inline-block"></span> Tasks
	</label>
	<label class="flex items-center gap-1.5 cursor-pointer">
		<input type="checkbox" bind:checked={showBirthdays} class="accent-purple-500" />
		<span class="w-3 h-3 rounded bg-purple-500 inline-block"></span> Birthdays
	</label>
	<label class="flex items-center gap-1.5 cursor-pointer">
		<input type="checkbox" bind:checked={showEvents} class="accent-orange-500" />
		<span class="w-3 h-3 rounded bg-orange-500 inline-block"></span> Events
	</label>
</div>

{#if loading}
	<p class="text-gray-400">Loading calendar...</p>
{:else if error}
	<p class="text-red-400">{error}</p>
{:else}
	<div class="grid grid-cols-7 gap-px bg-gray-700 rounded-lg overflow-hidden">
		{#each ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as day}
			<div class="bg-gray-800 p-2 text-center text-sm font-semibold text-gray-400">{day}</div>
		{/each}

		{#each Array(firstDay) as _, i}
			<div class="bg-gray-800/50 p-2 min-h-[100px]"></div>
		{/each}

		{#each Array(daysInMonth) as _, i}
			{@const day = i + 1}
			{@const dateStr = `${year}-${pad(month + 1)}-${pad(day)}`}
			{@const isToday = dateStr === today}
			{@const dayEvents = eventsByDate[dateStr] || []}
			<div class="bg-gray-800 p-2 min-h-[100px] {isToday ? 'ring-2 ring-blue-500 ring-inset' : ''}">
				<div class="flex items-center justify-between mb-1">
					<span class="text-sm {isToday ? 'text-blue-400 font-bold' : 'text-gray-400'}">{day}</span>
					<button onclick={() => openAddEvent(dateStr)} class="text-gray-600 hover:text-gray-300 text-xs" title="Add event">+</button>
				</div>
				<div class="space-y-1">
					{#each dayEvents.slice(0, 3) as event}
						<button
							onclick={() => (selectedEvent = event)}
							class="w-full text-left text-xs px-1.5 py-0.5 rounded {eventColor(event.type, event.done)} text-white truncate block"
							title={event.title}>
							<span class={event.done ? 'line-through opacity-80' : ''}>{eventIcon(event.type, event.done)} {event.title}</span>
						</button>
					{/each}
					{#if dayEvents.length > 3}
						<span class="text-xs text-gray-500">+{dayEvents.length - 3} more</span>
					{/if}
				</div>
			</div>
		{/each}
	</div>
{/if}

{#if selectedEvent}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => (selectedEvent = null)} onkeydown={(e) => { if (e.key === 'Escape') selectedEvent = null; }}>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="bg-gray-800 rounded-lg p-4 w-[350px]" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
			<div class="flex items-center gap-2 mb-3">
				<span class="w-3 h-3 rounded {eventColor(selectedEvent.type, selectedEvent.done)}"></span>
				<h3 class="font-bold {selectedEvent.done ? 'line-through text-gray-400' : ''}">{selectedEvent.title}</h3>
			</div>
			<div class="space-y-1 text-sm text-gray-300">
				<p><span class="text-gray-500">Type:</span> {selectedEvent.type === 'task_due' ? 'Task' : selectedEvent.type === 'birthday' ? 'Birthday' : 'Event'}</p>
				<p><span class="text-gray-500">Date:</span> {selectedEvent.date}</p>
				<p><span class="text-gray-500">When:</span> {daysUntil(selectedEvent.date)}</p>
				{#if selectedEvent.project}
					<p><span class="text-gray-500">Project:</span> {selectedEvent.project}</p>
				{/if}
				{#if selectedEvent.age_turning != null}
					<p><span class="text-gray-500">Age:</span> {selectedEvent.age_turning}</p>
				{/if}
				{#if selectedEvent.category}
					<p><span class="text-gray-500">Category:</span> {selectedEvent.category}</p>
				{/if}
			</div>
			<div class="mt-4 flex justify-between">
				{#if selectedEvent.type === 'event' && selectedEvent.event_id}
					<button onclick={() => deleteEvent(selectedEvent.event_id)} class="px-3 py-1 bg-red-600 hover:bg-red-700 rounded text-sm">Delete</button>
				{:else}
					<span></span>
				{/if}
				<button onclick={() => (selectedEvent = null)} class="px-3 py-1 bg-gray-700 rounded text-sm">Close</button>
			</div>
		</div>
	</div>
{/if}

{#if showAddEvent}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => (showAddEvent = false)} onkeydown={(e) => { if (e.key === 'Escape') showAddEvent = false; }}>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="bg-gray-800 rounded-lg p-6 w-[420px]" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
			<h3 class="text-lg font-bold mb-4">Add Event — {addEventDate}</h3>
			<div class="space-y-3">
				<div>
					<label for="ev-title" class="block text-sm text-gray-400 mb-1">Title</label>
					<input id="ev-title" bind:value={eventForm.title} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" placeholder="Dentist, Anniversary..." />
				</div>
				<div class="flex gap-3">
					<div class="flex-1">
						<label for="ev-date" class="block text-sm text-gray-400 mb-1">Date</label>
						<input id="ev-date" type="date" bind:value={eventForm.date} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
					<div class="flex-1">
						<label for="ev-time" class="block text-sm text-gray-400 mb-1">Time</label>
						<input id="ev-time" type="time" bind:value={eventForm.time} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
				</div>
				<div class="flex gap-3">
					<div class="flex-1">
						<label for="ev-cat" class="block text-sm text-gray-400 mb-1">Category</label>
						<input id="ev-cat" bind:value={eventForm.category} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" placeholder="anniversary, appointment..." />
					</div>
					<div class="flex-1">
						<label for="ev-rec" class="block text-sm text-gray-400 mb-1">Recurrence</label>
						<select id="ev-rec" bind:value={eventForm.recurrence} class="w-full p-2 bg-gray-700 border border-gray-600 rounded">
							<option value="">None (one-off)</option>
							<option value="yearly">Yearly</option>
							<option value="monthly">Monthly</option>
						</select>
					</div>
				</div>
				<div>
					<label for="ev-desc" class="block text-sm text-gray-400 mb-1">Description</label>
					<input id="ev-desc" bind:value={eventForm.description} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
				</div>
			</div>
			<div class="flex justify-end gap-2 mt-6">
				<button onclick={() => (showAddEvent = false)} class="px-3 py-1 bg-gray-700 rounded text-sm">Cancel</button>
				<button onclick={createEvent} class="px-3 py-1 bg-blue-600 hover:bg-blue-700 rounded text-sm">Add Event</button>
			</div>
		</div>
	</div>
{/if}
