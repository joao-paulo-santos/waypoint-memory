<script>
	import { toasts } from '$lib/toast.js';
</script>

<div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
	{#each $toasts as toast (toast.id)}
		<div class="px-4 py-2 rounded shadow-lg {toast.type === 'error' ? 'bg-red-600' : toast.type === 'success' ? 'bg-green-600' : 'bg-gray-700'} text-white text-sm">
			{toast.message}
		</div>
	{/each}
</div>
