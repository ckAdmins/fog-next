import { Plus } from "@phosphor-icons/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	flexRender,
	getCoreRowModel,
	getSortedRowModel,
	type SortingState,
	useReactTable,
} from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { RouteError } from "@/components/app/route-error";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { TableSkeleton } from "@/components/ui/table-skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useServerEvents } from "@/hooks/useServerEvents";
import { api } from "@/lib/api";
import type { Host, Image, Paginated, Task } from "@/types";
import { TaskDetailCard } from "@/components/tasks/task-detail-card";
import { TaskDialog } from "@/components/tasks/task-dialog";
import { makeTasksColumns } from "@/components/tasks/tasks-columns";

export const Route = createFileRoute("/_auth/tasks")({
	component: TasksPage,
	errorComponent: RouteError,
});

function TaskTable({
	tasks,
	isLoading,
	columns,
}: {
	tasks: Task[];
	isLoading: boolean;
	columns: ReturnType<typeof makeTasksColumns>;
}) {
	const [sorting, setSorting] = useState<SortingState>([]);

	const table = useReactTable({
		data: tasks,
		columns,
		getCoreRowModel: getCoreRowModel(),
		getSortedRowModel: getSortedRowModel(),
		state: { sorting },
		onSortingChange: setSorting,
	});

	if (isLoading) {
		return (
			<Table>
				<TableBody>
					<TableSkeleton columns={8} />
				</TableBody>
			</Table>
		);
	}

	if (tasks.length === 0) {
		return <EmptyState title="No tasks" />;
	}

	return (
		<div className="rounded-lg border">
			<Table>
				<TableHeader>
					{table.getHeaderGroups().map((hg) => (
						<TableRow key={hg.id}>
							{hg.headers.map((h) => (
								<TableHead
									key={h.id}
									aria-sort={
										h.column.getIsSorted() === "asc"
											? "ascending"
											: h.column.getIsSorted() === "desc"
												? "descending"
												: "none"
									}
								>
									{h.column.getCanSort() ? (
										<button
											type="button"
											className="flex items-center gap-1 cursor-pointer select-none"
											onClick={h.column.getToggleSortingHandler()}
										>
											{flexRender(h.column.columnDef.header, h.getContext())}
											{h.column.getIsSorted() === "asc" && (
												<span className="text-xs">▲</span>
											)}
											{h.column.getIsSorted() === "desc" && (
												<span className="text-xs">▼</span>
											)}
										</button>
									) : (
										flexRender(h.column.columnDef.header, h.getContext())
									)}
								</TableHead>
							))}
						</TableRow>
					))}
				</TableHeader>
				<TableBody>
					{table.getRowModel().rows.map((row) => (
						<TableRow key={row.id}>
							{row.getVisibleCells().map((cell) => (
								<TableCell key={cell.id}>
									{flexRender(cell.column.columnDef.cell, cell.getContext())}
								</TableCell>
							))}
						</TableRow>
					))}
				</TableBody>
			</Table>
		</div>
	);
}

function TasksPage() {
	const qc = useQueryClient();
	const [open, setOpen] = useState(false);
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(50);
	const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);

	useServerEvents();

	const tasksQuery = useQuery({
		queryKey: ["tasks", page, pageSize],
		queryFn: () =>
			api.get<Paginated<Task>>(`/tasks?page=${page}&limit=${pageSize}`),
	});

	const hostsQuery = useQuery({
		queryKey: ["hosts", "all"],
		queryFn: () => api.get<Paginated<Host>>("/hosts?page=1&limit=1000"),
	});

	const imagesQuery = useQuery({
		queryKey: ["images", "all"],
		queryFn: () => api.get<Paginated<Image>>("/images?page=1&limit=1000"),
	});

	const cancelMutation = useMutation({
		mutationFn: (id: string) => api.del<void>(`/tasks/${id}`),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["tasks"] });
			toast.success("Task cancelled");
		},
	});

	const columns = useMemo(
		() =>
			makeTasksColumns({
				onViewLogs: (id) => setSelectedTaskId(id),
				onCancel: (id) => cancelMutation.mutate(id),
			}),
		[cancelMutation.mutate],
	);

	const allTasks = tasksQuery.data?.data ?? [];
	const activeTasks = allTasks.filter((t) => t.state === "active");
	const queuedTasks = allTasks.filter((t) => t.state === "queued");
	const historyTasks = allTasks.filter((t) =>
		["complete", "failed", "canceled"].includes(t.state),
	);

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold">Tasks</h1>
					<p className="text-muted-foreground">
						Manage imaging and maintenance tasks
					</p>
				</div>
				<Button onClick={() => setOpen(true)}>
					<Plus data-icon="inline-start" />
					New Task
				</Button>
			</div>

			<Tabs defaultValue="active">
				<TabsList>
					<TabsTrigger value="active">
						Active ({activeTasks.length})
					</TabsTrigger>
					<TabsTrigger value="queued">
						Queued ({queuedTasks.length})
					</TabsTrigger>
					<TabsTrigger value="history">
						History ({historyTasks.length})
					</TabsTrigger>
				</TabsList>
				<TabsContent value="active">
					<TaskTable
						tasks={activeTasks}
						isLoading={tasksQuery.isLoading}
						columns={columns}
					/>
				</TabsContent>
				<TabsContent value="queued">
					<TaskTable
						tasks={queuedTasks}
						isLoading={tasksQuery.isLoading}
						columns={columns}
					/>
				</TabsContent>
				<TabsContent value="history">
					<TaskTable
						tasks={historyTasks}
						isLoading={tasksQuery.isLoading}
						columns={columns}
					/>
				</TabsContent>
			</Tabs>

			<Pagination
				page={page}
				pageSize={pageSize}
				total={tasksQuery.data?.total ?? 0}
				onPageChange={setPage}
				onPageSizeChange={(s) => {
					setPageSize(s);
					setPage(1);
				}}
			/>

			<TaskDialog
				open={open}
				onOpenChange={setOpen}
				hosts={hostsQuery.data?.data ?? []}
				images={imagesQuery.data?.data ?? []}
			/>

			<TaskDetailCard
				taskId={selectedTaskId}
				onClose={() => setSelectedTaskId(null)}
			/>
		</div>
	);
}
