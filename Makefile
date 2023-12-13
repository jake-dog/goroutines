.PHONY: test coverage
test:
	@go test -race -cover -v ./...

coverage:
	@go test ./... -coverprofile=cover.out
	@go tool cover -html=cover.out
