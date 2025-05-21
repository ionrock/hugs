HUGS_SRCS := $(shell find . -type f -name '*.go' -o -name '*.templ')

hugs: $(HUGS_SRCS)
	go mod tidy
	go build -o hugs main.go 