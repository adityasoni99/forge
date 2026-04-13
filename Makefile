.PHONY: lint test test-go test-ts ci
lint:
	go vet ./...
test:
	go test ./...

test-go: test

test-ts:
	cd harness && npm test

ci: test-go test-ts docker-build

.PHONY: forge-linux docker-build docker-build-arm

forge-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o forge-linux-amd64 ./cmd/forge/

docker-build: forge-linux
	cd harness && npm run build
	docker build -t forge:latest .

docker-build-arm:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o forge-linux-arm64 ./cmd/forge/
	cd harness && npm run build
	docker build --build-arg TARGETARCH=arm64 -t forge:latest .
