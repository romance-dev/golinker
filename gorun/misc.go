package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func tempDir() (string, func(), error) {
	created := []string{}
	cleanup := func() {
		for i := len(created) - 1; i >= 0; i-- {
			os.RemoveAll(created[i])
		}
	}

	tmpDir := ""
	if userSetTmpDir := os.Getenv("GOTMPDIR"); userSetTmpDir != "" {
		tmpDir = userSetTmpDir
		// Did GOTMPDIR already exist?
		if _, err := os.Stat(userSetTmpDir); err != nil {
			// directory didn't already exist
			err := os.MkdirAll(userSetTmpDir, 0700)
			if err != nil {
				return "", cleanup, fmt.Errorf("os.MkdirAll(%s): %s", userSetTmpDir, err.Error())
			}
			created = append(created, userSetTmpDir)
		}
	}

	tmpDir, err := os.MkdirTemp(tmpDir, "go-build")
	if err != nil {
		return "", cleanup, fmt.Errorf("os.MkdirTemp(%s): %s", tmpDir, err.Error())
	}
	created = append(created, tmpDir)
	return tmpDir, cleanup, nil
}

func runSimple(ctx context.Context, command string, path string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	expr := strings.Split(command, " ")
	for i, v := range expr {
		expr[i] = strings.TrimSpace(v)
	}
	cmd := exec.CommandContext(ctx, expr[0], expr[1:]...)
	cmd.Dir = path

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	slurp, _ := io.ReadAll(stderr)
	if len(slurp) == 0 {
		slurp, _ = io.ReadAll(stdout)
	}

	err = cmd.Wait()
	if err != nil {
		return "", fmt.Errorf("%w: %v", err, string(slurp))
	}

	return strings.TrimSpace(string(slurp)), nil
}

func build(ctx context.Context, goEXEPath string, outputPath string, args []string, wd *string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// Get compiler's go version
	goversion, err := runSimple(ctx, goEXEPath+" env GOVERSION", "")
	if err != nil {
		return fmt.Errorf("could not find compiler: %s", err.Error())
	}

	args = append([]string{"-o", outputPath}, args...)
	foundTagFlag := false
	for i := len(args) - 1; i >= 0; i-- {
		v := args[i]
		if strings.HasPrefix(v, "-tags") {
			if strings.HasPrefix(v, "-tags=") { // eg. "-tags=a,b,c"
				args[i] = args[i] + ",shrinkpkg" + goversion
			} else {
				// eg "-tags a,b,c"
				args[i+1] = args[i+1] + ",shrinkpkg" + goversion
			}
			// We are ignoring "-tags a b c" scenario because it is deprecated
			foundTagFlag = true
		}
	}

	if !foundTagFlag {
		args = append([]string{"-tags", "shrinkpkg" + goversion}, args...)
	}

	cmd := exec.CommandContext(ctx, goEXEPath, append([]string{"build"}, args...)...)

	// The working directory must be the directory of the package when running go build with a list of directories
	if wd != nil {
		cmd.Dir = *wd
	} else {
		if wd, err := os.Getwd(); err == nil {
			cmd.Dir = wd
		}
	}
	cmd.Env = os.Environ()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cmd.StderrPipe(): %s", err.Error())
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cmd.StdoutPipe(): %s", err.Error())
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cmd.Start(): %s", err.Error())
	}

	slurp, _ := io.ReadAll(stderr)
	if len(slurp) != 0 {
		fmt.Printf("\n%s\n", slurp)
	}

	slurp, _ = io.ReadAll(stdout)
	if len(slurp) != 0 {
		fmt.Printf("\n%s\n", slurp)
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
