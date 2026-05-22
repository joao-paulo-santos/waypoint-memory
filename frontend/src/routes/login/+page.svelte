<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	let loading = $state(true);
	let isRegister = $state(false);
	let username = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');

	onMount(async () => {
		try {
			const res = await fetch('/api/v1/auth/has-users');
			const data = await res.json();
			isRegister = !data.has_users;

			const statusRes = await fetch('/api/v1/auth/status');
			const statusData = await statusRes.json();
			if (statusData.authenticated) {
				goto('/');
				return;
			}
		} catch {}
		finally {
			loading = false;
		}
	});

	async function handleSubmit() {
		error = '';
		if (!username.trim() || !password) {
			error = 'Please fill in all fields';
			return;
		}
		if (isRegister) {
			if (password !== confirmPassword) {
				error = 'Passwords do not match';
				return;
			}
			if (password.length < 4) {
				error = 'Password must be at least 4 characters';
				return;
			}
			try {
				await fetch('/api/v1/auth/register', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ username, password })
				});
			} catch {
				error = 'Connection error';
				return;
			}
		}
		try {
			const res = await fetch('/api/v1/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ username, password })
			});
			if (!res.ok) {
				const data = await res.json().catch(() => ({}));
				error = data.error || 'Login failed';
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
{:else}
	<div class="flex items-center justify-center h-screen bg-gray-900">
		<div class="bg-gray-800 rounded-lg p-8 w-[400px]">
			<div class="text-center mb-6">
				<h1 class="text-2xl font-bold text-white">Waypoint Memory</h1>
				<p class="text-gray-300 text-sm mt-1">
					{isRegister ? 'Create an account to get started' : 'Enter your credentials to continue'}
				</p>
			</div>
			<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
				{#if isRegister}
					<div class="mb-4">
						<label for="username" class="block text-sm text-gray-300 mb-1">Username</label>
						<!-- svelte-ignore a11y_autofocus -->
						<input id="username" type="text" bind:value={username} class="w-full p-3 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-400" placeholder="Username" autofocus />
					</div>
				{:else}
					<div class="mb-4">
						<label for="username" class="block text-sm text-gray-300 mb-1">Username</label>
						<!-- svelte-ignore a11y_autofocus -->
						<input id="username" type="text" bind:value={username} class="w-full p-3 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-400" placeholder="Username" autofocus />
					</div>
				{/if}
				<div class="mb-4">
					<label for="password" class="block text-sm text-gray-300 mb-1">Password</label>
					<input id="password" type="password" bind:value={password} class="w-full p-3 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-400" placeholder="Password" />
				</div>
				{#if isRegister}
					<div class="mb-4">
						<label for="confirm" class="block text-sm text-gray-300 mb-1">Confirm Password</label>
						<input id="confirm" type="password" bind:value={confirmPassword} class="w-full p-3 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-400" placeholder="Confirm Password" />
					</div>
				{/if}
				{#if error}
					<p class="text-red-400 text-sm text-center mb-3">{error}</p>
				{/if}
				<button type="submit" class="w-full p-3 bg-blue-600 hover:bg-blue-700 rounded font-medium">
					{isRegister ? 'Create Account' : 'Login'}
				</button>
			</form>
		</div>
	</div>
{/if}
