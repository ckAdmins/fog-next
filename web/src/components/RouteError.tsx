import { useNavigate } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { ApiError } from "@/lib/api";

interface RouteErrorProps {
	error: unknown;
	info?: { componentStack?: string };
	reset: () => void;
}

function errorDetails(error: unknown): { title: string; description: string } {
	if (error instanceof ApiError) {
		if (error.status === 404) {
			return {
				title: "Not found",
				description: "The requested resource could not be found.",
			};
		}
		if (error.status === 401 || error.status === 403) {
			return {
				title: "Access denied",
				description: "You don't have permission to view this page.",
			};
		}
		if (error.status >= 500) {
			return {
				title: "Server error",
				description: "The server encountered an error. Please try again later.",
			};
		}
		return {
			title: "Request failed",
			description: error.message || "Something went wrong with the request.",
		};
	}

	if (error instanceof Error) {
		return {
			title: "Unexpected error",
			description: "An unexpected error occurred while loading this page.",
		};
	}

	return {
		title: "Something went wrong",
		description: "An unknown error occurred.",
	};
}

export function RouteError({ error, reset }: RouteErrorProps) {
	const navigate = useNavigate();
	const { title, description } = errorDetails(error);

	const message =
		error instanceof Error
			? error.message
			: error != null
				? String(error)
				: "Unknown error";

	return (
		<div className="flex min-h-[400px] items-center justify-center p-8">
			<Card className="max-w-md">
				<CardHeader>
					<CardTitle>{title}</CardTitle>
					<CardDescription>{description}</CardDescription>
				</CardHeader>
				<CardContent className="flex flex-col gap-4">
					<code className="rounded bg-muted p-2 text-xs font-mono break-all">
						{message}
					</code>
					<div className="flex gap-2">
						<Button
							variant="outline"
							size="sm"
							onClick={() => navigate({ to: "/dashboard" })}
						>
							Go to Dashboard
						</Button>
						<Button variant="default" size="sm" onClick={reset}>
							Try again
						</Button>
					</div>
				</CardContent>
			</Card>
		</div>
	);
}
