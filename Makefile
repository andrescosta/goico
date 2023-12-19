GOFMT_FILES = $(shell find . -type f -name '*.go' -not -path "*.pb.go")

lint:
	golangci-lint run ./...

vuln:
	govulncheck ./...

gofmt: $(GOFMT_FILES)  

$(GOFMT_FILES):
	@gofmt -s -w $@

release: gofmt lint vuln 

.PHONY: lint vuln release gofmt $(GOFMT_FILES)
