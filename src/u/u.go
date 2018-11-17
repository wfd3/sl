package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)



type Slist map[string]string

func checksum(pathname string) string {
	infile, inerr := os.Open(pathname)
	if inerr != nil {
		fmt.Println(inerr)
		os.Exit(1)
	}
	md5h := md5.New()
	io.Copy(md5h, infile)
	infile.Close()

	csum := md5h.Sum(nil)
	return fmt.Sprintf("%x", csum) // hex
}

func dump(sl Slist) {
	for k, v := range sl { 
		fmt.Printf("%s: %s\n", v, k)
	}
}

//////////////////////////////////////////////////////////////
// Return a path with all members containing spaces safely quoted
func QuotePath(pathname string) string {
	var splitpath []string

	if pathname == "" {
		return ""
	}

	splitpath = strings.Split(pathname, "/")
	for i, _ := range splitpath {
		if strings.Contains(splitpath[i], " ") {
			splitpath[i] = "\"" + splitpath[i] + "\""
		}
	}
	return strings.Join(splitpath, "/")
}

//////////////////////////////////////////////////////////////
// Directory tree walker

type Walker struct {
	s       Slist
	visited map[string]bool
	root    string
}

// read the directory tree, generate csums and sort.  May be a goroutine 
func ProcessPath(pathname string) Slist {
	sl := make(map[string]string)

	walkfn := func(pathname string, f os.FileInfo, err error) error {
		if f.Mode() & os.ModeType == 0 && os.ModeSymlink != 0 { // regular file
			csum := checksum(pathname)
			if duppath, present := sl[csum]; present == true {
				fmt.Printf("DUP: %s:%s -> %s\n", pathname, csum, duppath)
				return nil
			}
			sl[csum] = pathname
//			fmt.Printf("ADD: %s:%s\n", pathname, csum)
		} 
		return nil
	}
	
	filepath.Walk(pathname, walkfn)

	return sl
}



func main() {
	//	sl :=
	ProcessPath("/usr/local")
//	dump(sl)
}
