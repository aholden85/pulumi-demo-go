# Lamba Functions

## Requirements
Every Lambda function needs a `Makefile`. This file will be executed by the 
Pulumi code. An example makefile is included below:

```Makefile
.DEFAULT_GOAL := build

build_folder := bin
artifact_filename := main
artifact_path := ${build_folder}/${artifact_filename}

.PHONY: build
build: clean
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${artifact_path} main.go
	zip -j ${artifact_path}.zip ${artifact_path}

.PHONY: clean
clean:
ifneq ("$(wildcard ${artifact_path})","")
	@rm -f ${artifact_path}*
else
	@echo > /dev/null
endif
```