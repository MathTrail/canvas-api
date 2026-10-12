import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// The SPA, built into web/dist for Firebase Hosting to serve (R10). The plugin
// compiles JSX and, in the dev server, refreshes an edited component in place
// rather than reloading the page (R47). It compiles with React's automatic
// runtime unless told otherwise, which is what "react-jsx" in tsconfig.json
// says to the type checker and to the tests, which never load this file. It is
// told nothing, so that both stay at that default: a change to how JSX is
// compiled is made in both places.
export default defineConfig({
	root: import.meta.dirname,
	plugins: [react()],
	server: {
		// The address and the port the editor forwards from the container. A
		// server left to name localhost itself listens on IPv6's alone, which a
		// forwarded port does not reach, and one that moved to another port
		// when this one is taken would be out of reach as well.
		host: "127.0.0.1",
		port: 5173,
		strictPort: true,
	},
});
