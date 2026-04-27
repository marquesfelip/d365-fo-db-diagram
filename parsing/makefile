.PHONY: default run build test clean

# Variables
APP_NAME=app-parsing

# Tasks
default: run

run:
	air
build:
	@go build -o $(APP_NAME) main.go
test:
	@go test ./ ...
clean:
	@rm -f $(APP_NAME)
	@rm -rf ./docs
