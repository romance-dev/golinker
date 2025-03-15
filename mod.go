package golinker

import (
	"bufio"
	"bytes"
	"errors"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func stackTrace() []byte {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}

func isGoRun() bool {
	// Find out if application is being run using `go run ...`
	path, _ := os.Executable()

	// Due to caching, the technique below doesn't work
	// https://github.com/golang/go/issues/8451
	// if userSetTmpDir := os.Getenv("GOTMPDIR"); userSetTmpDir != "" {
	// 	if !strings.HasPrefix(path, userSetTmpDir) {
	// 		return false
	// 	}
	// }

	dir, _ := filepath.Split(path)
	segments := strings.Split(dir, string(filepath.Separator))

	goRun := false
	for _, v := range segments {
		if strings.HasPrefix(v, "go-build") {
			goRun = true
			break
		}
	}

	return goRun
}

var goModFile *modfile.File

func goModLocation() *modfile.File {
	if !isGoRun() {
		return nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(stackTrace()))

	// Look for main.main() in stack trace
	foundMain := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if foundMain {
			dir := filepath.Dir(line)
			// Check if directory is git managed
			if _, err := os.Stat(filepath.Join(dir, ".git")); errors.Is(err, os.ErrNotExist) {
				_parent := filepath.Dir(dir)
				if _, err := os.Stat(filepath.Join(_parent, ".git")); errors.Is(err, os.ErrNotExist) {
					_parent := filepath.Dir(_parent)
					if _, err := os.Stat(filepath.Join(_parent, ".git")); errors.Is(err, os.ErrNotExist) {
						_parent := filepath.Dir(_parent)
						if _, err := os.Stat(filepath.Join(_parent, ".git")); errors.Is(err, os.ErrNotExist) {
							_parent := filepath.Dir(_parent)
							if _, err := os.Stat(filepath.Join(_parent, ".git")); errors.Is(err, os.ErrNotExist) {
								// Package is not git managed
								return nil
							}
						}
					}
				}
			}

			goModPath := filepath.Join(dir, "go.mod")
			dat, err := os.ReadFile(goModPath)
			if err != nil {
				return nil
			}
			f, err := modfile.Parse(goModPath, dat, nil)
			if err != nil {
				return nil
			}
			return f
		} else {
			if strings.HasPrefix(line, "main.main()") {
				foundMain = true
			}
		}
	}
	return nil
}
