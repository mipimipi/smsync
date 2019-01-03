// Copyright (C) 2018 Michael Picht
//
// This file is part of go-lhlp (Go's little helper).
//
// go-lhlp is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-lhlp is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-lhlp. If not, see <http://www.gnu.org/licenses/>.

// Package file contains practical and handy functions for dealing with files
// and directories.
package file

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// Info extends the standard interface os.FileInfo
type Info interface {
	os.FileInfo
	Path() string // get complete file name
}

// info is an internal helper structire that implements FileInfo
type info struct {
	os.FileInfo
	path string // complete name of the file
}

// Path implements the Path() method, so that fileinfo implements FileInfo
func (inf info) Path() string { return inf.path }

// create FileInfo from os.FileInfo and a path
func newInfo(fi os.FileInfo, p string) Info { return info{fi, p} }

// ValidPropagate defines if validity shall be propagated to sub
// directories. Needed for Find()
type ValidPropagate int

// constants for Find(): propagation of validity to sub directories
const (
	NoneFromSuper    ValidPropagate = iota // no propagation to sub directories
	ValidFromSuper                         // propagate valid=true to sub directories
	InvalidFromSuper                       // propagate valid=false to sub directories
)

// Copy copies srcFn to dstFn. Prequisite is, that srcFn and (if existing)
// dstFn are regular files (i.e. no devices etc.). In case both files are the
// same, nothing is done. In case, dstFn is already existing it is overwritten.
func Copy(srcFn, dstFn string) error {
	var (
		err    error
		exists bool
		srcFi  os.FileInfo
		dstFi  os.FileInfo
		src    *os.File
		dst    *os.File
	)

	// check if source file exists
	exists, err = Exists(srcFn)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("CopyFile: Source file %s doesn't exist", srcFn)
	}

	// get fileinfo for source file
	srcFi, err = os.Stat(srcFn)
	if err != nil {
		return err
	}

	// make sure that source file is a regular file
	if !srcFi.Mode().IsRegular() {
		return fmt.Errorf("CopyFile: Non-regular source file %s (%q)", srcFi.Name(), srcFi.Mode().String())
	}

	// determine existence of dest file
	exists, err = Exists(dstFn)
	if err != nil {
		return err
	}
	if exists {
		// get fileinfo for dest file
		dstFi, err = os.Stat(dstFn)
		if err != nil {
			return err
		}
		// make sure that dest file is a regular file
		if !(dstFi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: Non-regular destination file %s (%q)", dstFi.Name(), dstFi.Mode().String())
		}
		// if source and dest file are the same: do nothing
		if os.SameFile(srcFi, dstFi) {
			return nil
		}
	}

	// open source file and defer closing
	src, err = os.Open(srcFn)
	if err != nil {
		return err
	}
	defer func() { err = src.Close() }()

	// create/open dest file and defer closing
	dst, err = os.OpenFile(dstFn, os.O_WRONLY|os.O_CREATE, srcFi.Mode())
	if err != nil {
		return err
	}
	defer func() { err = dst.Close() }()

	// copy content of source file to dest file
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	// flush dest file
	return dst.Sync()
}

// EscapePattern escapes special characters in pattern strings for usage in
// filepath.Glob() or filepath.Match()
// See: https://godoc.org/path/filepath#Match
func EscapePattern(s string) string {
	special := [...]string{"[", "?", "*"}
	for _, sp := range special {
		s = strings.Replace(s, sp, "\\"+sp, -1)
	}
	return s
}

// Exists returns true if path exists, otherwise false
func Exists(path string) (bool, error) {
	exists, _, err := ExistsInfo(path)
	return exists, err
}

// ExistsInfo returns true if path exists, otherwise false. In addition to
// Exists it also return file.Info
func ExistsInfo(path string) (bool, Info, error) {
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("Existence of file '%s' couldn't be determined: %v", path, err)
	}
	return true, newInfo(fi, path), nil
}

// Find traverses directory trees to find files that fulfill a certain filter
// condition. It starts at a list of root directories. The condition must be
// implemented in a function, which is passed to FindFiles as a parameter. This
// condition returns two boolean value. The first one determines if a certain
// entry is valid (i.e. fulfills the actual filter condition), the second
// determines (only in case of a directory) if the filter result (i.e. the first
// boolean) shall be propagated to the entries of the directory.
// numWorkers is the number of concurrent Go routines that FindFiles uses.
// FindFiles returns two string arrays: One contains the directories and one
// the files that fulfill the filter condition. Both lists contain the absolute
// paths.
// This function is inspired by the Concurrent Directory Traversal from the
// book "The Go Programming Language" by Alan A. A. Donovan & Brian W.
// Kernighan.
// See: https://github.com/adonovan/gopl.io/blob/master/ch8/du4/main.go
func Find(roots []string, filter func(Info, ValidPropagate) (bool, ValidPropagate), numWorkers int) (files *[]*Info) {
	var (
		descend func(Info, ValidPropagate, chan<- *Info) // func needs to be declared here since it calls itself recursively
		wg      sync.WaitGroup                           // waiting group for the concurrent traversal
		res     = make(chan *Info)                       // result channel to collect th relevant files
		sema    = make(chan struct{}, numWorkers)        // use as semaphore to restrict the number of Go routines

	)

	defer close(sema)

	// allocate result array
	files = new([]*Info)

	// function to retrieve the entries of a directory
	entries := func(dir Info) []os.FileInfo {
		// send to limited buffer and retrieve from buffer at the end
		sema <- struct{}{}
		defer func() { <-sema }()

		// retrieve directory entries
		entrs, err := ioutil.ReadDir(dir.Path())
		if err != nil {
			return nil
		}

		return entrs
	}

	// function to check if a directory is relevant. If yes, descend into that
	// directory
	checkDescendDir := func(fi os.FileInfo, dir string, vp ValidPropagate, res chan<- *Info) {
		inf := newInfo(fi, dir)
		valid, vpSub := filter(inf, vp)
		if valid {
			// send dir to result channel
			go func() {
				res <- &inf
			}()
		}
		// descend into directory
		if vpSub != InvalidFromSuper {
			wg.Add(1)
			go descend(inf, vpSub, res)
		}
	}

	// function to traverse the directory tree. Calls itself recursively
	descend = func(dir Info, vp ValidPropagate, res chan<- *Info) {
		defer wg.Done()

		// loop at the entries of dir
		for _, entr := range entries(dir) {
			// distinguish between dirs and files. Both need to fulfill the
			// filter condition. If they do, the entry is appended to the
			// corresponding array (either dirs or files)
			if entr.IsDir() {
				// check entr and descend
				checkDescendDir(entr, filepath.Join(dir.Path(), entr.Name()), vp, res)
			} else {
				// only regular files are relevant
				if !entr.Mode().IsRegular() {
					continue
				}
				// filter and add entry to files
				inf := newInfo(entr, filepath.Join(dir.Path(), entr.Name()))
				if valid, _ := filter(inf, vp); valid {
					// send file to result channel
					res <- &inf
				}
			}
		}
	}

	// start traversal for the root directories

	for _, root := range roots {
		// get file info for root directory
		fi, err := os.Stat(root)
		if err != nil {
			panic(err.Error())
		}
		// verify that root is a directory
		if !fi.IsDir() {
			continue
		}
		// check root and descend
		checkDescendDir(fi, root, NoneFromSuper, res)
	}

	// wait for traversals to be finalized
	go func() {
		wg.Wait()
		close(res)
	}()

	// retrieve results from channel
	for f := range res {
		*files = append(*files, f)
	}

	return files
}

// IsEmpty returns true if file or directory is empty, otherwise false
func IsEmpty(f string) (bool, error) {
	fi, err := os.Stat(f)
	if err != nil {
		return false, err
	}
	if fi.IsDir() {
		entries, err := ioutil.ReadDir(f)
		if err != nil {
			return false, err
		}
		return (len(entries) == 0), nil
	}
	if fi.Mode().IsRegular() {
		return (fi.Size() == 0), nil
	}
	return false, nil
}

// MkdirAll creates a directory named path, along with any necessary parents.
// In this regard, it behaves like the standard os.MkdirAll. In contrast to
// this, it doesn't complain if path already exists.
func MkdirAll(path string, perm os.FileMode) error {
	var err error

	if _, err = os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, perm)
	}
	return err
}

// PathRelCopy determines first a relative path that is lexically equivalent to
// path when joined to srcBase with an intervening separator. If this is
// successful, it returns a path joined from dstBase, a separator and the
// relative path from the previous step.
func PathRelCopy(srcBase, path, dstBase string) (string, error) {
	var (
		rel string
		err error
	)

	// determine the relative path using filepath.Rel()
	// See: https://godoc.org/path/filepath#Rel
	if rel, err = filepath.Rel(srcBase, path); err != nil {
		return "", fmt.Errorf("PathRelCopy: %v", err)
	}

	// if the relative path is empty: Just return dstBase
	if rel == "" {
		return dstBase, nil
	}

	// other wise join dstBase and the relative path
	return filepath.Join(dstBase, rel), nil
}

// PathTrunk returns the file path without the file extension.
// E.g. Trunk("/home/test/abc.mp3") return "/home/test/abc"
func PathTrunk(p string) string {
	return p[0 : len(p)-len(path.Ext(p))]
}

// RemoveEmpty removes a file or directory if it is empty. If it is not empty,
// RemoveEmpty returns and error
func RemoveEmpty(f string) error {

	var (
		empty bool
		err   error
	)

	if empty, err = IsEmpty(f); err != nil {
		return err
	}

	if empty {
		if err = os.Remove(f); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("file or directory %s is not empty: cannot be removed", f)
	}

	return nil
}

// Stat returns info about the file whose path is passed as
// parameter. In this regard, it is simlar to the standard function os.Stat.
// Different from it, FileStat return file info of type FileInfo, i.e. extended
// by Path(), which return the path of the file.
func Stat(path string) (Info, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return newInfo(fi, path), nil
}

// Suffix return the suffix of a file without the dot. If the file name
// contains no dot, an empty string is returned
func Suffix(f string) string {
	if len(path.Ext(f)) == 0 {
		return ""
	}
	return path.Ext(f)[1:]
}
