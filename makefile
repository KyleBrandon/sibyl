all: sibyl


sibyl: client server


client:
	@go build -o ./bin/client ./cmd/client/main.go

server:
	@go build -o ./bin/note-server ./cmd/note_server/main.go

clean:
	rm ./bin/*


deploy: server
	cp ./bin/server ~/bin/myserver
	
