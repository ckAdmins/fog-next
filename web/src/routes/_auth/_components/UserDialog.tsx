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
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { api } from "@/lib/api";
import type { User } from "@/types";

export const createSchema = z.object({
	username: z.string().min(1, "Username is required"),
	password: z.string().min(8, "Password must be at least 8 characters"),
	role: z.enum(["admin", "mobile", "readonly"]),
});

export const editSchema = z.object({
	username: z.string().min(1, "Username is required"),
	password: z.string(),
	role: z.enum(["admin", "mobile", "readonly"]),
});

export interface UserDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	editTarget: User | null;
}

export function UserDialog({
	open,
	onOpenChange,
	editTarget,
}: UserDialogProps) {
	const qc = useQueryClient();
	const isEdit = !!editTarget;

	const createMutation = useMutation({
		mutationFn: (values: z.infer<typeof createSchema>) =>
			api.post<User>("/users", values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["users"] });
			onOpenChange(false);
			toast.success("User created");
		},
	});

	const updateMutation = useMutation({
		mutationFn: ({
			id,
			values,
		}: {
			id: string;
			values: z.infer<typeof editSchema>;
		}) => {
			const body: Record<string, unknown> = {
				username: values.username,
				role: values.role,
			};
			if (values.password) body.password = values.password;
			return api.put<User>(`/users/${id}`, body);
		},
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["users"] });
			onOpenChange(false);
			toast.success("User updated");
		},
	});

	const form = useForm({
		defaultValues: {
			username: editTarget?.username ?? "",
			password: "",
			role: (editTarget?.role ?? "readonly") as "admin" | "mobile" | "readonly",
		},
		validators: {
			onBlur: isEdit ? editSchema : createSchema,
			onSubmit: isEdit ? editSchema : createSchema,
		},
		onSubmit: ({ value }) => {
			if (isEdit && editTarget) {
				updateMutation.mutate({ id: editTarget.id, values: value });
			} else {
				createMutation.mutate(value);
			}
		},
	});

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<div key={editTarget?.id ?? "__new__"}>
					<DialogHeader>
						<DialogTitle>{isEdit ? "Edit User" : "Add User"}</DialogTitle>
					</DialogHeader>
					<form
						onSubmit={(e) => {
							e.preventDefault();
							void form.handleSubmit();
						}}
					>
						<FieldGroup>
							<form.Field name="username">
								{(field) => {
									const isInvalid =
										field.state.meta.isTouched && !field.state.meta.isValid;
									return (
										<Field data-invalid={isInvalid}>
											<FieldLabel htmlFor={field.name}>Username</FieldLabel>
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
							<form.Field name="password">
								{(field) => {
									const isInvalid =
										field.state.meta.isTouched && !field.state.meta.isValid;
									return (
										<Field data-invalid={isInvalid}>
											<FieldLabel htmlFor={field.name}>
												{!isEdit
													? "Password"
													: "New Password (leave blank to keep)"}
											</FieldLabel>
											<Input
												id={field.name}
												name={field.name}
												type="password"
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
							<form.Field name="role">
								{(field) => (
									<Field>
										<FieldLabel>Role</FieldLabel>
										<RadioGroup
											value={field.state.value}
											onValueChange={(v) =>
												field.handleChange(v as "admin" | "mobile" | "readonly")
											}
										>
											<div className="flex items-center gap-2">
												<RadioGroupItem value="admin" id="role-admin" />
												<Label htmlFor="role-admin">Admin</Label>
											</div>
											<div className="flex items-center gap-2">
												<RadioGroupItem value="mobile" id="role-mobile" />
												<Label htmlFor="role-mobile">Mobile</Label>
											</div>
											<div className="flex items-center gap-2">
												<RadioGroupItem value="readonly" id="role-readonly" />
												<Label htmlFor="role-readonly">Read Only</Label>
											</div>
										</RadioGroup>
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
