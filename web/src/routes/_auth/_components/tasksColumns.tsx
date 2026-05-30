import { Note, X } from "@phosphor-icons/react";
import { createColumnHelper } from "@tanstack/react-table";
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
import type { Task } from "@/types";

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

interface MakeTasksColumnsOpts {
	onViewLogs: (id: string) => void;
	onCancel: (id: string) => void;
}

export function makeTasksColumns({
	onViewLogs,
	onCancel,
}: MakeTasksColumnsOpts) {
	return [
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
		col.display({
			id: "actions",
			cell: (info) => (
				<div className="flex items-center justify-end gap-1">
					{info.row.original.state !== "queued" && (
						<Button
							variant="ghost"
							size="icon-xs"
							aria-label="View task logs"
							onClick={() => onViewLogs(info.row.original.id)}
						>
							<Note />
						</Button>
					)}
					{["active", "queued"].includes(info.row.original.state) && (
						<AlertDialog>
							<AlertDialogTrigger
								render={
									<Button
										variant="ghost"
										size="icon-xs"
										aria-label="Cancel task"
									>
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
										onClick={() => onCancel(info.row.original.id)}
									>
										Cancel Task
									</AlertDialogAction>
								</AlertDialogFooter>
							</AlertDialogContent>
						</AlertDialog>
					)}
				</div>
			),
		}),
	];
}
