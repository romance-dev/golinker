package golinker

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
)

var onceDeps sync.Once
var deps map[string]string

// extractVersion returns the version tag or commit hash
func extractVersion(version string) string {
	// Check version type
	// 1. v1.2 (tag)
	// 2. v6.0.0-20250303095825-24047e466509 (psuedo)
	splits := strings.Split(version, "-")
	if len(splits) == 3 && semver.IsValid(splits[0]) {
		if _, err := time.Parse("20060102150405", splits[1]); err == nil {
			// valid timestamp
			if _, err := hex.DecodeString(splits[2]); err == nil {
				// valid sha
				return splits[2]
			}
		}
	}
	return version
}

// CheckDeps must only be called from within an init().
func CheckDeps(moduleName string, imports ...string) {
	onceDeps.Do(func() {
		_deps, ok := debug.ReadBuildInfo()
		if !ok {
			panic("couldn't get fetch build info")
		}

		// Dependencies baked into executable
		deps = map[string]string{}
		for _, mod := range _deps.Deps {
			if mod.Replace != nil {
				if _, exists := deps[mod.Replace.Path]; exists {
					panic(mod.Replace.Path + " already exists")
				}
				deps[mod.Replace.Path] = mod.Replace.Version
			} else {
				if _, exists := deps[mod.Path]; exists {
					panic(mod.Path + " already exists")
				}
				deps[mod.Path] = mod.Version
			}
		}
	})

	defer func() {
		if r := recover(); r != nil {
			table := tablewriter.NewWriter(os.Stderr)
			table.SetAutoWrapText(false)
			table.SetHeader([]string{"Dep (" + moduleName + ")", "Required", "Current", "*"})

			for _, i := range imports {
				splits := strings.SplitN(i, "=>", 2)
				splits = strings.SplitN(splits[len(splits)-1], "::", 2)
				ip := splits[0]
				_v := splits[1]
				_current := deps[ip]
				final := ""

				v, current := extractVersion(_v), extractVersion(_current)
				if v != current {
					final = "<=="
				}
				table.Append([]string{ip, _v, _current, final})
			}
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Dependency version mismatch: Exact Versions are required.")
			fmt.Fprintln(os.Stderr, "The replace directive can be used to pin dependencies to a single commit (rather than a minimum version).")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "(This is a limitation also inherent in the plugin pkg: https://pkg.go.dev/plugin#hdr-Warnings)")
			fmt.Fprintln(os.Stderr, "⮑ “Similar crashing problems are likely to arise unless all common dependencies of the application and its plugins are built from exactly the same source code.”")
			fmt.Fprintln(os.Stderr, "")

			table.Render()

			// Generate go mod lines
			f := &modfile.File{}
			f.AddModuleStmt(moduleName)
			replaceStmts := []string{}
			for _, i := range imports {
				splits := strings.SplitN(i, "=>", 2)
				require := splits[0]
				requireSplits := strings.SplitN(require, "::", 2)
				ip := requireSplits[0]
				ver := requireSplits[1]
				f.AddRequire(ip, ver)
				if len(splits) > 1 {
					replace := splits[1]
					replaceSplits := strings.SplitN(replace, "::", 2)
					replaceStmts = append(replaceStmts, fmt.Sprintf(`%s => %s %s`, ip, replaceSplits[0], replaceSplits[1]))
				} else {
					replaceStmts = append(replaceStmts, fmt.Sprintf(`%s => %s %s`, ip, ip, ver))
				}
			}
			modText, _ := f.Format()

			if pos := bytes.Index(modText, []byte("require")); pos != -1 {
				modText = modText[pos:]
			}

			if len(replaceStmts) > 0 {
				buf := bytes.NewBuffer(modText)
				buf.WriteString("\n// Pinned dependencies for " + moduleName + ":\nreplace (\n")
				for _, r := range replaceStmts {
					buf.WriteString("	" + r + "\n")
				}
				buf.WriteString(")")
				modText = buf.Bytes()
			}

			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "sample go.mod:")
			fmt.Fprintln(os.Stderr, string(modText))
			os.Exit(1)
		}
	}()

	for _, i := range imports {
		var ip string // import path
		var _v string // version (could be tag (ie. v1.0.0) or (psuedo) a1030444159b))

		splits := strings.SplitN(i, "=>", 2)
		if len(splits) > 1 {
			// Replace found
			splits := strings.SplitN(splits[1], "::", 2)
			ip = splits[0]
			_v = splits[1]
		} else {
			// No replacement OR artificial replacement we added in of itself
			splits := strings.SplitN(splits[0], "::", 2)
			ip = splits[0]
			_v = splits[1]
		}

		_val, exists := deps[ip]
		if !exists {
			// huh? The required dependency is not in the build.
			panic(ip)
		}

		if extractVersion(_v) != extractVersion(_val) {
			panic(ip)
		}
	}

	return
}

func GoVersionCheck(moduleName string, goBuildVersion string) {
	if rv := runtime.Version(); rv != goBuildVersion {
		fmt.Fprintf(os.Stderr, `module %s: was built using %s but application was built using: %s. Consider changing build tag.`, moduleName, goBuildVersion, rv)
		os.Exit(1)
	}
}
