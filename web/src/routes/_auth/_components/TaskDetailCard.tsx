import { AgentLogViewer } from "@/components/agent-log-viewer";
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";

export interface TaskDetailCardProps {
	taskId: string | null;
	onClose: () => void;
}

export function TaskDetailCard({ taskId, onClose }: TaskDetailCardProps) {
	return (
		<Dialog
			open={taskId !== null}
			onOpenChange={(open) => {
				if (!open) onClose();
			}}
		>
			<DialogContent className="max-w-dvw">
				<DialogHeader>
					<DialogTitle>Task Logs</DialogTitle>
				</DialogHeader>
				{taskId && <AgentLogViewer taskId={taskId} />}
			</DialogContent>
		</Dialog>
	);
}
