import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { AppSidebar } from "@/components/app/app-sidebar";
import { RouteError } from "@/components/app/route-error";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { useAuthStore } from "@/store/auth";

export const Route = createFileRoute("/_auth")({
	beforeLoad: () => {
		if (!useAuthStore.getState().isAuthenticated) {
			throw redirect({ to: "/login" });
		}
	},
	component: AuthLayout,
	errorComponent: RouteError,
});

function AuthLayout() {
	return (
		<SidebarProvider>
			<AppSidebar />
			<SidebarInset>
				<div className="flex flex-1 flex-col gap-4 p-6">
					<Outlet />
				</div>
			</SidebarInset>
		</SidebarProvider>
	);
}
