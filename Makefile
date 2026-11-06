.PHONY: build run run-staging test clean

# Build the Go binary and the React frontend.
# Output: bin/sifty (binary) and frontend/dist/ (frontend bundle).
build:
	mkdir -p bin
	cd backend && go build -o ../bin/sifty ./cmd/sifty
	npm --prefix frontend run build

# Run production server on port 8080 against ./data.
# Port is configured in data/config/runtime.yaml (port: 8080).
run: build
	./bin/sifty serve \
		--data-dir ./data \
		--dist ./frontend/dist \
		--workers-dir ./backend/workers

# Run staging server on port 8081 against ./data-staging.
# Port is configured in data-staging/config/runtime.yaml (port: 8081).
run-staging: build
	./bin/sifty serve \
		--data-dir ./data-staging \
		--dist ./frontend/dist \
		--workers-dir ./backend/workers

# Run Go tests with the race detector.
test:
	cd backend && go test -race ./...

# Remove build artifacts.
clean:
	rm -f bin/sifty
	rm -rf frontend/dist
