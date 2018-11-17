package slist

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Internal constants 
const (
	commentPrefix  string = "#"
	sourcesPrefix  string = "# Sources: "
	defaultMapSize int    = 4096
)

//////////////////////////////////////////////////////////////////////
// fileStat
const (
	TYPE_FILE      = iota
	TYPE_SYMLINK   = iota
	TYPE_DIRECTORY = iota
	TYPE_DEVICE    = iota
)

// Basic csum:pathname pair
type fileStat struct {
	ftype    byte
	hash     uint64
	pathname string
	csum     []byte // file csum, 0 if not file
	linkdest string
	size     int64
	dev      int32
	mode     uint16	// Should this be os.FileMode?
	ino      uint64
	uid      uint32
	gid      uint32
	ctime    int64
	mtime    int64
}

func checksum(pathname string) []byte {
	infile, inerr := os.Open(pathname)
	if inerr != nil {
		fmt.Println(inerr)
		os.Exit(1)
	}
	md5h := md5.New()
	io.Copy(md5h, infile)
	infile.Close()
	return md5h.Sum(nil)
}

func statEqual(a, b fileStat) bool {
	if a.ftype != b.ftype {
		return false
	}

	switch a.ftype {
	case TYPE_FILE:
		return a.mode == b.mode &&
			a.uid == b.uid &&
			a.gid == a.gid &&
			a.size == b.size &&
			a.mtime == b.mtime &&
			a.ctime == b.ctime
	case TYPE_DIRECTORY:
		return a.mode == b.mode &&
			a.uid == b.uid &&
			a.gid == a.gid &&
			a.mtime == b.mtime &&
			a.ctime == b.ctime
	case TYPE_SYMLINK:
		return a.pathname == b.pathname &&
			a.linkdest == b.linkdest
	case TYPE_DEVICE:
		return false
	}
	return false
}

const (
	COMPARE_ALL  = iota
	COMPARE_CSUM = iota
	COMPARE_STAT = iota
)

func fileStatEqual(a fileStat, b fileStat, method byte) bool {
	switch method {
	case COMPARE_ALL:
		return bytes.Equal(a.csum, b.csum) &&
			statEqual(a, b)
	case COMPARE_CSUM:
		return bytes.Equal(a.csum, b.csum)
	case COMPARE_STAT:
		return statEqual(a, b)
	}
	return false
}

// Format:
// xxx Need to document
// If the Sprintf()'s in FileStat.String() changes, change these too
const SEP string = ":"     // Field separator 
const FIELDS_FILE int = 11 // Number of fields
const FIELDS_SYMLINK int = 3
const FIELDS_DIRECTORY int = 7

func (s fileStat) String() string {
	var r string

	switch s.ftype {
	case TYPE_FILE:
		r = fmt.Sprintf("F:%032x:%d:%d:%o:%d:%d:%d:%d:%d:%s",
			s.csum, s.dev, s.ino, s.mode, s.uid, s.gid, s.size, 
			s.mtime, s.ctime, s.pathname)
	case TYPE_SYMLINK:
		r = fmt.Sprintf("S:%s:%s", s.linkdest, s.pathname)
	case TYPE_DIRECTORY:
		r = fmt.Sprintf("D:%o:%d:%d:%d:%d:%s",
			s.mode, s.uid, s.gid, s.mtime, s.ctime, 
			s.pathname)
	case TYPE_DEVICE:
	default:
		panic("Invalid ftype")
	}
	return r
}

// Parse a single slist into fileStat
func parseSlist(input string) fileStat {
	var s fileStat
	var split []string
	var i, fields int

	if input == "" {
		panic("input is empty string")
	}

	// what are we looking at?
	switch input[0] {
	case 'F':
		s.ftype = TYPE_FILE
		fields = FIELDS_FILE
	case 'S':
		s.ftype = TYPE_SYMLINK
		fields = FIELDS_SYMLINK
	case 'D':
		s.ftype = TYPE_DIRECTORY
		fields = FIELDS_DIRECTORY
	default:
		panic("Invalid type identifier")
	}

	// split and trip
	split = strings.SplitAfterN(input, SEP, fields)
	for i = 0; i < fields; i++ {
		split[i] = strings.TrimRight(split[i], SEP)
	}
	split[fields-1] = strings.TrimRight(split[fields-1], "\n")

	// and parse
	switch s.ftype {
	case TYPE_FILE:
		fmt.Sscanf(split[1], "%x", &s.csum) // hex
		fmt.Sscanf(split[2], "%d", &s.dev)
		fmt.Sscanf(split[3], "%d", &s.ino)
		fmt.Sscanf(split[4], "%o", &s.mode) // octal XXX
		fmt.Sscanf(split[5], "%d", &s.uid)
		fmt.Sscanf(split[6], "%d", &s.gid)
		fmt.Sscanf(split[7], "%d", &s.size)
		fmt.Sscanf(split[8], "%d", &s.mtime)
		fmt.Sscanf(split[9], "%d", &s.ctime)
		s.pathname = split[10] // Rest of the input string
	case TYPE_SYMLINK:
		s.linkdest = split[1]
		s.pathname = split[2]
	case TYPE_DIRECTORY:
		fmt.Sscanf(split[1], "%o", &s.mode) // octal XXX
		fmt.Sscanf(split[2], "%d", &s.uid)
		fmt.Sscanf(split[3], "%d", &s.gid)
		fmt.Sscanf(split[4], "%d", &s.mtime)
		fmt.Sscanf(split[5], "%d", &s.ctime)
		s.pathname = split[6]
	}

	return s
}

//////////////////////////////////////////////////////////////////////////
// Slists 

type Slist struct {
	sources []string // Where this slist was read from/saved to
	S       map[uint64]fileStat
}

func New() *Slist {
	var s *Slist = &Slist{}
	s.S = make(map[uint64]fileStat, defaultMapSize)
	return s
}

// Add a new entry 
func (q *Slist) Add(f fileStat) {

	h := fnv.New64()
	h.Write([]byte(f.String()))
	f.hash = h.Sum64()

	_, collision := q.S[f.hash]
	if collision {
		panic("Hash collision")
	}
	q.S[f.hash] = f
}

// Remove an entry
func (q *Slist) Remove(f fileStat) {
	delete(q.S, f.hash)
}

func (p *Slist) Len() int {
	return len(p.S)
}

func (q *Slist) Search(target fileStat) *fileStat {

	k, ok := q.S[target.hash]
	if ok {
		return &k
	}
	return nil
}

// Return a copy of an Slist
func (q *Slist) Copy() *Slist {
	var c *Slist

	c = New()
	c.sources = q.sources
	c.S = q.S
	return c
}

// Appends one slist to another
func (q *Slist) Append(r *Slist) *Slist {
	for _, v := range r.sources {
		q.sources = append(q.sources, v)
	}
	for _, v := range r.S {
		q.Add(v)
	}
	return q
}

// Print just the pathnames of files.
func (q *Slist) PrintFiles() {
	for _, v := range q.S {
		if v.ftype == TYPE_FILE {
			fmt.Println(v.pathname)
		}
	}
}

// Load an slist from a file.
func (q *Slist) Load(filename string) {
	var f *os.File
	var e error
	var s string

	f, e = os.Open(filename)
	if e != nil {
		fmt.Printf("Can't open file %s: %s\n", filename, e)
		panic("fatal error")
	}

	r := bufio.NewReader(f)
	for {
		s, e = r.ReadString('\n')
		if e == io.EOF {
			break
		}
		if e != nil {
			fmt.Fprintf(os.Stderr, "error reading from %s: %s\n",
				filename, e.Error())
			os.Exit(1)
		}

		s = strings.Trim(s, "\n")
		s = strings.TrimSpace(s)

		// The '# Sources: ' line.
		if strings.HasPrefix(s, sourcesPrefix) {
			l := len(sourcesPrefix)
			for _, v := range strings.SplitAfter(s[l:], " ") {
				q.sources = append(q.sources,
					strings.TrimSpace(v))
			}
		}
		// Skip other comments/metadata
		if strings.HasPrefix(s, commentPrefix) {
			continue
		}

		q.Add(parseSlist(s))
	}
	f.Close()
}

func (q *Slist) dump(f *os.File) {
	var s string

	fmt.Fprintf(f, "# SLIST VESION 0\n")
	fmt.Fprintf(f, "# File list contains %d entries\n", q.Len())
	fmt.Fprintf(f, sourcesPrefix)
	for _, v := range q.sources {
		s += v + " "
	}
	s = strings.TrimSpace(s)
	fmt.Fprintf(f, "%s\n", s)
	for _, v := range q.S {
		fmt.Fprintf(f, "%s\n", v)
	}
}

func (q *Slist) Dump() {
	q.dump(os.Stdout)
}

func (q *Slist) Save(filename string) {
	var f *os.File
	var e error

	f, e = os.Create(filename)
	if f == nil {
		fmt.Printf("Can't create file %s: %s\n", filename, e)
		panic("fatal error")
	}
	q.dump(f)
	f.Close()
}

/////////////////////////////////////////////////////////////////
// Set arithmetic 

func Difference(a *Slist, b *Slist) *Slist {
	var r *Slist

	r = New()
	for _, v := range a.S {
		if b.Search(v) == nil { // v not in list b
			r.Add(v)
		}
	}
	return r
}

func Intersection(a *Slist, b *Slist) *Slist {
	var r *Slist

	r = New()
	for _, v := range a.S {
		if b.Search(v) != nil { // v is in list b 
			r.Add(v)
		}
	}
	return r
}

func Subset(a *Slist, b *Slist) bool {
	if a.Len() > b.Len() {
		return false
	}
	for _, v := range a.S {
		if b.Search(v) == nil { // v is not in list b
			return false
		}
	}
	return true
}

func Union(a *Slist, b *Slist) *Slist {
	var u *Slist

	u = a.Copy()
	for _, v := range b.S {
		if a.Search(v) == nil { // element from b not in a
			u.Add(v)
		}
	}

	return u
}

func Disjoint(a *Slist, b *Slist) bool {
	l := Intersection(a, b)
	if l.Len() == 0 {
		return true
	}
	return false
}

func Equal(a *Slist, b *Slist) bool {
	if a.Len() != b.Len() {
		return false
	}

	for i, _ := range a.S {
		if !fileStatEqual(a.S[i], b.S[i], COMPARE_CSUM) {
			return false
		}
	}
	return true
}

// Impmement A - B - C - ... 
func MDifference(s []*Slist) *Slist {
	var diff *Slist

	if len(s) < 2 {
		panic("Invalid argument list")
	}
	diff = Difference(s[0], s[1])

	for i := 2; i < len(s); i++ {
		diff = Difference(diff, s[i])
	}
	return diff
}

// A + B + C + ...
func MUnion(s []*Slist) *Slist {
	var sum *Slist

	if len(s) < 2 {
		panic("Invalid argument list")
	}
	sum = Union(s[0], s[1])

	for i := 2; i < len(s); i++ {
		sum = Union(sum, s[i])
	}

	return sum
}

// A + B + C + ...
func MEqual(s []*Slist) bool {
	if len(s) < 2 {
		panic("Invalid argument list")
	}
	if !Equal(s[0], s[1]) {
		return false
	}

	for i := 1; i < len(s)-1; i++ {
		if !Equal(s[i], s[i+1]) {
			return false
		}
	}

	return true
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
	s       *Slist
	visited map[string]bool
	root    string
}


// read the directory tree, generate csums and sort.  May be a goroutine 
func (s *Slist) ProcessPath(pathname string, done chan bool) {
	var w Walker
	w.visited = make(map[string]bool)
	w.s = s
	s.sources = append(s.sources, pathname)

	walkfn := func(pathname string, f os.FileInfo, err error) error {
		var tmp fileStat
		if f.IsDir() {
			// Skip directories if we've already visited them
			if w.visited[pathname] {
				return filepath.SkipDir
			} 
			w.visited[pathname] = true
		} else if f.Mode() & os.ModeSymlink != 0 {
			tmp.ftype = TYPE_SYMLINK
			// error handling needed here
			s, _ := filepath.EvalSymlinks(pathname) 
			tmp.linkdest = QuotePath(s)
		} else if f.Mode() & os.ModeType == 0 { // regular file
			tmp.ftype = TYPE_FILE
			tmp.csum = checksum(pathname)
		} else {
			return nil
		}
		tmp.pathname = QuotePath(pathname)
		tmp.mode = uint16(f.Mode().Perm()) // Unix permission bits
		if f.Mode() & os.ModeSetuid != 0 {
			tmp.mode |= 04000
		}
		if f.Mode() & os.ModeSetgid != 0 {
			tmp.mode |= 02000
		}
		if f.Mode() & os.ModeSticky != 0 {
			tmp.mode |= 01000
		}
		tmp.size = f.Size()
		tmp.dev = f.Sys().(*syscall.Stat_t).Dev
		tmp.ino = f.Sys().(*syscall.Stat_t).Ino
		tmp.uid = f.Sys().(*syscall.Stat_t).Uid
		tmp.gid = f.Sys().(*syscall.Stat_t).Gid
		tmp.ctime, _ = f.Sys().(*syscall.Stat_t).Ctimespec.Unix()
		tmp.mtime, _ = f.Sys().(*syscall.Stat_t).Mtimespec.Unix()
		
		w.s.Add(tmp)
		return nil
	}
	
	filepath.Walk(pathname, walkfn)

	// Signal complete for goroutine
	if done != nil {
		done <- true
	}
}
