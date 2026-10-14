// The npm side of the repository's license rules, read from the lockfile of
// the SPA. Every package the lockfile pins — what the SPA is built from and
// the tools that build it — must carry a license from the allowed list; a
// build tool may be excepted by its name and its license, and only while it
// stays a tool. The packages the SPA is built from are listed, with their
// licenses, for the file of third-party licenses.
//
//	node scripts/licenses.ts check --allowed MIT,ISC --except lightningcss=MPL-2.0
//	node scripts/licenses.ts list

import { readFileSync } from "node:fs";
import { join } from "node:path";
import { parseArgs } from "node:util";

/** Lockfile is the part of package-lock.json the rules read. */
export type Lockfile = {
	packages: Record<
		string,
		{
			name?: string;
			version?: string;
			license?: string;
			dev?: boolean;
			optionalDependencies?: Record<string, string>;
		}
	>;
};

/** LockedPackage is one package the lockfile pins. */
export type LockedPackage = {
	name: string;
	version: string;
	/** license is the package's SPDX expression, empty when it names none. */
	license: string;
	/** shipped is true for a package the SPA can be built from. */
	shipped: boolean;
	/**
	 * optional names the packages it may use when there is one for the
	 * platform: a tool's build for each system, most often.
	 */
	optional: string[];
};

/** Exception lets a build tool carry one license outside the allowed list. */
export type Exception = { name: string; license: string };

const modules = "node_modules/";

/** lockedPackages lists every package the lockfile pins but the project. */
export function lockedPackages(lock: Lockfile): LockedPackage[] {
	return Object.entries(lock.packages)
		.filter(([path]) => path !== "")
		.map(([path, entry]) => ({
			name:
				entry.name ?? path.slice(path.lastIndexOf(modules) + modules.length),
			version: entry.version ?? "",
			license: entry.license ?? "",
			// Only a package every production path can do without is a tool.
			// One that is optional for production but needed in development is
			// marked neither way, and counts as shipped.
			shipped: entry.dev !== true,
			optional: Object.keys(entry.optionalDependencies ?? {}),
		}));
}

/**
 * satisfies says whether an SPDX license expression is met by the allowed
 * licenses alone: either side of an OR will do, both sides of an AND must, and
 * AND binds tighter. A license with an exception clause is not on any list, and
 * an expression that does not parse is met by nothing.
 */
export function satisfies(
	expression: string,
	allowed: ReadonlySet<string>,
): boolean {
	const tokens = expression.match(/\(|\)|[^\s()]+/g) ?? [];
	// The tokens that are never a name: neither a license nor an exception.
	const syntax: ReadonlySet<string> = new Set(["(", ")", "AND", "OR", "WITH"]);
	let at = 0;

	// Each operand is read before the operator's result is decided, so that
	// the whole expression is parsed whatever its first half settled.
	const anyOf = (): boolean => {
		let met = allOf();
		while (tokens[at] === "OR") {
			at++;
			const next = allOf();
			met = met || next;
		}
		return met;
	};
	const allOf = (): boolean => {
		let met = operand();
		while (tokens[at] === "AND") {
			at++;
			const next = operand();
			met = met && next;
		}
		return met;
	};
	const operand = (): boolean => {
		const token = tokens[at++];
		if (token === "(") {
			const met = anyOf();
			if (tokens[at++] !== ")") {
				throw new SyntaxError("unclosed parenthesis");
			}
			return met;
		}
		if (token === undefined || syntax.has(token)) {
			throw new SyntaxError(`a license expected, ${token ?? "the end"} found`);
		}
		if (tokens[at] === "WITH") {
			// The exception is a name too: an operator or a parenthesis in its
			// place is a sentence that does not parse.
			const exception = tokens[at + 1];
			if (exception === undefined || syntax.has(exception)) {
				throw new SyntaxError(
					`an exception expected, ${exception ?? "the end"} found`,
				);
			}
			at += 2;
			return false;
		}
		return allowed.has(token);
	};

	try {
		const met = anyOf();
		return met && at === tokens.length;
	} catch {
		return false;
	}
}

/**
 * refusals names every package whose license the rules do not allow: one
 * outside the allowed list that no exception covers. An exception covers the
 * build tool it names and the builds for each platform that tool itself lists
 * as optional, when they carry exactly its license — and no other package,
 * however alike its name.
 */
export function refusals(
	packages: readonly LockedPackage[],
	allowed: ReadonlySet<string>,
	exceptions: readonly Exception[],
): string[] {
	const coveredBy = (e: Exception) =>
		new Set([
			e.name,
			...packages
				.filter((p) => p.name === e.name && !p.shipped)
				.flatMap((p) => p.optional),
		]);
	const covered = exceptions.map((e) => ({
		license: e.license,
		names: coveredBy(e),
	}));
	const excepted = (p: LockedPackage) =>
		!p.shipped &&
		covered.some((c) => c.names.has(p.name) && p.license === c.license);
	return packages
		.filter((p) => !satisfies(p.license, allowed) && !excepted(p))
		.map(
			(p) =>
				`${p.name}@${p.version} is licensed ${p.license === "" ? "under nothing it names" : p.license}, which is not on the list; it is ${p.shipped ? "part of what the SPA is built from" : "a build tool"}`,
		);
}

/**
 * listing is one line per package the SPA is built from: its license, the
 * package at its exact version, and that version's page in the registry. The
 * columns are those of the Go modules above them in the same file.
 */
export function listing(packages: readonly LockedPackage[]): string[] {
	const lines = packages
		.filter((p) => p.shipped)
		.map((p) => ({ id: `${p.name}@${p.version}`, p }))
		.sort((a, b) => (a.id < b.id ? -1 : Number(a.id > b.id)))
		.map(
			({ id, p }) =>
				`${p.license.padEnd(13)} ${id.padEnd(46)} https://www.npmjs.com/package/${p.name}/v/${p.version}`,
		);
	return [...new Set(lines)];
}

/** parseExceptions reads exceptions written as name=license, comma-separated. */
export function parseExceptions(written: string): Exception[] {
	return written
		.split(",")
		.filter((part) => part !== "")
		.map((part) => {
			const [name, license, ...rest] = part.split("=");
			if (!name || !license || rest.length > 0) {
				throw new SyntaxError(`"${part}" is not name=license`);
			}
			return { name, license };
		});
}

const usage =
	"usage: licenses.ts list | check --allowed <license,...> [--except <name=license,...>]";

/**
 * run does what the command line asks of the lockfile, writing lines to out
 * and complaints to err, and returns the exit status.
 */
export function run(
	args: string[],
	lock: Lockfile,
	out: (line: string) => void,
	err: (line: string) => void,
): number {
	let parsed: ReturnType<typeof parseCommandLine>;
	let exceptions: Exception[];
	try {
		parsed = parseCommandLine(args);
		exceptions = parseExceptions(parsed.values.except ?? "");
	} catch (error) {
		err(`licenses: ${error instanceof Error ? error.message : String(error)}`);
		err(usage);
		return 2;
	}
	const [command, ...extra] = parsed.positionals;
	const packages = lockedPackages(lock);

	if (command === "list" && extra.length === 0) {
		for (const line of listing(packages)) {
			out(line);
		}
		return 0;
	}
	if (command === "check" && extra.length === 0 && parsed.values.allowed) {
		const allowed = new Set(parsed.values.allowed.split(","));
		const refused = refusals(packages, allowed, exceptions);
		for (const line of refused) {
			err(`licenses: ${line}`);
		}
		return refused.length === 0 ? 0 : 1;
	}
	err(usage);
	return 2;
}

function parseCommandLine(args: string[]) {
	return parseArgs({
		args,
		allowPositionals: true,
		options: {
			allowed: { type: "string" },
			except: { type: "string" },
		},
	});
}

if (import.meta.main) {
	const lock: Lockfile = JSON.parse(
		readFileSync(join(import.meta.dirname, "..", "package-lock.json"), "utf8"),
	);
	process.exitCode = run(
		process.argv.slice(2),
		lock,
		(line) => console.log(line),
		(line) => console.error(line),
	);
}
