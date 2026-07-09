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
import {
	makeStorageGroupColumns,
	makeStorageNodeColumns,
} from "@/components/storage/storage-columns";
import { StorageGroupDialog } from "@/components/storage/storage-group-dialog";
import { StorageNodeDialog } from "@/components/storage/storage-node-dialog";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { EmptyState } from "@/components/ui/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { api } from "@/lib/api";
import type { StorageGroup, StorageNode } from "@/types";

export const Route = createFileRoute("/_auth/storage")({
	component: StoragePage,
	errorComponent: RouteError,
});

function StoragePage() {
	const qc = useQueryClient();
	const [selectedGroup, setSelectedGroup] = useState<StorageGroup | null>(null);
	const [groupDialogOpen, setGroupDialogOpen] = useState(false);
	const [nodeDialogOpen, setNodeDialogOpen] = useState(false);
	const [groupEditTarget, setGroupEditTarget] = useState<StorageGroup | null>(
		null,
	);
	const [nodeEditTarget, setNodeEditTarget] = useState<StorageNode | null>(
		null,
	);

	const groupsQuery = useQuery({
		queryKey: ["storage-groups"],
		queryFn: () => api.get<{ data: StorageGroup[] }>("/storage/groups"),
	});

	const nodesQuery = useQuery({
		queryKey: ["storage-nodes", selectedGroup?.id],
		queryFn: () =>
			api.get<{ data: StorageNode[] }>(
				// biome-ignore lint/style/noNonNullAssertion: enabled only when selectedGroup is set
				`/storage/groups/${selectedGroup!.id}/nodes`,
			),
		enabled: !!selectedGroup,
	});

	const deleteGroupMutation = useMutation({
		mutationFn: (id: string) => api.del<void>(`/storage/groups/${id}`),
		onSuccess: (_, id) => {
			void qc.invalidateQueries({ queryKey: ["storage-groups"] });
			if (selectedGroup?.id === id) setSelectedGroup(null);
			toast.success("Storage group deleted");
		},
	});

	const deleteNodeMutation = useMutation({
		mutationFn: (id: string) => api.del<void>(`/storage/nodes/${id}`),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["storage-nodes", selectedGroup?.id],
			});
			toast.success("Node removed");
		},
	});

	const groups = groupsQuery.data?.data ?? [];
	const nodes = nodesQuery.data?.data ?? [];

	const sgColumns = makeStorageGroupColumns({
		onDelete: (id) => deleteGroupMutation.mutate(id),
		selectedGroupId: selectedGroup?.id,
	});

	const sgTable = useReactTable({
		data: groups,
		columns: sgColumns,
		getCoreRowModel: getCoreRowModel(),
	});

	const snColumns = makeStorageNodeColumns({
		onEdit: (node) => {
			setNodeEditTarget(node);
			setNodeDialogOpen(true);
		},
		onRemove: (id) => deleteNodeMutation.mutate(id),
	});

	const snTable = useReactTable({
		data: nodes,
		columns: snColumns,
		getCoreRowModel: getCoreRowModel(),
	});

	return (
		<div className="flex flex-col gap-6">
			<div>
				<h1 className="text-2xl font-bold">Storage</h1>
				<p className="text-muted-foreground">Manage storage groups and nodes</p>
			</div>

			<div className="grid grid-cols-1 gap-4 md:grid-cols-3">
				{/* Storage Groups */}
				<Card className="md:col-span-1">
					<CardHeader className="flex flex-row items-center justify-between">
						<CardTitle>Storage Groups</CardTitle>
						<Button
							size="sm"
							variant="ghost"
							onClick={() => {
								setGroupEditTarget(null);
								setGroupDialogOpen(true);
							}}
						>
							<Plus />
						</Button>
					</CardHeader>
					<CardContent className="p-0">
						{groups.length === 0 ? (
							<EmptyState title="No groups" />
						) : (
							<Table>
								<TableBody>
									{sgTable.getRowModel().rows.map((row) => (
										<TableRow
											key={row.id}
											className="cursor-pointer"
											data-selected={selectedGroup?.id === row.original.id}
											onClick={() => setSelectedGroup(row.original)}
										>
											{row.getVisibleCells().map((cell) => (
												<TableCell key={cell.id}>
													{flexRender(
														cell.column.columnDef.cell,
														cell.getContext(),
													)}
												</TableCell>
											))}
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</CardContent>
				</Card>

				{/* Nodes */}
				<Card className="md:col-span-2">
					<CardHeader className="flex flex-row items-center justify-between">
						<div>
							<CardTitle>
								{selectedGroup ? selectedGroup.name : "Nodes"}
							</CardTitle>
							<CardDescription>
								{selectedGroup
									? "Storage nodes in this group"
									: "Select a group to manage nodes"}
							</CardDescription>
						</div>
						{selectedGroup && (
							<Button
								size="sm"
								onClick={() => {
									setNodeEditTarget(null);
									setNodeDialogOpen(true);
								}}
							>
								<Plus data-icon="inline-start" />
								Add Node
							</Button>
						)}
					</CardHeader>
					<CardContent>
						{!selectedGroup ? null : nodesQuery.isLoading ? (
							<Skeleton className="h-24 w-full" />
						) : nodes.length === 0 ? (
							<EmptyState title="No nodes" />
						) : (
							<Table>
								<TableHeader>
									{snTable.getHeaderGroups().map((headerGroup) => (
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
									{snTable.getRowModel().rows.map((row) => (
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
									))}
								</TableBody>
							</Table>
						)}
					</CardContent>
				</Card>
			</div>

			<StorageGroupDialog
				open={groupDialogOpen}
				onOpenChange={(open) => {
					setGroupDialogOpen(open);
					if (!open) setGroupEditTarget(null);
				}}
				editTarget={groupEditTarget}
				groups={groups}
				selectedGroup={selectedGroup}
				setSelectedGroup={setSelectedGroup}
			/>

			{selectedGroup && (
				<StorageNodeDialog
					open={nodeDialogOpen}
					onOpenChange={(open) => {
						setNodeDialogOpen(open);
						if (!open) setNodeEditTarget(null);
					}}
					editTarget={nodeEditTarget}
					nodes={nodes}
					selectedGroup={selectedGroup}
				/>
			)}
		</div>
	);
}
