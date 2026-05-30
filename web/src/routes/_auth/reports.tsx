import { Download } from "@phosphor-icons/react";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	createColumnHelper,
	flexRender,
	getCoreRowModel,
	useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";
import { RouteError } from "@/components/RouteError";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Pagination } from "@/components/ui/pagination";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { TableSkeleton } from "@/components/ui/table-skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { api } from "@/lib/api";
import type { Host, ImagingLog, Inventory, Paginated } from "@/types";

export const Route = createFileRoute("/_auth/reports")({
	component: ReportsPage,
	errorComponent: RouteError,
});

function downloadCsv(filename: string, rows: string[][]) {
	const csv = rows
		.map((r) => r.map((c) => `"${String(c).replace(/"/g, '""')}"`).join(","))
		.join("\n");
	const url = URL.createObjectURL(new Blob([csv], { type: "text/csv" }));
	const a = document.createElement("a");
	a.href = url;
	a.download = filename;
	a.click();
	URL.revokeObjectURL(url);
}

function ImagingHistoryTab() {
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(50);

	const { data, isLoading } = useQuery({
		queryKey: ["imaging-logs", page, pageSize],
		queryFn: () =>
			api.get<Paginated<ImagingLog>>(
				`/reports/imaging?page=${page}&limit=${pageSize}`,
			),
	});

	const logs = data?.data ?? [];

	const logColHelper = createColumnHelper<ImagingLog>();

	const logColumns = [
		logColHelper.accessor("hostId", {
			header: "Host ID",
			cell: (info) => info.getValue(),
		}),
		logColHelper.accessor("imageId", {
			header: "Image",
			cell: (info) => info.getValue() ?? "—",
		}),
		logColHelper.accessor("type", {
			header: "Type",
			cell: (info) => info.getValue(),
		}),
		logColHelper.accessor("state", {
			header: "State",
			cell: (info) => {
				const state = info.getValue();
				return (
					<Badge
						variant={
							state === "complete"
								? "default"
								: state === "failed"
									? "destructive"
									: "secondary"
						}
					>
						{state}
					</Badge>
				);
			},
		}),
		logColHelper.accessor("durationSeconds", {
			header: "Duration",
			cell: (info) => (info.getValue() != null ? `${info.getValue()}s` : "—"),
		}),
		logColHelper.accessor("createdAt", {
			header: "Date",
			cell: (info) => (
				<span className="text-sm text-muted-foreground">
					{new Date(info.getValue()).toLocaleString()}
				</span>
			),
		}),
	];

	const logTable = useReactTable({
		data: logs,
		columns: logColumns,
		getCoreRowModel: getCoreRowModel(),
	});

	return (
		<div className="flex flex-col gap-4">
			<div className="flex justify-end">
				<Button
					variant="outline"
					size="sm"
					disabled={logs.length === 0}
					onClick={() =>
						downloadCsv("imaging-history.csv", [
							["Host", "Image", "Type", "State", "Duration", "Date"],
							...logs.map((l) => [
								l.hostId,
								l.imageId ?? "",
								l.type,
								l.state,
								String(l.durationSeconds ?? ""),
								l.createdAt,
							]),
						])
					}
				>
					<Download data-icon="inline-start" />
					Export CSV
				</Button>
			</div>
			<div className="rounded-lg border">
				<Table>
					<TableHeader>
						{logTable.getHeaderGroups().map((headerGroup) => (
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
						{isLoading ? (
							<TableSkeleton columns={6} />
						) : logTable.getRowModel().rows.length === 0 ? (
							<TableRow>
								<TableCell colSpan={6}>
									<EmptyState title="No imaging history" />
								</TableCell>
							</TableRow>
						) : (
							logTable.getRowModel().rows.map((row) => (
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
							))
						)}
					</TableBody>
				</Table>
			</div>
			<Pagination
				page={page}
				pageSize={pageSize}
				total={data?.total ?? 0}
				onPageChange={setPage}
				onPageSizeChange={(s) => {
					setPageSize(s);
					setPage(1);
				}}
			/>
		</div>
	);
}

function HostInventoryTab() {
	const hostsQuery = useQuery({
		queryKey: ["hosts", "all"],
		queryFn: () => api.get<Paginated<Host>>("/hosts?page=1&limit=1000"),
	});

	const inventoryQuery = useQuery({
		queryKey: ["inventory", "all"],
		queryFn: () =>
			api.get<{ data: (Inventory & { hostId: string })[] }>(
				"/reports/inventory",
			),
	});

	const hosts = hostsQuery.data?.data ?? [];
	const inventories = inventoryQuery.data?.data ?? [];

	// Join inventory with host names
	const rows = inventories.map((inv) => {
		const host = hosts.find((h) => h.id === inv.hostId);
		return { ...inv, hostName: host?.name ?? inv.hostId };
	});

	type InventoryRow = Inventory & { hostName: string };

	const invColHelper = createColumnHelper<InventoryRow>();

	const invColumns = [
		invColHelper.accessor("hostName", {
			header: "Host",
			cell: (info) => <span className="font-medium">{info.getValue()}</span>,
		}),
		invColHelper.accessor("cpuModel", {
			header: "CPU",
			cell: (info) =>
				`${info.getValue()} (${info.row.original.cpuCores} cores)`,
		}),
		invColHelper.accessor("ramMib", {
			header: "RAM",
			cell: (info) => `${info.getValue()} MiB`,
		}),
		invColHelper.accessor("hdModel", {
			header: "Disk",
			cell: (info) => `${info.getValue()} (${info.row.original.hdSizeGb} GB)`,
		}),
		invColHelper.accessor("osName", {
			header: "OS",
			cell: (info) => `${info.getValue()} ${info.row.original.osVersion}`,
		}),
	];

	const invTable = useReactTable({
		data: rows,
		columns: invColumns,
		getCoreRowModel: getCoreRowModel(),
	});

	return (
		<div className="flex flex-col gap-4">
			<div className="flex justify-end">
				<Button
					variant="outline"
					size="sm"
					disabled={rows.length === 0}
					onClick={() =>
						downloadCsv("host-inventory.csv", [
							[
								"Host",
								"CPU",
								"Cores",
								"RAM (MiB)",
								"Disk Model",
								"Disk (GB)",
								"OS",
								"Serial",
							],
							...rows.map((r) => [
								r.hostName,
								r.cpuModel,
								String(r.cpuCores),
								String(r.ramMib),
								r.hdModel,
								String(r.hdSizeGb),
								`${r.osName} ${r.osVersion}`,
								r.serial ?? "",
							]),
						])
					}
				>
					<Download data-icon="inline-start" />
					Export CSV
				</Button>
			</div>
			<div className="rounded-lg border">
				<Table>
					<TableHeader>
						{invTable.getHeaderGroups().map((headerGroup) => (
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
						{inventoryQuery.isLoading ? (
							<TableSkeleton columns={5} />
						) : invTable.getRowModel().rows.length === 0 ? (
							<TableRow>
								<TableCell colSpan={5}>
									<EmptyState title="No inventory data" />
								</TableCell>
							</TableRow>
						) : (
							invTable.getRowModel().rows.map((row) => (
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
							))
						)}
					</TableBody>
				</Table>
			</div>
		</div>
	);
}

function ReportsPage() {
	return (
		<div className="flex flex-col gap-6">
			<div>
				<h1 className="text-2xl font-bold">Reports</h1>
				<p className="text-muted-foreground">
					Imaging history and host inventory
				</p>
			</div>

			<Tabs defaultValue="imaging">
				<TabsList>
					<TabsTrigger value="imaging">Imaging History</TabsTrigger>
					<TabsTrigger value="inventory">Host Inventory</TabsTrigger>
				</TabsList>
				<TabsContent value="imaging">
					<ImagingHistoryTab />
				</TabsContent>
				<TabsContent value="inventory">
					<HostInventoryTab />
				</TabsContent>
			</Tabs>
		</div>
	);
}
