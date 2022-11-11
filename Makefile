fini: main.go
	GOOS=linux GOARCH=amd64 go build -o bin/fini-amd64-linux main.go
	GOOS=windows GOARCH=amd64 go build -o bin/fini-amd64.exe main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/fini-amd64-darwin main.go

clean:
	rm -f bin/fini-amd64-linux
	rm -f bin/fini-amd64.exe
	rm -f bin/fini-amd64-darwin
