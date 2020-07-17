build:
	go mod download
	go build -o manager

default: build

upgrade:
	go mod download
	go get -u -v
	go mod tidy
	go mod verify

test:
	go test

man: build
	./manager --help-man | man -l -

install:
	@if ! test -f manager;then echo 'run "make build" first'; exit 1; fi

ifneq ($(shell id -u), 0)
	@echo "You must be root to perform this action."
	@exit 1
endif
	@mkdir -p /usr/local/share/man/man8
	cp manager /usr/bin/manager
	/usr/bin/manager --help-man > manager.1
	install -Dm644 manager.1 /usr/share/man/man8/manager.8
	rm manager.1
	@echo Installed successfully!

uninstall:
ifneq ($(shell id -u), 0)
	@echo "You must be root to perform this action."
	@exit 1
endif
	rm /usr/bin/manager
	rm -f /usr/share/man/man8/manager.8
	@echo Uninstalled successfully!

clean:
	rm -f manager.1
	rm -f manager
	rm -f main
