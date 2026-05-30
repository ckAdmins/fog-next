import { Trash } from "@phosphor-icons/react";
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
import { Button } from "@/components/ui/button";
import type { Group, GroupMember, Host } from "@/types";

interface MakeGroupListColumnsOpts {
	onDelete: (id: string) => void;
}

export function makeGroupListColumns({ onDelete }: MakeGroupListColumnsOpts) {
	const col = createColumnHelper<Group>();

	return [
		col.accessor("name", {
			header: "",
			cell: (info) => (
				<div>
					<div className="font-medium">{info.getValue()}</div>
					{info.row.original.description && (
						<div className="text-xs text-muted-foreground">
							{info.row.original.description}
						</div>
					)}
				</div>
			),
		}),
		col.display({
			id: "actions",
			header: () => null,
			cell: (info) => (
				<div className="text-right">
					<AlertDialog>
						<AlertDialogTrigger
							render={
								<Button
									variant="ghost"
									size="icon-xs"
									aria-label="Delete group"
									onClick={(e) => e.stopPropagation()}
								>
									<Trash />
								</Button>
							}
						/>
						<AlertDialogContent>
							<AlertDialogHeader>
								<AlertDialogTitle>Delete group?</AlertDialogTitle>
								<AlertDialogDescription>
									This will delete "{info.row.original.name}".
								</AlertDialogDescription>
							</AlertDialogHeader>
							<AlertDialogFooter>
								<AlertDialogCancel>Cancel</AlertDialogCancel>
								<AlertDialogAction
									onClick={() => onDelete(info.row.original.id)}
								>
									Delete
								</AlertDialogAction>
							</AlertDialogFooter>
						</AlertDialogContent>
					</AlertDialog>
				</div>
			),
		}),
	];
}

interface MakeGroupMembersColumnsOpts {
	allHosts: Host[];
	onRemove: (hostId: string) => void;
}

export function makeGroupMembersColumns({
	allHosts,
	onRemove,
}: MakeGroupMembersColumnsOpts) {
	const col = createColumnHelper<GroupMember>();

	return [
		col.display({
			id: "host",
			header: "Host",
			cell: (info) => {
				const host = allHosts.find((h) => h.id === info.row.original.hostId);
				return host?.name ?? info.row.original.hostId;
			},
		}),
		col.display({
			id: "actions",
			header: () => null,
			cell: (info) => (
				<div className="text-right">
					<Button
						variant="ghost"
						size="icon-xs"
						aria-label="Remove member"
						onClick={() => onRemove(info.row.original.hostId)}
					>
						<Trash />
					</Button>
				</div>
			),
		}),
	];
}
