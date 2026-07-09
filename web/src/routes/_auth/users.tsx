import { Plus } from "@phosphor-icons/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	flexRender,
	getCoreRowModel,
	useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";
import { toast } from "sonner";
import { RouteError } from "@/components/app/route-error";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { TableSkeleton } from "@/components/ui/table-skeleton";
import { APITokenDialog } from "@/components/users/api-token-dialog";
import { UserDialog } from "@/components/users/user-dialog";
import { makeUsersColumns } from "@/components/users/users-columns";
import { api } from "@/lib/api";
import type { Paginated, User } from "@/types";

export const Route = createFileRoute("/_auth/users")({
	component: UsersPage,
	errorComponent: RouteError,
});

function UsersPage() {
	const qc = useQueryClient();
	const [createOpen, setCreateOpen] = useState(false);
	const [editTarget, setEditTarget] = useState<User | null>(null);
	const [apiToken, setApiToken] = useState<string | null>(null);

	const { data, isLoading } = useQuery({
		queryKey: ["users"],
		queryFn: () => api.get<Paginated<User>>("/users?page=1&limit=1000"),
	});

	const deleteMutation = useMutation({
		mutationFn: (id: string) => api.del<void>(`/users/${id}`),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["users"] });
			toast.success("User deleted");
		},
	});

	const regenTokenMutation = useMutation({
		mutationFn: (id: string) =>
			api.post<{ token: string }>(`/users/${id}/token`, {}),
		onSuccess: (res) => setApiToken(res.token),
	});

	const users = data?.data ?? [];
	const columns = makeUsersColumns({
		onCopyToken: (id) => regenTokenMutation.mutate(id),
		onEdit: setEditTarget,
		onDelete: (id) => deleteMutation.mutate(id),
	});

	const table = useReactTable({
		data: users,
		columns,
		getCoreRowModel: getCoreRowModel(),
	});

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold">Users</h1>
					<p className="text-muted-foreground">Manage user accounts</p>
				</div>
				<Button onClick={() => setCreateOpen(true)}>
					<Plus data-icon="inline-start" />
					Add User
				</Button>
			</div>

			<div className="rounded-lg border">
				<Table>
					<TableHeader>
						{table.getHeaderGroups().map((headerGroup) => (
							<TableRow key={headerGroup.id}>
								{headerGroup.headers.map((header) => (
									<TableHead key={header.id}>
										{header.isPlaceholder
											? null
											: flexRender(
													header.column.columnDef.header,
													header.getContext(),
												)}
									</TableHead>
								))}
							</TableRow>
						))}
					</TableHeader>
					<TableBody>
						{isLoading ? (
							<TableSkeleton columns={4} />
						) : table.getRowModel().rows.length === 0 ? (
							<TableRow>
								<TableCell colSpan={4}>
									<EmptyState title="No users" />
								</TableCell>
							</TableRow>
						) : (
							table.getRowModel().rows.map((row) => (
								<TableRow key={row.id}>
									{row.getVisibleCells().map((cell) => (
										<TableCell key={cell.id}>
											{flexRender(
												cell.column.columnDef.cell,
												cell.getContext(),
											)}
										</TableCell>
									))}
								</TableRow>
							))
						)}
					</TableBody>
				</Table>
			</div>

			<UserDialog
				open={createOpen || !!editTarget}
				onOpenChange={(o) => {
					if (!o) {
						setCreateOpen(false);
						setEditTarget(null);
					}
				}}
				editTarget={editTarget}
			/>

			<APITokenDialog
				open={!!apiToken}
				onOpenChange={(o) => !o && setApiToken(null)}
				token={apiToken}
			/>
		</div>
	);
}
