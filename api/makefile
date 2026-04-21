.PHONY: default run build test doc clean

# Variables
APP_NAME=d365-fo-diagram

# Tasks
default: run

run:
	cd "./cmd" && air
build:
	@go build -o $(APP_NAME) main.go
test:
	@go test ./ ...
doc:
	@swag init
clean:
	@rm -f $(APP_NAME)
	@rm -rf ./docs
