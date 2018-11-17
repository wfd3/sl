package main

import (
	"slist"
	"os"
	"fmt"
	"flag"
)


func usage() {
	fmt.Println("sldiffm [-full] file1 file2 [file3 ... fileN]")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	var full, help bool

	// Define local cmd line flags (global flags defined in func init()
	flag.BoolVar(&full, "full", false, "Show full slist entries")
	flag.BoolVar(&help, "help", false, "Help")

	// Parse flags
	flag.Parse();

	if (help) {
		usage()
	}

	var sl []*slist.Slist
	sl = make([]*slist.Slist, len(flag.Args()))
	for i, v := range flag.Args() {
		sl[i] = slist.New()
		sl[i].Load(v)
	}
	
	diff := slist.MDifference(sl)
	if full {
		diff.Dump()
	} else {
		diff.PrintFiles()
	}
}
