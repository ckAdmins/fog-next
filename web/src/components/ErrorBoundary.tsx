import { Component, type ErrorInfo, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface Props {
	children: ReactNode;
	fallback?: ReactNode;
}

interface State {
	hasError: boolean;
	error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
	constructor(props: Props) {
		super(props);
		this.state = { hasError: false, error: null };
	}

	static getDerivedStateFromError(error: Error): State {
		return { hasError: true, error };
	}

	componentDidCatch(error: Error, info: ErrorInfo) {
		console.error("ErrorBoundary caught:", error, info.componentStack);
	}

	render() {
		if (this.state.hasError) {
			if (this.props.fallback) return this.props.fallback;

			return (
				<div className="flex min-h-[400px] items-center justify-center p-8">
					<Card className="max-w-md border-destructive/30">
						<CardHeader>
							<CardTitle className="text-destructive">
								Something went wrong
							</CardTitle>
						</CardHeader>
						<CardContent className="flex flex-col gap-4">
							<p className="text-sm text-muted-foreground">
								An unexpected error occurred while rendering this page.
							</p>
							<code className="rounded bg-muted p-2 text-xs font-mono text-destructive break-all">
								{this.state.error?.message ?? "Unknown error"}
							</code>
							<div className="flex gap-2">
								<Button
									variant="outline"
									size="sm"
									onClick={() => (window.location.href = "/dashboard")}
								>
									Go to Dashboard
								</Button>
								<Button
									variant="default"
									size="sm"
									onClick={() =>
										this.setState({
											hasError: false,
											error: null,
										})
									}
								>
									Try again
								</Button>
							</div>
						</CardContent>
					</Card>
				</div>
			);
		}

		return this.props.children;
	}
}
