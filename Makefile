build:
	go build cmd/gitogram.go
run: build
	./gitogram
debug: build
	./gitogram -debug