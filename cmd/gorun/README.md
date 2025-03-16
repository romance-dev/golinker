# GORUN

The `go run` command is unsuitable for use with the `golinker` package because it strips the Symbols and other DEBUG information.

**gorun** emulates `go run` behavior but keeps the symbol table and other debug information (usually removed using the `-s -w` [linker flags](https://pkg.go.dev/cmd/link#hdr-Command_Line)).

> By default, 'go run' compiles the binary without generating the information used by debuggers, to reduce build time. To include debugger information in the binary, use 'go build'.
See: [Compile and run Go program](https://pkg.go.dev/cmd/go#hdr-Compile_and_run_Go_program)

It also automatically adds these flags:

1. `-ldflags=-checklinkname=0` (Required for golinker)
2. `-tags shrinkpkg$(go env GOVERSION)` (Shrinks executable file size)

## How to install

1. Run `go install`. This will place it in `$GOPATH/bin` directory.
2. Configure `$GOPATH/bin` as a [$PATH environment variable](https://en.wikipedia.org/wiki/PATH_(variable)) so you can run `gorun` from anywhere.

## How to use

1. You can use it like the official `go run` command but instead use `gorun`. It will pass environment variables and command line arguments to your newly built executable before running it. It will then delete the executable automatically.
2. You can also pass `-go {{ go version }}` flag to run it under an [older version of Go](https://go.dev/doc/manage-install). `-go 1.23.5` will compile your application using the `go 1.23.5` compiler.
3. Alternatively, you can copy the executable file and rename it: `gorun` **=>** `gorun1.23.5`.
4. When building using an older version of Go, make sure your `go.mod` file supports that version. Otherwise it will silently use the latest version.