import { Check, X } from "@phosphor-icons/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	createColumnHelper,
	flexRender,
	getCoreRowModel,
	useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";
import { toast } from "sonner";
import { RouteError } from "@/components/app/route-error";
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
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { EmptyState } from "@/components/ui/empty-state";
import { Field, FieldLabel } from "@/components/ui/field";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { TableSkeleton } from "@/components/ui/table-skeleton";
import { api } from "@/lib/api";
import type { Host, Paginated, PendingMAC } from "@/types";

export const Route = createFileRoute("/_auth/pending-macs")({
	component: PendingMacsPage,
	errorComponent: RouteError,
});

function PendingMacsPage() {
	const qc = useQueryClient();
	const [approveTarget, setApproveTarget] = useState<PendingMAC | null>(null);
	const [approveHostId, setApproveHostId] = useState("");

	const { data, isLoading } = useQuery({
		queryKey: ["pending-macs"],
		queryFn: () =>
			api.get<Paginated<PendingMAC>>("/pending-macs?page=1&limit=1000"),
	});

	const hostsQuery = useQuery({
		queryKey: ["hosts", "all"],
		queryFn: () => api.get<Paginated<Host>>("/hosts?page=1&limit=1000"),
	});

	const approveMutation = useMutation({
		mutationFn: ({ id, hostId }: { id: string; hostId: string }) =>
			api.post<void>(`/pending-macs/${id}/approve`, { hostId }),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["pending-macs"] });
			void qc.invalidateQueries({ queryKey: ["host-macs"] });
			setApproveTarget(null);
			setApproveHostId("");
			toast.success("MAC approved and assigned");
		},
	});

	const ignoreMutation = useMutation({
		mutationFn: (id: string) => api.del<void>(`/pending-macs/${id}`),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["pending-macs"] });
			toast.success("MAC ignored");
		},
	});

	const pendingMacs = data?.data ?? [];
	const hosts = hostsQuery.data?.data ?? [];

	const columnHelper = createColumnHelper<PendingMAC>();

	const columns = [
		columnHelper.accessor("mac", {
			header: "MAC Address",
			cell: (info) => <span className="font-mono">{info.getValue()}</span>,
		}),
		columnHelper.accessor("firstSeen", {
			header: "First Seen",
			cell: (info) => (
				<span className="text-sm text-muted-foreground">
					{new Date(info.getValue()).toLocaleString()}
				</span>
			),
		}),
		columnHelper.accessor("lastSeen", {
			header: "Last Seen",
			cell: (info) => (
				<span className="text-sm text-muted-foreground">
					{new Date(info.getValue()).toLocaleString()}
				</span>
			),
		}),
		columnHelper.display({
			id: "actions",
			header: () => null,
			cell: (info) => (
				<div className="flex justify-end gap-1">
					<Button
						variant="ghost"
						size="sm"
						onClick={() => {
							setApproveTarget(info.row.original);
							setApproveHostId("");
						}}
					>
						<Check data-icon="inline-start" />
						Approve
					</Button>
					<AlertDialog>
						<AlertDialogTrigger
							render={
								<Button variant="ghost" size="sm">
									<X data-icon="inline-start" />
									Ignore
								</Button>
							}
						/>
						<AlertDialogContent>
							<AlertDialogHeader>
								<AlertDialogTitle>Ignore MAC address?</AlertDialogTitle>
								<AlertDialogDescription>
									This will remove {info.row.original.mac} from the pending
									list.
								</AlertDialogDescription>
							</AlertDialogHeader>
							<AlertDialogFooter>
								<AlertDialogCancel>Cancel</AlertDialogCancel>
								<AlertDialogAction
									onClick={() => ignoreMutation.mutate(info.row.original.id)}
								>
									Ignore
								</AlertDialogAction>
							</AlertDialogFooter>
						</AlertDialogContent>
					</AlertDialog>
				</div>
			),
		}),
	];

	const table = useReactTable({
		data: pendingMacs,
		columns,
		getCoreRowModel: getCoreRowModel(),
	});

	return (
		<div className="flex flex-col gap-6">
			<div>
				<h1 className="text-2xl font-bold">Pending MACs</h1>
				<p className="text-muted-foreground">
					Unknown MAC addresses that have contacted the server
				</p>
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
									<EmptyState title="No pending MACs" />
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

			{/* Approve Dialog */}
			<Dialog
				open={!!approveTarget}
				onOpenChange={(o) => !o && setApproveTarget(null)}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Approve MAC Address</DialogTitle>
					</DialogHeader>
					<p className="text-sm text-muted-foreground">
						Assign{" "}
						<span className="font-mono font-medium">{approveTarget?.mac}</span>{" "}
						to a host.
					</p>
					<Field>
						<FieldLabel>Host</FieldLabel>
						<Select
							value={approveHostId}
							onValueChange={(v) => v !== null && setApproveHostId(v)}
						>
							<SelectTrigger>
								<SelectValue placeholder="Select host" />
							</SelectTrigger>
							<SelectContent>
								{hosts.map((h) => (
									<SelectItem key={h.id} value={h.id}>
										{h.name}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</Field>
					<DialogFooter>
						<Button variant="outline" onClick={() => setApproveTarget(null)}>
							Cancel
						</Button>
						<Button
							disabled={!approveHostId || approveMutation.isPending}
							onClick={() => {
								if (approveTarget && approveHostId) {
									approveMutation.mutate({
										id: approveTarget.id,
										hostId: approveHostId,
									});
								}
							}}
						>
							{approveMutation.isPending ? "Approving…" : "Approve"}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
}
