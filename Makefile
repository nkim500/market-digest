.PHONY: build migrate dashboard jobs test lint clean

BIN := bin
GO  := go

build: $(BIN)/jobs $(BIN)/dashboard

$(BIN)/jobs:
	$(GO) build -o $(BIN)/jobs ./jobs

$(BIN)/dashboard:
	$(GO) build -o $(BIN)/dashboard ./dashboard

migrate: $(BIN)/jobs
	./$(BIN)/jobs migrate

dashboard: $(BIN)/dashboard
	./$(BIN)/dashboard

jobs: $(BIN)/jobs

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...

clean:
	rm -rf $(BIN) data/*.db data/*.db-*
