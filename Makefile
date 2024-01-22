.PHONY: test coverage fmt lint
test:
	@go test -race -cover -v ./...

coverage:
	@go test ./... -coverprofile=cover.out
	@go tool cover -html=cover.out

lint:
	@golangci-lint run -v

fmt:
	@$(foreach f,$(wildcard $(PWD)/*.go),gofmt -s -w $(f);)
