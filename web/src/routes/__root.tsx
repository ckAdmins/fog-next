import { QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { createRootRoute, Outlet } from "@tanstack/react-router";
import { CommandPalette } from "@/components/CommandPalette";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useKeyboardShortcuts } from "@/hooks/useKeyboardShortcuts";
import { queryClient } from "@/lib/queryClient";

const RootLayout = () => {
	useKeyboardShortcuts();

	return (
		<QueryClientProvider client={queryClient}>
			<ThemeProvider>
				<TooltipProvider>
					<ErrorBoundary>
						<Outlet />
						<CommandPalette />
					</ErrorBoundary>
					<Toaster />
				</TooltipProvider>
			</ThemeProvider>
			<ReactQueryDevtools initialIsOpen={false} />
		</QueryClientProvider>
	);
};

export const Route = createRootRoute({ component: RootLayout });
