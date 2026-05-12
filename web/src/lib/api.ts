import { isTokenExpired, useAuthStore } from "@/store/auth";
import type { TokenPair } from "@/types";

const BASE = "/fog/api/v1";

export class ApiError extends Error {
	status: number;
	constructor(status: number, message: string) {
		super(message);
		this.name = "ApiError";
		this.status = status;
	}
}

// Singleton refresh promise — prevents concurrent refresh races.
let _refreshPromise: Promise<void> | null = null;

/**
 * Ensures a valid, non-expired access token is in the store.
 * If the current token is expired it exchanges the refresh token for a new pair.
 * Concurrent callers share the same in-flight request.
 * Throws an ApiError(401) and calls logout() when the refresh itself fails.
 */
export async function ensureFreshToken(): Promise<void> {
	// If not expired, nothing to do.
	if (!isTokenExpired()) return;
	// If there's no refresh token (e.g. not logged in yet), skip silently.
	if (!useAuthStore.getState().refreshToken) return;
	if (!_refreshPromise) {
		_refreshPromise = _doRefresh().finally(() => {
			_refreshPromise = null;
		});
	}
	return _refreshPromise;
}

async function _doRefresh(): Promise<void> {
	const { refreshToken, logout, login } = useAuthStore.getState();
	if (!refreshToken) {
		logout();
		throw new ApiError(401, "Session expired");
	}
	const res = await fetch(`${BASE}/auth/refresh`, {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ refreshToken }),
	});
	if (!res.ok) {
		logout();
		throw new ApiError(401, "Session expired");
	}
	const tokens = (await res.json()) as TokenPair;
	login(tokens);
}

function authHeaders(token: string | null): Record<string, string> {
	return token ? { Authorization: `Bearer ${token}` } : {};
}

// parseErrorBody extracts a human-readable message from a JSON error response.
// The server returns RFC 7807 Problem Details like {"detail": "message"}.
function parseErrorBody(text: string): string {
	try {
		const parsed = JSON.parse(text) as { detail?: string; error?: string; message?: string };
		return parsed.detail || parsed.error || parsed.message || text;
	} catch {
		return text;
	}
}

async function request<T>(
	method: string,
	path: string,
	body?: unknown,
): Promise<T> {
	await ensureFreshToken();

	const token = useAuthStore.getState().accessToken;
	const res = await fetch(`${BASE}${path}`, {
		method,
		headers: {
			"Content-Type": "application/json",
			...authHeaders(token),
		},
		body: body !== undefined ? JSON.stringify(body) : undefined,
	});

	// On 401, attempt one token refresh and retry.
	if (res.status === 401) {
		await _doRefresh();
		const freshToken = useAuthStore.getState().accessToken;
		const retry = await fetch(`${BASE}${path}`, {
			method,
			headers: {
				"Content-Type": "application/json",
				...authHeaders(freshToken),
			},
			body: body !== undefined ? JSON.stringify(body) : undefined,
		});
		if (!retry.ok) {
			const text = await retry.text().catch(() => retry.statusText);
			throw new ApiError(retry.status, parseErrorBody(text || retry.statusText));
		}
		if (retry.status === 204) return undefined as T;
		return retry.json() as Promise<T>;
	}

	if (!res.ok) {
		const text = await res.text().catch(() => res.statusText);
		throw new ApiError(res.status, parseErrorBody(text || res.statusText));
	}

	if (res.status === 204) return undefined as T;
	return res.json() as Promise<T>;
}

async function upload<T>(path: string, formData: FormData): Promise<T> {
	await ensureFreshToken();

	const token = useAuthStore.getState().accessToken;
	const res = await fetch(`${BASE}${path}`, {
		method: "POST",
		headers: authHeaders(token),
		body: formData,
	});

	// On 401, attempt one token refresh and retry.
	if (res.status === 401) {
		await _doRefresh();
		const freshToken = useAuthStore.getState().accessToken;
		const retry = await fetch(`${BASE}${path}`, {
			method: "POST",
			headers: authHeaders(freshToken),
			body: formData,
		});
		if (!retry.ok) {
			const text = await retry.text().catch(() => retry.statusText);
			throw new ApiError(retry.status, parseErrorBody(text || retry.statusText));
		}
		if (retry.status === 204) return undefined as T;
		return retry.json() as Promise<T>;
	}

	if (!res.ok) {
		const text = await res.text().catch(() => res.statusText);
		throw new ApiError(res.status, parseErrorBody(text || res.statusText));
	}

	if (res.status === 204) return undefined as T;
	return res.json() as Promise<T>;
}

export const api = {
	get: <T>(path: string) => request<T>("GET", path),
	post: <T>(path: string, body?: unknown) => request<T>("POST", path, body),
	put: <T>(path: string, body?: unknown) => request<T>("PUT", path, body),
	patch: <T>(path: string, body?: unknown) => request<T>("PATCH", path, body),
	del: <T>(path: string) => request<T>("DELETE", path),
	upload: <T>(path: string, formData: FormData) => upload<T>(path, formData),
};
