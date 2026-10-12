import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { App } from "./App";

let container: HTMLDivElement;
let root: Root;

beforeEach(() => {
	// act() carries a render through to the end only where the environment
	// says it is a test; elsewhere React warns that it does not.
	vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
	container = document.createElement("div");
	document.body.append(container);
	root = createRoot(container);
});

afterEach(() => {
	act(() => root.unmount());
	container.remove();
	vi.unstubAllGlobals();
});

describe("the placeholder page", () => {
	test("names the product and the build it came from", () => {
		act(() => root.render(<App version="v0.1.0-3-g1a2b3c4" />));

		expect(container.querySelector("h1")?.textContent).toBe("MathTrail Canvas");
		expect(container.querySelector("p")?.textContent).toBe(
			"Build v0.1.0-3-g1a2b3c4",
		);
	});
});
