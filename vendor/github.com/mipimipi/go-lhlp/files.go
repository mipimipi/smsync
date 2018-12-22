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

// Package lhlp contains practical and handy functions that are useful in many
// Go projects, but which are not part of the standards Go libraries.
package lhlp

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

// FileInfo extends the standard interface os.FileInfo
type FileInfo interface {
	os.FileInfo
	Path() string // get complete file name
}

// fileinfo is an internal helper structire that implements FileInfo
type fileinfo struct {
	os.FileInfo
	path string // complete name of the file
}

// Path implements the Path() method, so that fileinfo implements FileInfo
func (fi fileinfo) Path() string { return fi.path }

// create FileInfo from os.FileInfo and a path
func newFI(fi os.FileInfo, p string) FileInfo { return fileinfo{fi, p} }

// CopyFile copies srcFn to dstFn. Prequisite is, that srcFn and (if existing)
// dstFn are regular files (i.e. no devices etc.). In case both files are the
// same, nothing is done. In case, dstFn is already existing it is overwritten.
func CopyFile(srcFn, dstFn string) error {
	var (
		err    error
		exists bool
		srcFi  os.FileInfo
		dstFi  os.FileInfo
		src    *os.File
		dst    *os.File
	)

	// check if source file exists
	exists, err = FileExists(srcFn)
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
	exists, err = FileExists(dstFn)
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

// DirIsEmpty returns true is directory d is empty, otherwise false
func DirIsEmpty(d string) (bool, error) {
	entries, err := ioutil.ReadDir(d)
	if err != nil {
		return false, err
	}
	return (len(entries) == 0), nil
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

// FileExists returns true if filePath exists, otherwise false
func FileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("Existence of file '%s' couldn't be determined: %v", filePath, err)
	}
	return true, nil
}

// FileIsEmpty returns true is file f is empty, otherwise false
func FileIsEmpty(f string) (bool, error) {
	fi, err := os.Stat(f)
	if err != nil {
		return false, err
	}
	return (fi.Size() == 0), nil
}

// FileStat returns info about the file whose path is passed as
// parameter. In this regard, it is simlar to the standard function os.Stat.
// Different from it, FileStat return file info of type FileInfo, i.e. extended
// by Path(), which return the path of the file.
func FileStat(path string) (FileInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return newFI(fi, path), nil
}

// FileSuffix return the suffix of a file without the dot. If the file name
// contains no dot, an empty string is returned
func FileSuffix(f string) string {
	if len(path.Ext(f)) == 0 {
		return ""
	}
	return path.Ext(f)[1:]
}

// FindFiles traverses directory trees to find files and directories that
// fulfill a certain filter condition. It starts at a list of root
// directories. The condition must be implemented in a function, which is
// passed to FindFiles as a parameter. This condition returns two boolean value
// The first one determines if a certain entry is valid (i.e. fulfills the
// actual filter condition), the second determines (only in case of a
// directory) if FindFiles shall descend.
// numWorkers is the number of concurrent Go routines that FindFiles uses.
// FindFiles returns two string arrays: One contains the directories and one
// the files that fulfill the filter condition. Both lists contain the absolute
// paths.
// This function is inspired by the Concurrent Directory Traversal from the
// book "The Go Programming Language" by Alan A. A. Donovan & Brian W.
// Kernighan.
// See: https://github.com/adonovan/gopl.io/blob/master/ch8/du4/main.go
func FindFiles(roots []string, filter func(string) (bool, bool), numWorkers int) (*[]FileInfo, *[]FileInfo) {
	var (
		dirs    []FileInfo     // list of directories to be returned
		files   []FileInfo     // list of files to be returned
		descend func(string)   // func needs to be declared here since it calls itself recursively
		wg      sync.WaitGroup // waiting group for the concurrent traversal
	)

	// create buffered channel, used as semaphore to restrict the number of Go routines
	sema := make(chan struct{}, numWorkers)
	defer close(sema)

	// function to retrieve the entries of a directory
	entries := func(dir string) []os.FileInfo {
		// send to limited buffer and retrieve from buffer at the end
		sema <- struct{}{}
		defer func() { <-sema }()

		// retrieve directory entries
		entrs, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil
		}

		return entrs
	}

	// function to traverse the directory tree. Calls itself recursively
	descend = func(dir string) {
		defer wg.Done()

		// loop at the entries of dir
		for _, entr := range entries(dir) {
			// distinguish between dirs and files. Both need to fulfill the
			// filter condition. If they do, the entry is appended to the
			// corresponding array (either dirs or files)
			if entr.IsDir() {
				// filter and add entry to dirs
				subDir := filepath.Join(dir, entr.Name())
				isValid, goDown := filter(subDir)
				if isValid {
					// create extended FileInfo and append it to dirs array
					dirs = append(dirs, newFI(entr, subDir))
				}
				// traverse the next level
				if goDown {
					wg.Add(1)
					go descend(subDir)
				}
			} else {
				// only regular files are relevant
				if !entr.Mode().IsRegular() {
					continue
				}
				// filter and add entry to files
				file := filepath.Join(dir, entr.Name())
				if valid, _ := filter(file); valid {
					// create extended FileInfo and append it to dirs array
					files = append(files, newFI(entr, file))
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
		// start traversal for this directory
		wg.Add(1)
		go descend(root)
	}

	// wait for traversals to be finalized
	wg.Wait()

	return &dirs, &files
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
