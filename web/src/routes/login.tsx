import { useForm } from "@tanstack/react-form";
import { createFileRoute, redirect, useRouter } from "@tanstack/react-router";
import { toast } from "sonner";
import * as z from "zod";
import { RouteError } from "@/components/app/route-error";
import { Button } from "@/components/ui/button";
import {
	Field,
	FieldDescription,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { api } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import type { TokenPair } from "@/types";

export const Route = createFileRoute("/login")({
	beforeLoad: () => {
		if (useAuthStore.getState().isAuthenticated) {
			throw redirect({ to: "/dashboard" });
		}
	},
	component: LoginPage,
	errorComponent: RouteError,
});

const loginSchema = z.object({
	username: z.string().min(1, "Username is required"),
	password: z.string().min(1, "Password is required"),
});

function LoginPage() {
	const router = useRouter();
	const login = useAuthStore((s) => s.login);

	const form = useForm({
		defaultValues: { username: "", password: "" },
		validators: { onBlur: loginSchema, onSubmit: loginSchema },
		onSubmit: async ({ value }) => {
			try {
				const tokens = await api.post<TokenPair>("/auth/login", value);
				login(tokens);
				await router.navigate({ to: "/dashboard" });
			} catch (err) {
				toast.error(err instanceof Error ? err.message : "Login failed");
			}
		},
	});

	return (
		<div className="flex min-h-screen items-center justify-center bg-background p-4">
			<div className="flex w-full max-w-sm flex-col items-center gap-8">
				<div className="flex flex-col items-center gap-1.5">
					<span className="font-heading text-3xl font-light tracking-[0.3em] text-foreground">
						FOG
					</span>
					<span className="text-xs font-medium tracking-[0.12em] text-muted-foreground">
						NETWORK BOOT & IMAGING
					</span>
				</div>

				<div className="w-full border-t border-border/40" />

				<div className="w-full">
					<form
						onSubmit={(e) => {
							e.preventDefault();
							void form.handleSubmit();
						}}
					>
						<FieldGroup className="gap-4">
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
												autoComplete="username"
												autoFocus
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
											<FieldLabel htmlFor={field.name}>Password</FieldLabel>
											<Input
												id={field.name}
												name={field.name}
												type="password"
												value={field.state.value}
												onBlur={field.handleBlur}
												onChange={(e) => field.handleChange(e.target.value)}
												aria-invalid={isInvalid}
												autoComplete="current-password"
											/>
											{isInvalid && (
												<FieldError errors={field.state.meta.errors} />
											)}
										</Field>
									);
								}}
							</form.Field>

							<form.Subscribe selector={(s) => s.isSubmitting}>
								{(isSubmitting) => (
									<Button
										type="submit"
										className="w-full"
										disabled={isSubmitting}
									>
										{isSubmitting ? "Signing in\u2026" : "Sign in"}
									</Button>
								)}
							</form.Subscribe>
						</FieldGroup>
					</form>

					<FieldDescription className="mt-4 text-center text-[11px]">
						Use your FOG administrator credentials to continue.
					</FieldDescription>
				</div>
			</div>
		</div>
	);
}
