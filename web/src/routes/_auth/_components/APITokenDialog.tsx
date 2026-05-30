import { Copy } from "@phosphor-icons/react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";

export interface APITokenDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	token: string | null;
}

export function APITokenDialog({
	open,
	onOpenChange,
	token,
}: APITokenDialogProps) {
	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>API Token</DialogTitle>
				</DialogHeader>
				<p className="text-muted-foreground text-sm">
					Copy this token now — it won&apos;t be shown again.
				</p>
				<div className="bg-muted flex items-center gap-2 break-all rounded-md border p-3 font-mono text-sm">
					{token}
				</div>
				<DialogFooter>
					<Button
						variant="outline"
						onClick={() => {
							if (token) void navigator.clipboard.writeText(token);
							toast.success("Copied");
						}}
					>
						<Copy data-icon="inline-start" />
						Copy
					</Button>
					<Button onClick={() => onOpenChange(false)}>Done</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
