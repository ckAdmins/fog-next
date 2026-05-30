import { useForm } from "@tanstack/react-form";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import * as z from "zod";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import {
	Field,
	FieldContent,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { api } from "@/lib/api";
import type { Host } from "@/types";

const infoSchema = z.object({
	name: z.string().min(1, "Required"),
	ip: z.string().min(1, "Required"),
	description: z.string(),
	kernel: z.string(),
	init: z.string(),
	kernelArgs: z.string(),
	isEnabled: z.boolean(),
	useAad: z.boolean(),
	useWol: z.boolean(),
});

export function HostInfoCard({ host }: { host: Host }) {
	const qc = useQueryClient();

	const updateMutation = useMutation({
		mutationFn: (values: z.infer<typeof infoSchema>) =>
			api.put<Host>(`/hosts/${host.id}`, values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["host", host.id] });
			void qc.invalidateQueries({ queryKey: ["hosts"] });
			toast.success("Host updated");
		},
	});

	const infoForm = useForm({
		defaultValues: {
			name: host.name ?? "",
			ip: host.ip ?? "",
			description: host.description ?? "",
			kernel: host.kernel ?? "",
			init: host.init ?? "",
			kernelArgs: host.kernelArgs ?? "",
			isEnabled: host.isEnabled ?? true,
			useAad: host.useAad ?? false,
			useWol: host.useWol ?? false,
		},
		validators: { onBlur: infoSchema, onSubmit: infoSchema },
		onSubmit: ({ value }) => updateMutation.mutate(value),
	});

	return (
		<Card>
			<CardHeader>
				<CardTitle>Host Settings</CardTitle>
				<CardDescription>Update host configuration</CardDescription>
			</CardHeader>
			<CardContent>
				<form
					onSubmit={(e) => {
						e.preventDefault();
						void infoForm.handleSubmit();
					}}
				>
					<FieldGroup>
						{(
							[
								"name",
								"ip",
								"description",
								"kernel",
								"init",
								"kernelArgs",
							] as const
						).map((fieldName) => (
							<infoForm.Field key={fieldName} name={fieldName}>
								{(field) => {
									const isInvalid =
										field.state.meta.isTouched && !field.state.meta.isValid;
									return (
										<Field data-invalid={isInvalid}>
											<FieldLabel htmlFor={field.name} className="capitalize">
												{fieldName.replace(/([A-Z])/g, " $1")}
											</FieldLabel>
											<Input
												id={field.name}
												name={field.name}
												value={field.state.value as string}
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
							</infoForm.Field>
						))}

						{(["isEnabled", "useAad", "useWol"] as const).map((fieldName) => {
							const labels: Record<string, string> = {
								isEnabled: "Enabled",
								useAad: "Use AAD",
								useWol: "Use Wake-on-LAN",
							};
							return (
								<infoForm.Field key={fieldName} name={fieldName}>
									{(field) => (
										<Field orientation="horizontal">
											<FieldContent>
												<FieldLabel htmlFor={field.name}>
													{labels[fieldName]}
												</FieldLabel>
											</FieldContent>
											<Switch
												id={field.name}
												checked={field.state.value as boolean}
												onCheckedChange={field.handleChange}
											/>
										</Field>
									)}
								</infoForm.Field>
							);
						})}

						<infoForm.Subscribe selector={(s) => s.isSubmitting}>
							{(isSubmitting) => (
								<Button type="submit" disabled={isSubmitting}>
									{isSubmitting ? "Saving…" : "Save Changes"}
								</Button>
							)}
						</infoForm.Subscribe>
					</FieldGroup>
				</form>
			</CardContent>
		</Card>
	);
}
