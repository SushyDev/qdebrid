TARGET = main

run:
	go run $(TARGET).go

debug:
	dlv debug -l 127.0.0.1:2345 --headless $(TARGET).go --
.PHONY: run debug
