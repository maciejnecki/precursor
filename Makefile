# VERSION is the human-readable build identifier baked into release builds. It is
# the nearest git tag when one exists, falling back to the short commit hash.
VERSION := $(shell git describe --tags --always)

# build produces a release bundle with the git version injected; the app shows it
# in the settings footer. Dev runs (make dev) keep the compiled-in "dev" default.
.PHONY: build
build:
	wails build -ldflags "-X main.version=$(VERSION)"

# dev runs the app with live reload and the version left as "dev".
.PHONY: dev
dev:
	wails dev
