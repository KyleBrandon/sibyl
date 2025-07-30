all: pdf_server notes_server

pdf_server:
	@go build -o ./bin/pdf-server ./cmd/pdf-server/main.go

notes_server:
	@go build -o ./bin/notes-server ./cmd/note_server/main.go

clean:
	rm -f ./bin/*
	
