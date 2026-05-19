<script>
	import api from '$lib/api';
	import { onMount } from 'svelte';

	let birthdays = $state([]);
	let loading = $state(true);
	let error = $state('');

	let showModal = $state(false);
	let newName = $state('');
	let newDate = $state('');
	let newYear = $state('');
	let newNotes = $state('');

	async function load() {
		loading = true;
		error = '';
		try {
			const data = await api.get('/api/v1/birthdays/upcoming?days=365');
			birthdays = data.birthdays || [];
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(load);

	async function createBirthday() {
		if (!newName.trim() || !newDate) return;
		try {
			const body = { name: newName, date: newDate.slice(5) };
			if (newYear) body.year = parseInt(newYear);
			if (newNotes) body.notes = newNotes;
			await api.post('/api/v1/birthdays', body);
			newName = '';
			newDate = '';
			newYear = '';
			newNotes = '';
			showModal = false;
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function deleteBirthday(bid) {
		if (!confirm('Delete this birthday?')) return;
		try {
			await api.del(`/api/v1/birthdays/${bid}`);
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	function formatDaysUntil(d) {
		if (d === 0) return 'Today!';
		if (d === 1) return 'Tomorrow';
		return `In ${d} days`;
	}

	function monthGroup(birthdays) {
		const groups = {};
		for (const b of birthdays) {
			const month = new Date(b.occurrence_date).toLocaleString('default', { month: 'long', year: 'numeric' });
			if (!groups[month]) groups[month] = [];
			groups[month].push(b);
		}
		return Object.entries(groups);
	}
</script>

<div class="mb-4 flex items-center justify-between">
	<h2 class="text-2xl font-bold">Birthdays</h2>
	<button onclick={() => (showModal = true)} class="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 rounded text-sm">+ Add Birthday</button>
</div>

{#if loading}
	<p class="text-gray-400">Loading birthdays...</p>
{:else if error}
	<p class="text-red-400">{error}</p>
{:else if birthdays.length === 0}
	<p class="text-gray-500">No upcoming birthdays. Add one to get started.</p>
{:else}
	<div class="space-y-6">
		{#each monthGroup(birthdays) as [month, items]}
			<div>
				<h3 class="text-lg font-semibold text-gray-300 mb-2">{month}</h3>
				<div class="space-y-2">
					{#each items as b (b.id)}
						<div class="bg-gray-800 rounded-lg p-3 flex items-center justify-between">
							<div class="flex items-center gap-3">
								<div class="text-2xl">&#127874;</div>
								<div>
									<div class="flex items-center gap-2">
										<span class="font-medium">{b.name}</span>
										{#if b.age_turning != null}
											<span class="text-xs px-1.5 py-0.5 bg-purple-900 text-purple-300 rounded">turning {b.age_turning}</span>
										{/if}
									</div>
									<div class="text-xs text-gray-500">
										{b.occurrence_date}
										{#if b.contact_id}
											&middot; <a href="/contacts/{b.contact_id}" class="text-blue-400 hover:underline">View contact</a>
										{/if}
									</div>
								</div>
							</div>
							<div class="flex items-center gap-3">
								<span class="text-sm {b.days_until === 0 ? 'text-green-400 font-bold' : b.days_until <= 7 ? 'text-yellow-400' : 'text-gray-400'}">{formatDaysUntil(b.days_until)}</span>
								<button onclick={() => deleteBirthday(b.id)} class="text-gray-500 hover:text-red-400 text-sm" title="Delete">&#128465;</button>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/each}
	</div>
{/if}

{#if showModal}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => (showModal = false)}>
		<div class="bg-gray-800 rounded-lg p-6 w-[400px]" onclick={(e) => e.stopPropagation()}>
			<h3 class="text-lg font-bold mb-4">Add Birthday</h3>
			<div class="space-y-3">
				<div>
					<label for="bday-name" class="block text-sm text-gray-400 mb-1">Name</label>
					<input id="bday-name" bind:value={newName} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" placeholder="Person's name" />
				</div>
				<div>
					<label for="bday-date" class="block text-sm text-gray-400 mb-1">Date</label>
					<input id="bday-date" type="date" bind:value={newDate} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
				</div>
				<div>
					<label for="bday-year" class="block text-sm text-gray-400 mb-1">Birth Year (optional)</label>
					<input id="bday-year" type="number" bind:value={newYear} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" placeholder="1990" />
				</div>
				<div>
					<label for="bday-notes" class="block text-sm text-gray-400 mb-1">Notes (optional)</label>
					<input id="bday-notes" bind:value={newNotes} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" placeholder="Gift ideas, etc." />
				</div>
			</div>
			<div class="flex justify-end gap-2 mt-6">
				<button onclick={() => (showModal = false)} class="px-3 py-1 bg-gray-700 rounded text-sm">Cancel</button>
				<button onclick={createBirthday} class="px-3 py-1 bg-blue-600 hover:bg-blue-700 rounded text-sm">Add</button>
			</div>
		</div>
	</div>
{/if}
