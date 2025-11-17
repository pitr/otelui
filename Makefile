.PHONY: build run

build:
	go build -o tmp/otelui main.go

run:
	tmp/otelui
