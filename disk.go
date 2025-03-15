package golinker

import (
	"bytes"
	"compress/gzip"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func getTempDir() string {
	tempDir := os.TempDir()

	// Create a file to test if permissions allow storage
	randN := strconv.Itoa(rand.Int())
	randFile := filepath.Join(tempDir, randN)
	err := os.WriteFile(randFile, []byte{}, 0644)
	if err == nil {
		os.Remove(randFile)
		return tempDir
	}

	// os.TempDir failed so try directory of executable
	exePath, err1 := os.Executable()
	if err1 != nil {
		panic(pkgname + ": os.WriteFile(" + tempDir + "): " + err.Error())
	}

	tempDir = filepath.Dir(exePath)
	randFile = filepath.Join(tempDir, randN)
	err1 = os.WriteFile(randFile, []byte{}, 0644)
	if err1 != nil {
		panic(pkgname + ": os.WriteFile(" + tempDir + "): " + err1.Error())
	}
	os.Remove(randFile)

	return tempDir
}

// writeBytesToDisk writes the package's object file to disk and returns the location
func writeBytesToDisk(pkg []byte, fullPackageName string) string {
	// Create a temp directory
	tempDir := getTempDir()
	dst := filepath.Join(tempDir, strings.ReplaceAll(fullPackageName, "/", "_")+"_"+strconv.Itoa(rand.Int())+".golinker")

	// Assume gzipped
	zr, err := gzip.NewReader(bytes.NewReader(pkg))
	if err != nil {
		// Not a valid gzip file. Assume it's a raw object file.
		err := os.WriteFile(dst, pkg, 0666)
		if err != nil {
			panic(pkgname + ": os.WriteFile(" + dst + "): " + err.Error())
		}
		toRemove = append(toRemove, dst)
		return dst
	}
	defer zr.Close()

	f, err := os.Create(dst)
	if err != nil {
		panic(pkgname + ": os.Create(" + dst + "): " + err.Error())
	}
	toRemove = append(toRemove, dst)
	defer f.Close()

	// Write file to disk
	_, err = io.Copy(f, zr)
	if err != nil {
		panic(pkgname + ": io.Copy(): " + err.Error())
	}
	return dst
}
