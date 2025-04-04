package golinker

import (
	"sync"

	"github.com/fatih/color"
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
		deps = nil

		// Print out startup messages
		for _, m := range startupMessages {
			switch m.color {
			case "black":
				color.Black(m.message)
			case "red":
				color.Red(m.message)
			case "green":
				color.Green(m.message)
			case "yellow":
				color.Yellow(m.message)
			case "blue":
				color.Blue(m.message)
			case "magenta":
				color.Magenta(m.message)
			case "cyan":
				color.Cyan(m.message)
			case "white":
				color.White(m.message)
			default:
				color.Black(m.color + "::" + m.message)
			}
		}
		startupMessages = nil
	})
	return singletonLinker
}
