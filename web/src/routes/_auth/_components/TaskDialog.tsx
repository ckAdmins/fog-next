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
	FieldContent,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "@/components/ui/field";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { api } from "@/lib/api";
import type { Host, Image, Task } from "@/types";

const TASK_TYPES = [
	"deploy",
	"capture",
	"debug_deploy",
	"debug_capture",
	"multicast",
	"wipe",
	"memtest",
	"disk_test",
] as const;

export type TaskType = (typeof TASK_TYPES)[number];

export const IMAGE_TASK_TYPES: TaskType[] = [
	"deploy",
	"capture",
	"debug_deploy",
	"debug_capture",
	"multicast",
];

const taskSchema = z.object({
	type: z.enum(TASK_TYPES, { message: "Task type is required" }),
	hostId: z.string().min(1, "Host is required"),
	imageId: z.string(),
	isShutdown: z.boolean(),
	isForced: z.boolean(),
});

export interface TaskDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	hosts: Host[];
	images: Image[];
}

export function TaskDialog({
	open,
	onOpenChange,
	hosts,
	images,
}: TaskDialogProps) {
	const qc = useQueryClient();

	const createMutation = useMutation({
		mutationFn: (values: z.infer<typeof taskSchema>) =>
			api.post<Task>("/tasks", {
				...values,
				imageId: values.imageId || undefined,
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["tasks"] });
			onOpenChange(false);
			toast.success("Task created");
		},
	});

	const form = useForm({
		defaultValues: {
			type: "deploy" as TaskType,
			hostId: "",
			imageId: "",
			isShutdown: false,
			isForced: false,
		},
		validators: { onBlur: taskSchema, onSubmit: taskSchema },
		onSubmit: ({ value }) => createMutation.mutate(value),
	});

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>New Task</DialogTitle>
				</DialogHeader>
				<form
					onSubmit={(e) => {
						e.preventDefault();
						void form.handleSubmit();
					}}
				>
					<FieldGroup>
						<form.Field name="type">
							{(field) => {
								const isInvalid =
									field.state.meta.isTouched && !field.state.meta.isValid;
								return (
									<Field data-invalid={isInvalid}>
										<FieldLabel>Task Type</FieldLabel>
										<Select
											value={field.state.value}
											onValueChange={(v) => field.handleChange(v as TaskType)}
										>
											<SelectTrigger aria-invalid={isInvalid}>
												<SelectValue placeholder="Select type" />
											</SelectTrigger>
											<SelectContent>
												{TASK_TYPES.map((t) => (
													<SelectItem key={t} value={t}>
														{t}
													</SelectItem>
												))}
											</SelectContent>
										</Select>
										{isInvalid && (
											<FieldError errors={field.state.meta.errors} />
										)}
									</Field>
								);
							}}
						</form.Field>

						<form.Field name="hostId">
							{(field) => {
								const isInvalid =
									field.state.meta.isTouched && !field.state.meta.isValid;
								return (
									<Field data-invalid={isInvalid}>
										<FieldLabel>Host</FieldLabel>
										<Select
											value={field.state.value}
											onValueChange={(v) => v !== null && field.handleChange(v)}
										>
											<SelectTrigger aria-invalid={isInvalid}>
												<SelectValue placeholder="Select host" />
											</SelectTrigger>
											<SelectContent>
												{hosts.map((h) => (
													<SelectItem key={h.id} value={h.id}>
														{h.name}
													</SelectItem>
												))}
											</SelectContent>
										</Select>
										{isInvalid && (
											<FieldError errors={field.state.meta.errors} />
										)}
									</Field>
								);
							}}
						</form.Field>

						<form.Subscribe selector={(s) => s.values.type}>
							{(taskType) =>
								IMAGE_TASK_TYPES.includes(taskType as TaskType) ? (
									<form.Field name="imageId">
										{(field) => (
											<Field>
												<FieldLabel>Image</FieldLabel>
												<Select
													value={field.state.value}
													onValueChange={(v) =>
														v !== null && field.handleChange(v)
													}
												>
													<SelectTrigger>
														<SelectValue placeholder="Select image" />
													</SelectTrigger>
													<SelectContent>
														{images.map((img) => (
															<SelectItem key={img.id} value={img.id}>
																{img.name}
															</SelectItem>
														))}
													</SelectContent>
												</Select>
											</Field>
										)}
									</form.Field>
								) : null
							}
						</form.Subscribe>

						{(["isShutdown", "isForced"] as const).map((fieldName) => {
							const labels: Record<string, string> = {
								isShutdown: "Shutdown after task",
								isForced: "Force (skip queue)",
							};
							return (
								<form.Field key={fieldName} name={fieldName}>
									{(field) => (
										<Field orientation="horizontal">
											<FieldContent>
												<FieldLabel htmlFor={field.name}>
													{labels[fieldName]}
												</FieldLabel>
											</FieldContent>
											<Switch
												id={field.name}
												checked={field.state.value}
												onCheckedChange={field.handleChange}
											/>
										</Field>
									)}
								</form.Field>
							);
						})}
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
									{isSubmitting ? "Creating…" : "Create Task"}
								</Button>
							)}
						</form.Subscribe>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
