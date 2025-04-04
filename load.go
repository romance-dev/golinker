package golinker

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/pkujhd/goloader"
)

func TypeOf(i interface{}) reflect.Type {
	return reflect.TypeOf(i)
}

type Ptr = unsafe.Pointer

type CodeModule = goloader.CodeModule

// Var represents an exported package-level variable (of pointer type).
type Var struct {

	// The variable's name.
	// eg. var GlobalVar *string
	Name string

	// Ptr is the unsafe.Pointer of the pointer-type variable.
	// eg. Ptr(&GlobalVar)
	Ptr Ptr
}

// Load returns a function that lazy-loads a CodeModule specific to a package.
// It should be stored in a variable in the package. It will always return the same
// CodeModule. The CodeModule must not be unloaded unless you know that you will
// never use the package again. The CodeModule is derived from the Linker.
//
// ptrs represents package-level variables which will be initialized to point
// to the equivalent variable in the "backing-package".
func Load(fullPackageName string, pattern string, ptrs ...Var) func() *CodeModule {
	var (
		once   sync.Once
		valid  bool
		p      interface{}
		result *CodeModule
	)
	g := func() {
		defer func() {
			p = recover()
			if !valid {
				panic(p)
			}
		}()
		result = func() *CodeModule {
			codeModule, err := goloader.Load(linker(), symPtr)
			if err != nil {
				panic(fmt.Sprintf(`%s: %s: Load error: %s`, pkgname, fullPackageName, err.Error()))
			}
			for _, p := range ptrs {
				name := fullPackageName + "." + fmt.Sprintf(pattern, p.Name)
				q := (*(*func() unsafe.Pointer)(SymbolPtr(name, codeModule)))()
				*(*unsafe.Pointer)(p.Ptr) = q
			}
			return codeModule
		}()
		valid = true
	}

	return func() *CodeModule {
		once.Do(g)
		if !valid {
			panic(p)
		}
		return result
	}
}
