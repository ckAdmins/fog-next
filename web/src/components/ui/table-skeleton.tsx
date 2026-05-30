import { Skeleton } from "@/components/ui/skeleton";
import { TableCell, TableRow } from "@/components/ui/table";

interface TableSkeletonProps {
	columns: number;
	rows?: number;
}

export function TableSkeleton({ columns, rows = 5 }: TableSkeletonProps) {
	return Array.from({ length: rows }).map((_, i) => (
		// biome-ignore lint/suspicious/noArrayIndexKey: static array, stable order
		<TableRow key={`skel-r-${i}`}>
			{Array.from({ length: columns }).map((_, j) => (
				// biome-ignore lint/suspicious/noArrayIndexKey: static array, stable order
				<TableCell key={`skel-c-${i}-${j}`}>
					<Skeleton className="h-4 w-full" />
				</TableCell>
			))}
		</TableRow>
	));
}
