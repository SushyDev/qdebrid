TARGET = main

run:
	go run $(TARGET).go
debug:
	dlv debug -l 127.0.0.1:2345 --headless $(TARGET).go --
build:
	CGO_ENABLED=0 go build $(TARGET).go

.PHONY: run debug build
.DEFAULT_GOAL := build
