package golinker

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"unsafe"

	"github.com/pkujhd/goloader"
)

const pkgname = "golinker"

type toLoadObj struct {
	objpath string
	pkgName string
}

type startupMessage struct {
	message string
	color   string
}

var startupMessages = map[string]startupMessage{} // modulename => message details
var toLoad = []toLoadObj{}                        // pending object files that need to be loaded
var toRemove = []string{}                         // delete out these files after loading

var symPtr = make(map[string]uintptr)

func init() {
	err := goloader.RegSymbol(symPtr)
	if err != nil {
		panic(pkgname + ": goloader.RegSymbol: " + err.Error())
	}
}

func cleanup() {
	for _, p := range toRemove {
		os.Remove(p)
	}
	toRemove = []string{}
}

// LoadMessage schedules a startup message to be displayed when a module gets initialized.
func LoadMessage(moduleName, message string) {
	if message == "" {
		return
	}
	if pos := strings.Index(message, "::"); pos == -1 {
		// pos not found
		startupMessages[moduleName] = startupMessage{
			message: message,
			color:   "black",
		}
	} else {
		// pos found
		startupMessages[moduleName] = startupMessage{
			message: message[pos+2:],
			color:   message[0:pos],
		}
	}
}

// LoadObject loads an object file to be processed by the linker.
// fullPackageName must include the module name at the start.
// object can be a path to an existing object file or the raw data of an object file.
func LoadObject(fullPackageName string, object interface{}) {
	switch pkg := object.(type) {
	case string:
		if _, err := os.Stat(pkg); errors.Is(err, os.ErrNotExist) {
			panic(fmt.Sprintf("%s: object file: %s for: %s does not exist", pkgname, pkg, fullPackageName))
		}
		toLoad = append(toLoad, toLoadObj{
			objpath: pkg,
			pkgName: fullPackageName,
		})
	case []byte:
		toLoad = append(toLoad, toLoadObj{
			objpath: writeBytesToDisk(pkg, fullPackageName),
			pkgName: fullPackageName,
		})
	case map[string][]byte:
		p, exists := pkg[strings.TrimPrefix(runtime.Version(), "go")]
		if !exists {
			panic(fmt.Sprintf("%s: %s is unavailable for %s", pkgname, fullPackageName, runtime.Version()))
		}
		toLoad = append(toLoad, toLoadObj{
			objpath: writeBytesToDisk(p, fullPackageName),
			pkgName: fullPackageName,
		})
	default:
		_ = object.(string)
	}
}

func RegTypes(typs ...interface{}) {
	if len(typs) > 0 {
		goloader.RegTypes(symPtr, typs...)
	}
}

func RegSymbolWithPath(path string) {
	goloader.RegSymbolWithPath(symPtr, path)
}

// SymbolPtr returns the memory-address of a symbol.
func SymbolPtr(fullSymbolName string, codeModule *CodeModule) unsafe.Pointer {
	fnPtr := codeModule.Syms[fullSymbolName]
	if fnPtr == 0 {
		panic(pkgname + ": could not find symbol: " + fullSymbolName)
	}
	funcPtrContainer := (uintptr)(unsafe.Pointer(&fnPtr))
	return unsafe.Pointer(&funcPtrContainer)
}

// Run_main runs the main() function of a package.
func Run_main(fullImportPath string, codeModule *CodeModule) {
	if !strings.HasSuffix(fullImportPath, ".main") {
		fullImportPath = fullImportPath + ".main"
	}
	rf := SymbolPtr(fullImportPath, codeModule)
	(*(*func())(rf))()
}
