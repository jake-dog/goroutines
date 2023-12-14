.PHONY: test coverage fmt
test:
	@go test -race -cover -v ./...

coverage:
	@go test ./... -coverprofile=cover.out
	@go tool cover -html=cover.out

fmt:
	@$(foreach f,$(wildcard $(PWD)/*.go),gofmt -w $(f);)
