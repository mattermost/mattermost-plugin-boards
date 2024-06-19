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
FOCALBOARD_PLUGIN_PATH=$(MATTERMOST_PLUGINS_PATH)/focalboard

export GO111MODULE=on

ASSETS_DIR ?= assets

RACE = -race

## Define the default target (make all)
.PHONY: default
default: all

# Verify environment, and define PLUGIN_ID, PLUGIN_VERSION, HAS_SERVER and HAS_WEBAPP as needed.
include build/setup.mk

BUNDLE_NAME ?= $(PLUGIN_ID)-$(PLUGIN_VERSION).tar.gz

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
	cd webapp && npm run lint
	cd webapp && npm run check-types
endif

ifneq ($(HAS_SERVER),)
	@if ! [ -x "$$(command -v golangci-lint)" ]; then \
		echo "golangci-lint is not installed. Please see https://github.com/golangci/golangci-lint#install-golangci-lint for installation instructions."; \
		exit 1; \
	fi; \

	@echo Running golangci-lint
	golangci-lint run ./...
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

# ## Ensures NPM dependencies are installed without having to run this all the time.
# webapp/node_modules: $(wildcard webapp/package.json)
# ifneq ($(HAS_WEBAPP),)
# 	cd webapp && $(NPM) install
# 	touch $@
# endif

# ## Builds the webapp, if it exists.
# .PHONY: webapp
# webapp: webapp/node_modules
# ifneq ($(HAS_WEBAPP),)
# ifeq ($(MM_DEBUG),)
# 	cd webapp && $(NPM) run build;
# else
# 	cd webapp && $(NPM) run debug;
# endif
# endif

## Generates a tar bundle of the plugin for install.
.PHONY: bundle
bundle:
	rm -rf dist/
	mkdir -p dist/$(PLUGIN_ID)
	cp $(MANIFEST_FILE) dist/$(PLUGIN_ID)/
	cp -r ../webapp/pack dist/$(PLUGIN_ID)/
ifneq ($(wildcard $(ASSETS_DIR)/.),)
	cp -r $(ASSETS_DIR) dist/$(PLUGIN_ID)/
endif
ifneq ($(HAS_PUBLIC),)
	cp -r public dist/$(PLUGIN_ID)/public/
endif
ifneq ($(HAS_SERVER),)
	mkdir -p dist/$(PLUGIN_ID)/server
	cp -r server/dist dist/$(PLUGIN_ID)/server/
endif
ifneq ($(HAS_WEBAPP),)
	mkdir -p dist/$(PLUGIN_ID)/webapp
	cp -r webapp/dist dist/$(PLUGIN_ID)/webapp/
endif
	cd dist && tar -cvzf $(BUNDLE_NAME) $(PLUGIN_ID)

	@echo plugin built at: dist/$(BUNDLE_NAME)

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
	cp -r ../webapp/pack $(FOCALBOARD_PLUGIN_PATH)/
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

ci: webapp-ci server-test ## Simulate CI, locally.

# server: ## Build server for local environment.
# 	$(eval LDFLAGS += -X "github.com/mattermost/focalboard/server/model.Edition=dev")
# 	cd server; go build -ldflags '$(LDFLAGS)' -tags '$(BUILD_TAGS)' -o ../bin/focalboard-server ./main

# server-mac: ## Build server for Mac.
# 	mkdir -p bin/mac
# 	$(eval LDFLAGS += -X "github.com/mattermost/focalboard/server/model.Edition=mac")
# ifeq ($(FB_PROD),)
# 	cd server; env GOOS=darwin GOARCH=$(MAC_GO_ARCH) go build -ldflags '$(LDFLAGS)' -tags '$(BUILD_TAGS)' -o ../bin/mac/focalboard-server ./main
# else
# # Always build x86 for production, to work on both Apple Silicon and legacy Macs
# 	cd server; env GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -ldflags '$(LDFLAGS)' -tags '$(BUILD_TAGS)' -o ../bin/mac/focalboard-server ./main
# endif

# server-linux: ## Build server for Linux.
# 	mkdir -p bin/linux
# 	$(eval LDFLAGS += -X "github.com/mattermost/focalboard/server/model.Edition=linux")
# 	cd server; env GOOS=linux GOARCH=$(arch) go build -ldflags '$(LDFLAGS)' -tags '$(BUILD_TAGS)' -o ../bin/linux/focalboard-server ./main

# server-docker: ## Build server for Docker Architectures.
# 	mkdir -p bin/docker
# 	$(eval LDFLAGS += -X "github.com/mattermost/focalboard/server/model.Edition=linux")
# 	cd server; env GOOS=$(os) GOARCH=$(arch) go build -ldflags '$(LDFLAGS)' -tags '$(BUILD_TAGS)' -o ../bin/docker/focalboard-server ./main

# server-win: ## Build server for Windows.
# 	$(eval LDFLAGS += -X "github.com/mattermost/focalboard/server/model.Edition=win")
# 	cd server; env GOOS=windows GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -tags '$(BUILD_TAGS)' -o ../bin/win/focalboard-server.exe ./main

# server-dll: ## Build server as Windows DLL.
# 	$(eval LDFLAGS += -X "github.com/mattermost/focalboard/server/model.Edition=win")
# 	cd server; env GOOS=windows GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -tags '$(BUILD_TAGS)' -buildmode=c-shared -o ../bin/win-dll/focalboard-server.dll ./main

# server-linux-package: server-linux webapp
# 	rm -rf package
# 	mkdir -p package/${PACKAGE_FOLDER}/bin
# 	cp bin/linux/focalboard-server package/${PACKAGE_FOLDER}/bin
# 	cp -R webapp/pack package/${PACKAGE_FOLDER}/pack
# 	cp server-config.json package/${PACKAGE_FOLDER}/config.json
# 	cp NOTICE.txt package/${PACKAGE_FOLDER}
# 	cp webapp/NOTICE.txt package/${PACKAGE_FOLDER}/webapp-NOTICE.txt
# 	mkdir -p dist
# 	cd package && tar -czvf ../dist/focalboard-server-linux-amd64.tar.gz ${PACKAGE_FOLDER}
# 	rm -rf package

# server-linux-package-docker:
# 	rm -rf package
# 	mkdir -p package/${PACKAGE_FOLDER}/bin
# 	cp bin/linux/focalboard-server package/${PACKAGE_FOLDER}/bin
# 	cp -R webapp/pack package/${PACKAGE_FOLDER}/pack
# 	cp server-config.json package/${PACKAGE_FOLDER}/config.json
# 	cp NOTICE.txt package/${PACKAGE_FOLDER}
# 	cp webapp/NOTICE.txt package/${PACKAGE_FOLDER}/webapp-NOTICE.txt
# 	mkdir -p dist
# 	cd package && tar -czvf ../dist/focalboard-server-linux-$(arch).tar.gz ${PACKAGE_FOLDER}
# 	rm -rf package

generate: ## Install and run code generators.
	cd server; go install github.com/golang/mock/mockgen@v1.6.0
	cd server; go generate ./...

server-lint: ## Run linters on server code.
	@if ! [ -x "$$(command -v golangci-lint)" ]; then \
		echo "golangci-lint is not installed. Please see https://github.com/golangci/golangci-lint#install-golangci-lint for installation instructions."; \
		exit 1; \
	fi;
	cd server; golangci-lint run ./...
	cd mattermost-plugin; golangci-lint run ./...

modd-precheck:
	@if ! [ -x "$$(command -v modd)" ]; then \
		echo "modd is not installed. Please see https://github.com/cortesi/modd#install for installation instructions"; \
		exit 1; \
	fi; \

# watch: modd-precheck ## Run both server and webapp watching for changes
# 	env FOCALBOARD_BUILD_TAGS='$(BUILD_TAGS)' modd

# watch-single-user: modd-precheck ## Run both server and webapp in single user mode watching for changes
# 	env FOCALBOARDSERVER_ARGS=--single-user FOCALBOARD_BUILD_TAGS='$(BUILD_TAGS)' modd

# watch-server-test: modd-precheck ## Run server tests watching for changes
# 	env FOCALBOARD_BUILD_TAGS='$(BUILD_TAGS)' modd -f modd-servertest.conf

server-test: server-test-sqlite server-test-mysql server-test-mariadb server-test-postgres ## Run server tests

server-test-sqlite: export FOCALBOARD_UNIT_TESTING=1

server-test-sqlite: ## Run server tests using sqlite
	cd server; go test -tags '$(BUILD_TAGS)' -race -v -coverpkg=./... -coverprofile=server-sqlite-profile.coverage -count=1 -timeout=30m ./...
	cd server; go tool cover -func server-sqlite-profile.coverage

server-test-mini-sqlite: export FOCALBOARD_UNIT_TESTING=1

server-test-mini-sqlite: ## Run server tests using sqlite
	cd server/integrationtests; go test -tags '$(BUILD_TAGS)' $(RACE) -v -count=1 -timeout=30m ./...

server-test-mysql: export FOCALBOARD_UNIT_TESTING=1
server-test-mysql: export FOCALBOARD_STORE_TEST_DB_TYPE=mysql
server-test-mysql: export FOCALBOARD_STORE_TEST_DOCKER_PORT=44446

server-test-mysql: ## Run server tests using mysql
	@echo Starting docker container for mysql
	docker-compose -f ./docker-testing/docker-compose-mysql.yml down -v --remove-orphans
	docker-compose -f ./docker-testing/docker-compose-mysql.yml run start_dependencies
	cd server; go test -tags '$(BUILD_TAGS)' -race -v -coverpkg=./... -coverprofile=server-mysql-profile.coverage -count=1 -timeout=30m ./...
	cd server; go tool cover -func server-mysql-profile.coverage
	cd mattermost-plugin/server; go test -tags '$(BUILD_TAGS)' -race -v -coverpkg=./... -coverprofile=plugin-mysql-profile.coverage -count=1 -timeout=30m ./...
	cd mattermost-plugin/server; go tool cover -func plugin-mysql-profile.coverage
	docker-compose -f ./docker-testing/docker-compose-mysql.yml down -v --remove-orphans

server-test-mariadb: export FOCALBOARD_UNIT_TESTING=1
server-test-mariadb: export FOCALBOARD_STORE_TEST_DB_TYPE=mariadb
server-test-mariadb: export FOCALBOARD_STORE_TEST_DOCKER_PORT=44445

server-test-mariadb: templates-archive ## Run server tests using mysql
	@echo Starting docker container for mariadb
	docker-compose -f ./docker-testing/docker-compose-mariadb.yml down -v --remove-orphans
	docker-compose -f ./docker-testing/docker-compose-mariadb.yml run start_dependencies
	cd server; go test -tags '$(BUILD_TAGS)' -race -v -coverpkg=./... -coverprofile=server-mariadb-profile.coverage -count=1 -timeout=30m ./...
	cd server; go tool cover -func server-mariadb-profile.coverage
	cd mattermost-plugin/server; go test -tags '$(BUILD_TAGS)' -race -v -coverpkg=./... -coverprofile=plugin-mariadb-profile.coverage -count=1 -timeout=30m ./...
	cd mattermost-plugin/server; go tool cover -func plugin-mariadb-profile.coverage
	docker-compose -f ./docker-testing/docker-compose-mariadb.yml down -v --remove-orphans

server-test-postgres: export FOCALBOARD_UNIT_TESTING=1
server-test-postgres: export FOCALBOARD_STORE_TEST_DB_TYPE=postgres
server-test-postgres: export FOCALBOARD_STORE_TEST_DOCKER_PORT=44447

server-test-postgres: ## Run server tests using postgres
	@echo Starting docker container for postgres
	docker-compose -f ./docker-testing/docker-compose-postgres.yml down -v --remove-orphans
	docker-compose -f ./docker-testing/docker-compose-postgres.yml run start_dependencies
	cd server; go test -tags '$(BUILD_TAGS)' -race -v -coverpkg=./... -coverprofile=server-postgres-profile.coverage -count=1 -timeout=30m ./...
	cd server; go tool cover -func server-postgres-profile.coverage
	cd mattermost-plugin/server; go test -tags '$(BUILD_TAGS)' -race -v -coverpkg=./... -coverprofile=plugin-postgres-profile.coverage -count=1 -timeout=30m ./...
	cd mattermost-plugin/server; go tool cover -func plugin-postgres-profile.coverage
	docker-compose -f ./docker-testing/docker-compose-postgres.yml down -v --remove-orphans

webapp: ## Build webapp.
	cd webapp; npm run pack

webapp-ci: ## Webapp CI: linting & testing.
	cd webapp; npm run check
	cd mattermost-plugin/webapp; npm run lint
	cd webapp; npm run test
	cd mattermost-plugin/webapp; npm run test
	cd webapp; npm run cypress:ci

webapp-test: ## jest tests for webapp
	cd webapp; npm run test

watch-plugin: modd-precheck ## Run and upload the plugin to a development server
	env FOCALBOARD_BUILD_TAGS='$(BUILD_TAGS)' modd -f modd-watchplugin.conf

live-watch-plugin: modd-precheck ## Run and update locally the plugin in the development server
	cd mattermost-plugin; make live-watch

# mac-app: server-mac webapp ## Build Mac application.
# 	rm -rf mac/temp
# 	rm -rf mac/dist
# 	rm -rf mac/resources/bin
# 	rm -rf mac/resources/pack
# 	mkdir -p mac/resources/bin
# 	cp bin/mac/focalboard-server mac/resources/bin/focalboard-server
# 	cp app-config.json mac/resources/config.json
# 	cp -R webapp/pack mac/resources/pack
# 	mkdir -p mac/temp
# 	xcodebuild archive -workspace mac/Focalboard.xcworkspace -scheme Focalboard -archivePath mac/temp/focalboard.xcarchive CODE_SIGN_IDENTITY="" CODE_SIGNING_REQUIRED="NO" CODE_SIGNING_ALLOWED="NO" \
# 		|| { echo "xcodebuild failed, did you install the full Xcode and not just the CLI tools?"; exit 1; }
# 	mkdir -p mac/dist
# 	cp -R mac/temp/focalboard.xcarchive/Products/Applications/Focalboard.app mac/dist/
# 	# xcodebuild -exportArchive -archivePath mac/temp/focalboard.xcarchive -exportPath mac/dist -exportOptionsPlist mac/export.plist
# 	cp NOTICE.txt mac/dist
# 	cp webapp/NOTICE.txt mac/dist/webapp-NOTICE.txt
# 	cd mac/dist; zip -r focalboard-mac.zip Focalboard.app MIT-COMPILED-LICENSE.md NOTICE.txt webapp-NOTICE.txt

# win-wpf-app: server-dll webapp ## Build Windows WPF application.
# 	cd win-wpf && ./build.bat
# 	cd win-wpf && ./package.bat
# 	cd win-wpf && ./package-zip.bat

# linux-app: webapp ## Build Linux application.
# 	rm -rf linux/temp
# 	rm -rf linux/dist
# 	mkdir -p linux/dist
# 	mkdir -p linux/temp/focalboard-app
# 	cp app-config.json linux/temp/focalboard-app/config.json
# 	cp NOTICE.txt linux/temp/focalboard-app/
# 	cp webapp/NOTICE.txt linux/temp/focalboard-app/webapp-NOTICE.txt
# 	cp -R webapp/pack linux/temp/focalboard-app/pack
# 	cd linux; make build
# 	cp -R linux/bin/focalboard-app linux/temp/focalboard-app/
# 	cd linux/temp; tar -zcf ../dist/focalboard-linux.tar.gz focalboard-app
# 	rm -rf linux/temp

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

cleanall: clean ## Clean all build artifacts and dependencies.
	rm -rf webapp/node_modules
