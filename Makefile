.PHONY: deps
deps:
	go get ./...

.PHONY: test
test:
	go test -v ./...