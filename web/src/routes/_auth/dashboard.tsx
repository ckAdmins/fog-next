import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { RouteError } from "@/components/app/route-error";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import type { Host, Image, Paginated, Task } from "@/types";

export const Route = createFileRoute("/_auth/dashboard")({
	component: DashboardPage,
	errorComponent: RouteError,
});

/* ─── Instrument stat card ───────────────────────────────────── */

function StatCard({
	title,
	value,
	secondary,
	barColor,
	index,
}: {
	title: string;
	value: number | undefined;
	secondary: string;
	barColor: string;
	index: number;
}) {
	return (
		<div
			className="stat-card animate-card-enter"
			style={{ animationDelay: `${index * 100}ms` }}
		>
			<div
				className="stat-card-bar bg-border/40 transition-colors duration-300"
				style={{ backgroundColor: `oklch(from ${barColor} l c h / 0.25)` }}
			/>
			<div className="flex flex-1 flex-col justify-between gap-3 px-4 py-4">
				<div className="flex flex-col gap-0.5">
					<span className="stat-card-value">
						{value ?? (
							<Skeleton className="inline-block h-8 w-16 align-middle" />
						)}
					</span>
					<span className="stat-card-label">{title}</span>
				</div>
				<span className="stat-card-secondary">{secondary}</span>
			</div>
		</div>
	);
}

/* ─── Recent activity feed ─────────────────────────────────────── */

function timeAgo(dateStr: string): string {
	const diff = Date.now() - new Date(dateStr).getTime();
	const mins = Math.floor(diff / 60000);
	if (mins < 1) return "just now";
	if (mins === 1) return "1m ago";
	if (mins < 60) return `${mins}m ago`;
	const hrs = Math.floor(mins / 60);
	if (hrs === 1) return "1h ago";
	return `${hrs}h ago`;
}

const stateDot: Record<string, string> = {
	active: "bg-green",
	pending: "bg-amber",
	queued: "bg-amber",
	completed: "bg-green",
	failed: "bg-destructive",
	cancelled: "bg-muted-foreground",
};

const stateLabel: Record<string, string> = {
	active: "Running",
	pending: "Pending",
	queued: "Queued",
	completed: "Done",
	failed: "Failed",
	cancelled: "Cancelled",
};

function ActivityFeed({ tasks }: { tasks: Task[] | undefined }) {
	if (!tasks || tasks.length === 0) {
		return (
			<div className="mt-1 text-sm text-muted-foreground">
				No recent activity
			</div>
		);
	}

	return (
		<div className="mt-1">
			{tasks.slice(0, 8).map((task) => (
				<div key={task.id} className="activity-item">
					<div
						className={`activity-dot ${stateDot[task.state] || "bg-muted-foreground"} ${task.state === "active" ? "animate-pulse-dot" : ""}`}
					/>
					<span className="min-w-0 flex-1 truncate font-medium text-foreground">
						{task.name}
					</span>
					<span className="hidden text-xs text-muted-foreground sm:inline">
						{stateLabel[task.state] || task.state}
					</span>
					<span className="shrink-0 text-xs tabular-nums text-muted-foreground/60">
						{timeAgo(task.createdAt)}
					</span>
				</div>
			))}
		</div>
	);
}

/* ─── Dashboard page ──────────────────────────────────────────── */

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

	const recentTasksQuery = useQuery({
		queryKey: ["tasks", "recent", 8],
		queryFn: () => api.get<Paginated<Task>>("/tasks?page=1&limit=8"),
	});

	const totalHosts = hostsQuery.data?.total ?? 0;
	const totalImages = imagesQuery.data?.total ?? 0;
	const activeCount = activeTasksQuery.data?.total ?? 0;

	return (
		<div className="flex flex-col gap-6 animate-page-enter">
			<div>
				<h1 className="text-xl font-heading font-semibold tracking-tight">
					Operations
				</h1>
				<p className="mt-0.5 text-xs tracking-[0.12em] text-muted-foreground">
					{totalHosts} host{totalHosts !== 1 ? "s" : ""} registered
					{" \u00b7 "}
					{activeCount} active task{activeCount !== 1 ? "s" : ""}
				</p>
			</div>

			<div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
				<StatCard
					title="HOSTS"
					value={totalHosts}
					secondary={hostsQuery.isLoading ? "" : `${totalHosts} registered`}
					barColor="oklch(0.72 0.09 210)"
					index={0}
				/>
				<StatCard
					title="IMAGES"
					value={totalImages}
					secondary={imagesQuery.isLoading ? "" : `${totalImages} available`}
					barColor="oklch(0.65 0.15 160)"
					index={1}
				/>
				<StatCard
					title="RUNNING"
					value={activeCount}
					secondary={
						activeTasksQuery.isLoading
							? ""
							: activeCount === 0
								? "No active tasks"
								: `${activeCount} task${activeCount !== 1 ? "s" : ""} in progress`
					}
					barColor="oklch(0.75 0.14 85)"
					index={2}
				/>
			</div>

			<div className="mt-2">
				<h2 className="text-xs font-semibold tracking-[0.12em] text-muted-foreground">
					RECENT ACTIVITY
				</h2>
				<ActivityFeed tasks={recentTasksQuery.data?.data} />
			</div>
		</div>
	);
}
