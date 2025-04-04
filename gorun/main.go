package main

import (
	"fmt"
	l "log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
)

var Version string = "1.0.0"

var rxBase = regexp.MustCompile(`^.+(?:1.)(\d{1,}\.{0,1}\d*)$`)

var gopath string
var goEXEPath = "go"

var log = l.New(os.Stderr, "gorun: ", 0)

func init() {
	var err error
	gopath, err = runSimple(nil, "go env GOPATH", "")
	if err != nil {
		log.Fatalln(err)
	}
	// Test if executable name gives clue to intended go version
	arg0 := os.Args[0]
	if runtime.GOOS == "windows" {
		arg0 = strings.TrimSuffix(arg0, ".exe")
	}
	res := rxBase.FindStringSubmatch(filepath.Base(arg0))
	if len(res) == 2 {
		goEXEPath = filepath.Join(gopath, "bin", "go1."+res[1])
	}
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "-v" {
		fmt.Println(filepath.Base(os.Args[0]) + " version: " + Version)
		os.Exit(0)
	}

	appArgs := []string{} // args to pass on to newly built app

	// https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies
	flagsThatExpectPath := map[string]struct{}{
		"-C":       {}, // dir
		"-modfile": {}, // file
		"-overlay": {}, // file
		"-pgo":     {}, // file
		"-pkgdir":  {}, // dir
		"-o":       {}, // dir
	}

	var foundPath *int // first path found
	var endPath *int   // last adjoining path found
	var wd *string
	for i, arg := range args {
		if foundPath == nil {
			if strings.HasPrefix(arg, "-") {
				// We found a flag
				continue
			}

			// Check if arg is a path to a directory or file
			path, err := filepath.EvalSymlinks(arg)
			if err != nil {
				// not a valid file or directory. It could be a value for a flag
				continue
			}

			f, _ := os.Stat(path)
			if f.IsDir() {
				wd = &path
			}

			// We found a file or directory
			if i == 0 {
				foundPath = &i
				endPath = &i
				continue
			}
			_, exists := flagsThatExpectPath[args[i-1]]
			if exists {
				// The path belongs to a flag that expects a path
				continue
			}

			foundPath = &i
			endPath = &i
			// if strings.HasSuffix(arg, ".go") {
			// 	goFiles = true
			// }
		} else {
			// We have already found a path. We are checking if the next is also a path
			if strings.HasPrefix(arg, "-") {
				// We found a flag
				break
			}

			// Check if arg is a path to a directory or file
			_, err := filepath.EvalSymlinks(arg)
			if err != nil {
				// not a valid file or directory. It could be a command for the newly built app
				break
			}
			endPath = &i
		}
	}

	if foundPath != nil {
		appArgs = append(appArgs, args[*endPath+1:]...) // Must come before next line
		args = args[0 : *endPath+1]
	}

	// Obtain a temp directory to store build
	tmpDir, cleanup, err := tempDir()
	if err != nil {
		cleanup()
		log.Fatalln(err)
	}
	defer cleanup()

	// Find go compiler
	for i := len(args) - 1; i >= 0; i-- {
		v := args[i]
		if strings.HasPrefix(v, "-go") {
			splits := strings.SplitAfterN(v, "=", 2)
			if len(splits) == 1 {
				// No '=' found
				if i == len(args)-1 {
					cleanup()
					log.Fatalln("no go version provided to -go:")
					return
				}
				next := args[i+1]
				goEXEPath = filepath.Join(gopath, "bin", "go"+next)
				args = append(args[:i+1], args[i+2:]...) // Remove arg after -go
			} else {
				goEXEPath = filepath.Join(gopath, "bin", "go"+splits[len(splits)-1])
			}
			args = append(args[:i], args[i+1:]...) // Remove -go
		} else if strings.HasPrefix(v, "-ldflags") {
			splits := strings.SplitAfterN(v, "=", 2)
			rhs := splits[len(splits)-1]
			// Remove '' or "" from rhs ?
			splits = strings.Split(rhs, " ")
			for _, v := range splits {
				if strings.TrimSpace(v) == "-s" || strings.TrimSpace(v) == "-w" {
					cleanup()
					log.Fatalln("can't contain ldflags: '-s' or '-w'")
					return
				}
			}
		}
	}
	args = append([]string{"-ldflags=-checklinkname=0"}, args...) // prepend

	// https://emretanriverdi.medium.com/graceful-shutdown-in-go-c106fe1a99d9
	// https://itsfoss.com/linux-exit-codes/
	// https://learn.microsoft.com/en-us/cpp/c-runtime-library/signal-constants?view=msvc-170
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-c
		switch sig {
		case os.Interrupt:
			cleanup()
			os.Exit(130)
		case syscall.SIGTERM:
			cleanup()
			os.Exit(143)
		}
	}()

	outputPath := "run" // Determine this based on go.mod?
	if runtime.GOOS == "windows" {
		outputPath = outputPath + ".exe"
	}
	outputPath = filepath.Join(tmpDir, outputPath)

	err = build(nil, goEXEPath, outputPath, args, wd)
	if err != nil {
		cleanup()
		log.Fatalf("could not build: %s", err.Error())
		return
	}

	// Run the built executable
	cmd := exec.Command(outputPath, appArgs...)
	if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}
	// cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		// fmt.Println("err", err)
	}
}
