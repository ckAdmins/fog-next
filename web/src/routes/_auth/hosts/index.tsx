import { Plus } from "@phosphor-icons/react";
import { useForm } from "@tanstack/react-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import {
	createColumnHelper,
	flexRender,
	getCoreRowModel,
	useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";
import { toast } from "sonner";
import * as z from "zod";
import { RouteError } from "@/components/RouteError";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { EmptyState } from "@/components/ui/empty-state";
import {
	Field,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
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
import type { Host, Paginated } from "@/types";

export const Route = createFileRoute("/_auth/hosts/")({
	component: HostsPage,
	errorComponent: RouteError,
});

const col = createColumnHelper<Host>();

const columns = [
	col.accessor("name", { header: "Name" }),
	col.accessor("ip", { header: "IP Address" }),
	col.accessor("isEnabled", {
		header: "Status",
		cell: (info) =>
			info.getValue() ? (
				<Badge variant="default">Enabled</Badge>
			) : (
				<Badge variant="secondary">Disabled</Badge>
			),
	}),
	col.accessor("lastContact", {
		header: "Last Contact",
		cell: (info) => {
			const v = info.getValue();
			return v ? new Date(v).toLocaleString() : "—";
		},
	}),
];

const createSchema = z.object({
	name: z.string().min(1, "Name is required"),
	ip: z.string().min(1, "IP address is required"),
	description: z.string(),
});

function HostsPage() {
	const navigate = useNavigate();
	const qc = useQueryClient();
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(25);
	const [open, setOpen] = useState(false);

	const { data, isLoading } = useQuery({
		queryKey: ["hosts", page, pageSize],
		queryFn: () =>
			api.get<Paginated<Host>>(`/hosts?page=${page}&limit=${pageSize}`),
	});

	const createMutation = useMutation({
		mutationFn: (values: z.infer<typeof createSchema>) =>
			api.post<Host>("/hosts", values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["hosts"] });
			setOpen(false);
			toast.success("Host created");
		},
	});

	const form = useForm({
		defaultValues: { name: "", ip: "", description: "" },
		validators: { onBlur: createSchema, onSubmit: createSchema },
		onSubmit: ({ value }) => createMutation.mutate(value),
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
					<h1 className="text-2xl font-bold">Hosts</h1>
					<p className="text-muted-foreground">Manage registered hosts</p>
				</div>
				<Button onClick={() => setOpen(true)}>
					<Plus data-icon="inline-start" />
					Add Host
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
									<EmptyState title="No hosts found" />
								</TableCell>
							</TableRow>
						) : (
							table.getRowModel().rows.map((row) => (
								<TableRow
									key={row.id}
									className="cursor-pointer"
									onClick={() =>
										navigate({
											to: "/hosts/$id",
											params: { id: row.original.id },
										})
									}
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

			<Dialog open={open} onOpenChange={setOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Add Host</DialogTitle>
					</DialogHeader>
					<form
						onSubmit={(e) => {
							e.preventDefault();
							void form.handleSubmit();
						}}
					>
						<FieldGroup>
							<form.Field name="name">
								{(field) => {
									const isInvalid =
										field.state.meta.isTouched && !field.state.meta.isValid;
									return (
										<Field data-invalid={isInvalid}>
											<FieldLabel htmlFor={field.name}>Name</FieldLabel>
											<Input
												id={field.name}
												name={field.name}
												value={field.state.value}
												onBlur={field.handleBlur}
												onChange={(e) => field.handleChange(e.target.value)}
												aria-invalid={isInvalid}
											/>
											{isInvalid && (
												<FieldError errors={field.state.meta.errors} />
											)}
										</Field>
									);
								}}
							</form.Field>
							<form.Field name="ip">
								{(field) => {
									const isInvalid =
										field.state.meta.isTouched && !field.state.meta.isValid;
									return (
										<Field data-invalid={isInvalid}>
											<FieldLabel htmlFor={field.name}>IP Address</FieldLabel>
											<Input
												id={field.name}
												name={field.name}
												value={field.state.value}
												onBlur={field.handleBlur}
												onChange={(e) => field.handleChange(e.target.value)}
												aria-invalid={isInvalid}
											/>
											{isInvalid && (
												<FieldError errors={field.state.meta.errors} />
											)}
										</Field>
									);
								}}
							</form.Field>
							<form.Field name="description">
								{(field) => (
									<Field>
										<FieldLabel htmlFor={field.name}>Description</FieldLabel>
										<Input
											id={field.name}
											name={field.name}
											value={field.state.value}
											onBlur={field.handleBlur}
											onChange={(e) => field.handleChange(e.target.value)}
										/>
									</Field>
								)}
							</form.Field>
						</FieldGroup>
						<DialogFooter className="mt-4">
							<Button
								type="button"
								variant="outline"
								onClick={() => setOpen(false)}
							>
								Cancel
							</Button>
							<form.Subscribe selector={(s) => s.isSubmitting}>
								{(isSubmitting) => (
									<Button type="submit" disabled={isSubmitting}>
										{isSubmitting ? "Creating…" : "Create"}
									</Button>
								)}
							</form.Subscribe>
						</DialogFooter>
					</form>
				</DialogContent>
			</Dialog>
		</div>
	);
}
