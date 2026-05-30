import { create } from "zustand";
import { persist } from "zustand/middleware";
import { decodeJWT } from "@/lib/jwt";

interface JWTClaims {
	uid: string;
	sub: string;
	role: string;
}

interface TokenPair {
	accessToken: string;
	refreshToken: string;
	expiresAt: string;
}

export type Role = "admin" | "mobile" | "readonly";

interface AuthState {
	accessToken: string | null;
	refreshToken: string | null;
	expiresAt: string | null;
	role: Role | null;
	isAuthenticated: boolean;
	login: (tokens: TokenPair) => void;
	logout: () => void;
}

export const useAuthStore = create<AuthState>()(
	persist(
		(set) => ({
			accessToken: null,
			refreshToken: null,
			expiresAt: null,
			role: null,
			isAuthenticated: false,

			login: (tokens) => {
				const claims = decodeJWT<JWTClaims>(tokens.accessToken);
				set({
					accessToken: tokens.accessToken,
					refreshToken: tokens.refreshToken,
					expiresAt: tokens.expiresAt,
					role: (claims?.role as Role) ?? null,
					isAuthenticated: true,
				});
			},

			logout: () => {
				set({
					accessToken: null,
					refreshToken: null,
					expiresAt: null,
					role: null,
					isAuthenticated: false,
				});
			},
		}),
		{ name: "fog-auth" },
	),
);

export function isTokenExpired(): boolean {
	const { expiresAt } = useAuthStore.getState();
	if (!expiresAt) return true;
	return new Date(expiresAt).getTime() - 10_000 <= Date.now();
}

export function isAdmin(): boolean {
	const role = useAuthStore.getState().role;
	return role === "admin";
}
