<script>
	import { page } from '$app/stores';
	import '../app.css';

	let { children } = $props();
</script>

<div class="flex h-screen bg-gray-900 text-gray-100">
	<nav class="w-64 bg-gray-800 border-r border-gray-700 p-4 flex flex-col shrink-0">
		<div class="mb-8">
			<h1 class="text-xl font-bold text-white">Waypoint</h1>
		</div>
		<div class="flex flex-col gap-2">
			<a href="/" class="nav-link" class:active={$page.url.pathname === '/'}>Dashboard</a>
			<a href="/projects" class="nav-link" class:active={$page.url.pathname.startsWith('/projects')}>Projects</a>
			<a href="/calendar" class="nav-link" class:active={$page.url.pathname === '/calendar'}>Calendar</a>
			<a href="/contacts" class="nav-link" class:active={$page.url.pathname.startsWith('/contacts')}>Contacts</a>
			<a href="/birthdays" class="nav-link" class:active={$page.url.pathname === '/birthdays'}>Birthdays</a>
			<a href="/settings" class="nav-link" class:active={$page.url.pathname === '/settings'}>Settings</a>
		</div>
	</nav>
	<main class="flex-1 overflow-auto p-6">
		{@render children()}
	</main>
</div>

<style>
	.nav-link {
		padding: 0.5rem 0.75rem;
		border-radius: 0.375rem;
		color: #d1d5db;
		text-decoration: none;
		display: block;
	}
	.nav-link:hover {
		background-color: #374151;
		color: white;
	}
	.nav-link.active {
		background-color: #374151;
		color: white;
	}
</style>
