# FIPS-compliant Linux/amd64 build. Linux/amd64 is the only platform
# mattermost-server's FIPS release path supports.

FIPS_IMAGE ?= cgr.dev/mattermost.com/go-msft-fips:1.26.3-dev@sha256:48ab99fede7fb33e132a0636072971e1ec4a69520865bfa1e4b517ee9cfdef34
# Same filename as the non-FIPS bundle on purpose: `make upload-to-server`
# (called from .github/actions/e2e-test) expects `dist/$(BUNDLE_NAME)`. The
# FIPS variant lives in dist-fips/, so there's no collision. delivery-platform's
# release pipeline renames to `<plugin>-<version>+<sha>-fips.tar.gz` itself.
BUNDLE_NAME_FIPS ?= $(BUNDLE_NAME)
FIPS_BIN := server/dist-fips/plugin-linux-amd64-fips

# Empty by default. Inheriting the plugin's release LDFLAGS (`-ldflags="-s -w"`)
# would strip the symbol table, and verify-fips below needs symbols intact
# for `go tool nm` to find the OpenSSL integration.
FIPS_GO_BUILD_LDFLAGS ?=

# GO_BUILD_* are Make-substituted into the single-quoted inner script.
# Env-var passing (`-e VAR=...`) doesn't survive a second pass of word
# splitting on values with embedded quotes like `-gcflags "all=-N -l"`.
# Boards sets GO_BUILD_FLAGS with single-quoted -ldflags, which breaks the
# single-quoted docker inner script and drops -o (go then tries to write "server").
# Pass build metadata via env vars and use double-quoted -ldflags instead.
.PHONY: server-fips
server-fips: templates-archive
	mkdir -p server/dist-fips
	docker run --rm \
	  --entrypoint="" \
	  -v $(PWD):/plugin \
	  -w /plugin/server \
	  -e BUILD_NUMBER=$(BUILD_NUMBER) \
	  -e BUILD_DATE="$(BUILD_DATE)" \
	  -e BUILD_HASH=$(BUILD_HASH) \
	  $(FIPS_IMAGE) \
	  /bin/sh -c 'CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
	    go build -trimpath -buildvcs=false \
	    -ldflags "-X github.com/mattermost/focalboard/server/model.BuildNumber=$${BUILD_NUMBER} -X github.com/mattermost/focalboard/server/model.BuildDate=$${BUILD_DATE} -X github.com/mattermost/focalboard/server/model.BuildHash=$${BUILD_HASH} -X github.com/mattermost/focalboard/server/model.Edition=plugin" \
	    $(GO_BUILD_GCFLAGS) $(FIPS_GO_BUILD_LDFLAGS) \
	    -tags requirefips \
	    -o dist-fips/plugin-linux-amd64-fips'

# Mirrors mattermost-server/build/release.mk. Each check is its own recipe
# line so a failure exits with the right status; joined with `; \`, only
# the last command's exit propagates and a failed check would be masked.
.PHONY: verify-fips
verify-fips:
	@test -f $(FIPS_BIN) || (echo "verify-fips: $(FIPS_BIN) not built" && exit 1)
	$(GO) version -m $(FIPS_BIN) | grep -q "GOEXPERIMENT=systemcrypto" || (echo "ERROR: missing GOEXPERIMENT=systemcrypto" && exit 1)
	$(GO) version -m $(FIPS_BIN) | grep "\-tags" | grep -q "requirefips" || (echo "ERROR: missing -tags=requirefips" && exit 1)
	$(GO) tool nm $(FIPS_BIN) | grep -qE "func_go_openssl_OpenSSL_version|_mkcgo_OpenSSL_version" || (echo "ERROR: missing OpenSSL integration" && exit 1)
	@echo "verify-fips: OK"

# Depends on verify-fips so a direct `make bundle-fips` can't package an
# unverified binary. Phony-target dedup means dist-fips / dist-all still
# run verify-fips exactly once.
.PHONY: bundle-fips
bundle-fips: verify-fips
	rm -rf server/dist-fips-staged
	mkdir -p server/dist-fips-staged
	cp $(FIPS_BIN) server/dist-fips-staged/plugin-linux-amd64
	$(MAKE) bundle BUNDLE_DIR=dist-fips BUNDLE_NAME=$(BUNDLE_NAME_FIPS) SERVER_DIST_SRC=server/dist-fips-staged

.PHONY: dist-fips
dist-fips: apply server-fips verify-fips webapp bundle-fips

# Flat prerequisite list so `apply` and `webapp` run once, not twice.
.PHONY: dist-all
dist-all: apply webapp server server-fips verify-fips bundle bundle-fips
