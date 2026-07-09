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
import type { Snapin } from "@/types";

export const snapinSchema = z.object({
	name: z.string().min(1, "Name is required"),
	description: z.string(),
	runOrder: z.number().int().min(0),
	timeout: z.number().int().min(0),
});

export interface SnapinDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	editTarget: Snapin | null;
}

export function SnapinDialog({
	open,
	onOpenChange,
	editTarget,
}: SnapinDialogProps) {
	const qc = useQueryClient();

	const createMutation = useMutation({
		mutationFn: (values: z.infer<typeof snapinSchema>) =>
			api.post<Snapin>("/snapins", values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["snapins"] });
			onOpenChange(false);
			toast.success("Snapin created");
		},
	});

	const updateMutation = useMutation({
		mutationFn: ({
			id,
			values,
		}: {
			id: string;
			values: z.infer<typeof snapinSchema>;
		}) => api.put<Snapin>(`/snapins/${id}`, values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["snapins"] });
			onOpenChange(false);
			toast.success("Snapin updated");
		},
	});

	const form = useForm({
		defaultValues: { name: "", description: "", runOrder: 0, timeout: 300 },
		validators: { onBlur: snapinSchema, onSubmit: snapinSchema },
		onSubmit: ({ value }) => createMutation.mutate(value),
	});

	const editForm = useForm({
		defaultValues: {
			name: editTarget?.name ?? "",
			description: editTarget?.description ?? "",
			runOrder: editTarget?.runOrder ?? 0,
			timeout: editTarget?.timeout ?? 300,
		},
		validators: { onBlur: snapinSchema, onSubmit: snapinSchema },
		onSubmit: ({ value }) => {
			if (editTarget)
				updateMutation.mutate({ id: editTarget.id, values: value });
		},
	});

	const isEditing = !!editTarget;

	function SnapinFormFields({ formInstance }: { formInstance: typeof form }) {
		return (
			<FieldGroup>
				<formInstance.Field name="name">
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
								{isInvalid && <FieldError errors={field.state.meta.errors} />}
							</Field>
						);
					}}
				</formInstance.Field>
				<formInstance.Field name="description">
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
				</formInstance.Field>
				<formInstance.Field name="runOrder">
					{(field) => (
						<Field>
							<FieldLabel htmlFor={field.name}>Run Order</FieldLabel>
							<Input
								id={field.name}
								name={field.name}
								type="number"
								value={field.state.value}
								onBlur={field.handleBlur}
								onChange={(e) => field.handleChange(Number(e.target.value))}
							/>
						</Field>
					)}
				</formInstance.Field>
				<formInstance.Field name="timeout">
					{(field) => (
						<Field>
							<FieldLabel htmlFor={field.name}>Timeout (seconds)</FieldLabel>
							<Input
								id={field.name}
								name={field.name}
								type="number"
								value={field.state.value}
								onBlur={field.handleBlur}
								onChange={(e) => field.handleChange(Number(e.target.value))}
							/>
						</Field>
					)}
				</formInstance.Field>
			</FieldGroup>
		);
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>{isEditing ? "Edit Snapin" : "Add Snapin"}</DialogTitle>
				</DialogHeader>
				<form
					onSubmit={(e) => {
						e.preventDefault();
						if (isEditing) {
							void editForm.handleSubmit();
						} else {
							void form.handleSubmit();
						}
					}}
				>
					<SnapinFormFields formInstance={isEditing ? editForm : form} />
					<DialogFooter className="mt-4">
						<Button
							type="button"
							variant="outline"
							onClick={() => onOpenChange(false)}
						>
							Cancel
						</Button>
						{isEditing ? (
							<editForm.Subscribe selector={(s) => s.isSubmitting}>
								{(isSubmitting) => (
									<Button type="submit" disabled={isSubmitting}>
										{isSubmitting ? "Saving…" : "Save"}
									</Button>
								)}
							</editForm.Subscribe>
						) : (
							<form.Subscribe selector={(s) => s.isSubmitting}>
								{(isSubmitting) => (
									<Button type="submit" disabled={isSubmitting}>
										{isSubmitting ? "Creating…" : "Create"}
									</Button>
								)}
							</form.Subscribe>
						)}
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
