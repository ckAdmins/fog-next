import { useQuery } from "@tanstack/react-query";
import { AgentLogViewer } from "@/components/app/agent-log-viewer";
import { Badge } from "@/components/ui/badge";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import type { Task } from "@/types";

export function TaskCard({ hostId }: { hostId: string }) {
	const taskQuery = useQuery({
		queryKey: ["host-task", hostId],
		queryFn: () => api.get<Task | null>(`/hosts/${hostId}/task`),
	});

	return (
		<>
			<Card>
				<CardHeader>
					<CardTitle>Current Task</CardTitle>
				</CardHeader>
				<CardContent>
					{taskQuery.isLoading ? (
						<Skeleton className="h-16 w-full" />
					) : !taskQuery.data ? (
						<p className="text-muted-foreground">No active task</p>
					) : (
						<div className="flex flex-col gap-2">
							<div className="flex items-center justify-between">
								<span className="font-medium">{taskQuery.data.type}</span>
								<Badge>{taskQuery.data.state}</Badge>
							</div>
							<div className="text-sm text-muted-foreground">
								Progress: {taskQuery.data.percentComplete}%
							</div>
						</div>
					)}
				</CardContent>
			</Card>

			<Card>
				<CardHeader>
					<CardTitle>Agent Logs</CardTitle>
					<CardDescription>
						Live log output forwarded from fos-agent during imaging tasks
					</CardDescription>
				</CardHeader>
				<CardContent>
					{taskQuery.isLoading ? (
						<p className="text-muted-foreground text-sm">Loading task…</p>
					) : !taskQuery.data ? (
						<p className="text-muted-foreground text-sm">
							No task found. Logs appear here when an imaging task runs.
						</p>
					) : (
						<AgentLogViewer taskId={taskQuery.data.id} />
					)}
				</CardContent>
			</Card>
		</>
	);
}
