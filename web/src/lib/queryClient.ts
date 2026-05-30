import { QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

export const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			staleTime: 30_000,
			gcTime: 5 * 60_000,
			retry: 1,
		},
		mutations: {
			onError: (err) => {
				toast.error(err instanceof Error ? err.message : "An error occurred");
			},
		},
	},
});
