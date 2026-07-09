import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { RouteError } from "@/components/app/route-error";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { api } from "@/lib/api";
import type { Host } from "@/types";
import { HostInfoCard } from "@/components/hosts/host-info-card";
import { InventoryCard } from "@/components/hosts/inventory-card";
import { MACAddressesCard } from "@/components/hosts/mac-addresses-card";
import { TaskCard } from "@/components/hosts/task-card";

export const Route = createFileRoute("/_auth/hosts/$id")({
	component: HostDetailPage,
	errorComponent: RouteError,
});

function HostDetailPage() {
	const { id } = Route.useParams();

	const hostQuery = useQuery({
		queryKey: ["host", id],
		queryFn: () => api.get<Host>(`/hosts/${id}`),
	});

	if (hostQuery.isLoading) {
		return (
			<div className="flex flex-col gap-4">
				<Skeleton className="h-8 w-48" />
				<Skeleton className="h-64 w-full" />
			</div>
		);
	}

	const host = hostQuery.data;

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
				</TabsList>

				<TabsContent value="info">
					<HostInfoCard host={host} />
				</TabsContent>

				<TabsContent value="macs">
					<MACAddressesCard hostId={id} />
				</TabsContent>

				<TabsContent value="inventory">
					<InventoryCard hostId={id} />
				</TabsContent>

				<TabsContent value="tasks">
					<div className="flex flex-col gap-6">
						<TaskCard hostId={id} />
					</div>
				</TabsContent>
			</Tabs>
		</div>
	);
}
