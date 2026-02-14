.PHONY: generate build dev test build-arm64 build-armv7 build-all clean

generate:
	templ generate

build: generate
	CGO_ENABLED=0 go build -o dungeon-revealer .

dev: generate
	go run . --data-dir=./data --dm-password=admin

test:
	CGO_ENABLED=0 go test ./...

build-arm64: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o dungeon-revealer-linux-arm64 .

build-armv7: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o dungeon-revealer-linux-armv7 .

build-all: generate
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/dungeon-revealer-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o dist/dungeon-revealer-linux-arm64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o dist/dungeon-revealer-linux-armv7 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/dungeon-revealer-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/dungeon-revealer-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/dungeon-revealer-windows-amd64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o dist/dungeon-revealer-windows-arm64.exe .

clean:
	rm -f dungeon-revealer dungeon-revealer-*
	rm -rf dist
