import { defineConfig } from "vitest/config";

// The tests load no plugin, and Vitest reads no vite.config.ts beside this
// file: they compile JSX from tsconfig.json, whose "react-jsx" is the default
// the build's plugin compiles with too (vite.config.ts).
export default defineConfig({
	test: {
		environment: "happy-dom",
		include: ["src/**/*.test.{ts,tsx}", "scripts/**/*.test.ts"],
		coverage: {
			provider: "v8",
			// The code, not its tests or its declarations. The entry point only
			// starts what the tests cover.
			include: ["src/**/*.{ts,tsx}", "scripts/**/*.ts"],
			exclude: ["**/*.test.{ts,tsx}", "**/*.d.ts", "src/main.tsx"],
			reportsDirectory: "coverage",
			// Paths are written from the repository's root, where the readers of
			// the report look for the files.
			reporter: [["lcovonly", { projectRoot: ".." }]],
		},
	},
});
