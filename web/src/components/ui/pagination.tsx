import { CaretLeft, CaretRight } from "@phosphor-icons/react";
import {
	type ChangeEvent,
	type KeyboardEvent,
	useCallback,
	useState,
} from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";

interface PaginationProps {
	page: number;
	pageSize: number;
	total: number;
	onPageChange: (page: number) => void;
	onPageSizeChange?: (size: number) => void;
	pageSizeOptions?: number[];
}

const DEFAULT_PAGE_SIZES = [10, 25, 50, 100];

export function Pagination({
	page,
	pageSize,
	total,
	onPageChange,
	onPageSizeChange,
	pageSizeOptions = DEFAULT_PAGE_SIZES,
}: PaginationProps) {
	const totalPages = Math.max(1, Math.ceil(total / pageSize));
	const [jumpValue, setJumpValue] = useState("");

	const goToPage = useCallback(
		(target: number) => {
			const clamped = Math.max(1, Math.min(target, totalPages));
			if (clamped !== page) onPageChange(clamped);
		},
		[page, totalPages, onPageChange],
	);

	const handleJump = () => {
		const n = Number.parseInt(jumpValue, 10);
		if (Number.isFinite(n)) goToPage(n);
		setJumpValue("");
	};

	const handleJumpKey = (e: KeyboardEvent<HTMLInputElement>) => {
		if (e.key === "Enter") handleJump();
	};

	return (
		<div className="flex items-center justify-between gap-4">
			{onPageSizeChange && (
				<div className="flex items-center gap-2">
					<Label className="text-xs text-muted-foreground">Rows</Label>
					<Select
						value={String(pageSize)}
						onValueChange={(v) => onPageSizeChange(Number(v))}
					>
						<SelectTrigger className="h-8 w-[70px] text-xs">
							<SelectValue />
						</SelectTrigger>
						<SelectContent>
							{pageSizeOptions.map((s) => (
								<SelectItem key={s} value={String(s)}>
									{s}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</div>
			)}

			<div className="flex items-center gap-2 ml-auto">
				<Button
					variant="outline"
					size="sm"
					disabled={page <= 1}
					onClick={() => onPageChange(page - 1)}
					aria-label="Previous page"
				>
					<CaretLeft />
					Prev
				</Button>

				<span className="min-w-[80px] text-center text-sm text-muted-foreground tabular-nums">
					{page} / {totalPages}
				</span>

				<Button
					variant="outline"
					size="sm"
					disabled={page >= totalPages}
					onClick={() => onPageChange(page + 1)}
					aria-label="Next page"
				>
					Next
					<CaretRight />
				</Button>
			</div>

			<div className="flex items-center gap-1.5">
				<input
					type="number"
					min={1}
					max={totalPages}
					placeholder="#"
					value={jumpValue}
					onChange={(e: ChangeEvent<HTMLInputElement>) =>
						setJumpValue(e.target.value)
					}
					onKeyDown={handleJumpKey}
					aria-label="Jump to page"
					className="h-8 w-12 rounded-md border bg-transparent px-2 text-center text-xs tabular-nums [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
				/>
				<Button
					variant="ghost"
					size="sm"
					className="h-8 px-2 text-xs"
					onClick={handleJump}
				>
					Go
				</Button>
			</div>
		</div>
	);
}
