.PHONY: test test-sim race bench lint ci build run

build:
	mkdir -p /tmp/extralibs
	ln -sf /usr/lib/x86_64-linux-gnu/libXxf86vm.so.1 /tmp/extralibs/libXxf86vm.so
	CGO_LDFLAGS="-L/tmp/extralibs" go build -o evolution ./cmd/evolution/

run: build
	./evolution

# Alle Tests (braucht X11-Header für render/ui/cmd)
test:
	go test ./...

# Nur sim-Packages (kein X11 nötig — für lokale Entwicklung ohne X11-Header)
test-sim:
	go test ./config/... ./gen/... ./sim/... ./testworld/...
	go test -tags noebiten ./render/...

race:
	go test -race ./sim/...

bench:
	go test -run='^$$' -bench=. -benchmem ./sim/...

ci: test race
	go run ./tools/check_ebiten_imports.go ./...
	go run ./tools/check_global_rand.go ./sim/...
