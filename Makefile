HUGS_SRCS := $(shell find . -type f -name '*.go' -o -name '*.templ')

hugs: $(HUGS_SRCS)
	go tool templ fmt .
	go tool templ generate .
	go mod tidy	
	go build -o hugs main.go 

install_templ:
	go install github.com/a-h/templ/cmd/templ@latest	