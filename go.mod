module github.com/MathTrail/canvas-api

go 1.27.2

// The SPA's packages in web/node_modules are not this module's, but an npm
// package may carry Go files of its own, and ./... would then build, test and
// lint them as packages of this module. GO_FILES in the justfile and the
// pre-commit hook keep gofmt out of node_modules in the same way.
ignore node_modules
