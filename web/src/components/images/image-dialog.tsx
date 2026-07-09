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
import type { Image } from "@/types";

const imageSchema = z.object({
	name: z.string().min(1, "Name is required"),
	description: z.string(),
	path: z.string().min(1, "Path is required"),
});

interface ImageDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	editTarget: Image | null;
}

export function ImageDialog({
	open,
	onOpenChange,
	editTarget,
}: ImageDialogProps) {
	const qc = useQueryClient();
	const isEdit = !!editTarget;

	const createMutation = useMutation({
		mutationFn: (values: z.infer<typeof imageSchema>) =>
			api.post<Image>("/images", values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["images"] });
			onOpenChange(false);
			toast.success("Image created");
		},
	});

	const updateMutation = useMutation({
		mutationFn: ({
			id,
			values,
		}: {
			id: string;
			values: z.infer<typeof imageSchema>;
		}) => api.put<Image>(`/images/${id}`, values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["images"] });
			onOpenChange(false);
			toast.success("Image updated");
		},
	});

	const form = useForm({
		defaultValues: {
			name: editTarget?.name ?? "",
			description: editTarget?.description ?? "",
			path: editTarget?.path ?? "",
		},
		validators: { onBlur: imageSchema, onSubmit: imageSchema },
		onSubmit: ({ value }) => {
			if (isEdit && editTarget) {
				updateMutation.mutate({ id: editTarget.id, values: value });
			} else {
				createMutation.mutate(value);
			}
		},
	});

	function ImageFormFields({ formInstance }: { formInstance: typeof form }) {
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
				<formInstance.Field name="path">
					{(field) => {
						const isInvalid =
							field.state.meta.isTouched && !field.state.meta.isValid;
						return (
							<Field data-invalid={isInvalid}>
								<FieldLabel htmlFor={field.name}>Path</FieldLabel>
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
			</FieldGroup>
		);
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<div key={editTarget?.id ?? "__new__"}>
					<DialogHeader>
						<DialogTitle>{isEdit ? "Edit Image" : "Add Image"}</DialogTitle>
					</DialogHeader>
					<form
						onSubmit={(e) => {
							e.preventDefault();
							void form.handleSubmit();
						}}
					>
						<ImageFormFields formInstance={form} />
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
