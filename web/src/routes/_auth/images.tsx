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
import { ImageDialog } from "@/components/images/image-dialog";
import { makeImagesColumns } from "@/components/images/images-columns";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
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
import type { Image, Paginated } from "@/types";

export const Route = createFileRoute("/_auth/images")({
	component: ImagesPage,
	errorComponent: RouteError,
});

function ImagesPage() {
	const qc = useQueryClient();
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(25);
	const [open, setOpen] = useState(false);
	const [editTarget, setEditTarget] = useState<Image | null>(null);

	const { data, isLoading } = useQuery({
		queryKey: ["images", page, pageSize],
		queryFn: () =>
			api.get<Paginated<Image>>(`/images?page=${page}&limit=${pageSize}`),
	});

	const deleteMutation = useMutation({
		mutationFn: (id: string) => api.del<void>(`/images/${id}`),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["images"] });
			toast.success("Image deleted");
		},
	});

	const columns = makeImagesColumns({
		onEdit: (image) => {
			setEditTarget(image);
			setOpen(true);
		},
		onDelete: (id) => deleteMutation.mutate(id),
	});

	const table = useReactTable({
		data: data?.data ?? [],
		columns,
		getCoreRowModel: getCoreRowModel(),
	});

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold">Images</h1>
					<p className="text-muted-foreground">Manage disk images</p>
				</div>
				<Button onClick={() => setOpen(true)}>
					<Plus data-icon="inline-start" />
					Add Image
				</Button>
			</div>

			<div className="rounded-lg border">
				<Table>
					<TableHeader>
						{table.getHeaderGroups().map((hg) => (
							<TableRow key={hg.id}>
								{hg.headers.map((h) => (
									<TableHead key={h.id}>
										{flexRender(h.column.columnDef.header, h.getContext())}
									</TableHead>
								))}
							</TableRow>
						))}
					</TableHeader>
					<TableBody>
						{isLoading ? (
							<TableSkeleton columns={columns.length} />
						) : table.getRowModel().rows.length === 0 ? (
							<TableRow>
								<TableCell colSpan={columns.length}>
									<EmptyState title="No images found" />
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

			<Pagination
				page={page}
				pageSize={pageSize}
				total={data?.total ?? 0}
				onPageChange={setPage}
				onPageSizeChange={(s) => {
					setPageSize(s);
					setPage(1);
				}}
			/>

			<ImageDialog
				open={open}
				onOpenChange={(o) => {
					setOpen(o);
					if (!o) setEditTarget(null);
				}}
				editTarget={editTarget}
			/>
		</div>
	);
}
