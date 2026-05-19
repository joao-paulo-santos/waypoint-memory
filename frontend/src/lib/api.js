const BASE = '';

/** @param {string} method @param {string} path @param {any} body */
async function request(method, path, body) {
	const opts = { method, headers: { 'Content-Type': 'application/json' } };
	if (body) opts.body = JSON.stringify(body);
	const res = await fetch(`${BASE}${path}`, opts);
	if (res.status === 204) return null;
	if (!res.ok) {
		const err = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(err.error || res.statusText);
	}
	return res.json();
}

export default {
	get: (/** @type {string} */ path) => request('GET', path),
	post: (/** @type {string} */ path, /** @type {any} */ body) => request('POST', path, body),
	put: (/** @type {string} */ path, /** @type {any} */ body) => request('PUT', path, body),
	del: (/** @type {string} */ path) => request('DELETE', path),
	patch: (/** @type {string} */ path, /** @type {any} */ body) => request('PATCH', path, body)
};
