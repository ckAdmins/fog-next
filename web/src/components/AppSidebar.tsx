import type { Icon } from "@phosphor-icons/react";
import { Gear, SignOut, WifiX } from "@phosphor-icons/react";
import { Link, useRouter } from "@tanstack/react-router";
import { Fragment } from "react";
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarGroupContent,
	SidebarGroupLabel,
	SidebarHeader,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
	SidebarSeparator,
} from "@/components/ui/sidebar";
import { useOnlineStatus } from "@/hooks/useOnlineStatus";
import { navShortcuts } from "@/lib/shortcuts";
import { useAuthStore } from "@/store/auth";

const sections = navShortcuts.reduce<
	Record<string, { to: string; label: string; icon: Icon; shortcut: string }[]>
>((acc, s) => {
	if (!acc[s.section]) acc[s.section] = [];
	acc[s.section].push({
		to: s.to,
		label: s.label,
		icon: s.icon,
		shortcut: s.shortcut,
	});
	return acc;
}, {});

export function AppSidebar() {
	const router = useRouter();
	const logout = useAuthStore((s) => s.logout);
	const role = useAuthStore((s) => s.role);
	const isAdmin = role === "admin";
	const online = useOnlineStatus();

	const visibleSections = Object.entries(sections).filter(([name]) => {
		if (isAdmin) return true;
		return name === "Navigate";
	});

	const handleLogout = () => {
		logout();
		void router.navigate({ to: "/login" });
	};

	return (
		<Sidebar>
			<SidebarHeader>
				<div className="flex items-center gap-2.5 px-2 py-1.5">
					<div className="flex size-7 items-center justify-center rounded-md bg-primary/10">
						<Gear className="size-4 text-primary" weight="duotone" />
					</div>
					<span className="font-heading text-sm font-semibold tracking-tight">
						FOG Next
					</span>
				</div>
			</SidebarHeader>

			<SidebarContent>
				{visibleSections.map(([sectionName, items], idx) => (
					<Fragment key={sectionName}>
						{idx > 0 && <SidebarSeparator />}
						<SidebarGroup>
							<SidebarGroupLabel className="sidebar-group-label">
								{sectionName}
							</SidebarGroupLabel>
							<SidebarGroupContent>
								<SidebarMenu>
									{items.map(({ to, label, icon: Icon, shortcut }) => (
										<SidebarMenuItem key={to}>
											<SidebarMenuButton
												render={
													<Link
														to={to}
														activeProps={{
															className:
																"bg-sidebar-accent text-sidebar-accent-foreground font-medium",
														}}
													>
														<Icon weight="duotone" />
														<span>{label}</span>
														<kbd className="ml-auto hidden font-mono text-[9px] text-sidebar-foreground/30 group-hover/menu-button:text-sidebar-foreground/50 sm:inline-flex">
															g {shortcut}
														</kbd>
													</Link>
												}
											/>
										</SidebarMenuItem>
									))}
								</SidebarMenu>
							</SidebarGroupContent>
						</SidebarGroup>
					</Fragment>
				))}
			</SidebarContent>

			<SidebarFooter>
				{!online && (
					<div className="flex items-center gap-2 px-2 py-1.5 text-xs text-destructive">
						<WifiX className="size-3.5" />
						<span>Offline</span>
					</div>
				)}
				<SidebarMenu>
					<SidebarMenuItem>
						<SidebarMenuButton onClick={handleLogout}>
							<SignOut />
							<span>Sign out</span>
						</SidebarMenuButton>
					</SidebarMenuItem>
					<SidebarMenuItem>
						<div className="flex items-center gap-2 px-2 py-1">
							<kbd className="inline-flex h-5 select-none items-center rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground/50">
								⌘K
							</kbd>
							<span className="text-[10px] text-muted-foreground/50">
								Command palette
							</span>
						</div>
					</SidebarMenuItem>
				</SidebarMenu>
			</SidebarFooter>
		</Sidebar>
	);
}
