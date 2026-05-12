import { AgentLogViewer } from "@/components/agent-log-viewer";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
	AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import {
	Field,
	FieldContent,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "@/components/ui/field";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useServerEvents } from "@/hooks/useServerEvents";
import { api } from "@/lib/api";
import type { Host, Image, Paginated, Task } from "@/types";
import { Note, Plus, X } from "@phosphor-icons/react";
import { useForm } from "@tanstack/react-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	createColumnHelper,
	flexRender,
	getCoreRowModel,
	useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";
import { toast } from "sonner";
import * as z from "zod";

export const Route = createFileRoute("/_auth/tasks")({
	component: TasksPage,
});

const TASK_TYPES = [
	"deploy",
	"capture",
	"debug_deploy",
	"debug_capture",
	"multicast",
	"wipe",
	"memtest",
	"disk_test",
] as const;

type TaskType = (typeof TASK_TYPES)[number];

const IMAGE_TASK_TYPES: TaskType[] = [
	"deploy",
	"capture",
	"debug_deploy",
	"debug_capture",
	"multicast",
];

const taskSchema = z.object({
	type: z.enum(TASK_TYPES, { message: "Task type is required" }),
	hostId: z.string().min(1, "Host is required"),
	imageId: z.string(),
	isShutdown: z.boolean(),
	isForced: z.boolean(),
});

const col = createColumnHelper<Task>();

function taskStateColor(state: string) {
	switch (state) {
		case "active":
			return "default";
		case "queued":
			return "secondary";
		case "complete":
			return "outline";
		case "failed":
			return "destructive";
		case "canceled":
			return "secondary";
		default:
			return "secondary";
	}
}

const columns = [
	col.accessor("type", { header: "Type" }),
	col.accessor("state", {
		header: "State",
		cell: (info) => (
			<Badge
				variant={
					taskStateColor(info.getValue()) as
						| "default"
						| "secondary"
						| "outline"
						| "destructive"
				}
			>
				{info.getValue()}
			</Badge>
		),
	}),
	col.accessor("hostId", { header: "Host ID" }),
	col.accessor("percentComplete", {
		header: "Progress",
		cell: (info) => `${info.getValue()}%`,
	}),
	col.accessor("createdAt", {
		header: "Created",
		cell: (info) => new Date(info.getValue()).toLocaleString(),
	}),
];

function TaskTable({
	tasks,
	isLoading,
	onCancel,
	onViewLogs,
}: {
	tasks: Task[];
	isLoading: boolean;
	onCancel: (id: string) => void;
	onViewLogs: (id: string) => void;
}) {
	const table = useReactTable({
		data: tasks,
		columns,
		getCoreRowModel: getCoreRowModel(),
	});

	if (isLoading) {
		return (
			<div className="py-8 text-center text-muted-foreground">Loading…</div>
		);
	}

	if (tasks.length === 0) {
		return (
			<div className="py-8 text-center text-muted-foreground">No tasks</div>
		);
	}

	return (
		<div className="rounded-lg border">
			<Table>
				<TableHeader>
					{table.getHeaderGroups().map((hg) => (
						<TableRow key={hg.id}>
							{hg.headers.map((h) => (
								<TableHead key={h.id}>
									{flexRender(h.column.columnDef.header, h.getContext())}
								</TableHead>
							))}
							<TableHead />
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
							<TableCell className="text-right">
								<div className="flex items-center justify-end gap-1">
									{row.original.state !== "queued" && (
										<Button
											variant="ghost"
											size="icon-xs"
											onClick={() => onViewLogs(row.original.id)}
										>
											<Note />
										</Button>
									)}
									{["active", "queued"].includes(row.original.state) && (
										<AlertDialog>
											<AlertDialogTrigger
												render={
													<Button variant="ghost" size="icon-xs">
														<X />
													</Button>
												}
											/>
											<AlertDialogContent>
												<AlertDialogHeader>
													<AlertDialogTitle>Cancel task?</AlertDialogTitle>
													<AlertDialogDescription>
														This will cancel the running task.
													</AlertDialogDescription>
												</AlertDialogHeader>
												<AlertDialogFooter>
													<AlertDialogCancel>Keep</AlertDialogCancel>
													<AlertDialogAction
														onClick={() => onCancel(row.original.id)}
													>
														Cancel Task
													</AlertDialogAction>
												</AlertDialogFooter>
											</AlertDialogContent>
										</AlertDialog>
									)}
								</div>
							</TableCell>
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
	const [logDialogTaskId, setLogDialogTaskId] = useState<string | null>(null);

	useServerEvents();

	const tasksQuery = useQuery({
		queryKey: ["tasks", page],
		queryFn: () => api.get<Paginated<Task>>(`/tasks?page=${page}&limit=50`),
	});

	const hostsQuery = useQuery({
		queryKey: ["hosts", "all"],
		queryFn: () => api.get<Paginated<Host>>("/hosts?page=1&limit=1000"),
	});

	const imagesQuery = useQuery({
		queryKey: ["images", "all"],
		queryFn: () => api.get<Paginated<Image>>("/images?page=1&limit=1000"),
	});

	const createMutation = useMutation({
		mutationFn: (values: z.infer<typeof taskSchema>) =>
			api.post<Task>("/tasks", {
				...values,
				imageId: values.imageId || undefined,
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["tasks"] });
			setOpen(false);
			toast.success("Task created");
		},
		onError: (err) =>
			toast.error(err instanceof Error ? err.message : "Failed"),
	});

	const cancelMutation = useMutation({
		mutationFn: (id: string) => api.del<void>(`/tasks/${id}`),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["tasks"] });
			toast.success("Task cancelled");
		},
		onError: (err) =>
			toast.error(err instanceof Error ? err.message : "Failed"),
	});

	const form = useForm({
		defaultValues: {
			type: "deploy" as TaskType,
			hostId: "",
			imageId: "",
			isShutdown: false,
			isForced: false,
		},
		validators: { onSubmit: taskSchema },
		onSubmit: ({ value }) => createMutation.mutate(value),
	});

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
						onCancel={(id) => cancelMutation.mutate(id)}
						onViewLogs={(id) => setLogDialogTaskId(id)}
					/>
				</TabsContent>
				<TabsContent value="queued">
					<TaskTable
						tasks={queuedTasks}
						isLoading={tasksQuery.isLoading}
						onCancel={(id) => cancelMutation.mutate(id)}
						onViewLogs={(id) => setLogDialogTaskId(id)}
					/>
				</TabsContent>
				<TabsContent value="history">
					<TaskTable
						tasks={historyTasks}
						isLoading={tasksQuery.isLoading}
						onCancel={() => {}}
						onViewLogs={(id) => setLogDialogTaskId(id)}
					/>
				</TabsContent>
			</Tabs>

			{tasksQuery.data && tasksQuery.data.total > 50 && (
				<div className="flex items-center justify-end gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={page <= 1}
						onClick={() => setPage((p) => p - 1)}
					>
						Previous
					</Button>
					<span className="text-sm text-muted-foreground">Page {page}</span>
					<Button
						variant="outline"
						size="sm"
						disabled={page >= Math.ceil(tasksQuery.data.total / 50)}
						onClick={() => setPage((p) => p + 1)}
					>
						Next
					</Button>
				</div>
			)}

			<Dialog open={open} onOpenChange={setOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>New Task</DialogTitle>
					</DialogHeader>
					<form
						onSubmit={(e) => {
							e.preventDefault();
							void form.handleSubmit();
						}}
					>
						<FieldGroup>
							<form.Field name="type">
								{(field) => {
									const isInvalid =
										field.state.meta.isTouched && !field.state.meta.isValid;
									return (
										<Field data-invalid={isInvalid}>
											<FieldLabel>Task Type</FieldLabel>
											<Select
												value={field.state.value}
												onValueChange={(v) => field.handleChange(v as TaskType)}
											>
												<SelectTrigger aria-invalid={isInvalid}>
													<SelectValue placeholder="Select type" />
												</SelectTrigger>
												<SelectContent>
													{TASK_TYPES.map((t) => (
														<SelectItem key={t} value={t}>
															{t}
														</SelectItem>
													))}
												</SelectContent>
											</Select>
											{isInvalid && (
												<FieldError errors={field.state.meta.errors} />
											)}
										</Field>
									);
								}}
							</form.Field>

							<form.Field name="hostId">
								{(field) => {
									const isInvalid =
										field.state.meta.isTouched && !field.state.meta.isValid;
									return (
										<Field data-invalid={isInvalid}>
											<FieldLabel>Host</FieldLabel>
											<Select
												value={field.state.value}
												onValueChange={(v) =>
													v !== null && field.handleChange(v)
												}
											>
												<SelectTrigger aria-invalid={isInvalid}>
													<SelectValue placeholder="Select host" />
												</SelectTrigger>
												<SelectContent>
													{hostsQuery.data?.data.map((h) => (
														<SelectItem key={h.id} value={h.id}>
															{h.name}
														</SelectItem>
													))}
												</SelectContent>
											</Select>
											{isInvalid && (
												<FieldError errors={field.state.meta.errors} />
											)}
										</Field>
									);
								}}
							</form.Field>

							<form.Subscribe selector={(s) => s.values.type}>
								{(taskType) =>
									IMAGE_TASK_TYPES.includes(taskType as TaskType) ? (
										<form.Field name="imageId">
											{(field) => (
												<Field>
													<FieldLabel>Image</FieldLabel>
													<Select
														value={field.state.value}
														onValueChange={(v) =>
															v !== null && field.handleChange(v)
														}
													>
														<SelectTrigger>
															<SelectValue placeholder="Select image" />
														</SelectTrigger>
														<SelectContent>
															{imagesQuery.data?.data.map((img) => (
																<SelectItem key={img.id} value={img.id}>
																	{img.name}
																</SelectItem>
															))}
														</SelectContent>
													</Select>
												</Field>
											)}
										</form.Field>
									) : null
								}
							</form.Subscribe>

							{(["isShutdown", "isForced"] as const).map((fieldName) => {
								const labels: Record<string, string> = {
									isShutdown: "Shutdown after task",
									isForced: "Force (skip queue)",
								};
								return (
									<form.Field key={fieldName} name={fieldName}>
										{(field) => (
											<Field orientation="horizontal">
												<FieldContent>
													<FieldLabel htmlFor={field.name}>
														{labels[fieldName]}
													</FieldLabel>
												</FieldContent>
												<Switch
													id={field.name}
													checked={field.state.value}
													onCheckedChange={field.handleChange}
												/>
											</Field>
										)}
									</form.Field>
								);
							})}
						</FieldGroup>
						<DialogFooter className="mt-4">
							<Button
								type="button"
								variant="outline"
								onClick={() => setOpen(false)}
							>
								Cancel
							</Button>
							<form.Subscribe selector={(s) => s.isSubmitting}>
								{(isSubmitting) => (
									<Button type="submit" disabled={isSubmitting}>
										{isSubmitting ? "Creating…" : "Create Task"}
									</Button>
								)}
							</form.Subscribe>
						</DialogFooter>
					</form>
				</DialogContent>
			</Dialog>

			<Dialog
				open={logDialogTaskId !== null}
				onOpenChange={(open) => {
					if (!open) setLogDialogTaskId(null);
				}}
			>
				<DialogContent className="max-w-5xl">
					<DialogHeader>
						<DialogTitle>Task Logs</DialogTitle>
					</DialogHeader>
					{logDialogTaskId && <AgentLogViewer taskId={logDialogTaskId} />}
				</DialogContent>
			</Dialog>
		</div>
	);
}
