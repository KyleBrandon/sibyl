all: sibyl


sibyl: client server


client:
	@go build -o ./bin/client ./cmd/client/main.go

server:
	@go build -o ./bin/server ./cmd/server/main.go

clean:
	rm ./bin/*


deply_server: server
	cp ./bin/server ~/bin/myserver
	
