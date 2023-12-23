FORMAT_FILES = $(shell find . -type f -name '*.go' -not -path "*.pb.go")

.PHONY: lint vuln release format $(FORMAT_FILES)

lint:
	golangci-lint run ./...

vuln:
	govulncheck ./...

format: $(FORMAT_FILES)  

$(FORMAT_FILES):
	@gofumpt -w $@

release: format lint vuln 