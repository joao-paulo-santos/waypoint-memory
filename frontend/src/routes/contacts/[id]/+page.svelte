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

	async function load() {
		loading = true;
		error = '';
		try {
			contact = await api.get(`/api/v1/contacts/${id}`);
			form = { ...contact };
			if (contact.birthday_date) {
				const parts = contact.birthday_date.split('-');
				form.birthday_month = parts[0] || '';
				form.birthday_day = parts[1] || '';
			} else {
				form.birthday_month = '';
				form.birthday_day = '';
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
			for (const key of ['first_name', 'last_name', 'email', 'phone', 'company', 'role', 'notes', 'tags', 'birthday_notes']) {
				if (form[key] !== contact[key]) body[key] = form[key];
			}
			const newDate = (form.birthday_month && form.birthday_day) ? `${form.birthday_month}-${form.birthday_day}` : '';
			if (newDate !== (contact.birthday_date || '')) body.birthday_date = newDate;
			const byr = form.birthday_year ? parseInt(form.birthday_year) : null;
			const origByr = contact.birthday_year || null;
			if (byr !== origByr) body.birthday_year = byr;
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

	function parseTags(tags) {
		if (!tags) return [];
		return tags.split(',').map((t) => t.trim()).filter(Boolean);
	}

	function formatBirthday(c) {
		if (!c.birthday_date) return '';
		const months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
		const parts = c.birthday_date.split('-');
		if (parts.length !== 2) return c.birthday_date;
		const m = parseInt(parts[0]) - 1;
		const d = parts[1];
		let str = `${months[m]} ${d}`;
		if (c.birthday_year) str += `, ${c.birthday_year}`;
		return str;
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
					<button onclick={() => { editing = false; form = { ...contact, birthday_month: contact.birthday_date ? contact.birthday_date.split('-')[0] : '', birthday_day: contact.birthday_date ? contact.birthday_date.split('-')[1] : '' }; }} class="px-3 py-1.5 bg-gray-700 rounded text-sm">Cancel</button>
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
				<div class="border-t border-gray-700 pt-3">
					<h4 class="text-sm font-semibold text-purple-400 mb-2">Birthday</h4>
					<div class="flex gap-3">
						<div class="flex-1">
							<label for="ed-bmonth" class="block text-sm text-gray-400 mb-1">Month</label>
							<select id="ed-bmonth" bind:value={form.birthday_month} class="w-full p-2 bg-gray-700 border border-gray-600 rounded">
								<option value="">--</option>
								<option value="01">January</option>
								<option value="02">February</option>
								<option value="03">March</option>
								<option value="04">April</option>
								<option value="05">May</option>
								<option value="06">June</option>
								<option value="07">July</option>
								<option value="08">August</option>
								<option value="09">September</option>
								<option value="10">October</option>
								<option value="11">November</option>
								<option value="12">December</option>
							</select>
						</div>
						<div class="flex-1">
							<label for="ed-bday" class="block text-sm text-gray-400 mb-1">Day</label>
							<select id="ed-bday" bind:value={form.birthday_day} class="w-full p-2 bg-gray-700 border border-gray-600 rounded">
								<option value="">--</option>
								{#each Array(31) as _, i}
									{@const d = String(i + 1).padStart(2, '0')}
									<option value={d}>{i + 1}</option>
								{/each}
							</select>
						</div>
						<div class="flex-1">
							<label for="ed-byear" class="block text-sm text-gray-400 mb-1">Birth Year</label>
							<input id="ed-byear" type="number" bind:value={form.birthday_year} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" placeholder="1990" />
						</div>
					</div>
					<div class="mt-2">
						<label for="ed-bnotes" class="block text-sm text-gray-400 mb-1">Birthday Notes</label>
						<input id="ed-bnotes" bind:value={form.birthday_notes} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" placeholder="Gift ideas, preferences..." />
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
				{#if contact.birthday_date}
					<div class="flex">
						<span class="w-24 text-gray-500 text-sm shrink-0">Birthday</span>
						<span class="text-sm">&#127874; {formatBirthday(contact)}</span>
					</div>
				{/if}
				{#if contact.birthday_notes}
					<div class="flex">
						<span class="w-24 text-gray-500 text-sm shrink-0">Bday Notes</span>
						<span class="text-sm">{contact.birthday_notes}</span>
					</div>
				{/if}
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
	</div>
{/if}
