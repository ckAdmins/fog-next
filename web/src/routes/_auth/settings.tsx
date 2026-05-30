import { Check, MagnifyingGlass } from "@phosphor-icons/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { RouteError } from "@/components/RouteError";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import type { GlobalSetting, ListResponse } from "@/types";

export const Route = createFileRoute("/_auth/settings")({
	component: SettingsPage,
	errorComponent: RouteError,
});

function SettingRow({ setting }: { setting: GlobalSetting }) {
	const qc = useQueryClient();
	const [value, setValue] = useState(setting.value);
	const [focused, setFocused] = useState(false);

	const mutation = useMutation({
		mutationFn: () =>
			api.put<GlobalSetting>(`/settings/${setting.key}`, { value }),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["settings"] });
			toast.success(`Saved "${setting.key}"`);
		},
	});

	const dirty = value !== setting.value;

	return (
		<div
			className={`group relative flex flex-col gap-1.5 rounded-md border p-3 transition-all duration-150 ${focused ? "border-primary/50 bg-accent/30" : "border-transparent hover:border-border hover:bg-accent/10"}`}
		>
			<div className="flex items-start justify-between gap-3">
				<div className="min-w-0 flex-1">
					<div className="flex items-center gap-2 mb-0.5">
						<code className="text-xs font-mono text-muted-foreground bg-muted px-1.5 py-0.5 rounded-sm">
							{setting.key}
						</code>
						{setting.description && (
							<span className="text-xs text-muted-foreground truncate">
								{setting.description}
							</span>
						)}
					</div>
					<Input
						id={`setting-${setting.key}`}
						value={value}
						onChange={(e) => setValue(e.target.value)}
						onFocus={() => setFocused(true)}
						onBlur={() => setFocused(false)}
						className="h-8 text-sm font-mono"
						onKeyDown={(e) => {
							if (e.key === "Enter" && dirty && !mutation.isPending) {
								mutation.mutate();
							}
							if (e.key === "Escape") {
								setValue(setting.value);
								(e.target as HTMLInputElement).blur();
							}
						}}
					/>
				</div>
				{dirty && (
					<Button
						size="sm"
						variant="default"
						className="h-8 shrink-0"
						disabled={mutation.isPending}
						onClick={() => mutation.mutate()}
					>
						{mutation.isPending ? (
							<span className="animate-pulse">...</span>
						) : (
							<>
								<Check />
								<span className="hidden sm:inline ml-1">Save</span>
							</>
						)}
					</Button>
				)}
			</div>
			{dirty && (
				<div className="flex items-center gap-2">
					<Badge variant="secondary" className="text-[10px] h-4 px-1">
						modified
					</Badge>
					<span className="text-[10px] text-muted-foreground">
						Enter to save · Esc to cancel
					</span>
				</div>
			)}
		</div>
	);
}

function SettingsPage() {
	const [search, setSearch] = useState("");

	const { data, isLoading, isError } = useQuery({
		queryKey: ["settings"],
		queryFn: async () => {
			const resp = await api.get<ListResponse<GlobalSetting>>("/settings");
			return resp.data;
		},
	});

	const settings = data ?? [];

	const filtered = useMemo(() => {
		if (!search.trim()) return settings;
		const q = search.toLowerCase();
		return settings.filter(
			(s) =>
				s.key.toLowerCase().includes(q) ||
				(s.description || "").toLowerCase().includes(q) ||
				(s.category || "").toLowerCase().includes(q),
		);
	}, [settings, search]);

	const grouped = useMemo(() => {
		return filtered.reduce<Record<string, GlobalSetting[]>>((acc, s) => {
			const cat = s.category || "General";
			if (!acc[cat]) acc[cat] = [];
			acc[cat].push(s);
			return acc;
		}, {});
	}, [filtered]);

	if (isLoading) {
		return (
			<div className="flex flex-col gap-6">
				<div>
					<div className="h-8 w-48 bg-muted rounded mb-2 animate-pulse" />
					<div className="h-4 w-64 bg-muted rounded animate-pulse" />
				</div>
				{[1, 2, 3].map((i) => (
					<Card key={i}>
						<CardHeader>
							<Skeleton className="h-6 w-32" />
						</CardHeader>
						<CardContent className="flex flex-col gap-3">
							{[1, 2].map((j) => (
								<Skeleton key={j} className="h-16 w-full" />
							))}
						</CardContent>
					</Card>
				))}
			</div>
		);
	}

	if (isError) {
		return (
			<div className="flex flex-col items-center justify-center gap-4 py-16">
				<p className="text-destructive font-medium">Failed to load settings</p>
				<Button variant="outline" onClick={() => window.location.reload()}>
					Retry
				</Button>
			</div>
		);
	}

	const categories = Object.entries(grouped);
	const totalSettings = settings.length;

	return (
		<div className="flex flex-col gap-6 animate-in fade-in slide-in-from-top-2 duration-300">
			<div className="flex flex-col gap-1">
				<h1 className="text-2xl font-heading font-bold tracking-tight">
					Settings
				</h1>
				<p className="text-sm text-muted-foreground">
					{totalSettings} global configuration parameter
					{totalSettings !== 1 ? "s" : ""}
				</p>
			</div>

			<div className="relative">
				<MagnifyingGlass className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
				<Input
					placeholder="Filter settings by key, description, or category..."
					value={search}
					onChange={(e) => setSearch(e.target.value)}
					className="pl-9 h-9 text-sm"
				/>
			</div>

			{search && filtered.length === 0 && (
				<p className="text-sm text-muted-foreground text-center py-8">
					No settings match "{search}"
				</p>
			)}

			<div className="flex flex-col gap-4">
				{categories.map(([category, items], catIdx) => (
					<Card
						key={category}
						className="animate-in fade-in slide-in-from-top-2 duration-300 border-border/50"
						style={{ animationDelay: `${catIdx * 50}ms` }}
					>
						<CardHeader className="pb-0">
							<div className="flex items-center gap-3">
								<div className="size-2 rounded-full bg-primary/60" />
								<CardTitle className="text-base font-heading font-semibold tracking-tight">
									{category}
								</CardTitle>
								<Badge
									variant="secondary"
									className="text-[10px] font-mono h-4.5"
								>
									{items.length}
								</Badge>
							</div>
						</CardHeader>
						<CardContent className="flex flex-col gap-2 pt-4">
							{items.map((s) => (
								<SettingRow key={s.key} setting={s} />
							))}
						</CardContent>
					</Card>
				))}
			</div>
		</div>
	);
}
