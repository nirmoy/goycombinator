SHELL := /usr/bin/env bash
CWD := $(shell pwd)
BIN := goyc

SOURCES := $(shell find  . -name '*.go')

.PHONY: clean


all: $(BIN)

$(BIN): $(SOURCES)
	 GO111MODULE=on go build -o $(BIN) main.go

clean:
	rm -f $(BIN)
