// dev is what a build that was told no version calls itself.
const dev = "dev";

/**
 * buildVersion is the version of the build the SPA came from — a release, or
 * how far past one the build is — or "dev" when the build did not say. The
 * page shows it, so that which build Hosting serves can be read off the page
 * itself.
 */
export function buildVersion(): string {
	return import.meta.env.VITE_VERSION || dev;
}
