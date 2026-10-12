import { describe, expect, test } from "vitest";
import {
	type LockedPackage,
	type Lockfile,
	listing,
	lockedPackages,
	parseExceptions,
	refusals,
	run,
	satisfies,
} from "./licenses.ts";

const allowed = new Set(["MIT", "ISC", "Apache-2.0"]);

describe("satisfies", () => {
	test.each([
		["MIT", true],
		["GPL-3.0", false],
		["MIT OR GPL-3.0", true],
		["GPL-3.0 OR MIT", true],
		["(MIT OR GPL-3.0)", true],
		["MIT AND ISC", true],
		["MIT AND GPL-3.0", false],
		["MIT AND (ISC OR GPL-3.0)", true],
		// AND binds tighter than OR.
		["GPL-3.0 OR MIT AND ISC", true],
		["MIT AND GPL-3.0 OR BSD-4-Clause", false],
		["GPL-2.0 WITH Classpath-exception-2.0", false],
		["MIT WITH Some-exception OR ISC", true],
		["GPL-2.0+", false],
		// Nothing, or a sentence that does not parse, is met by nothing.
		["", false],
		["(MIT", false],
		["MIT OR", false],
		["MIT)", false],
		["OR MIT", false],
		// WITH names an exception; an operator or a parenthesis in its place
		// is a sentence that does not parse, whatever the other branch says.
		["MIT WITH", false],
		["MIT WITH ) OR ISC", false],
		["MIT WITH OR OR ISC", false],
		["MIT WITH ( OR ISC", false],
		["MIT WITH Some-exception WITH Other OR ISC", false],
		["(MIT) WITH Some-exception OR ISC", false],
	])("%j is %s", (expression, want) => {
		expect(satisfies(expression, allowed)).toBe(want);
	});
});

// The license field is written by each package's authors, so the parser is
// held to three properties over every expression these tokens make, up to five
// of them, whether it parses or not, rather than over the cases a table
// thought of: an empty list accepts nothing, a longer list accepts no less,
// and nothing is accepted that is not a sentence at all.
describe("satisfies, over every expression of up to five tokens", () => {
	const tokens = ["MIT", "GPL-3.0", "AND", "OR", "WITH", "(", ")"];
	const none = new Set<string>();
	const mit = new Set(["MIT"]);
	const both = new Set(["MIT", "GPL-3.0"]);

	// wellFormed judges a sentence by what may follow what, a judge of its
	// own that shares no code with the parser: an operand opens the sentence
	// and follows an operator or "("; WITH follows a license and comes before
	// its exception; parentheses close, and the sentence ends on an operand.
	function wellFormed(parts: readonly string[]): boolean {
		let depth = 0;
		let last: "open" | "license" | "with" | "exception" | "close" = "open";
		for (const part of parts) {
			const operator = part === "AND" || part === "OR";
			if (last === "with") {
				if (operator || ["WITH", "(", ")"].includes(part)) return false;
				last = "exception";
			} else if (last === "open") {
				if (operator || part === "WITH" || part === ")") return false;
				if (part === "(") depth++;
				last = part === "(" ? "open" : "license";
			} else if (operator) {
				last = "open";
			} else if (part === ")" && depth > 0) {
				depth--;
				last = "close";
			} else if (part === "WITH" && last === "license") {
				last = "with";
			} else {
				return false;
			}
		}
		return depth === 0 && ["license", "exception", "close"].includes(last);
	}

	function* expressions(length: number): Generator<string[]> {
		if (length === 0) {
			yield [];
			return;
		}
		for (const head of tokens) {
			for (const tail of expressions(length - 1)) {
				yield [head, ...tail];
			}
		}
	}

	test("accepts nothing on an empty list, no less on a longer one, and only sentences", () => {
		const broken: string[] = [];
		for (let length = 1; length <= 5; length++) {
			for (const parts of expressions(length)) {
				const expression = parts.join(" ");
				if (satisfies(expression, none)) {
					broken.push(`${expression}: met with no license allowed`);
				}
				if (satisfies(expression, mit) && !satisfies(expression, both)) {
					broken.push(`${expression}: met by MIT, not by MIT and GPL-3.0`);
				}
				if (satisfies(expression, both) && !wellFormed(parts)) {
					broken.push(`${expression}: met, though it is no sentence`);
				}
			}
		}
		expect(broken).toEqual([]);
	});
});

const lock: Lockfile = {
	packages: {
		"": { name: "canvas-web", license: "MIT" },
		"node_modules/preact": { version: "10.29.8", license: "MIT" },
		"node_modules/@scope/lib": { version: "2.0.0", license: "ISC" },
		"node_modules/a/node_modules/b": { version: "1.0.0", license: "MIT" },
		"node_modules/aliased": {
			name: "real-name",
			version: "3.0.0",
			license: "MIT",
		},
		"node_modules/tool": {
			version: "1.0.0",
			license: "MIT",
			dev: true,
			optionalDependencies: { "tool-linux-x64": "1.0.0" },
		},
		"node_modules/nameless": { version: "0.1.0" },
	},
};

describe("lockedPackages", () => {
	test("names every package the lockfile pins, the project aside", () => {
		expect(lockedPackages(lock)).toEqual([
			pkg("preact", "MIT", true, "10.29.8"),
			pkg("@scope/lib", "ISC", true, "2.0.0"),
			pkg("b", "MIT", true),
			pkg("real-name", "MIT", true, "3.0.0"),
			pkg("tool", "MIT", false, "1.0.0", ["tool-linux-x64"]),
			pkg("nameless", "", true, "0.1.0"),
		]);
	});
});

function pkg(
	name: string,
	license: string,
	shipped: boolean,
	version = "1.0.0",
	optional: string[] = [],
): LockedPackage {
	return { name, version, license, shipped, optional };
}

describe("refusals", () => {
	const exceptions = [{ name: "lightningcss", license: "MPL-2.0" }];

	test("allows what the list allows", () => {
		expect(
			refusals([pkg("a", "MIT", true), pkg("b", "ISC", false)], allowed, []),
		).toEqual([]);
	});

	test("refuses a license outside the list, and one that names nothing", () => {
		const refused = refusals(
			[pkg("copyleft", "GPL-3.0", true), pkg("silent", "", false)],
			allowed,
			[],
		);
		expect(refused).toHaveLength(2);
		expect(refused[0]).toContain("copyleft@1.0.0 is licensed GPL-3.0");
		expect(refused[0]).toContain("part of what the SPA is built from");
		expect(refused[1]).toContain("silent@1.0.0");
		expect(refused[1]).toContain("a build tool");
	});

	// The tool, as the lockfile has it: a build tool that lists its build for
	// one platform as optional.
	const tool = pkg("lightningcss", "MPL-2.0", false, "1.0.0", [
		"lightningcss-linux-x64-gnu",
	]);

	test("excepts the tool it names and the builds the tool lists", () => {
		expect(
			refusals(
				[tool, pkg("lightningcss-linux-x64-gnu", "MPL-2.0", false)],
				allowed,
				exceptions,
			),
		).toEqual([]);
	});

	test.each([
		["once it is shipped", pkg("lightningcss", "MPL-2.0", true)],
		["under another license", pkg("lightningcss", "GPL-3.0", false)],
		[
			"for a name that only starts the same",
			pkg("lightningcssx", "MPL-2.0", false),
		],
		[
			"for a package the tool does not list, however alike its name",
			pkg("lightningcss-plugin", "MPL-2.0", false),
		],
		[
			"for a build the tool lists, once that build is shipped",
			pkg("lightningcss-linux-x64-gnu", "MPL-2.0", true),
		],
	])("does not except it %s", (_, refused) => {
		expect(refusals([tool, refused], allowed, exceptions)).toHaveLength(1);
	});
});

describe("listing", () => {
	test("lists what ships, once each, in order, with its registry page", () => {
		const lines = listing([
			pkg("zod", "MIT", true, "4.6.5"),
			pkg("tool", "MIT", false),
			pkg("@scope/lib", "ISC", true, "2.0.0"),
			pkg("zod", "MIT", true, "4.6.5"),
		]);
		expect(lines).toEqual([
			`${"ISC".padEnd(13)} ${"@scope/lib@2.0.0".padEnd(46)} https://www.npmjs.com/package/@scope/lib/v/2.0.0`,
			`${"MIT".padEnd(13)} ${"zod@4.6.5".padEnd(46)} https://www.npmjs.com/package/zod/v/4.6.5`,
		]);
	});
});

describe("parseExceptions", () => {
	test("reads name=license pairs", () => {
		expect(parseExceptions("a=MPL-2.0,b=CC0-1.0")).toEqual([
			{ name: "a", license: "MPL-2.0" },
			{ name: "b", license: "CC0-1.0" },
		]);
		expect(parseExceptions("")).toEqual([]);
	});

	test.each(["a", "=MPL-2.0", "a=", "a=b=c"])("refuses %j", (written) => {
		expect(() => parseExceptions(written)).toThrow(SyntaxError);
	});
});

describe("run", () => {
	function capture(args: string[]) {
		const out: string[] = [];
		const err: string[] = [];
		const status = run(
			args,
			lock,
			(line) => out.push(line),
			(line) => err.push(line),
		);
		return { status, out, err };
	}

	test("list writes what ships", () => {
		const { status, out } = capture(["list"]);
		expect(status).toBe(0);
		expect(out).toHaveLength(5);
		expect(out.some((line) => line.includes("tool@"))).toBe(false);
	});

	test("check passes a lockfile the list allows", () => {
		const err: string[] = [];
		const status = run(
			["check", "--allowed", "MIT,ISC"],
			{ packages: { "node_modules/preact": { version: "1", license: "MIT" } } },
			() => {},
			(line) => err.push(line),
		);
		expect(status).toBe(0);
		expect(err).toEqual([]);
	});

	test("check refuses an exception that is not name=license", () => {
		const { status, err } = capture([
			"check",
			"--allowed",
			"MIT,ISC",
			"--except",
			"nameless=",
		]);
		expect(status).toBe(2);
		expect(err[0]).toContain("is not name=license");
	});

	test("check fails, naming what it refused", () => {
		const { status, err } = capture(["check", "--allowed", "MIT,ISC"]);
		expect(status).toBe(1);
		expect(err).toEqual([
			expect.stringContaining(
				"nameless@0.1.0 is licensed under nothing it names",
			),
		]);
	});

	test.each([
		[[]],
		[["check"]],
		[["list", "extra"]],
		[["publish"]],
		[["check", "--bogus"]],
	])("%j is not a command it knows", (args) => {
		const { status, err } = capture(args);
		expect(status).toBe(2);
		expect(err.at(-1)).toContain("usage:");
	});
});
