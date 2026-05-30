export function decodeJWT<T = Record<string, unknown>>(
	token: string,
): T | null {
	try {
		const payload = token.split(".")[1];
		if (!payload) return null;
		return JSON.parse(atob(payload));
	} catch {
		return null;
	}
}
