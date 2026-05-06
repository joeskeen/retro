default:
  @just --list

build:
  go build -o retro ./cmd/cli

test:
  go test ./...

clean:
  rm -f retro

run EXAMPLE="example/cat":
  ./retro run {{EXAMPLE}}

build-image PATH:
  ./retro build {{PATH}}

install: build
  cp retro ~/bin/retro

rm IMAGE:
  ./retro rm {{IMAGE}}