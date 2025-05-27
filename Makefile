HUGS_SRCS := $(shell find . -type f -name '*.go' -o -name '*.templ')

hugs: $(HUGS_SRCS)
	go mod tidy
	go tool templ fmt .
	go tool templ generate .
	go build -o hugs main.go

install_templ:
	go install github.com/a-h/templ/cmd/templ@latest

local: hugs
	./hugs -d ../ionrock.github.io

server:
	air \
	--build.cmd "make hugs" \
	--build.bin "make local" \
	--build.delay "100" \
	--build.include_ext "go,templ" \
	--build.stop_on_error "false" \

bootstrap:
	go install github.com/air-verse/air@latest
