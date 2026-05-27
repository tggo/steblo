.PHONY: test bench lint fuzz cover

# steblo (core) and bleveuk are separate modules: core has zero dependencies;
# bleveuk pulls in Bleve. Test both.
test:
	go test ./...
	cd bleveuk && go test ./...

cover:
	go test -coverprofile=cover.out . && go tool cover -func=cover.out | tail -1

bench:
	go test -bench=. -benchmem -count=10 .

fuzz:
	go test -run x -fuzz FuzzStemNoCrash -fuzztime 60s .
	go test -run x -fuzz FuzzStemLengthMonotonic -fuzztime 60s .

lint:
	go vet ./...
	cd bleveuk && go vet ./...
	@command -v golangci-lint >/dev/null && golangci-lint run || echo "golangci-lint not installed, skipping"
