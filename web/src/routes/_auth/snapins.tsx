import { Plus } from "@phosphor-icons/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	flexRender,
	getCoreRowModel,
	useReactTable,
} from "@tanstack/react-table";
import { useMemo, useRef, useState } from "react";
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
import { api } from "@/lib/api";
import type { Paginated, Snapin } from "@/types";
import { SnapinDialog } from "@/components/snapins/snapin-dialog";
import { makeSnapinsColumns } from "@/components/snapins/snapins-columns";

export const Route = createFileRoute("/_auth/snapins")({
	component: SnapinsPage,
	errorComponent: RouteError,
});

function SnapinsPage() {
	const qc = useQueryClient();
	const [createOpen, setCreateOpen] = useState(false);
	const [editTarget, setEditTarget] = useState<Snapin | null>(null);
	const [uploadTarget, setUploadTarget] = useState<Snapin | null>(null);
	const fileInputRef = useRef<HTMLInputElement>(null);

	const { data, isLoading } = useQuery({
		queryKey: ["snapins"],
		queryFn: () => api.get<Paginated<Snapin>>("/snapins?page=1&limit=1000"),
	});

	const deleteMutation = useMutation({
		mutationFn: (id: string) => api.del<void>(`/snapins/${id}`),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["snapins"] });
			toast.success("Snapin deleted");
		},
	});

	const uploadMutation = useMutation({
		mutationFn: ({ id, file }: { id: string; file: File }) => {
			const formData = new FormData();
			formData.append("file", file);
			return api.upload<Snapin>(`/snapins/${id}/upload`, formData);
		},
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["snapins"] });
			setUploadTarget(null);
			toast.success("File uploaded");
		},
	});

	const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		const file = e.target.files?.[0];
		if (!file || !uploadTarget) return;
		uploadMutation.mutate({ id: uploadTarget.id, file });
	};

	const snapins = data?.data ?? [];

	const columns = useMemo(
		() =>
			makeSnapinsColumns({
				onUpload: (snapin) => {
					setUploadTarget(snapin);
					fileInputRef.current?.click();
				},
				onEdit: setEditTarget,
				onDelete: (id) => deleteMutation.mutate(id),
			}),
		[deleteMutation],
	);

	const snapinTable = useReactTable({
		data: snapins,
		columns,
		getCoreRowModel: getCoreRowModel(),
	});

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold">Snapins</h1>
					<p className="text-muted-foreground">
						Scripts and files deployed to hosts
					</p>
				</div>
				<Button onClick={() => setCreateOpen(true)}>
					<Plus data-icon="inline-start" />
					Add Snapin
				</Button>
			</div>

			<div className="rounded-lg border">
				<Table>
					<TableHeader>
						{snapinTable.getHeaderGroups().map((headerGroup) => (
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
							<TableSkeleton columns={6} />
						) : snapinTable.getRowModel().rows.length === 0 ? (
							<TableRow>
								<TableCell colSpan={6}>
									<EmptyState title="No snapins" />
								</TableCell>
							</TableRow>
						) : (
							snapinTable.getRowModel().rows.map((row) => (
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

			<input
				ref={fileInputRef}
				type="file"
				className="hidden"
				onChange={handleFileChange}
			/>

			<SnapinDialog
				open={createOpen || !!editTarget}
				onOpenChange={(o) => {
					if (!o) {
						setCreateOpen(false);
						setEditTarget(null);
					}
				}}
				editTarget={editTarget}
			/>
		</div>
	);
}
