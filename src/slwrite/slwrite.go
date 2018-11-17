package main

import (
	"slist"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func usage() {
	fmt.Println("slwrite: -d directory [-o output]")
	flag.PrintDefaults()
	os.Exit(1)
}

func dirExists(pathname string) bool {

	fi, err := os.Lstat(pathname)
	if err != nil {
		fmt.Printf("os.Lstat: %s\n", err.Error());
		return false
	}
	return fi.IsDir()
}

func main() {
	var a *slist.Slist
	var aDir, output string

	// Define local cmd line flags (global flags defined in func init()
	flag.StringVar(&aDir, "d", "./", "Directory")
	flag.StringVar(&output, "o", "", "Output path, stdout if omitted")

	// Parse flags
	flag.Parse()

	if aDir == "" {
		panic("Source directory can't be empty")
	}
	if !dirExists(aDir) {
		fmt.Printf("Error: directory '%s' doesn't exist\n", aDir)
		os.Exit(1)
	}
	aDir, _ = filepath.Abs(aDir)

	a = slist.New()
	a.ProcessPath(aDir, nil)

	if output != "" {
		a.Save(output)
	} else {
		a.Dump()
	}
}
