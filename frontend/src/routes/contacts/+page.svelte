<script>
	import api from '$lib/api';
	import { onMount } from 'svelte';

	let contacts = $state([]);
	let loading = $state(true);
	let error = $state('');
	let search = $state('');

	let showModal = $state(false);
	let form = $state({ first_name: '', last_name: '', email: '', phone: '', company: '', role: '', notes: '', tags: '' });

	async function load() {
		loading = true;
		error = '';
		try {
			const data = await api.get('/api/v1/contacts');
			contacts = data.contacts || [];
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(load);

	let filtered = $derived.by(() => {
		if (!search.trim()) return contacts;
		const q = search.toLowerCase();
		return contacts.filter((c) =>
			(c.first_name || '').toLowerCase().includes(q) ||
			(c.last_name || '').toLowerCase().includes(q) ||
			(c.company || '').toLowerCase().includes(q) ||
			(c.email || '').toLowerCase().includes(q) ||
			(c.tags || '').toLowerCase().includes(q)
		);
	});

	async function createContact() {
		if (!form.first_name.trim()) return;
		try {
			await api.post('/api/v1/contacts', form);
			form = { first_name: '', last_name: '', email: '', phone: '', company: '', role: '', notes: '', tags: '' };
			showModal = false;
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	function parseTags(tags) {
		if (!tags) return [];
		return tags.split(',').map((t) => t.trim()).filter(Boolean);
	}
</script>

<div class="mb-4 flex items-center justify-between">
	<h2 class="text-2xl font-bold">Contacts</h2>
	<button onclick={() => (showModal = true)} class="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 rounded text-sm">+ Add Contact</button>
</div>

<div class="mb-4">
	<input bind:value={search} class="w-full max-w-md p-2 bg-gray-800 border border-gray-700 rounded text-sm" placeholder="Search by name, company, email, or tags..." />
</div>

{#if loading}
	<p class="text-gray-400">Loading contacts...</p>
{:else if error}
	<p class="text-red-400">{error}</p>
{:else if filtered.length === 0}
	<p class="text-gray-500">{search ? 'No contacts match your search.' : 'No contacts yet. Add one to get started.'}</p>
{:else}
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
		{#each filtered as contact (contact.id)}
			<a href="/contacts/{contact.id}" class="block bg-gray-800 rounded-lg p-4 hover:bg-gray-750 border border-gray-700 hover:border-gray-600 transition-colors">
				<div class="flex items-start justify-between">
					<div>
						<h3 class="font-semibold">{contact.first_name} {contact.last_name || ''}</h3>
						{#if contact.role || contact.company}
							<p class="text-sm text-gray-400">{contact.role || ''}{contact.role && contact.company ? ' at ' : ''}{contact.company || ''}</p>
						{/if}
					</div>
				</div>
				{#if contact.email}
					<p class="text-sm text-gray-500 mt-2">{contact.email}</p>
				{/if}
				{#if parseTags(contact.tags).length > 0}
					<div class="flex gap-1 mt-2 flex-wrap">
						{#each parseTags(contact.tags) as tag}
							<span class="text-xs px-1.5 py-0.5 bg-gray-700 rounded">{tag}</span>
						{/each}
					</div>
				{/if}
			</a>
		{/each}
	</div>
{/if}

{#if showModal}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => (showModal = false)}>
		<div class="bg-gray-800 rounded-lg p-6 w-[500px] max-h-[80vh] overflow-y-auto" onclick={(e) => e.stopPropagation()}>
			<h3 class="text-lg font-bold mb-4">Add Contact</h3>
			<div class="space-y-3">
				<div class="flex gap-3">
					<div class="flex-1">
						<label for="c-fname" class="block text-sm text-gray-400 mb-1">First Name</label>
						<input id="c-fname" bind:value={form.first_name} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
					<div class="flex-1">
						<label for="c-lname" class="block text-sm text-gray-400 mb-1">Last Name</label>
						<input id="c-lname" bind:value={form.last_name} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
				</div>
				<div class="flex gap-3">
					<div class="flex-1">
						<label for="c-email" class="block text-sm text-gray-400 mb-1">Email</label>
						<input id="c-email" type="email" bind:value={form.email} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
					<div class="flex-1">
						<label for="c-phone" class="block text-sm text-gray-400 mb-1">Phone</label>
						<input id="c-phone" bind:value={form.phone} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
				</div>
				<div class="flex gap-3">
					<div class="flex-1">
						<label for="c-company" class="block text-sm text-gray-400 mb-1">Company</label>
						<input id="c-company" bind:value={form.company} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
					<div class="flex-1">
						<label for="c-role" class="block text-sm text-gray-400 mb-1">Role</label>
						<input id="c-role" bind:value={form.role} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
				</div>
				<div>
					<label for="c-tags" class="block text-sm text-gray-400 mb-1">Tags (comma separated)</label>
					<input id="c-tags" bind:value={form.tags} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" placeholder="friend, work, family" />
				</div>
				<div>
					<label for="c-notes" class="block text-sm text-gray-400 mb-1">Notes</label>
					<textarea id="c-notes" bind:value={form.notes} rows="2" class="w-full p-2 bg-gray-700 border border-gray-600 rounded"></textarea>
				</div>
			</div>
			<div class="flex justify-end gap-2 mt-6">
				<button onclick={() => (showModal = false)} class="px-3 py-1 bg-gray-700 rounded text-sm">Cancel</button>
				<button onclick={createContact} class="px-3 py-1 bg-blue-600 hover:bg-blue-700 rounded text-sm">Add Contact</button>
			</div>
		</div>
	</div>
{/if}
