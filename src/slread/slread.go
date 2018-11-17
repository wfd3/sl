package main

import (
	"slist"
	"flag"
	"fmt"
	"os"
)

func usage() {
	fmt.Println("slread: -s slist.file [-o output]")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	var pathname, output string

	// Define local cmd line flags (global flags defined in func init()
	flag.StringVar(&pathname, "s", "", "Source slist")
	flag.StringVar(&output, "o", "", "Output file")

	// Parse flags
	flag.Parse();
	
	if pathname == "" {
		panic("Pathname can't be empty")
	}


	var a *slist.Slist
	a = slist.New()
	a.Load(pathname)
	if output == "" {
		a.Dump()
	} else {
		a.Save(output)
	}
}
