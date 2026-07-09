import { useForm } from "@tanstack/react-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import * as z from "zod";
import { Button } from "@/components/ui/button";
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
import { api } from "@/lib/api";
import type { StorageGroup, StorageNode } from "@/types";

const nodeSchema = z.object({
	name: z.string().min(1, "Required"),
	description: z.string(),
	ip: z.string().min(1, "Required"),
	path: z.string().min(1, "Required"),
	maxClients: z.number().int().min(1),
	bandwidthMbps: z.number().int().min(0),
});

export interface StorageNodeDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	editTarget: StorageNode | null;
	nodes: StorageNode[];
	selectedGroup: StorageGroup;
}

export function StorageNodeDialog({
	open,
	onOpenChange,
	editTarget,
	selectedGroup,
}: StorageNodeDialogProps) {
	const qc = useQueryClient();
	const isEdit = !!editTarget;

	const createNodeMutation = useMutation({
		mutationFn: (values: z.infer<typeof nodeSchema>) =>
			api.post<StorageNode>(
				`/storage/groups/${selectedGroup.id}/nodes`,
				values,
			),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["storage-nodes", selectedGroup.id],
			});
			onOpenChange(false);
			toast.success("Node added");
		},
	});

	const updateNodeMutation = useMutation({
		mutationFn: ({
			id,
			values,
		}: {
			id: string;
			values: z.infer<typeof nodeSchema>;
		}) => api.put<StorageNode>(`/storage/nodes/${id}`, values),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["storage-nodes", selectedGroup.id],
			});
			onOpenChange(false);
			toast.success("Node updated");
		},
	});

	const form = useForm({
		defaultValues: {
			name: editTarget?.name ?? "",
			description: editTarget?.description ?? "",
			ip: editTarget?.ip ?? "",
			path: editTarget?.path ?? "",
			maxClients: editTarget?.maxClients ?? 10,
			bandwidthMbps: editTarget?.bandwidthMbps ?? 0,
		},
		validators: { onBlur: nodeSchema, onSubmit: nodeSchema },
		onSubmit: ({ value }) => {
			if (isEdit && editTarget) {
				updateNodeMutation.mutate({ id: editTarget.id, values: value });
			} else {
				createNodeMutation.mutate(value);
			}
		},
	});

	function NodeFormFields({ formInstance }: { formInstance: typeof form }) {
		return (
			<FieldGroup>
				{(["name", "ip", "path", "description"] as const).map((fieldName) => (
					<formInstance.Field key={fieldName} name={fieldName}>
						{(field) => {
							const isInvalid =
								field.state.meta.isTouched && !field.state.meta.isValid;
							return (
								<Field data-invalid={isInvalid}>
									<FieldLabel htmlFor={field.name} className="capitalize">
										{fieldName}
									</FieldLabel>
									<Input
										id={field.name}
										name={field.name}
										value={field.state.value as string}
										onBlur={field.handleBlur}
										onChange={(e) => field.handleChange(e.target.value)}
										aria-invalid={isInvalid}
									/>
									{isInvalid && <FieldError errors={field.state.meta.errors} />}
								</Field>
							);
						}}
					</formInstance.Field>
				))}
				{(["maxClients", "bandwidthMbps"] as const).map((fieldName) => {
					const labels: Record<string, string> = {
						maxClients: "Max Clients",
						bandwidthMbps: "Bandwidth (Mbps)",
					};
					return (
						<formInstance.Field key={fieldName} name={fieldName}>
							{(field) => (
								<Field>
									<FieldLabel htmlFor={field.name}>
										{labels[fieldName]}
									</FieldLabel>
									<Input
										id={field.name}
										name={field.name}
										type="number"
										value={field.state.value as number}
										onBlur={field.handleBlur}
										onChange={(e) => field.handleChange(Number(e.target.value))}
									/>
								</Field>
							)}
						</formInstance.Field>
					);
				})}
			</FieldGroup>
		);
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<div key={editTarget?.id ?? "__new__"}>
					<DialogHeader>
						<DialogTitle>{isEdit ? "Edit Node" : "Add Node"}</DialogTitle>
					</DialogHeader>
					<form
						onSubmit={(e) => {
							e.preventDefault();
							void form.handleSubmit();
						}}
					>
						<NodeFormFields formInstance={form} />
						<DialogFooter className="mt-4">
							<Button
								type="button"
								variant="outline"
								onClick={() => onOpenChange(false)}
							>
								Cancel
							</Button>
							<form.Subscribe selector={(s) => s.isSubmitting}>
								{(isSubmitting) => (
									<Button type="submit" disabled={isSubmitting}>
										{isSubmitting
											? isEdit
												? "Saving…"
												: "Adding…"
											: isEdit
												? "Save"
												: "Add Node"}
									</Button>
								)}
							</form.Subscribe>
						</DialogFooter>
					</form>
				</div>
			</DialogContent>
		</Dialog>
	);
}
