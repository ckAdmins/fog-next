import {
	ChartBar,
	Cpu,
	FolderOpen,
	Gauge,
	Gear,
	HardDrive,
	HouseSimple,
	Package,
	SignOut,
	Sliders,
	Users,
	WifiMedium,
} from "@phosphor-icons/react";
import { Link, useRouter } from "@tanstack/react-router";
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
import { useAuthStore } from "@/store/auth";

const navItems = [
	{ to: "/dashboard", label: "Dashboard", icon: HouseSimple, shortcut: "d" },
	{ to: "/hosts", label: "Hosts", icon: Cpu, shortcut: "h" },
	{ to: "/images", label: "Images", icon: HardDrive, shortcut: "i" },
	{ to: "/tasks", label: "Tasks", icon: Gauge, shortcut: "t" },
	{ to: "/groups", label: "Groups", icon: FolderOpen, shortcut: "g" },
	{ to: "/snapins", label: "Snapins", icon: Package, shortcut: "s" },
	{ to: "/storage", label: "Storage", icon: WifiMedium, shortcut: null },
] as const;

const adminItems = [
	{ to: "/users", label: "Users", icon: Users, shortcut: "u" },
	{ to: "/settings", label: "Settings", icon: Sliders, shortcut: "," },
] as const;

const reportItems = [
	{ to: "/reports", label: "Reports", icon: ChartBar, shortcut: "r" },
] as const;

export function AppSidebar() {
	const router = useRouter();
	const logout = useAuthStore((s) => s.logout);

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
				<SidebarGroup>
					<SidebarGroupLabel className="sidebar-group-label">
						Navigation
					</SidebarGroupLabel>
					<SidebarGroupContent>
						<SidebarMenu>
							{navItems.map(({ to, label, icon: Icon, shortcut }) => (
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
												{shortcut != null && (
													<kbd className="ml-auto hidden font-mono text-[9px] text-sidebar-foreground/30 group-hover/menu-button:text-sidebar-foreground/50 sm:inline-flex">
														g {shortcut}
													</kbd>
												)}
											</Link>
										}
									/>
								</SidebarMenuItem>
							))}
						</SidebarMenu>
					</SidebarGroupContent>
				</SidebarGroup>

				<SidebarSeparator />

				<SidebarGroup>
					<SidebarGroupLabel className="sidebar-group-label">
						Administration
					</SidebarGroupLabel>
					<SidebarGroupContent>
						<SidebarMenu>
							{adminItems.map(({ to, label, icon: Icon, shortcut }) => (
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
												{shortcut != null && (
													<kbd className="ml-auto hidden font-mono text-[9px] text-sidebar-foreground/30 sm:inline-flex">
														g {shortcut}
													</kbd>
												)}
											</Link>
										}
									/>
								</SidebarMenuItem>
							))}
						</SidebarMenu>
					</SidebarGroupContent>
				</SidebarGroup>

				<SidebarSeparator />

				<SidebarGroup>
					<SidebarGroupContent>
						<SidebarMenu>
							{reportItems.map(({ to, label, icon: Icon, shortcut }) => (
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
												{shortcut != null && (
													<kbd className="ml-auto hidden font-mono text-[9px] text-sidebar-foreground/30 sm:inline-flex">
														g {shortcut}
													</kbd>
												)}
											</Link>
										}
									/>
								</SidebarMenuItem>
							))}
						</SidebarMenu>
					</SidebarGroupContent>
				</SidebarGroup>
			</SidebarContent>

			<SidebarFooter>
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
