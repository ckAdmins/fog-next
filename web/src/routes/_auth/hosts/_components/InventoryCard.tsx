import { useQuery } from "@tanstack/react-query";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { api } from "@/lib/api";
import type { Inventory } from "@/types";

export function InventoryCard({ hostId }: { hostId: string }) {
	const inventoryQuery = useQuery({
		queryKey: ["host-inventory", hostId],
		queryFn: () => api.get<Inventory>(`/hosts/${hostId}/inventory`),
	});

	return (
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
					<p className="text-muted-foreground">No inventory data available</p>
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
	);
}
