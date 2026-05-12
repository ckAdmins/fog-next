import { Plus, Trash } from "@phosphor-icons/react";
import { useForm } from "@tanstack/react-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";
import * as z from "zod";
import { AgentLogViewer } from "@/components/agent-log-viewer";
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
	FieldContent,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { api } from "@/lib/api";
import type { Host, HostMAC, Inventory, Task } from "@/types";

export const Route = createFileRoute("/_auth/hosts/$id")({
	component: HostDetailPage,
});

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

const macSchema = z.object({
	mac: z.string().min(1, "MAC address is required"),
	description: z.string(),
});

function HostDetailPage() {
	const { id } = Route.useParams();
	const qc = useQueryClient();
	const [macOpen, setMacOpen] = useState(false);

	const hostQuery = useQuery({
		queryKey: ["host", id],
		queryFn: () => api.get<Host>(`/hosts/${id}`),
	});

	const macsQuery = useQuery({
		queryKey: ["host-macs", id],
		queryFn: () => api.get<{ data: HostMAC[] }>(`/hosts/${id}/macs`),
	});

	const inventoryQuery = useQuery({
		queryKey: ["host-inventory", id],
		queryFn: () => api.get<Inventory>(`/hosts/${id}/inventory`),
	});

	const taskQuery = useQuery({
		queryKey: ["host-task", id],
		queryFn: () => api.get<Task | null>(`/hosts/${id}/task`),
	});

	const updateMutation = useMutation({
		mutationFn: (values: z.infer<typeof infoSchema>) =>
			api.put<Host>(`/hosts/${id}`, values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["host", id] });
			void qc.invalidateQueries({ queryKey: ["hosts"] });
			toast.success("Host updated");
		},
		onError: (err) =>
			toast.error(err instanceof Error ? err.message : "Update failed"),
	});

	const deleteMacMutation = useMutation({
		mutationFn: (macId: string) => api.del<void>(`/hosts/${id}/macs/${macId}`),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["host-macs", id] });
			toast.success("MAC removed");
		},
		onError: (err) =>
			toast.error(err instanceof Error ? err.message : "Failed to remove MAC"),
	});

	const addMacMutation = useMutation({
		mutationFn: (values: z.infer<typeof macSchema>) =>
			api.post<HostMAC>(`/hosts/${id}/macs`, values),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["host-macs", id] });
			setMacOpen(false);
			toast.success("MAC added");
		},
		onError: (err) =>
			toast.error(err instanceof Error ? err.message : "Failed to add MAC"),
	});

	const host = hostQuery.data;

	const infoForm = useForm({
		defaultValues: {
			name: host?.name ?? "",
			ip: host?.ip ?? "",
			description: host?.description ?? "",
			kernel: host?.kernel ?? "",
			init: host?.init ?? "",
			kernelArgs: host?.kernelArgs ?? "",
			isEnabled: host?.isEnabled ?? true,
			useAad: host?.useAad ?? false,
			useWol: host?.useWol ?? false,
		},
		validators: { onSubmit: infoSchema },
		onSubmit: ({ value }) => updateMutation.mutate(value),
	});

	const macForm = useForm({
		defaultValues: { mac: "", description: "" },
		validators: { onSubmit: macSchema },
		onSubmit: ({ value }) => addMacMutation.mutate(value),
	});

	if (hostQuery.isLoading) {
		return (
			<div className="flex flex-col gap-4">
				<Skeleton className="h-8 w-48" />
				<Skeleton className="h-64 w-full" />
			</div>
		);
	}

	if (!host) {
		return <p className="text-muted-foreground">Host not found</p>;
	}

	return (
		<div className="flex flex-col gap-6">
			<div>
				<h1 className="text-2xl font-bold">{host.name}</h1>
				<p className="text-muted-foreground">{host.ip}</p>
			</div>

			<Tabs defaultValue="info">
				<TabsList>
					<TabsTrigger value="info">Info</TabsTrigger>
					<TabsTrigger value="macs">MAC Addresses</TabsTrigger>
					<TabsTrigger value="inventory">Inventory</TabsTrigger>
					<TabsTrigger value="tasks">Tasks</TabsTrigger>
					<TabsTrigger value="logs">Logs</TabsTrigger>
				</TabsList>

				{/* ─── Info Tab ──────────────────────────────────────────────────── */}
				<TabsContent value="info">
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
													field.state.meta.isTouched &&
													!field.state.meta.isValid;
												return (
													<Field data-invalid={isInvalid}>
														<FieldLabel
															htmlFor={field.name}
															className="capitalize"
														>
															{fieldName.replace(/([A-Z])/g, " $1")}
														</FieldLabel>
														<Input
															id={field.name}
															name={field.name}
															value={field.state.value as string}
															onBlur={field.handleBlur}
															onChange={(e) =>
																field.handleChange(e.target.value)
															}
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

									{(["isEnabled", "useAad", "useWol"] as const).map(
										(fieldName) => {
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
										},
									)}

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
				</TabsContent>

				{/* ─── MAC Addresses Tab ────────────────────────────────────── */}
				<TabsContent value="macs">
					<Card>
						<CardHeader className="flex flex-row items-center justify-between">
							<div>
								<CardTitle>MAC Addresses</CardTitle>
								<CardDescription>
									Network interfaces for this host
								</CardDescription>
							</div>
							<Button size="sm" onClick={() => setMacOpen(true)}>
								<Plus data-icon="inline-start" />
								Add MAC
							</Button>
						</CardHeader>
						<CardContent>
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>MAC</TableHead>
										<TableHead>Description</TableHead>
										<TableHead>Primary</TableHead>
										<TableHead />
									</TableRow>
								</TableHeader>
								<TableBody>
									{macsQuery.data?.data.map((mac) => (
										<TableRow key={mac.id}>
											<TableCell className="font-mono">{mac.mac}</TableCell>
											<TableCell>{mac.description || "—"}</TableCell>
											<TableCell>
												{mac.isPrimary && <Badge>Primary</Badge>}
											</TableCell>
											<TableCell className="text-right">
												<AlertDialog>
													<AlertDialogTrigger
														render={
															<Button variant="ghost" size="icon-xs">
																<Trash />
															</Button>
														}
													/>
													<AlertDialogContent>
														<AlertDialogHeader>
															<AlertDialogTitle>
																Remove MAC address?
															</AlertDialogTitle>
															<AlertDialogDescription>
																This will remove {mac.mac} from this host.
															</AlertDialogDescription>
														</AlertDialogHeader>
														<AlertDialogFooter>
															<AlertDialogCancel>Cancel</AlertDialogCancel>
															<AlertDialogAction
																onClick={() => deleteMacMutation.mutate(mac.id)}
															>
																Remove
															</AlertDialogAction>
														</AlertDialogFooter>
													</AlertDialogContent>
												</AlertDialog>
											</TableCell>
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
													<FieldLabel htmlFor={field.name}>
														MAC Address
													</FieldLabel>
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
												<FieldLabel htmlFor={field.name}>
													Description
												</FieldLabel>
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
				</TabsContent>

				{/* ─── Inventory Tab ────────────────────────────────────────── */}
				<TabsContent value="inventory">
					<Card>
						<CardHeader>
							<CardTitle>Hardware Inventory</CardTitle>
							<CardDescription>Detected hardware information</CardDescription>
						</CardHeader>
						<CardContent>
							{inventoryQuery.isLoading ? (
								<div className="flex flex-col gap-2">
									{Array.from({ length: 6 }).map((_, i) => (
										// biome-ignore lint/suspicious/noArrayIndexKey: skeleton placeholders are stable, index key is acceptable
										<Skeleton key={i} className="h-5 w-full" />
									))}
								</div>
							) : !inventoryQuery.data ? (
								<p className="text-muted-foreground">
									No inventory data available
								</p>
							) : (
								<div className="flex flex-col gap-3">
									{(
										[
											[
												"CPU",
												`${inventoryQuery.data.cpuModel} (${inventoryQuery.data.cpuCores} cores @ ${inventoryQuery.data.cpuFreqMhz} MHz)`,
											],
											["RAM", `${inventoryQuery.data.ramMib} MiB`],
											[
												"Disk",
												`${inventoryQuery.data.hdModel} (${inventoryQuery.data.hdSizeGb} GB)`,
											],
											["Manufacturer", inventoryQuery.data.manufacturer],
											["Product", inventoryQuery.data.product],
											["Serial", inventoryQuery.data.serial],
											["UUID", inventoryQuery.data.uuid],
											["BIOS", inventoryQuery.data.biosVersion],
											[
												"OS",
												`${inventoryQuery.data.osName} ${inventoryQuery.data.osVersion}`,
											],
										] as [string, string][]
									).map(([label, value]) => (
										<div key={label}>
											<div className="flex justify-between text-sm">
												<span className="font-medium">{label}</span>
												<span className="text-muted-foreground">{value}</span>
											</div>
											<Separator className="mt-2" />
										</div>
									))}
								</div>
							)}
						</CardContent>
					</Card>
				</TabsContent>

				{/* ─── Tasks Tab ────────────────────────────────────────────── */}
				<TabsContent value="tasks">
					<Card>
						<CardHeader>
							<CardTitle>Current Task</CardTitle>
						</CardHeader>
						<CardContent>
							{taskQuery.isLoading ? (
								<Skeleton className="h-16 w-full" />
							) : !taskQuery.data ? (
								<p className="text-muted-foreground">No active task</p>
							) : (
								<div className="flex flex-col gap-2">
									<div className="flex items-center justify-between">
										<span className="font-medium">{taskQuery.data.type}</span>
										<Badge>{taskQuery.data.state}</Badge>
									</div>
									<div className="text-sm text-muted-foreground">
										Progress: {taskQuery.data.percentComplete}%
									</div>
								</div>
							)}
						</CardContent>
					</Card>
				</TabsContent>

				{/* ─── Logs Tab ─────────────────────────────────────────────── */}
				<TabsContent value="logs">
					<Card>
						<CardHeader>
							<CardTitle>Agent Logs</CardTitle>
							<CardDescription>
								Live log output forwarded from fos-agent during imaging tasks
							</CardDescription>
						</CardHeader>
						<CardContent>
							{taskQuery.isLoading ? (
								<p className="text-muted-foreground text-sm">Loading task…</p>
							) : !taskQuery.data ? (
								<p className="text-muted-foreground text-sm">
									No task found. Logs appear here when an imaging task runs.
								</p>
							) : (
								<AgentLogViewer taskId={taskQuery.data.id} />
							)}
						</CardContent>
					</Card>
				</TabsContent>
			</Tabs>
		</div>
	);
}
