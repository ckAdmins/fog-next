import { Plus } from "@phosphor-icons/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	flexRender,
	getCoreRowModel,
	useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";
import { toast } from "sonner";
import { RouteError } from "@/components/app/route-error";
import { GroupDialog } from "@/components/groups/group-dialog";
import {
	makeGroupListColumns,
	makeGroupMembersColumns,
} from "@/components/groups/groups-columns";
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
import { EmptyState } from "@/components/ui/empty-state";
import { Field, FieldLabel } from "@/components/ui/field";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { api } from "@/lib/api";
import type { Group, GroupMember, Host, Paginated } from "@/types";

export const Route = createFileRoute("/_auth/groups")({
	component: GroupsPage,
	errorComponent: RouteError,
});

function GroupsPage() {
	const qc = useQueryClient();
	const [createOpen, setCreateOpen] = useState(false);
	const [selectedGroup, setSelectedGroup] = useState<Group | null>(null);
	const [addMemberOpen, setAddMemberOpen] = useState(false);
	const [addMemberHostId, setAddMemberHostId] = useState<string | null>("");

	const groupsQuery = useQuery({
		queryKey: ["groups"],
		queryFn: () => api.get<Paginated<Group>>("/groups?page=1&limit=1000"),
	});

	const membersQuery = useQuery({
		queryKey: ["group-members", selectedGroup?.id],
		queryFn: () =>
			// biome-ignore lint/style/noNonNullAssertion: enabled only when selectedGroup is set
			api.get<{ data: GroupMember[] }>(`/groups/${selectedGroup!.id}/members`),
		enabled: !!selectedGroup,
	});

	const hostsQuery = useQuery({
		queryKey: ["hosts", "all"],
		queryFn: () => api.get<Paginated<Host>>("/hosts?page=1&limit=1000"),
	});

	const deleteMutation = useMutation({
		mutationFn: (id: string) => api.del<void>(`/groups/${id}`),
		onSuccess: (_, id) => {
			void qc.invalidateQueries({ queryKey: ["groups"] });
			if (selectedGroup?.id === id) setSelectedGroup(null);
			toast.success("Group deleted");
		},
	});

	const addMemberMutation = useMutation({
		mutationFn: (hostId: string) =>
			// biome-ignore lint/style/noNonNullAssertion: mutates only when selectedGroup is set
			api.post<void>(`/groups/${selectedGroup!.id}/members`, { hostId }),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["group-members", selectedGroup?.id],
			});
			setAddMemberOpen(false);
			setAddMemberHostId("");
			toast.success("Member added");
		},
	});

	const removeMemberMutation = useMutation({
		mutationFn: (hostId: string) =>
			// biome-ignore lint/style/noNonNullAssertion: mutates only when selectedGroup is set
			api.del<void>(`/groups/${selectedGroup!.id}/members/${hostId}`),
		onSuccess: () => {
			void qc.invalidateQueries({
				queryKey: ["group-members", selectedGroup?.id],
			});
			toast.success("Member removed");
		},
	});

	const groups = groupsQuery.data?.data ?? [];
	const members = membersQuery.data?.data ?? [];

	const allHosts = hostsQuery.data?.data ?? [];
	const memberHostIds = new Set(members.map((m) => m.hostId));
	const availableHosts = allHosts.filter((h) => !memberHostIds.has(h.id));

	const groupTable = useReactTable({
		data: groups,
		columns: makeGroupListColumns({
			onDelete: (id) => deleteMutation.mutate(id),
		}),
		getCoreRowModel: getCoreRowModel(),
	});

	const memberTable = useReactTable({
		data: members,
		columns: makeGroupMembersColumns({
			allHosts,
			onRemove: (hostId) => removeMemberMutation.mutate(hostId),
		}),
		getCoreRowModel: getCoreRowModel(),
	});

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center justify-between">
				<div>
					<h1 className="text-2xl font-bold">Groups</h1>
					<p className="text-muted-foreground">Organize hosts into groups</p>
				</div>
				<Button onClick={() => setCreateOpen(true)}>
					<Plus data-icon="inline-start" />
					New Group
				</Button>
			</div>

			<div className="grid grid-cols-1 gap-4 md:grid-cols-3">
				<Card className="md:col-span-1">
					<CardHeader>
						<CardTitle>Groups</CardTitle>
					</CardHeader>
					<CardContent className="p-0">
						{groups.length === 0 ? (
							<EmptyState title="No groups" />
						) : (
							<Table>
								<TableBody>
									{groupTable.getRowModel().rows.map((row) => (
										<TableRow
											key={row.id}
											className="cursor-pointer"
											data-selected={selectedGroup?.id === row.original.id}
											onClick={() => setSelectedGroup(row.original)}
										>
											{row.getVisibleCells().map((cell) => (
												<TableCell key={cell.id}>
													{flexRender(
														cell.column.columnDef.cell,
														cell.getContext(),
													)}
												</TableCell>
											))}
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</CardContent>
				</Card>

				<Card className="md:col-span-2">
					<CardHeader className="flex flex-row items-center justify-between">
						<div>
							<CardTitle>
								{selectedGroup ? selectedGroup.name : "Members"}
							</CardTitle>
							<CardDescription>
								{selectedGroup
									? "Hosts in this group"
									: "Select a group to view members"}
							</CardDescription>
						</div>
						{selectedGroup && (
							<Button size="sm" onClick={() => setAddMemberOpen(true)}>
								<Plus data-icon="inline-start" />
								Add Host
							</Button>
						)}
					</CardHeader>
					<CardContent>
						{!selectedGroup ? null : membersQuery.isLoading ? (
							<Skeleton className="h-24 w-full" />
						) : members.length === 0 ? (
							<EmptyState title="No members" />
						) : (
							<Table>
								<TableHeader>
									{memberTable.getHeaderGroups().map((headerGroup) => (
										<TableRow key={headerGroup.id}>
											{headerGroup.headers.map((header) => (
												<TableHead key={header.id}>
													{header.isPlaceholder
														? null
														: flexRender(
																header.column.columnDef.header,
																header.getContext(),
															)}
												</TableHead>
											))}
										</TableRow>
									))}
								</TableHeader>
								<TableBody>
									{memberTable.getRowModel().rows.map((row) => (
										<TableRow key={row.id}>
											{row.getVisibleCells().map((cell) => (
												<TableCell key={cell.id}>
													{flexRender(
														cell.column.columnDef.cell,
														cell.getContext(),
													)}
												</TableCell>
											))}
										</TableRow>
									))}
								</TableBody>
							</Table>
						)}
					</CardContent>
				</Card>
			</div>

			<GroupDialog open={createOpen} onOpenChange={setCreateOpen} />

			<Dialog open={addMemberOpen} onOpenChange={setAddMemberOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Add Host to Group</DialogTitle>
					</DialogHeader>
					<div className="flex flex-col gap-4">
						<Field>
							<FieldLabel>Host</FieldLabel>
							<Select
								value={addMemberHostId}
								onValueChange={setAddMemberHostId}
							>
								<SelectTrigger>
									<SelectValue placeholder="Select host" />
								</SelectTrigger>
								<SelectContent>
									{availableHosts.map((h) => (
										<SelectItem key={h.id} value={h.id}>
											{h.name}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</Field>
					</div>
					<DialogFooter>
						<Button
							type="button"
							variant="outline"
							onClick={() => setAddMemberOpen(false)}
						>
							Cancel
						</Button>
						<Button
							disabled={!addMemberHostId || addMemberMutation.isPending}
							onClick={() =>
								addMemberHostId && addMemberMutation.mutate(addMemberHostId)
							}
						>
							{addMemberMutation.isPending ? "Adding…" : "Add"}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
}
