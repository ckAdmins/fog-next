import { Copy, Pencil, Trash } from "@phosphor-icons/react";
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
import type { User } from "@/types";

const col = createColumnHelper<User>();

interface MakeUsersColumnsOpts {
	onCopyToken: (id: string) => void;
	onEdit: (user: User) => void;
	onDelete: (id: string) => void;
}

export function makeUsersColumns({
	onCopyToken,
	onEdit,
	onDelete,
}: MakeUsersColumnsOpts) {
	return [
		col.accessor("username", {
			header: "Username",
			cell: (info) => <span className="font-medium">{info.getValue()}</span>,
		}),
		col.accessor("role", {
			header: "Role",
			cell: (info) => (
				<Badge variant={info.getValue() === "admin" ? "default" : "secondary"}>
					{info.getValue()}
				</Badge>
			),
		}),
		col.accessor("lastLogin", {
			header: "Last Login",
			cell: (info) => {
				const v = info.getValue();
				return (
					<span className="text-muted-foreground text-sm">
						{v ? new Date(v).toLocaleString() : "Never"}
					</span>
				);
			},
		}),
		col.display({
			id: "actions",
			header: () => null,
			cell: (info) => (
				<div className="flex justify-end gap-1">
					<Button
						variant="ghost"
						size="icon-xs"
						aria-label="Regenerate API token"
						onClick={() => onCopyToken(info.row.original.id)}
					>
						<Copy />
					</Button>
					<Button
						variant="ghost"
						size="icon-xs"
						aria-label="Edit user"
						onClick={() => onEdit(info.row.original)}
					>
						<Pencil />
					</Button>
					<AlertDialog>
						<AlertDialogTrigger
							render={
								<Button variant="ghost" size="icon-xs" aria-label="Delete user">
									<Trash />
								</Button>
							}
						/>
						<AlertDialogContent>
							<AlertDialogHeader>
								<AlertDialogTitle>Delete user?</AlertDialogTitle>
								<AlertDialogDescription>
									This will permanently delete &quot;
									{info.row.original.username}&quot;.
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
