import { Cpu, HardDrive, ListChecks } from "@phosphor-icons/react";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { RouteError } from "@/components/app/route-error";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import type { Host, Image, Paginated, Task } from "@/types";

export const Route = createFileRoute("/_auth/dashboard")({
	component: DashboardPage,
	errorComponent: RouteError,
});

function StatCard({
	title,
	description,
	value,
	icon: Icon,
	isLoading,
	index,
}: {
	title: string;
	description: string;
	value: number | undefined;
	icon: React.ElementType;
	isLoading: boolean;
	index: number;
}) {
	return (
		<Card
			className="card-stat animate-card-enter border-border/60"
			style={{ animationDelay: `${index * 80}ms` }}
		>
			<CardHeader className="flex flex-row items-center justify-between pb-2">
				<CardTitle className="text-sm font-heading font-medium tracking-tight">
					{title}
				</CardTitle>
				<div className="rounded-md bg-primary/10 p-1.5">
					<Icon className="size-3.5 text-primary" weight="duotone" />
				</div>
			</CardHeader>
			<CardContent>
				{isLoading ? (
					<Skeleton className="h-9 w-20" />
				) : (
					<div className="text-3xl font-heading font-bold tabular-nums tracking-tight">
						{value ?? 0}
					</div>
				)}
				<CardDescription className="text-xs mt-0.5">
					{description}
				</CardDescription>
			</CardContent>
		</Card>
	);
}

function DashboardPage() {
	const hostsQuery = useQuery({
		queryKey: ["hosts", 1, 1],
		queryFn: () => api.get<Paginated<Host>>("/hosts?page=1&limit=1"),
	});

	const imagesQuery = useQuery({
		queryKey: ["images", 1, 1],
		queryFn: () => api.get<Paginated<Image>>("/images?page=1&limit=1"),
	});

	const activeTasksQuery = useQuery({
		queryKey: ["tasks", "active"],
		queryFn: () => api.get<Paginated<Task>>("/tasks?state=active&limit=1"),
	});

	return (
		<div className="flex flex-col gap-6 animate-page-enter">
			<div>
				<h1 className="text-2xl font-heading font-bold">Dashboard</h1>
				<p className="text-sm text-muted-foreground mt-0.5">
					Overview of your FOG environment
				</p>
			</div>

			<div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
				<StatCard
					title="Total Hosts"
					description="Registered managed hosts"
					value={hostsQuery.data?.total}
					icon={Cpu}
					isLoading={hostsQuery.isLoading}
					index={0}
				/>
				<StatCard
					title="Disk Images"
					description="Available deployment images"
					value={imagesQuery.data?.total}
					icon={HardDrive}
					isLoading={imagesQuery.isLoading}
					index={1}
				/>
				<StatCard
					title="Active Tasks"
					description="Currently running tasks"
					value={activeTasksQuery.data?.total}
					icon={ListChecks}
					isLoading={activeTasksQuery.isLoading}
					index={2}
				/>
			</div>

			{/* Keyboard shortcut hint */}
			<div className="mt-2">
				<p className="text-[11px] text-muted-foreground/60">
					Press{" "}
					<kbd className="inline-flex h-4.5 select-none items-center rounded border bg-muted/50 px-1 font-mono text-[10px] font-medium text-muted-foreground">
						⌘K
					</kbd>{" "}
					to open command palette ·{" "}
					<kbd className="inline-flex h-4.5 select-none items-center rounded border bg-muted/50 px-1 font-mono text-[10px] font-medium text-muted-foreground">
						g d
					</kbd>{" "}
					for quick nav
				</p>
			</div>
		</div>
	);
}
