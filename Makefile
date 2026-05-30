.PHONY: build run clean tidy

build:
	go build -o bin/phibot ./cmd/phibot

run:
	go run ./cmd/phibot

run-debug:
	go run ./cmd/phibot -debug

tidy:
	go mod tidy

clean:
	rm -rf bin/
