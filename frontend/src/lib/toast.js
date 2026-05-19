import { writable } from 'svelte/store';

export const toasts = writable([]);

export function addToast(message, type = 'info') {
	const id = Date.now();
	toasts.update(t => [...t, { id, message, type }]);
	setTimeout(() => {
		toasts.update(t => t.filter(x => x.id !== id));
	}, 3000);
}
