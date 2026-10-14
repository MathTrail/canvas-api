/**
 * App is the SPA's placeholder page, until the epic MVP draws the sheet: the
 * product's working name and the build the page came from.
 */
export function App({ version }: Readonly<{ version: string }>) {
	return (
		<main>
			<h1>MathTrail Canvas</h1>
			<p>Build {version}</p>
		</main>
	);
}
