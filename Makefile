BINARY := arte-rss-subfeeds
COVERPROFILE := coverage.out

.PHONY: build test coverage clean

build:
	go build -o $(BINARY) .

test:
	go test ./...

coverage:
	go test -coverprofile=$(COVERPROFILE) ./...
	go tool cover -func=$(COVERPROFILE)

clean:
	rm -f $(BINARY) $(COVERPROFILE)
