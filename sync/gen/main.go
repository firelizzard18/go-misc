/*
Package main generates definitions for github.com/firelizzard18/go-misc/sync.

Usage:

	go run /path/to/main.go [safe|atomic] <package> <Name> <type> <file>

Examples (relative to github.com/firelizzard18/go-misc/sync):

	go run ../main.go safe chansync Int int safe.int.go

Generates github.com/firelizzard18/go-misc/blob/master/sync/safe.int.go

	go run ../main.go atomic chansync Int int atomic.int.go

Generates github.com/firelizzard18/go-misc/blob/master/sync/atomic.int.go
*/
package main

import (
	"os"
	"fmt"
	"io"
)

const (
	s_header = `package %s

/*
* CODE GENERATED AUTOMATICALLY WITH github.com/firelizzard18/go-misc/sync/gen
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/`

	s_safe = `

// Safe%[1]s is a concurrency-safe %[2]s.
type Safe%[1]s interface {
	// Read returns the internal %[2]s value.
	Read() %[2]s
	// Write sets the internal %[2]s value to val and returns the previous
	// value.
	Write(val %[2]s) %[2]s
}

type safe%[1]s struct {
	read chan %[2]s
	write chan *safe%[1]sWrite
}

type safe%[1]sWrite struct {
	val %[2]s
	ret chan %[2]s
}

// NewSafe%[1]s returns a new safe %[2]s.
func NewSafe%[1]s(val %[2]s) Safe%[1]s {
	s := &safe%[1]s {
		read: make(chan %[2]s),
		write: make(chan *safe%[1]sWrite),
	}

	go func() {
		for {
			last := val
			select {
			case s.read <- val:
				// nothing else to do
			case wr := <- s.write:
				val = wr.val
				wr.ret <- last
			}
		}
	}()

	return s
}

func (s *safe%[1]s) Read() %[2]s {
	return <- s.read
}

func (s *safe%[1]s) Write(val %[2]s) %[2]s {
	ret := make(chan %[2]s)
	s.write <- &safe%[1]sWrite{val: val, ret: ret}
	return <- ret
}
`

	s_atomic = `

// Atomic%[1]s is a concurrency-safe, atomic %[2]s.
type Atomic%[1]s interface {
	// Read returns the internal %[2]s value.
	Read() %[2]s
	// Write sets the internal %[2]s value to val, if and only if the current
	// interal value matches old. Write returns whether or not the write was
	// successful.
	Write(old, val %[2]s) bool
}

type atomic%[1]s struct {
	read chan %[2]s
	write chan *atomic%[1]sWrite
}

type atomic%[1]sWrite struct {
	old, val %[2]s
	ret chan bool
}

// NewAtomic%[1]s returns a new atomic %[2]s.
func NewAtomic%[1]s(val %[2]s) Atomic%[1]s {
	a := &atomic%[1]s {
		read: make(chan %[2]s),
		write: make(chan *atomic%[1]sWrite),
	}

	go func() {
		for {
			last := val
			select {
			case a.read <- val:
				// nothing else to do
			case wr := <- a.write:
				if wr.old != last {
					wr.ret <- false
				} else {
					val = wr.val
					wr.ret <- true
				}
			}
		}
	}()

	return a
}

func (a *atomic%[1]s) Read() %[2]s {
	return <- a.read
}

func (a *atomic%[1]s) Write(old, val %[2]s) bool {
	ret := make(chan bool)
	a.write <- &atomic%[1]sWrite{old: old, val: val, ret: ret}
	return <- ret
}
`
)

func main() {
	if len(os.Args) == 1 {
		usage()
	} else if (len(os.Args)) != 6 {
		badargs()
	}
	
	kind := os.Args[1]
	pkg := os.Args[2]
	name := os.Args[3]
	typ := os.Args[4]
	file := os.Args[5]

	switch kind {
	case "safe":
		f := open(file)
		defer f.Close()
		header(pkg, f)
		safe(name, typ, f)
	case "atomic":
		f := open(file)
		defer f.Close()
		header(pkg, f)
		atomic(name, typ, f)
	default:
		badargs()
	}
}

func usage() {
	fmt.Println("gen [safe|atomic] <package> <Name> <type> <file>")
	os.Exit(1)
}

func badargs() {
	fmt.Println("Bad arguments")
	usage()
}

func open(file string) *os.File {
	if f, err := os.Create(file); err != nil {
		panic(err)
	} else {
		return f
	}
}

func header(pkg string, f io.Writer) {
	if _, err := f.Write([]byte(fmt.Sprintf(s_header, pkg))); err != nil {
		panic(err)
	}
}

func safe(name, typ string, f io.Writer) {
	if _, err := f.Write([]byte(fmt.Sprintf(s_safe, name, typ))); err != nil {
		panic(err)
	}
}

func atomic(name, typ string, f io.Writer) {
	if _, err := f.Write([]byte(fmt.Sprintf(s_atomic, name, typ))); err != nil {
		panic(err)
	}
}