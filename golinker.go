package golinker

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/pkujhd/goloader"
)

const pkgname = "golinker"

type toLoadObj struct {
	objpath string
	pkgName string
}

var toLoad = []toLoadObj{} // pending objects that need to be loaded
var toRemove = []string{}  // delete out these files when application terminates

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

// LoadObject loads an object file to be processed by the linker.
// fullPackageName must include the module name at the start.
// object can be a path to an existing object file or the raw data to an object file.
func LoadObject(fullPackageName string, object any) {
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

var onceLinker = sync.OnceValue(func() *goloader.Linker {
	defer cleanup()
	fileLocs := []string{}
	pkgNames := []string{}

	for _, v := range toLoad {
		fileLocs = append(fileLocs, v.objpath)
		pkgNames = append(pkgNames, v.pkgName)
	}
	toLoad = []toLoadObj{}

	linker, err := goloader.ReadObjs(fileLocs, pkgNames)
	if err != nil {
		panic(pkgname + ": Link error: " + err.Error())
	}

	goModFile = goModLocation()
	return linker
})

// Linker returns the Linker. It will lazy-load a Linker
// and always return the same one.
func Linker() *goloader.Linker {
	return onceLinker()
}

func RegTypes(typs ...any) {
	goloader.RegTypes(symPtr, typs...)
}

func RegSymbolWithPath(path string) {
	goloader.RegSymbolWithPath(symPtr, path)
}

// SymbolPtr returns the memory-address to a symbol.
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
