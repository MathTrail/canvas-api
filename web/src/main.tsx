import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { buildVersion } from "./version";

const root = document.getElementById("root");
if (root === null) {
	throw new Error("index.html has no #root for the SPA to draw into");
}
createRoot(root).render(
	<StrictMode>
		<App version={buildVersion()} />
	</StrictMode>,
);
