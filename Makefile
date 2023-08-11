build:
	env GOOS=linux GOARCH=amd64 go build -o dist/tm-linter main.go