import type { Icon } from "@phosphor-icons/react";
import {
	ChartBar,
	Cpu,
	FolderOpen,
	Gauge,
	HardDrive,
	HouseSimple,
	Package,
	Sliders,
	Users,
	WifiMedium,
} from "@phosphor-icons/react";

export interface NavShortcut {
	id: string;
	label: string;
	to: string;
	shortcut: string;
	section: string;
	icon: Icon;
}

export const navShortcuts: NavShortcut[] = [
	{
		id: "dashboard",
		label: "Dashboard",
		to: "/dashboard",
		shortcut: "d",
		section: "Navigate",
		icon: HouseSimple,
	},
	{
		id: "hosts",
		label: "Hosts",
		to: "/hosts",
		shortcut: "h",
		section: "Navigate",
		icon: Cpu,
	},
	{
		id: "images",
		label: "Images",
		to: "/images",
		shortcut: "i",
		section: "Navigate",
		icon: HardDrive,
	},
	{
		id: "tasks",
		label: "Tasks",
		to: "/tasks",
		shortcut: "t",
		section: "Navigate",
		icon: Gauge,
	},
	{
		id: "groups",
		label: "Groups",
		to: "/groups",
		shortcut: "g",
		section: "Navigate",
		icon: FolderOpen,
	},
	{
		id: "snapins",
		label: "Snapins",
		to: "/snapins",
		shortcut: "s",
		section: "Navigate",
		icon: Package,
	},
	{
		id: "storage",
		label: "Storage",
		to: "/storage",
		shortcut: "e",
		section: "Navigate",
		icon: WifiMedium,
	},
	{
		id: "users",
		label: "Users",
		to: "/users",
		shortcut: "u",
		section: "Administration",
		icon: Users,
	},
	{
		id: "settings",
		label: "Settings",
		to: "/settings",
		shortcut: ",",
		section: "Administration",
		icon: Sliders,
	},
	{
		id: "pending-macs",
		label: "Pending MACs",
		to: "/pending-macs",
		shortcut: "p",
		section: "Administration",
		icon: WifiMedium,
	},
	{
		id: "reports",
		label: "Reports",
		to: "/reports",
		shortcut: "r",
		section: "Reports",
		icon: ChartBar,
	},
];

export const goRoutes: Record<string, string> = Object.fromEntries(
	navShortcuts.map(({ shortcut, to }) => [shortcut, to]),
);

export function shortcutsBySection(): Map<string, NavShortcut[]> {
	return navShortcuts.reduce((acc, s) => {
		const list = acc.get(s.section);
		if (list) {
			list.push(s);
		} else {
			acc.set(s.section, [s]);
		}
		return acc;
	}, new Map<string, NavShortcut[]>());
}
