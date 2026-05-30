import { Database, type Icon } from "@phosphor-icons/react";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";

interface EmptyStateProps {
	icon?: Icon;
	title: string;
	description?: string;
	action?: {
		label: string;
		onClick: () => void;
	};
	children?: ReactNode;
}

export function EmptyState({
	icon: Icon = Database,
	title,
	description,
	action,
	children,
}: EmptyStateProps) {
	return (
		<div className="flex flex-col items-center justify-center py-12 text-center">
			<Icon className="size-10 text-muted-foreground/40" weight="duotone" />
			<p className="mt-4 text-sm font-medium text-muted-foreground">{title}</p>
			{description && (
				<p className="mt-1 text-xs text-muted-foreground/60">{description}</p>
			)}
			{action && (
				<Button
					variant="outline"
					size="sm"
					className="mt-4"
					onClick={action.onClick}
				>
					{action.label}
				</Button>
			)}
			{children}
		</div>
	);
}
