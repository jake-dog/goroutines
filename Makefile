.PHONY: test coverage fmt bench
test:
	@go test -race -cover -v ./...

bench:
	@go test -bench=. -benchmem $(if $(BENCHTIME),-benchtime=$(BENCHTIME))

coverage:
	@go test ./... -coverprofile=cover.out
	@go tool cover -html=cover.out

fmt:
	@$(foreach f,$(wildcard $(PWD)/*.go),gofmt -w $(f);)
