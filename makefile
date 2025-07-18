all: sibyl


sibyl: client note_server gcp_server


client:
	@go build -o ./bin/client ./cmd/client/main.go

note_server:
	@go build -o ./bin/note-server ./cmd/note_server/main.go

gcp_server:
	@go build -o ./bin/gcp-server ./cmd/gcp_server/main.go

clean:
	rm ./bin/*


deploy: server
	cp ./bin/server ~/bin/myserver
	
