package golinker

import (
	"sync"

	"github.com/pkujhd/goloader"
)

// Linker returns the Linker. It will lazy-load a Linker and always return the same one.
func Linker() func() *goloader.Linker {
	var (
		once   sync.Once
		valid  bool
		p      interface{}
		result *goloader.Linker
	)
	g := func() {
		defer func() {
			p = recover()
			if !valid {
				panic(p)
			}
		}()
		result = func() *goloader.Linker {
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
		}()
		valid = true
	}

	return func() *goloader.Linker {
		once.Do(g)
		if !valid {
			panic(p)
		}
		return result
	}
}
