import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import type { AgentLog } from "@/types";

interface Props {
	taskId: string;
}

type ListResponse = { data: AgentLog[] };

const LEVEL_VARIANT: Record<
	string,
	"default" | "secondary" | "outline" | "destructive"
> = {
	DEBUG: "secondary",
	INFO: "default",
	WARN: "outline",
	ERROR: "destructive",
};

export function AgentLogViewer({ taskId }: Props) {
	const bottomRef = useRef<HTMLDivElement | null>(null);

	const { data, isLoading } = useQuery({
		queryKey: ["task-logs", taskId],
		queryFn: () => api.get<ListResponse>(`/tasks/${taskId}/logs`),
		refetchInterval: 3000,
	});

	const logs = data?.data ?? [];

	// Auto-scroll to bottom whenever the log list grows.
	// biome-ignore lint/correctness/useExhaustiveDependencies: logs.length is used as a scroll trigger
	useEffect(() => {
		bottomRef.current?.scrollIntoView({ behavior: "smooth" });
	}, [logs.length]);

	if (isLoading) {
		return <Skeleton className="h-48 w-full" />;
	}

	if (logs.length === 0) {
		return (
			<EmptyState
				title="No log entries yet"
				description="Logs appear here while the agent is running."
			/>
		);
	}

	return (
		<ScrollArea className="h-[480px] w-full rounded-md border bg-black/90">
			<div className="p-3 font-mono text-xs text-green-300 space-y-0.5">
				{logs.map((entry) => (
					<div key={entry.id} className="flex gap-2 items-baseline">
						<span className="shrink-0 text-muted-foreground/60 tabular-nums">
							{new Date(entry.loggedAt)
								.toISOString()
								.replace("T", " ")
								.replace("Z", "")}
						</span>
						<Badge
							className="shrink-0 text-[10px] px-1 py-0 font-mono"
							variant={LEVEL_VARIANT[entry.level.toUpperCase()] ?? "secondary"}
						>
							{entry.level.toUpperCase()}
						</Badge>
						<span className="break-all">{entry.message}</span>
						{entry.attrs && Object.keys(entry.attrs).length > 0 && (
							<span className="text-muted-foreground/50 break-all">
								{Object.entries(entry.attrs)
									.map(([k, v]) => `${k}=${String(v)}`)
									.join(" ")}
							</span>
						)}
					</div>
				))}
				<div ref={bottomRef} />
			</div>
		</ScrollArea>
	);
}
