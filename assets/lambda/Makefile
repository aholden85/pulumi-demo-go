.DEFAULT_GOAL := build

build_folder := bin
artifact_filename := main
artifact_path := ${build_folder}/${artifact_filename}

lambda_functions = \
	facts \
	images \
	pats

.PHONY: clean
clean:
	@$(foreach func, ${lambda_functions}, \
		find . -path "./${func}/bin/*" | xargs rm; \
	)

.PHONY: build
build: clean
	@$(foreach func, ${lambda_functions}, \
		cd ${func}; \
		env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${artifact_path} main.go; \
		zip -j ${artifact_path}.zip ${artifact_path}; \
		cd ..; \
	)