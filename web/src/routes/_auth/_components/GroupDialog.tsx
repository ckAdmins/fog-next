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
import type { Group } from "@/types";

export const groupSchema = z.object({
	name: z.string().min(1, "Name is required"),
	description: z.string(),
});

export interface GroupDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
}

export function GroupDialog({ open, onOpenChange }: GroupDialogProps) {
	const qc = useQueryClient();

	const createMutation = useMutation({
		mutationFn: (values: z.infer<typeof groupSchema>) =>
			api.post<Group>("/groups", values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["groups"] });
			onOpenChange(false);
			toast.success("Group created");
		},
	});

	const form = useForm({
		defaultValues: { name: "", description: "" },
		validators: { onBlur: groupSchema, onSubmit: groupSchema },
		onSubmit: ({ value }) => createMutation.mutate(value),
	});

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>New Group</DialogTitle>
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
									{isSubmitting ? "Creating…" : "Create"}
								</Button>
							)}
						</form.Subscribe>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
