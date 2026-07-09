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
import type { StorageGroup } from "@/types";

export const groupSchema = z.object({
	name: z.string().min(1, "Required"),
	description: z.string(),
});

export interface StorageGroupDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	editTarget: StorageGroup | null;
	groups: StorageGroup[];
	selectedGroup: StorageGroup | null;
	setSelectedGroup: (group: StorageGroup | null) => void;
}

export function StorageGroupDialog({
	open,
	onOpenChange,
	editTarget,
}: StorageGroupDialogProps) {
	const qc = useQueryClient();
	const isEdit = !!editTarget;

	const createGroupMutation = useMutation({
		mutationFn: (values: z.infer<typeof groupSchema>) =>
			api.post<StorageGroup>("/storage/groups", values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["storage-groups"] });
			onOpenChange(false);
			toast.success("Storage group created");
		},
	});

	const editGroupMutation = useMutation({
		mutationFn: ({
			id,
			values,
		}: {
			id: string;
			values: z.infer<typeof groupSchema>;
		}) => api.put<StorageGroup>(`/storage/groups/${id}`, values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["storage-groups"] });
			onOpenChange(false);
			toast.success("Storage group updated");
		},
	});

	const form = useForm({
		defaultValues: {
			name: editTarget?.name ?? "",
			description: editTarget?.description ?? "",
		},
		validators: { onBlur: groupSchema, onSubmit: groupSchema },
		onSubmit: ({ value }) => {
			if (isEdit && editTarget) {
				editGroupMutation.mutate({ id: editTarget.id, values: value });
			} else {
				createGroupMutation.mutate(value);
			}
		},
	});

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<div key={editTarget?.id ?? "__new__"}>
					<DialogHeader>
						<DialogTitle>
							{isEdit ? "Edit Storage Group" : "New Storage Group"}
						</DialogTitle>
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
												: "Creating…"
											: isEdit
												? "Save"
												: "Create"}
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
