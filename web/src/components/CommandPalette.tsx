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
	X,
} from "@phosphor-icons/react";
import { useNavigate } from "@tanstack/react-router";
import {
	type KeyboardEvent,
	useCallback,
	useEffect,
	useRef,
	useState,
} from "react";
import { Input } from "@/components/ui/input";

interface CommandItem {
	id: string;
	label: string;
	icon: React.ElementType;
	to: string;
	shortcut?: string;
	section: string;
}

const commands: CommandItem[] = [
	{
		id: "dashboard",
		label: "Dashboard",
		icon: HouseSimple,
		to: "/dashboard",
		shortcut: "g d",
		section: "Navigate",
	},
	{
		id: "hosts",
		label: "Hosts",
		icon: Cpu,
		to: "/hosts",
		shortcut: "g h",
		section: "Navigate",
	},
	{
		id: "images",
		label: "Images",
		icon: HardDrive,
		to: "/images",
		shortcut: "g i",
		section: "Navigate",
	},
	{
		id: "tasks",
		label: "Tasks",
		icon: Gauge,
		to: "/tasks",
		shortcut: "g t",
		section: "Navigate",
	},
	{
		id: "groups",
		label: "Groups",
		icon: FolderOpen,
		to: "/groups",
		shortcut: "g g",
		section: "Navigate",
	},
	{
		id: "snapins",
		label: "Snapins",
		icon: Package,
		to: "/snapins",
		shortcut: "g s",
		section: "Navigate",
	},
	{
		id: "storage",
		label: "Storage",
		icon: WifiMedium,
		to: "/storage",
		section: "Navigate",
	},
	{
		id: "users",
		label: "Users",
		icon: Users,
		to: "/users",
		shortcut: "g u",
		section: "Administration",
	},
	{
		id: "settings",
		label: "Settings",
		icon: Sliders,
		to: "/settings",
		shortcut: "g ,",
		section: "Administration",
	},
	{
		id: "pending-macs",
		label: "Pending MACs",
		icon: WifiMedium,
		to: "/pending-macs",
		section: "Administration",
	},
	{
		id: "reports",
		label: "Reports",
		icon: ChartBar,
		to: "/reports",
		section: "Reports",
	},
];

export function CommandPalette() {
	const [open, setOpen] = useState(false);
	const [query, setQuery] = useState("");
	const [selectedIdx, setSelectedIdx] = useState(0);
	const inputRef = useRef<HTMLInputElement>(null);
	const listRef = useRef<HTMLDivElement>(null);
	const navigate = useNavigate();

	const filtered = query.trim()
		? commands.filter(
				(c) =>
					c.label.toLowerCase().includes(query.toLowerCase()) ||
					c.id.toLowerCase().includes(query.toLowerCase()) ||
					c.section.toLowerCase().includes(query.toLowerCase()),
			)
		: commands;

	const grouped = filtered.reduce<Record<string, CommandItem[]>>((acc, c) => {
		if (!acc[c.section]) acc[c.section] = [];
		acc[c.section].push(c);
		return acc;
	}, {});

	const entries = Object.entries(grouped);

	const handleSelect = useCallback(
		(id: string) => {
			const cmd = commands.find((c) => c.id === id);
			if (cmd) {
				navigate({ to: cmd.to });
				setOpen(false);
				setQuery("");
			}
		},
		[navigate],
	);

	// Global keyboard shortcut: Cmd/Ctrl+K to open palette
	useEffect(() => {
		const handler = (e: globalThis.KeyboardEvent) => {
			if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
				e.preventDefault();
				setOpen((prev) => !prev);
				setSelectedIdx(0);
			}
			if (e.key === "Escape" && open) {
				setOpen(false);
			}
		};
		window.addEventListener("keydown", handler);
		return () => window.removeEventListener("keydown", handler);
	}, [open]);

	// Focus input when opened
	useEffect(() => {
		if (open) {
			setTimeout(() => inputRef.current?.focus(), 50);
		} else {
			setQuery("");
			setSelectedIdx(0);
		}
	}, [open]);

	// Keyboard navigation within the palette
	const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
		switch (e.key) {
			case "ArrowDown": {
				e.preventDefault();
				setSelectedIdx((prev) => (prev + 1) % filtered.length);
				break;
			}
			case "ArrowUp": {
				e.preventDefault();
				setSelectedIdx((prev) => (prev === 0 ? filtered.length - 1 : prev - 1));
				break;
			}
			case "Enter": {
				e.preventDefault();
				if (filtered[selectedIdx]) {
					handleSelect(filtered[selectedIdx].id);
				}
				break;
			}
			case "Escape": {
				setOpen(false);
				break;
			}
		}
	};

	if (!open) return null;

	return (
		<div className="fixed inset-0 z-50">
			{/* Backdrop */}
			<button
				type="button"
				className="absolute inset-0 bg-background/80 backdrop-blur-sm border-0 cursor-default"
				onClick={() => setOpen(false)}
				onKeyDown={(e) => {
					if (e.key === "Escape") setOpen(false);
				}}
				aria-label="Close command palette"
			/>

			{/* Palette */}
			<div className="absolute left-1/2 top-[20%] w-full max-w-lg -translate-x-1/2 animate-in fade-in slide-in-from-top-2 duration-150">
				<div className="rounded-lg border bg-card shadow-2xl overflow-hidden">
					{/* Search */}
					<div className="flex items-center border-b px-3">
						<Input
							ref={inputRef}
							value={query}
							onChange={(e) => {
								setQuery(e.target.value);
								setSelectedIdx(0);
							}}
							onKeyDown={handleKeyDown}
							placeholder="Type a command or search..."
							className="border-0 bg-transparent h-12 text-sm focus-visible:ring-0 focus-visible:ring-offset-0"
						/>
						<kbd className="pointer-events-none hidden h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground sm:flex">
							<X className="size-2" />
						</kbd>
					</div>

					{/* Results */}
					<div ref={listRef} className="max-h-72 overflow-y-auto p-2">
						{entries.length === 0 && (
							<p className="text-center text-sm text-muted-foreground py-8">
								No results found.
							</p>
						)}
						{entries.map(([section, items]) => (
							<div key={section}>
								<div className="px-2 py-1.5 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/60">
									{section}
								</div>
								{items.map((item) => {
									const flatIdx = filtered.indexOf(item);
									const isSelected = flatIdx === selectedIdx;
									return (
										<button
											key={item.id}
											type="button"
											className={`flex w-full items-center gap-3 rounded-md px-2 py-2 text-sm transition-colors ${isSelected ? "bg-accent text-accent-foreground" : "text-foreground hover:bg-accent/50"}`}
											onClick={() => handleSelect(item.id)}
											onMouseEnter={() => setSelectedIdx(flatIdx)}
										>
											<item.icon className="size-4 shrink-0 text-muted-foreground" />
											<span className="flex-1 text-left">{item.label}</span>
											{item.shortcut && (
												<kbd className="pointer-events-none inline-flex h-5 select-none items-center gap-0.5 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground">
													{item.shortcut.replace(" ", " ")}
												</kbd>
											)}
										</button>
									);
								})}
							</div>
						))}
					</div>

					{/* Footer */}
					<div className="flex items-center gap-3 border-t px-4 py-2 text-[10px] text-muted-foreground/60">
						<span>
							<kbd className="inline-flex h-4 select-none items-center rounded border bg-muted px-1 font-mono text-[10px] font-medium">
								↑↓
							</kbd>{" "}
							navigate
						</span>
						<span>
							<kbd className="inline-flex h-4 select-none items-center rounded border bg-muted px-1 font-mono text-[10px] font-medium">
								enter
							</kbd>{" "}
							select
						</span>
						<span>
							<kbd className="inline-flex h-4 select-none items-center rounded border bg-muted px-1 font-mono text-[10px] font-medium">
								esc
							</kbd>{" "}
							close
						</span>
						<span className="ml-auto">
							<kbd className="inline-flex h-4 select-none items-center rounded border bg-muted px-1 font-mono text-[10px] font-medium">
								⌘K
							</kbd>{" "}
							toggle
						</span>
					</div>
				</div>
			</div>
		</div>
	);
}
