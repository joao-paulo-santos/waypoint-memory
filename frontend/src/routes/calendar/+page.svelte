<script>
	import api from '$lib/api';
	import { onMount } from 'svelte';

	let currentDate = $state(new Date());
	let events = $state([]);
	let loading = $state(true);
	let error = $state('');
	let selectedEvent = $state(null);

	let year = $derived(currentDate.getFullYear());
	let month = $derived(currentDate.getMonth());
	let monthName = $derived(currentDate.toLocaleString('default', { month: 'long', year: 'numeric' }));

	let firstDay = $derived(new Date(year, month, 1).getDay());
	let daysInMonth = $derived(new Date(year, month + 1, 0).getDate());
	let today = $derived(new Date().toISOString().slice(0, 10));

	let from = $derived(`${year}-${String(month + 1).padStart(2, '0')}-01`);
	let to = $derived(`${year}-${String(month + 1).padStart(2, '0')}-${String(daysInMonth).padStart(2, '0')}`);

	let eventsByDate = $derived.by(() => {
		const map = {};
		for (const e of events) {
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

	function eventColor(type) {
		switch (type) {
			case 'task_due': return 'bg-blue-500';
			case 'task_done': return 'bg-green-500';
			case 'birthday': return 'bg-purple-500';
			case 'recurring_event': return 'bg-orange-500';
			default: return 'bg-gray-500';
		}
	}

	function eventIcon(type) {
		switch (type) {
			case 'task_due': return '';
			case 'task_done': return '\u2713';
			case 'birthday': return '\uD83C\uDF82';
			case 'recurring_event': return '\u21BB';
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
				<div class="text-sm mb-1 {isToday ? 'text-blue-400 font-bold' : 'text-gray-400'}">{day}</div>
				<div class="space-y-1">
					{#each dayEvents.slice(0, 3) as event}
						<button
							onclick={() => (selectedEvent = event)}
							class="w-full text-left text-xs px-1.5 py-0.5 rounded {eventColor(event.type)} text-white truncate block"
							title={event.title}>
							{eventIcon(event.type)} {event.title}
						</button>
					{/each}
					{#if dayEvents.length > 3}
						<span class="text-xs text-gray-500">+{dayEvents.length - 3} more</span>
					{/if}
				</div>
			</div>
		{/each}
	</div>

	<div class="mt-4 flex gap-4 text-xs text-gray-500">
		<span class="flex items-center gap-1"><span class="w-3 h-3 rounded bg-blue-500 inline-block"></span> Task Due</span>
		<span class="flex items-center gap-1"><span class="w-3 h-3 rounded bg-green-500 inline-block"></span> Task Done</span>
		<span class="flex items-center gap-1"><span class="w-3 h-3 rounded bg-purple-500 inline-block"></span> Birthday</span>
		<span class="flex items-center gap-1"><span class="w-3 h-3 rounded bg-orange-500 inline-block"></span> Recurring</span>
	</div>
{/if}

{#if selectedEvent}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => (selectedEvent = null)}>
		<div class="bg-gray-800 rounded-lg p-4 w-[350px]" onclick={(e) => e.stopPropagation()}>
			<div class="flex items-center gap-2 mb-3">
				<span class="w-3 h-3 rounded {eventColor(selectedEvent.type)}"></span>
				<h3 class="font-bold">{selectedEvent.title}</h3>
			</div>
			<div class="space-y-1 text-sm text-gray-300">
				<p><span class="text-gray-500">Type:</span> {selectedEvent.type.replace(/_/g, ' ')}</p>
				<p><span class="text-gray-500">Date:</span> {selectedEvent.date}</p>
				<p><span class="text-gray-500">When:</span> {daysUntil(selectedEvent.date)}</p>
				{#if selectedEvent.project}
					<p><span class="text-gray-500">Project:</span> {selectedEvent.project}</p>
				{/if}
				{#if selectedEvent.age_turning != null}
					<p><span class="text-gray-500">Age:</span> {selectedEvent.age_turning}</p>
				{/if}
				{#if selectedEvent.recurrence}
					<p><span class="text-gray-500">Recurrence:</span> {selectedEvent.recurrence}</p>
				{/if}
				{#if selectedEvent.category}
					<p><span class="text-gray-500">Category:</span> {selectedEvent.category}</p>
				{/if}
			</div>
			<div class="mt-4 flex justify-end">
				<button onclick={() => (selectedEvent = null)} class="px-3 py-1 bg-gray-700 rounded text-sm">Close</button>
			</div>
		</div>
	</div>
{/if}
