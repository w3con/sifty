.PHONY: build run run-staging test clean

# Build the Go binary and the React frontend.
# Output: bin/sifty (binary) and sifty/dist/ (frontend bundle).
build:
	mkdir -p bin
	cd sifty && go build -o ../bin/sifty ./cmd/sifty
	npm --prefix sifty/web run build

# Run production server on port 8080 against ./data.
# Port is configured in data/config/runtime.yaml (port: 8080).
run: build
	./bin/sifty serve \
		--data-dir ./data \
		--dist ./sifty/dist \
		--workers-dir ./sifty/workers

# Run staging server on port 8081 against ./data-staging.
# Port is configured in data-staging/config/runtime.yaml (port: 8081).
run-staging: build
	./bin/sifty serve \
		--data-dir ./data-staging \
		--dist ./sifty/dist \
		--workers-dir ./sifty/workers

# Run Go tests with the race detector.
test:
	cd sifty && go test -race ./...

# Remove build artifacts.
clean:
	rm -f bin/sifty
	rm -rf sifty/dist
