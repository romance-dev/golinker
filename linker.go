package golinker

import (
	"sync"

	"github.com/pkujhd/goloader"
)

var onceLinker sync.Once
var singletonLinker *goloader.Linker
func linker() *goloader.Linker { 
	onceLinker.Do(func() {
		defer cleanup()
		fileLocs := []string{}
		pkgNames := []string{}

		for _, v := range toLoad {
			fileLocs = append(fileLocs, v.objpath)
			pkgNames = append(pkgNames, v.pkgName)
		}
		toLoad = []toLoadObj{}

		var err error
		singletonLinker, err = goloader.ReadObjs(fileLocs, pkgNames)
		if err != nil {
			panic(pkgname + ": Link error: " + err.Error())
		}
		goModFile = goModLocation()
	})
	return singletonLinker
}