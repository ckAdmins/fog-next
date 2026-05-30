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
import type { Image } from "@/types";

const col = createColumnHelper<Image>();

interface MakeImagesColumnsOpts {
	onEdit: (image: Image) => void;
	onDelete: (id: string) => void;
}

export function makeImagesColumns({ onEdit, onDelete }: MakeImagesColumnsOpts) {
	return [
		col.accessor("name", { header: "Name" }),
		col.accessor("description", { header: "Description" }),
		col.accessor("path", { header: "Path" }),
		col.accessor("sizeBytes", {
			header: "Size",
			cell: (info) => {
				const bytes = info.getValue();
				if (!bytes) return "—";
				return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
			},
		}),
		col.accessor("isEnabled", {
			header: "Status",
			cell: (info) =>
				info.getValue() ? (
					<Badge variant="default">Enabled</Badge>
				) : (
					<Badge variant="secondary">Disabled</Badge>
				),
		}),
		col.display({
			id: "actions",
			cell: ({ row }) => (
				<div className="flex justify-end gap-1">
					<Button
						variant="ghost"
						size="icon-xs"
						aria-label="Edit image"
						onClick={() => onEdit(row.original)}
					>
						<Pencil />
					</Button>
					<AlertDialog>
						<AlertDialogTrigger
							render={
								<Button
									variant="ghost"
									size="icon-xs"
									aria-label="Delete image"
								>
									<Trash />
								</Button>
							}
						/>
						<AlertDialogContent>
							<AlertDialogHeader>
								<AlertDialogTitle>Delete image?</AlertDialogTitle>
								<AlertDialogDescription>
									This will permanently delete "{row.original.name}".
								</AlertDialogDescription>
							</AlertDialogHeader>
							<AlertDialogFooter>
								<AlertDialogCancel>Cancel</AlertDialogCancel>
								<AlertDialogAction onClick={() => onDelete(row.original.id)}>
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
