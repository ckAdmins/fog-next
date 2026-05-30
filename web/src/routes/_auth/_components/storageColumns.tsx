import { Pencil, Trash } from "@phosphor-icons/react";
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
import type { StorageGroup, StorageNode } from "@/types";

export function makeStorageGroupColumns({
	onDelete,
}: {
	onDelete: (id: string) => void;
	selectedGroupId?: string;
}) {
	const helper = createColumnHelper<StorageGroup>();
	return [
		helper.accessor("name", {
			header: "",
			cell: (info) => <div className="font-medium">{info.getValue()}</div>,
		}),
		helper.display({
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
									aria-label="Delete storage group"
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
									This will delete "{info.row.original.name}" and all its nodes.
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

export function makeStorageNodeColumns({
	onEdit,
	onRemove,
}: {
	onEdit: (node: StorageNode) => void;
	onRemove: (id: string) => void;
}) {
	const helper = createColumnHelper<StorageNode>();
	return [
		helper.accessor("name", {
			header: "Name",
			cell: (info) => <span className="font-medium">{info.getValue()}</span>,
		}),
		helper.accessor("ip", {
			header: "IP",
			cell: (info) => <span className="font-mono">{info.getValue()}</span>,
		}),
		helper.accessor("path", {
			header: "Path",
			cell: (info) => info.getValue(),
		}),
		helper.accessor("maxClients", {
			header: "Clients",
			cell: (info) => info.getValue(),
		}),
		helper.accessor("isOnline", {
			header: "Status",
			cell: (info) =>
				info.getValue() ? (
					<Badge>Online</Badge>
				) : (
					<Badge variant="secondary">Offline</Badge>
				),
		}),
		helper.display({
			id: "actions",
			header: () => null,
			cell: (info) => (
				<div className="flex justify-end gap-1">
					<Button
						variant="ghost"
						size="icon-xs"
						aria-label="Edit storage node"
						onClick={() => onEdit(info.row.original)}
					>
						<Pencil />
					</Button>
					<AlertDialog>
						<AlertDialogTrigger
							render={
								<Button
									variant="ghost"
									size="icon-xs"
									aria-label="Remove storage node"
								>
									<Trash />
								</Button>
							}
						/>
						<AlertDialogContent>
							<AlertDialogHeader>
								<AlertDialogTitle>Remove node?</AlertDialogTitle>
								<AlertDialogDescription>
									This will remove "{info.row.original.name}" from the storage
									group.
								</AlertDialogDescription>
							</AlertDialogHeader>
							<AlertDialogFooter>
								<AlertDialogCancel>Cancel</AlertDialogCancel>
								<AlertDialogAction
									onClick={() => onRemove(info.row.original.id)}
								>
									Remove
								</AlertDialogAction>
							</AlertDialogFooter>
						</AlertDialogContent>
					</AlertDialog>
				</div>
			),
		}),
	];
}
