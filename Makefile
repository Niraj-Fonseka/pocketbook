
run: go-build run-binary

go-build:
	@echo "\033[31m***** Building Go binary ******\033[0m"
	@go build -o pocketbook-app .

run-binary:
	@echo "\033[31m***** Running app ******\033[0m"
	@./pocketbook-app

docker-build:
	@echo "\033[31m***** Building Docker image ******\033[0m"
	@docker build -t pocketbook .