all: pdf_server notes_server

pdf_server:
	@go build -o ./bin/pdf-server ./cmd/pdfserver/main.go

notes_server:
	@go build -o ./bin/notes-server ./cmd/noteserver/main.go

clean:
	rm -f ./bin/*
	
