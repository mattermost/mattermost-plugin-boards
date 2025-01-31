.PHONY: prebuild clean cleanall ci server server-mac server-linux server-win server-linux-package generate watch-server webapp mac-app win-app-wpf linux-app modd-precheck templates-archive

PACKAGE_FOLDER = focalboard

# Build Flags
BUILD_NUMBER ?= $(BUILD_NUMBER:)
BUILD_DATE = $(shell date -u)
BUILD_HASH = $(shell git rev-parse HEAD)
# If we don't set the build number it defaults to dev
ifeq ($(BUILD_NUMBER),)
	BUILD_NUMBER := dev
	BUILD_DATE := n/a
endif

MM_SERVER_PATH ?= $(MM_SERVER_PATH:)
ifeq ($(MM_SERVER_PATH),)
	MM_SERVER_PATH := ../mattermost
endif

BUILD_TAGS += json1 sqlite3

LDFLAGS += -X "github.com/mattermost/focalboard/server/model.BuildNumber=$(BUILD_NUMBER)"
LDFLAGS += -X "github.com/mattermost/focalboard/server/model.BuildDate=$(BUILD_DATE)"
LDFLAGS += -X "github.com/mattermost/focalboard/server/model.BuildHash=$(BUILD_HASH)"
LDFLAGS += -X "github.com/mattermost/focalboard/server/model.Edition=plugin"

GO ?= $(shell command -v go 2> /dev/null)
NPM ?= $(shell command -v npm 2> /dev/null)
CURL ?= $(shell command -v curl 2> /dev/null)
MM_DEBUG ?=
MANIFEST_FILE ?= plugin.json
GOPATH ?= $(shell go env GOPATH)
GO_TEST_FLAGS ?= -race
GO_BUILD_FLAGS ?= -ldflags '$(LDFLAGS)'
MM_UTILITIES_DIR ?= ../mattermost-utilities
DLV_DEBUG_PORT := 2346
MATTERMOST_PLUGINS_PATH=$(MM_SERVER_PATH)/plugins
BOARD_PLUGIN_PATH=$(MATTERMOST_PLUGINS_PATH)/boards
PLUGIN_NAME=boards

export GO111MODULE=on

ASSETS_DIR ?= assets

RACE = -race

## Define the default target (make all)
.PHONY: default
default: all

# Verify environment, and define PLUGIN_ID, PLUGIN_VERSION, HAS_SERVER and HAS_WEBAPP as needed.
include build/setup.mk

BUNDLE_NAME ?= $(PLUGIN_NAME)-$(PLUGIN_VERSION).tar.gz

# Include custom makefile, if present
ifneq ($(wildcard build/custom.mk),)
	include build/custom.mk
endif

## Checks the code style, tests, builds and bundles the plugin.
.PHONY: all
all: check-style test dist

## Propagates plugin manifest information into the server/ and webapp/ folders.
.PHONY: apply
apply:
	./build/bin/manifest apply

## Runs eslint and golangci-lint
.PHONY: check-style
check-style: webapp/node_modules
	@echo Checking for style guide compliance

ifneq ($(HAS_WEBAPP),)
	cd webapp && npm run check
	cd webapp && npm run check-types
endif

ifneq ($(HAS_SERVER),)
	@if ! [ -x "$$(command -v golangci-lint)" ]; then \
		echo "golangci-lint is not installed. Please see https://github.com/golangci/golangci-lint#install-golangci-lint for installation instructions."; \
		exit 1; \
	fi; \

	@echo Running golangci-lint
	cd server && golangci-lint run ./...
	$(GO) install github.com/mattermost/mattermost-govet/v2@3f08281c344327ac09364f196b15f9a81c7eff08
	$(GO) vet -vettool=$(GOBIN)/mattermost-govet -license -license.year=2020 ./server/...
endif

templates-archive: ## Build templates archive file
	cd ./server/assets/build-template-archive; go run -tags '$(BUILD_TAGS)' main.go --dir="../templates-boardarchive" --out="../templates.boardarchive"

## Builds the server, if it exists, for all supported architectures.
.PHONY: server
server: templates-archive
ifneq ($(HAS_SERVER),)
	mkdir -p server/dist;
ifeq ($(MM_DEBUG),)
	cd server && env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -o dist/plugin-linux-amd64;
	cd server && env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -o dist/plugin-linux-arm64;
	cd server && env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -o dist/plugin-darwin-amd64;
	cd server && env CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -o dist/plugin-darwin-arm64;
	cd server && env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -o dist/plugin-windows-amd64.exe;
else
	$(info DEBUG mode is on; to disable, unset MM_DEBUG)

	cd server && env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -gcflags "all=-N -l" -trimpath -o dist/plugin-darwin-amd64;
	cd server && env CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GO_BUILD_FLAGS) -gcflags "all=-N -l" -trimpath -o dist/plugin-darwin-arm64;
	cd server && env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -gcflags "all=-N -l" -trimpath -o dist/plugin-linux-amd64;
	cd server && env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GO_BUILD_FLAGS) -gcflags "all=-N -l" -trimpath -o dist/plugin-linux-arm64;
	cd server && env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -gcflags "all=-N -l" -trimpath -o dist/plugin-windows-amd64.exe;
endif
endif

## Ensures NPM dependencies are installed without having to run this all the time.
webapp/node_modules: $(wildcard webapp/package.json)
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) install
	touch $@
endif

## Generate dist and pack files for the webapp.
.PHONY: webapp
webapp: webapp/node_modules
ifneq ($(HAS_WEBAPP),)
ifeq ($(MM_DEBUG),)
	cd webapp && $(NPM) run build;
else
	cd webapp && $(NPM) run debug;
endif
endif
	cd webapp; npm run pack

## Generates a tar bundle of the plugin for install.
.PHONY: bundle
bundle:
	rm -rf dist/
	mkdir -p dist/$(PLUGIN_NAME)
	cp $(MANIFEST_FILE) dist/$(PLUGIN_NAME)/
	cp -r webapp/pack dist/$(PLUGIN_NAME)/
ifneq ($(wildcard LICENSE.txt),)
	cp -r LICENSE.txt dist/$(PLUGIN_ID)/
endif
ifneq ($(wildcard NOTICE.txt),)
	cp -r NOTICE.txt dist/$(PLUGIN_ID)/
endif
ifneq ($(wildcard $(ASSETS_DIR)/.),)
	cp -r $(ASSETS_DIR) dist/$(PLUGIN_NAME)/
endif
ifneq ($(HAS_PUBLIC),)
	cp -r public dist/$(PLUGIN_NAME)/public/
endif
ifneq ($(HAS_SERVER),)
	mkdir -p dist/$(PLUGIN_NAME)/server
	cp -r server/dist dist/$(PLUGIN_NAME)/server/
endif
ifneq ($(HAS_WEBAPP),)
	mkdir -p dist/$(PLUGIN_NAME)/webapp
	cp -r webapp/dist dist/$(PLUGIN_NAME)/webapp/
endif
	cd dist && tar -cvzf $(BUNDLE_NAME) $(PLUGIN_NAME)

	@echo plugin built at: dist/$(BUNDLE_NAME)

info: ## Display build information
	@echo "Build Number: $(BUILD_NUMBER)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Build Hash: $(BUILD_HASH)"
	@echo "Plugin ID: $(PLUGIN_ID)"
	@echo "Plugin Version: $(PLUGIN_VERSION)"
	@echo "Bundle Name: $(BUNDLE_NAME)"

## Builds and bundles the plugin.
.PHONY: dist
dist:	apply server webapp bundle

## Builds and installs the plugin to a server.
.PHONY: deploy
deploy: dist
	./build/bin/pluginctl deploy $(PLUGIN_ID) dist/$(BUNDLE_NAME)

## Builds and installs the plugin to a server, updating the webapp automatically when changed.
.PHONY: watch
watch: apply server bundle
ifeq ($(MM_DEBUG),)
	cd webapp && $(NPM) run build:watch
else
	cd webapp && $(NPM) run debug:watch
endif

## Installs a previous built plugin with updated webpack assets to a server.
.PHONY: deploy-from-watch
deploy-from-watch: bundle
	./build/bin/pluginctl deploy $(PLUGIN_ID) dist/$(BUNDLE_NAME)

## Setup dlv for attaching, identifying the plugin PID for other targets.
.PHONY: setup-attach
setup-attach:
	$(eval PLUGIN_PID := $(shell ps aux | grep "plugins/${PLUGIN_ID}" | grep -v "grep" | awk -F " " '{print $$2}'))
	$(eval NUM_PID := $(shell echo -n ${PLUGIN_PID} | wc -w))

	@if [ ${NUM_PID} -gt 2 ]; then \
		echo "** There is more than 1 plugin process running. Run 'make kill reset' to restart just one."; \
		exit 1; \
	fi

## Check if setup-attach succeeded.
.PHONY: check-attach
check-attach:
	@if [ -z ${PLUGIN_PID} ]; then \
		echo "Could not find plugin PID; the plugin is not running. Exiting."; \
		exit 1; \
	else \
		echo "Located Plugin running with PID: ${PLUGIN_PID}"; \
	fi

## Attach dlv to an existing plugin instance.
.PHONY: attach
attach: setup-attach check-attach
	dlv attach ${PLUGIN_PID}

## Attach dlv to an existing plugin instance, exposing a headless instance on $DLV_DEBUG_PORT.
.PHONY: attach-headless
attach-headless: setup-attach check-attach
	dlv attach ${PLUGIN_PID} --listen :$(DLV_DEBUG_PORT) --headless=true --api-version=2 --accept-multiclient

## Detach dlv from an existing plugin instance, if previously attached.
.PHONY: detach
detach: setup-attach
	@DELVE_PID=$(shell ps aux | grep "dlv attach ${PLUGIN_PID}" | grep -v "grep" | awk -F " " '{print $$2}') && \
	if [ "$$DELVE_PID" -gt 0 ] > /dev/null 2>&1 ; then \
		echo "Located existing delve process running with PID: $$DELVE_PID. Killing." ; \
		kill -9 $$DELVE_PID ; \
	fi

## Runs any lints and unit tests defined for the server and webapp, if they exist.
.PHONY: test
test: export FOCALBOARD_UNIT_TESTING=1
test: webapp/node_modules
ifneq ($(HAS_SERVER),)
	$(GO) test -v $(GO_TEST_FLAGS) ./server/...
endif
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run test;
endif
ifneq ($(wildcard ./build/sync/plan/.),)
	cd ./build/sync && $(GO) test -v $(GO_TEST_FLAGS) ./...
endif

## Creates a coverage report for the server code.
.PHONY: coverage
coverage: webapp/node_modules
ifneq ($(HAS_SERVER),)
	$(GO) test $(GO_TEST_FLAGS) -coverprofile=server/coverage.txt ./server/...
	$(GO) tool cover -html=server/coverage.txt
endif

## Extract strings for translation from the source code.
.PHONY: i18n-extract
i18n-extract:
ifneq ($(HAS_WEBAPP),)
ifeq ($(HAS_MM_UTILITIES),)
	@echo "You must clone github.com/mattermost/mattermost-utilities repo in .. to use this command"
else
	cd $(MM_UTILITIES_DIR) && npm install && npm run babel && node mmjstool/build/index.js i18n extract-webapp --webapp-dir $(PWD)/webapp
endif
endif

## Disable the plugin.
.PHONY: disable
disable: detach
	./build/bin/pluginctl disable $(PLUGIN_ID)

## Enable the plugin.
.PHONY: enable
enable:
	./build/bin/pluginctl enable $(PLUGIN_ID)

## Reset the plugin, effectively disabling and re-enabling it on the server.
.PHONY: reset
reset: detach
	./build/bin/pluginctl reset $(PLUGIN_ID)

## Kill all instances of the plugin, detaching any existing dlv instance.
.PHONY: kill
kill: detach
	$(eval PLUGIN_PID := $(shell ps aux | grep "plugins/${PLUGIN_ID}" | grep -v "grep" | awk -F " " '{print $$2}'))

	@for PID in ${PLUGIN_PID}; do \
		echo "Killing plugin pid $$PID"; \
		kill -9 $$PID; \
	done; \

## Clean removes all build artifacts.
.PHONY: clean
clean:
	rm -rf bin
	rm -rf dist
	rm -rf webapp/pack
ifneq ($(HAS_SERVER),)
	rm -fr server/coverage.txt
	rm -fr server/dist
endif
ifneq ($(HAS_WEBAPP),)
	rm -fr webapp/junit.xml
	rm -fr webapp/dist
	rm -fr webapp/node_modules
endif
	rm -fr build/bin/

## Sync directory with a starter template
sync:
ifndef STARTERTEMPLATE_PATH
	@echo STARTERTEMPLATE_PATH is not set.
	@echo Set STARTERTEMPLATE_PATH to a local clone of https://github.com/mattermost/mattermost-plugin-starter-template and retry.
	@exit 1
endif
	cd ${STARTERTEMPLATE_PATH} && go run ./build/sync/main.go ./build/sync/plan.yml $(PWD)

## Watch webapp and server changes and redeploy locally using local filesystem (MM_SERVER_PATH)
.PHONY: live-watch
live-watch:
	make -j2 live-watch-server live-watch-webapp

## Watch server changes and redeploy locally using local filesystem (MM_SERVER_PATH)
.PHONY: live-watch-server
live-watch-server: apply
	cd ../ && modd -f mattermost-plugin/modd.conf

## Watch webapp changes and redeploy locally using local filesystem (MM_SERVER_PATH)
.PHONY: live-watch-webapp
live-watch-webapp: apply
	cd webapp && $(NPM) run live-watch

.PHONY: deploy-to-mattermost-directory
deploy-to-mattermost-directory:
	./build/bin/pluginctl disable $(PLUGIN_ID)
	mkdir -p $(FOCALBOARD_PLUGIN_PATH)
	cp $(MANIFEST_FILE) $(FOCALBOARD_PLUGIN_PATH)/
	cp -r webapp/pack $(FOCALBOARD_PLUGIN_PATH)/
	cp -r $(ASSETS_DIR) $(FOCALBOARD_PLUGIN_PATH)/
	cp -r public $(FOCALBOARD_PLUGIN_PATH)/
	mkdir -p $(FOCALBOARD_PLUGIN_PATH)/server
	cp -r server/dist $(FOCALBOARD_PLUGIN_PATH)/server/
	mkdir -p $(FOCALBOARD_PLUGIN_PATH)/webapp
	cp -r webapp/dist $(FOCALBOARD_PLUGIN_PATH)/webapp/
	./build/bin/pluginctl enable $(PLUGIN_ID)
	@echo plugin built at: $(FOCALBOARD_PLUGIN_PATH)

# Help documentation à la https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@cat Makefile build/*.mk | grep -v '\.PHONY' |  grep -v '\help:' | grep -B1 -E '^[a-zA-Z0-9_.-]+:.*' | sed -e "s/:.*//" | sed -e "s/^## //" |  grep -v '\-\-' | sed '1!G;h;$$!d' | awk 'NR%2{printf "\033[36m%-30s\033[0m",$$0;next;}1' | sort

ifeq ($(OS),Windows_NT)
	RACE := ''
endif

# MAC cpu architecture
ifeq ($(shell uname -m),am64)
	MAC_GO_ARCH := arm64
else
	MAC_GO_ARCH := amd64
endif

all: webapp server ## Build server and webapp.

prebuild: ## Run prebuild actions (install dependencies etc.).
	cd webapp; npm install

ci: webapp-ci  ## Simulate CI, locally.

generate: ## Install and run code generators.
	cd server; go install github.com/golang/mock/mockgen@v1.6.0
	cd server; go generate ./...

server-ci: server-lint

server-lint: ## Run linters on server code.
	@if ! [ -x "$$(command -v golangci-lint)" ]; then \
		echo "golangci-lint is not installed. Please see https://github.com/golangci/golangci-lint#install-golangci-lint for installation instructions."; \
		exit 1; \
	fi;
	cd server; golangci-lint run ./...

modd-precheck:
	@if ! [ -x "$$(command -v modd)" ]; then \
		echo "modd is not installed. Please see https://github.com/cortesi/modd#install for installation instructions"; \
		exit 1; \
	fi; \

webapp-ci: ## Webapp CI: linting & testing.
	cd webapp; npm run check
	cd webapp; npm run test
	cd webapp; npm run check-types

webapp-test: ## jest tests for webapp
	cd webapp; npm run test

watch-plugin: modd-precheck ## Run and upload the plugin to a development server
	env FOCALBOARD_BUILD_TAGS='$(BUILD_TAGS)' modd -f modd-watchplugin.conf

live-watch-plugin: modd-precheck ## Run and update locally the plugin in the development server
	make live-watch

server-test: ## Run server tests
	@echo Starting tests for server
	cd server; go test -tags '$(BUILD_TAGS)' -race -v -coverpkg=./... -coverprofile=plugin-profile.coverage -count=1 -timeout=30m ./...

swagger: ## Generate swagger API spec and clients based on it.
	mkdir -p server/swagger/docs
	mkdir -p server/swagger/clients
	cd server && swagger generate spec -m -o ./swagger/swagger.yml

	cd server/swagger && openapi-generator generate -i swagger.yml -g html2 -o docs/html
	cd server/swagger && openapi-generator generate -i swagger.yml -g go -o clients/go
	cd server/swagger && openapi-generator generate -i swagger.yml -g javascript -o clients/javascript
	cd server/swagger && openapi-generator generate -i swagger.yml -g typescript-fetch -o clients/typescript
	cd server/swagger && openapi-generator generate -i swagger.yml -g swift5 -o clients/swift
	cd server/swagger && openapi-generator generate -i swagger.yml -g python -o clients/python

# ====================================================================================
# Used for semver bumping
PROTECTED_BRANCH := main
APP_NAME    := $(shell basename -s .git `git config --get remote.origin.url`)
CURRENT_VERSION := $(shell git describe --abbrev=0 --tags)
VERSION_PARTS := $(subst ., ,$(subst v,,$(subst -rc, ,$(CURRENT_VERSION))))
MAJOR := $(word 1,$(VERSION_PARTS))
MINOR := $(word 2,$(VERSION_PARTS))
PATCH := $(word 3,$(VERSION_PARTS))
RC := $(shell echo $(CURRENT_VERSION) | grep -oE 'rc[0-9]+' | sed 's/rc//')
# Check if current branch is protected
define check_protected_branch
	@current_branch=$$(git rev-parse --abbrev-ref HEAD); \
	if ! echo "$(PROTECTED_BRANCH)" | grep -wq "$$current_branch" && ! echo "$$current_branch" | grep -q "^release"; then \
		echo "Error: Tagging is only allowed from $(PROTECTED_BRANCH) or release branches. You are on $$current_branch branch."; \
		exit 1; \
	fi
endef
# Check if there are pending pulls
define check_pending_pulls
	@git fetch; \
	current_branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$(git rev-parse HEAD)" != "$$(git rev-parse origin/$$current_branch)" ]; then \
		echo "Error: Your branch is not up to date with upstream. Please pull the latest changes before performing a release"; \
		exit 1; \
	fi
endef
# Prompt for approval
define prompt_approval
	@read -p "About to bump $(APP_NAME) to version $(1), approve? (y/n) " userinput; \
	if [ "$$userinput" != "y" ]; then \
		echo "Bump aborted."; \
		exit 1; \
	fi
endef
# ====================================================================================

.PHONY: patch minor major patch-rc minor-rc major-rc

patch: ## to bump patch version (semver)
	$(call check_protected_branch)
	$(call check_pending_pulls)
	@$(eval PATCH := $(shell echo $$(($(PATCH)+1))))
	$(call prompt_approval,$(MAJOR).$(MINOR).$(PATCH))
	@echo Bumping $(APP_NAME) to Patch version $(MAJOR).$(MINOR).$(PATCH)
	git tag -s -a v$(MAJOR).$(MINOR).$(PATCH) -m "Bumping $(APP_NAME) to Patch version $(MAJOR).$(MINOR).$(PATCH)"
	git push origin v$(MAJOR).$(MINOR).$(PATCH)
	@echo Bumped $(APP_NAME) to Patch version $(MAJOR).$(MINOR).$(PATCH)

minor: ## to bump minor version (semver)
	$(call check_protected_branch)
	$(call check_pending_pulls)
	@$(eval MINOR := $(shell echo $$(($(MINOR)+1))))
	@$(eval PATCH := 0)
	$(call prompt_approval,$(MAJOR).$(MINOR).$(PATCH))
	@echo Bumping $(APP_NAME) to Minor version $(MAJOR).$(MINOR).$(PATCH)
	git tag -s -a v$(MAJOR).$(MINOR).$(PATCH) -m "Bumping $(APP_NAME) to Minor version $(MAJOR).$(MINOR).$(PATCH)"
	git push origin v$(MAJOR).$(MINOR).$(PATCH)
	@echo Bumped $(APP_NAME) to Minor version $(MAJOR).$(MINOR).$(PATCH)

major: ## to bump major version (semver)
	$(call check_protected_branch)
	$(call check_pending_pulls)
	$(eval MAJOR := $(shell echo $$(($(MAJOR)+1))))
	$(eval MINOR := 0)
	$(eval PATCH := 0)
	$(call prompt_approval,$(MAJOR).$(MINOR).$(PATCH))
	@echo Bumping $(APP_NAME) to Major version $(MAJOR).$(MINOR).$(PATCH)
	git tag -s -a v$(MAJOR).$(MINOR).$(PATCH) -m "Bumping $(APP_NAME) to Major version $(MAJOR).$(MINOR).$(PATCH)"
	git push origin v$(MAJOR).$(MINOR).$(PATCH)
	@echo Bumped $(APP_NAME) to Major version $(MAJOR).$(MINOR).$(PATCH)

patch-rc: ## to bump patch release candidate version (semver)
	$(call check_protected_branch)
	$(call check_pending_pulls)
	@$(eval RC := $(shell echo $$(($(RC)+1))))
	$(call prompt_approval,$(MAJOR).$(MINOR).$(PATCH)-rc$(RC))
	@echo Bumping $(APP_NAME) to Patch RC version $(MAJOR).$(MINOR).$(PATCH)-rc$(RC)
	git tag -s -a v$(MAJOR).$(MINOR).$(PATCH)-rc$(RC) -m "Bumping $(APP_NAME) to Patch RC version $(MAJOR).$(MINOR).$(PATCH)-rc$(RC)"
	git push origin v$(MAJOR).$(MINOR).$(PATCH)-rc$(RC)
	@echo Bumped $(APP_NAME) to Patch RC version $(MAJOR).$(MINOR).$(PATCH)-rc$(RC)

minor-rc: ## to bump minor release candidate version (semver)
	$(call check_protected_branch)
	$(call check_pending_pulls)
	@$(eval MINOR := $(shell echo $$(($(MINOR)+1))))
	@$(eval PATCH := 0)
	@$(eval RC := 1)
	$(call prompt_approval,$(MAJOR).$(MINOR).$(PATCH)-rc$(RC))
	@echo Bumping $(APP_NAME) to Minor RC version $(MAJOR).$(MINOR).$(PATCH)-rc$(RC)
	git tag -s -a v$(MAJOR).$(MINOR).$(PATCH)-rc$(RC) -m "Bumping $(APP_NAME) to Minor RC version $(MAJOR).$(MINOR).$(PATCH)-rc$(RC)"
	git push origin v$(MAJOR).$(MINOR).$(PATCH)-rc$(RC)
	@echo Bumped $(APP_NAME) to Minor RC version $(MAJOR).$(MINOR).$(PATCH)-rc$(RC)

major-rc: ## to bump major release candidate version (semver)
	$(call check_protected_branch)
	$(call check_pending_pulls)
	@$(eval MAJOR := $(shell echo $$(($(MAJOR)+1))))
	@$(eval MINOR := 0)
	@$(eval PATCH := 0)
	@$(eval RC := 1)
	$(call prompt_approval,$(MAJOR).$(MINOR).$(PATCH)-rc$(RC))
	@echo Bumping $(APP_NAME) to Major RC version $(MAJOR).$(MINOR).$(PATCH)-rc$(RC)
	git tag -s -a v$(MAJOR).$(MINOR).$(PATCH)-rc$(RC) -m "Bumping $(APP_NAME) to Major RC version $(MAJOR).$(MINOR).$(PATCH)-rc$(RC)"
	git push origin v$(MAJOR).$(MINOR).$(PATCH)-rc$(RC)
	@echo Bumped $(APP_NAME) to Major RC version $(MAJOR).$(MINOR).$(PATCH)-rc$(RC)
