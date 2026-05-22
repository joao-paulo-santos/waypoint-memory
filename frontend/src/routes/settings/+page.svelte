<script>
	import api from '$lib/api';
	import { onMount } from 'svelte';

	let tokens = $state([]);
	let loading = $state(true);
	let error = $state('');

	let showCreate = $state(false);
	let tokenName = $state('');
	let tokenPermissions = $state('');
	let newToken = $state(null);
	let copied = $state(false);

	let currentUser = $state(null);

	async function load() {
		loading = true;
		error = '';
		try {
			const [tData, aData] = await Promise.all([
				api.get('/api/v1/tokens'),
				api.get('/api/v1/auth/status')
			]);
			tokens = tData.tokens || [];
			if (aData.authenticated) currentUser = aData.user;
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(load);

	async function createToken() {
		if (!tokenName.trim()) return;
		try {
			const body = { name: tokenName };
			if (tokenPermissions) body.permissions = tokenPermissions;
			const result = await api.post('/api/v1/tokens', body);
			newToken = result;
			copied = false;
			tokenName = '';
			tokenPermissions = '';
			showCreate = false;
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function deleteToken(id) {
		if (!confirm('Revoke this token? Any applications using it will lose access.')) return;
		try {
			await api.del(`/api/v1/tokens/${id}`);
			await load();
		} catch (e) {
			alert(e.message);
		}
	}

	async function copyToken() {
		if (newToken?.token) {
			await navigator.clipboard.writeText(newToken.token);
			copied = true;
		}
	}

	function formatDate(d) {
		if (!d) return 'Never';
		return d.slice(0, 10);
	}
</script>

<h2 class="text-2xl font-bold mb-6">Settings</h2>

<div class="max-w-3xl space-y-8">
	<div>
		<h3 class="text-lg font-semibold mb-3">Account</h3>
		{#if currentUser}
			<p class="text-sm text-gray-400">Logged in as <span class="text-white font-medium">{currentUser.username}</span></p>
		{:else}
			<p class="text-sm text-gray-400">Loading user info...</p>
		{/if}
	</div>

	<div>
		<h3 class="text-lg font-semibold mb-3">API Tokens</h3>
		<p class="text-sm text-gray-400 mb-4">Manage API tokens for MCP server authentication.</p>

		<div class="mb-4">
			<button onclick={() => (showCreate = true)} class="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 rounded text-sm">+ Generate Token</button>
		</div>

		{#if newToken}
			<div class="bg-yellow-900/30 border border-yellow-600 rounded-lg p-4 mb-4">
				<h4 class="font-bold text-yellow-400 mb-2">New Token Created</h4>
				<p class="text-sm text-yellow-300 mb-3">Copy this token now. It will not be shown again.</p>
				<div class="flex gap-2">
					<code class="flex-1 p-2 bg-gray-900 rounded text-sm font-mono break-all">{newToken.token}</code>
					<button onclick={copyToken} class="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-sm shrink-0">
						{copied ? 'Copied!' : 'Copy'}
					</button>
				</div>
				<div class="mt-2 text-sm text-gray-400">
					<span class="font-medium">{newToken.info.name}</span> &middot; prefix: <code class="bg-gray-700 px-1 rounded">{newToken.info.prefix}...</code>
				</div>
				<button onclick={() => (newToken = null)} class="mt-3 text-sm text-gray-400 hover:text-gray-300">Dismiss</button>
			</div>
		{/if}

		{#if loading}
			<p class="text-gray-400">Loading tokens...</p>
		{:else if error}
			<p class="text-red-400">{error}</p>
		{:else if tokens.length === 0}
			<p class="text-gray-500 text-sm">No API tokens. Generate one to connect MCP clients.</p>
		{:else}
			<div class="bg-gray-800 rounded-lg divide-y divide-gray-700">
				{#each tokens as token (token.id)}
					<div class="p-4 flex items-center justify-between">
						<div>
							<div class="flex items-center gap-2">
								<span class="font-medium">{token.name}</span>
								<code class="text-xs px-1.5 py-0.5 bg-gray-700 rounded">{token.prefix}...</code>
							</div>
							<div class="text-xs text-gray-500 mt-1">
								Created {formatDate(token.created_at)}
								{#if token.last_used}
									&middot; Last used {formatDate(token.last_used)}
								{/if}
							</div>
						</div>
						<button onclick={() => deleteToken(token.id)} class="px-3 py-1 bg-red-600/20 hover:bg-red-600 text-red-400 hover:text-white rounded text-sm">Revoke</button>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>

{#if showCreate}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={() => (showCreate = false)} onkeydown={(e) => { if (e.key === 'Escape') showCreate = false; }}>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="bg-gray-800 rounded-lg p-6 w-[400px]" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
			<h3 class="text-lg font-bold mb-4">Generate API Token</h3>
			<div class="space-y-3">
				<div>
					<label for="tok-name" class="block text-sm text-gray-400 mb-1">Token Name</label>
					<input id="tok-name" bind:value={tokenName} class="w-full p-2 bg-gray-700 border border-gray-600 rounded" placeholder="e.g. Claude Desktop" />
				</div>
				<div>
					<label for="tok-perm" class="block text-sm text-gray-400 mb-1">Permissions (JSON, optional)</label>
					<input id="tok-perm" bind:value={tokenPermissions} class="w-full p-2 bg-gray-700 border border-gray-600 rounded font-mono text-sm" placeholder='["read","write"]' />
				</div>
			</div>
			<div class="flex justify-end gap-2 mt-6">
				<button onclick={() => (showCreate = false)} class="px-3 py-1 bg-gray-700 rounded text-sm">Cancel</button>
				<button onclick={createToken} class="px-3 py-1 bg-blue-600 hover:bg-blue-700 rounded text-sm">Generate</button>
			</div>
		</div>
	</div>
{/if}
