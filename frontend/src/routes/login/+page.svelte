<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	let password = $state('');
	let error = $state('');
	let loading = $state(true);
	let noPassword = $state(false);

	onMount(async () => {
		try {
			const res = await fetch('/api/v1/auth/status');
			const data = await res.json();
			if (!data.has_password) {
				noPassword = true;
				goto('/');
				return;
			}
		} catch {}
		finally {
			loading = false;
		}
	});

	async function handleLogin() {
		error = '';
		try {
			const res = await fetch('/api/v1/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ password })
			});
			if (!res.ok) {
				error = 'Invalid password';
				return;
			}
			goto('/');
		} catch {
			error = 'Connection error';
		}
	}
</script>

{#if loading}
	<div class="flex items-center justify-center h-screen bg-gray-900 text-gray-400">Loading...</div>
{:else if !noPassword}
	<div class="flex items-center justify-center h-screen bg-gray-900">
		<div class="bg-gray-800 rounded-lg p-8 w-[400px]">
			<div class="text-center mb-6">
				<h1 class="text-2xl font-bold">Waypoint</h1>
				<p class="text-gray-400 text-sm mt-1">Enter your password to continue</p>
			</div>
			<form onsubmit={(e) => { e.preventDefault(); handleLogin(); }}>
				<div class="mb-4">
					<input
						type="password"
						bind:value={password}
						class="w-full p-3 bg-gray-700 border border-gray-600 rounded text-center"
						placeholder="Password"
						autofocus
					/>
				</div>
				{#if error}
					<p class="text-red-400 text-sm text-center mb-3">{error}</p>
				{/if}
				<button type="submit" class="w-full p-3 bg-blue-600 hover:bg-blue-700 rounded font-medium">Login</button>
			</form>
		</div>
	</div>
{/if}
