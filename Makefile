APP = promtop
# WORKING_PATH = /go/src/$(APP)
# DOCKER_CMD = docker run --rm -it -e GOCACHE=/tmp --user $$(id -u):$$(id -g) -v $$PWD:$(WORKING_PATH) -v $$GOPATH/pkg:/go/pkg -v $$GOPATH/bin:/go/bin -w $(WORKING_PATH) golang:1.18-buster
WORKING_PATH = .
DOCKER_CMD =
VERSION ?= 0.0.1
SOURCES = $(wildcard **/*.go)

$(APP): $(SOURCES)
	$(DOCKER_CMD) go build -ldflags="-X 'main.version=$(VERSION)'" -o $(WORKING_PATH)/$(APP) main.go

build: $(APP)

run: build
	PROMTOP_PROMETHEUS_URL=http://prometheus.lan:9090 ./$(APP)

clean:
	rm -f $(APP)

fmt:
	$(DOCKER_CMD) gofmt -s -w $(WORKING_PATH)

test:
	$(DOCKER_CMD) go test

watch:
	fd -e go | entr make --no-print-directory --always-make

install: build
	sudo cp $(APP) /usr/local/bin/$(APP)
	sudo chmod +x /usr/local/bin/$(APP)
