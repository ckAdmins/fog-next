import { useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";
import { goRoutes } from "@/lib/shortcuts";

export function useKeyboardShortcuts() {
	const navigate = useNavigate();

	useEffect(() => {
		let pendingGo = false;

		const isEditable = (target: EventTarget | null): boolean => {
			if (!(target instanceof HTMLElement)) return false;
			if (target.isContentEditable) return true;
			const tag = target.tagName.toLowerCase();
			if (tag === "input" || tag === "textarea" || tag === "select") {
				return true;
			}
			return !!target.closest(
				'input, textarea, select, [contenteditable="true"]',
			);
		};

		const handler = (e: KeyboardEvent) => {
			// Skip if any modifier is held
			if (e.metaKey || e.ctrlKey || e.altKey) return;
			// Skip in editable fields
			if (isEditable(e.target)) return;

			const key = e.key.toLowerCase();

			// "g" prefix: wait for next key
			if (!pendingGo && key === "g") {
				pendingGo = true;
				e.preventDefault();
				return;
			}

			if (pendingGo) {
				pendingGo = false;
				const route = goRoutes[key];
				if (route) {
					e.preventDefault();
					navigate({ to: route });
				}
				return;
			}
		};

		// Reset pending on any non-matching key
		const resetOnAny = () => {
			pendingGo = false;
		};

		window.addEventListener("keydown", handler);
		window.addEventListener("keyup", resetOnAny);

		return () => {
			window.removeEventListener("keydown", handler);
			window.removeEventListener("keyup", resetOnAny);
		};
	}, [navigate]);
}
