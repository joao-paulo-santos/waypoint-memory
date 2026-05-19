<script>
	import api from '$lib/api';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	let id = $derived($page.params.id);
	let contact = $state(null);
	let loading = $state(true);
	let error = $state('');

	let editing = $state(false);
	let form = $state({});

	let birthday = $state(null);
	let showBirthdayModal = $state(false);
	let bdayForm = $state({ name: '', date: '', year: '', notes: '' });

	async function load() {
		loading = true;
		error = '';
		try {
			contact = await api.get(`/api/v1/contacts/${id}`);
			form = { ...contact };

			const bdata = await api.get('/api/v1/birthdays');
			const all = bdata.birthdays || [];
			birthday = all.find((b) => b.contact_id && b.contact_id === contact.id) || null;
			if (birthday) {
				const year = birthday.year || new Date().getFullYear();
				bdayForm = { name: birthday.name, date: `${year}-${birthday.date}`, year: birthday.year || '', notes: birthday.notes || '' };
			}
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(load);

	async function saveContact() {
		try {
			const body = {};
			for (const key of ['first_name', 'last_name', 'email', 'phone', 'company', 'role', 'notes', 'tags']) {
				if (form[key] !== contact[key]) body[key] = form[key];
			}
			if (Object.keys(body).length > 0) {
				contact = await api.put(`/api/v1/contacts/${id}`, body);
			}
			editing = false;
		} catch (e) {
			alert(e.message);
		}
	}

	async function deleteContact() {
		if (!confirm('Delete this contact? This cannot be undone.')) return;
		try {
			await api.del(`/api/v1/contacts/${id}`);
			goto('/contacts');
		} catch (e) {
			alert(e.message);
		}
	}

	async function addBirthday() {
		if (!bdayForm.date) return;
		try {
			const body = { name: bdayForm.name, date: bdayForm.date.slice(5), contact_id: parseInt(id) };
			if (bdayForm.year) body.year = parseInt(bdayForm.year);
			if (bdayForm.notes) body.notes = bdayForm.notes;
			await api.post('/api/v1/birthdays', body);
			showBirthdayModal = false;
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function deleteBirthday() {
		if (!birthday || !confirm('Remove this birthday?')) return;
		try {
			await api.del(`/api/v1/birthdays/${birthday.id}`);
			birthday = null;
		} catch (e) {
			alert(e.message);
		}
	}

	function parseTags(tags) {
		if (!tags) return [];
		return tags.split(',').map((t) => t.trim()).filter(Boolean);
	}

	function field(label, key) {
		return { label, value: contact?.[key] };
	}
</script>

<div class="mb-4 flex items-center gap-3">
	<a href="/contacts" class="text-gray-400 hover:text-white text-sm">&larr; Contacts</a>
</div>

{#if loading}
	<p class="text-gray-400">Loading contact...</p>
{:else if error}
	<p class="text-red-400">{error}</p>
{:else if contact}
	<div class="max-w-2xl">
		<div class="flex items-center justify-between mb-6">
			<h2 class="text-2xl font-bold">{contact.first_name} {contact.last_name || ''}</h2>
			<div class="flex gap-2">
				{#if editing}
					<button onclick={saveContact} class="px-3 py-1.5 bg-green-600 hover:bg-green-700 rounded text-sm">Save</button>
					<button onclick={() => { editing = false; form = { ...contact }; }} class="px-3 py-1.5 bg-gray-700 rounded text-sm">Cancel</button>
				{:else}
					<button onclick={() => (editing = true)} class="px-3 py-1.5 bg-gray-700 hover:bg-gray-600 rounded text-sm">Edit</button>
					<button onclick={deleteContact} class="px-3 py-1.5 bg-red-600 hover:bg-red-700 rounded text-sm">Delete</button>
				{/if}
			</div>
		</div>

		{#if editing}
			<div class="bg-gray-800 rounded-lg p-4 space-y-3">
				<div class="flex gap-3">
					<div class="flex-1">
						<label for="ed-fname" class="block text-sm text-gray-400 mb-1">First Name</label>
						<input id="ed-fname" bind:value={form.first_name} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
					<div class="flex-1">
						<label for="ed-lname" class="block text-sm text-gray-400 mb-1">Last Name</label>
						<input id="ed-lname" bind:value={form.last_name} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
				</div>
				<div class="flex gap-3">
					<div class="flex-1">
						<label for="ed-email" class="block text-sm text-gray-400 mb-1">Email</label>
						<input id="ed-email" type="email" bind:value={form.email} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
					<div class="flex-1">
						<label for="ed-phone" class="block text-sm text-gray-400 mb-1">Phone</label>
						<input id="ed-phone" bind:value={form.phone} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
				</div>
				<div class="flex gap-3">
					<div class="flex-1">
						<label for="ed-company" class="block text-sm text-gray-400 mb-1">Company</label>
						<input id="ed-company" bind:value={form.company} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
					<div class="flex-1">
						<label for="ed-role" class="block text-sm text-gray-400 mb-1">Role</label>
						<input id="ed-role" bind:value={form.role} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
					</div>
				</div>
				<div>
					<label for="ed-tags" class="block text-sm text-gray-400 mb-1">Tags (comma separated)</label>
					<input id="ed-tags" bind:value={form.tags} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
				</div>
				<div>
					<label for="ed-notes" class="block text-sm text-gray-400 mb-1">Notes</label>
					<textarea id="ed-notes" bind:value={form.notes} rows="3" class="w-full p-2 bg-gray-700 border border-gray-600 rounded"></textarea>
				</div>
			</div>
		{:else}
			<div class="bg-gray-800 rounded-lg p-4 space-y-3">
				{#each [['Email', 'email'], ['Phone', 'phone'], ['Company', 'company'], ['Role', 'role']] as [label, key]}
					{#if contact[key]}
						<div class="flex">
							<span class="w-24 text-gray-500 text-sm shrink-0">{label}</span>
							<span class="text-sm">{contact[key]}</span>
						</div>
					{/if}
				{/each}
				{#if parseTags(contact.tags).length > 0}
					<div class="flex">
						<span class="w-24 text-gray-500 text-sm shrink-0">Tags</span>
						<div class="flex gap-1 flex-wrap">
							{#each parseTags(contact.tags) as tag}
								<span class="text-xs px-1.5 py-0.5 bg-gray-700 rounded">{tag}</span>
							{/each}
						</div>
					</div>
				{/if}
				{#if contact.notes}
					<div class="flex">
						<span class="w-24 text-gray-500 text-sm shrink-0">Notes</span>
						<span class="text-sm whitespace-pre-wrap">{contact.notes}</span>
					</div>
				{/if}
			</div>
		{/if}

		<div class="mt-6">
			<div class="flex items-center justify-between mb-3">
				<h3 class="text-lg font-semibold">Birthday</h3>
				{#if !birthday}
					<button onclick={() => {
						bdayForm = { name: `${contact.first_name} ${contact.last_name || ''}`.trim(), date: '', year: '', notes: '' };
						showBirthdayModal = true;
					}} class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm">+ Add Birthday</button>
				{/if}
			</div>
			{#if birthday}
				<div class="bg-gray-800 rounded-lg p-3 flex items-center justify-between">
					<div class="flex items-center gap-2">
						<span class="text-xl">&#127874;</span>
						<div>
							<span class="font-medium">{birthday.name}</span>
							<span class="text-sm text-gray-400 ml-2">{birthday.date}</span>
							{#if birthday.year}
								<span class="text-xs text-gray-500 ml-1">(born {birthday.year})</span>
							{/if}
						</div>
					</div>
					<button onclick={deleteBirthday} class="text-gray-500 hover:text-red-400 text-sm" title="Remove birthday">&#128465;</button>
				</div>
			{:else}
				<p class="text-gray-500 text-sm">No birthday linked to this contact.</p>
			{/if}
		</div>
	</div>
{/if}

{#if showBirthdayModal}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => (showBirthdayModal = false)}>
		<div class="bg-gray-800 rounded-lg p-6 w-[400px]" onclick={(e) => e.stopPropagation()}>
			<h3 class="text-lg font-bold mb-4">Add Birthday for {contact?.first_name}</h3>
			<div class="space-y-3">
				<div>
					<label for="bd-modal-name" class="block text-sm text-gray-400 mb-1">Name</label>
					<input id="bd-modal-name" bind:value={bdayForm.name} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
				</div>
				<div>
					<label for="bd-modal-date" class="block text-sm text-gray-400 mb-1">Date</label>
					<input id="bd-modal-date" type="date" bind:value={bdayForm.date} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
				</div>
				<div>
					<label for="bd-modal-year" class="block text-sm text-gray-400 mb-1">Birth Year (optional)</label>
					<input id="bd-modal-year" type="number" bind:value={bdayForm.year} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" />
				</div>
			</div>
			<div class="flex justify-end gap-2 mt-6">
				<button onclick={() => (showBirthdayModal = false)} class="px-3 py-1 bg-gray-700 rounded text-sm">Cancel</button>
				<button onclick={addBirthday} class="px-3 py-1 bg-blue-600 hover:bg-blue-700 rounded text-sm">Add</button>
			</div>
		</div>
	</div>
{/if}
