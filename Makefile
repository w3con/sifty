.PHONY: build build-staging run run-staging test clean

# Isolated environment build targets. Each environment owns its binary, bundle
# and worker tree; publication refuses to replace artifacts while a server or
# continuous preview owns that environment. Build candidates live under
# .local/deploy and are never written over a running environment.
PROD_PORT ?= 8080
STAGING_PORT ?= 8081
PROD_BIN ?= bin/sifty
STAGING_BIN ?= bin/sifty-staging
PROD_DIST ?= frontend/dist
STAGING_DIST ?= frontend/dist-staging
PROD_WORKERS ?= backend/workers-prod
STAGING_WORKERS ?= backend/workers-staging

build:
	bin/sift-deploy --env prod --build-only

build-staging:
	bin/sift-deploy --env staging --build-only

# Run production against its isolated artifacts. Stop any old server first;
# build-only refuses to publish over a running environment.
run: build
	./$(PROD_BIN) serve \
		--data-dir ./data \
		--dist ./$(PROD_DIST) \
		--workers-dir ./$(PROD_WORKERS) \
		--port $(PROD_PORT) --public-port $(PROD_PORT)

# Run staging against staging-only artifacts and data. A continuous preview
# owns port 8081 and its own generations; stop it before this target.
run-staging: build-staging
	./$(STAGING_BIN) serve \
		--data-dir ./data-staging \
		--dist ./$(STAGING_DIST) \
		--workers-dir ./$(STAGING_WORKERS) \
		--port $(STAGING_PORT) --public-port $(STAGING_PORT)

# Run Go tests with the race detector. Automated suites are paused by owner
# instruction; this target remains for an explicit request.
test:
	cd backend && go test -race ./...

# Remove generated environment artifacts. Deploy recovery state and runtime
# data are deliberately not removed.
clean:
	rm -f bin/sifty bin/sifty-staging
	rm -rf frontend/dist frontend/dist-staging backend/workers-prod backend/workers-staging
