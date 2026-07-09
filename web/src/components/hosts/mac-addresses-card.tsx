import { Plus, Trash } from "@phosphor-icons/react";
import { useForm } from "@tanstack/react-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
	createColumnHelper,
	flexRender,
	getCoreRowModel,
	useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";
import { toast } from "sonner";
import * as z from "zod";
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
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import {
	Field,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { api } from "@/lib/api";
import type { HostMAC } from "@/types";

const macSchema = z.object({
	mac: z.string().min(1, "MAC address is required"),
	description: z.string(),
});

export function MACAddressesCard({ hostId }: { hostId: string }) {
	const qc = useQueryClient();
	const [macOpen, setMacOpen] = useState(false);

	const macsQuery = useQuery({
		queryKey: ["host-macs", hostId],
		queryFn: () => api.get<{ data: HostMAC[] }>(`/hosts/${hostId}/macs`),
	});

	const macs = macsQuery.data?.data ?? [];

	const macColHelper = createColumnHelper<HostMAC>();

	const macColumns = [
		macColHelper.accessor("mac", {
			header: "MAC",
			cell: (info) => <span className="font-mono">{info.getValue()}</span>,
		}),
		macColHelper.accessor("description", {
			header: "Description",
			cell: (info) => info.getValue() || "—",
		}),
		macColHelper.accessor("isPrimary", {
			header: "Primary",
			cell: (info) => (info.getValue() ? <Badge>Primary</Badge> : null),
		}),
		macColHelper.display({
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
									aria-label="Remove MAC address"
								>
									<Trash />
								</Button>
							}
						/>
						<AlertDialogContent>
							<AlertDialogHeader>
								<AlertDialogTitle>Remove MAC address?</AlertDialogTitle>
								<AlertDialogDescription>
									This will remove {info.row.original.mac} from this host.
								</AlertDialogDescription>
							</AlertDialogHeader>
							<AlertDialogFooter>
								<AlertDialogCancel>Cancel</AlertDialogCancel>
								<AlertDialogAction
									onClick={() => deleteMacMutation.mutate(info.row.original.id)}
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

	const macTable = useReactTable({
		data: macs,
		columns: macColumns,
		getCoreRowModel: getCoreRowModel(),
	});

	const deleteMacMutation = useMutation({
		mutationFn: (macId: string) =>
			api.del<void>(`/hosts/${hostId}/macs/${macId}`),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["host-macs", hostId] });
			toast.success("MAC removed");
		},
	});

	const addMacMutation = useMutation({
		mutationFn: (values: z.infer<typeof macSchema>) =>
			api.post<HostMAC>(`/hosts/${hostId}/macs`, values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["host-macs", hostId] });
			setMacOpen(false);
			toast.success("MAC added");
		},
	});

	const macForm = useForm({
		defaultValues: { mac: "", description: "" },
		validators: { onBlur: macSchema, onSubmit: macSchema },
		onSubmit: ({ value }) => addMacMutation.mutate(value),
	});

	return (
		<>
			<Card>
				<CardHeader className="flex flex-row items-center justify-between">
					<div>
						<CardTitle>MAC Addresses</CardTitle>
						<CardDescription>Network interfaces for this host</CardDescription>
					</div>
					<Button size="sm" onClick={() => setMacOpen(true)}>
						<Plus data-icon="inline-start" />
						Add MAC
					</Button>
				</CardHeader>
				<CardContent>
					<Table>
						<TableHeader>
							{macTable.getHeaderGroups().map((headerGroup) => (
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
							{macTable.getRowModel().rows.map((row) => (
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
				</CardContent>
			</Card>

			<Dialog open={macOpen} onOpenChange={setMacOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Add MAC Address</DialogTitle>
					</DialogHeader>
					<form
						onSubmit={(e) => {
							e.preventDefault();
							void macForm.handleSubmit();
						}}
					>
						<FieldGroup>
							<macForm.Field name="mac">
								{(field) => {
									const isInvalid =
										field.state.meta.isTouched && !field.state.meta.isValid;
									return (
										<Field data-invalid={isInvalid}>
											<FieldLabel htmlFor={field.name}>MAC Address</FieldLabel>
											<Input
												id={field.name}
												name={field.name}
												value={field.state.value}
												onBlur={field.handleBlur}
												onChange={(e) => field.handleChange(e.target.value)}
												aria-invalid={isInvalid}
												placeholder="AA:BB:CC:DD:EE:FF"
											/>
											{isInvalid && (
												<FieldError errors={field.state.meta.errors} />
											)}
										</Field>
									);
								}}
							</macForm.Field>
							<macForm.Field name="description">
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
							</macForm.Field>
						</FieldGroup>
						<DialogFooter className="mt-4">
							<Button
								type="button"
								variant="outline"
								onClick={() => setMacOpen(false)}
							>
								Cancel
							</Button>
							<macForm.Subscribe selector={(s) => s.isSubmitting}>
								{(isSubmitting) => (
									<Button type="submit" disabled={isSubmitting}>
										{isSubmitting ? "Adding…" : "Add"}
									</Button>
								)}
							</macForm.Subscribe>
						</DialogFooter>
					</form>
				</DialogContent>
			</Dialog>
		</>
	);
}
