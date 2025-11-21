.PHONY: build run

SOURCES       = $(shell find . -name '*.go')

run: build
	tmp/otelui

build: $(SOURCES)
	go build -o tmp/otelui main.go
