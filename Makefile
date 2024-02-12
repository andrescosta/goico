FORMAT_FILES = $(shell find . -type f -name '*.go' -not -path "*.pb.go")

MKDIR_REPO_CMD = mkdir -p reports 
ifeq ($(OS),Windows_NT)
ifneq ($(MSYSTEM), MSYS)
# -new-item exit with an error if "-Force" is not passed and the directory exists. 
	MKDIR_REPO_CMD = pwsh -noprofile -command "new-item reports -ItemType Directory -Force -ErrorAction silentlycontinue | Out-Null"
endif
endif

.PHONY: init test test_coverage test_html lint vuln release format $(FORMAT_FILES)

lint:
	@golangci-lint run ./...

vuln:
	@govulncheck ./...

test:
	go test -timeout 30s -count=1 -race ./...

test_coverage: init
	go test ./... -coverprofile=./reports/coverage.out

test_html: test_coverage
	go tool cover -html=./reports/coverage.out

format: $(FORMAT_FILES)  

$(FORMAT_FILES):
	@gofumpt -w $@

init:
	@$(MKDIR_REPO_CMD) 

release: format test lint vuln
