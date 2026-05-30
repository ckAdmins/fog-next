import { Pencil, Trash, Upload } from "@phosphor-icons/react";
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
import type { Snapin } from "@/types";

const col = createColumnHelper<Snapin>();

export interface MakeSnapinsColumnsOpts {
	onUpload: (snapin: Snapin) => void;
	onEdit: (snapin: Snapin) => void;
	onDelete: (id: string, name: string) => void;
}

export function makeSnapinsColumns({
	onUpload,
	onEdit,
	onDelete,
}: MakeSnapinsColumnsOpts) {
	return [
		col.accessor("name", {
			header: "Name",
			cell: (info) => <span className="font-medium">{info.getValue()}</span>,
		}),
		col.accessor("description", {
			header: "Description",
			cell: (info) => info.getValue() || "—",
		}),
		col.accessor("runOrder", {
			header: "Run Order",
			cell: (info) => info.getValue(),
		}),
		col.accessor("timeout", {
			header: "Timeout",
			cell: (info) => `${info.getValue()}s`,
		}),
		col.accessor("fileName", {
			header: "File",
			cell: (info) =>
				info.getValue() ? (
					<Badge variant="secondary">{info.getValue()}</Badge>
				) : (
					<span className="text-muted-foreground text-xs">No file</span>
				),
		}),
		col.display({
			id: "actions",
			header: () => null,
			cell: (info) => (
				<div className="flex justify-end gap-1">
					<Button
						variant="ghost"
						size="icon-xs"
						aria-label="Upload file"
						onClick={() => onUpload(info.row.original)}
					>
						<Upload />
					</Button>
					<Button
						variant="ghost"
						size="icon-xs"
						aria-label="Edit snapin"
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
									aria-label="Delete snapin"
								>
									<Trash />
								</Button>
							}
						/>
						<AlertDialogContent>
							<AlertDialogHeader>
								<AlertDialogTitle>Delete snapin?</AlertDialogTitle>
								<AlertDialogDescription>
									This will permanently delete &quot;{info.row.original.name}
									&quot;.
								</AlertDialogDescription>
							</AlertDialogHeader>
							<AlertDialogFooter>
								<AlertDialogCancel>Cancel</AlertDialogCancel>
								<AlertDialogAction
									onClick={() =>
										onDelete(info.row.original.id, info.row.original.name)
									}
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
