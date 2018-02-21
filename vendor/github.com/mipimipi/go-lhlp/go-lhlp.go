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
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Contains checks if the array a contains the element e.
// inspired by: https://stackoverflow.com/questions/10485743/contains-method-for-a-slice
func Contains(a interface{}, e interface{}) bool {
	arr := reflect.ValueOf(a)

	if arr.Kind() == reflect.Slice {
		for i := 0; i < arr.Len(); i++ {
			// XXX - panics if slice element points to an unexported struct field
			// see https://golang.org/pkg/reflect/#Value.Interface
			if arr.Index(i).Interface() == e {
				return true
			}
		}
	}

	return false
}

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

// DurToHms converts a duration in a string that shows hours, minutes and
// seconds. The concrete format of the returned string is determined by
// the format string. Since DurToHms retrieves hours, minutes abd seconds
// fro the duration as intergers, the format string needs to contain %d's.
// DurToHms replaces the first %d by hours, the second by minutes and the
// last by seconds
func DurToHms(d time.Duration, format string) string {
	// a is an aray of length 2: a[0] is time in full seconds, a[1] contains
	// the sub second time
	a := strings.Split(strconv.FormatFloat(d.Seconds(), 'f', 6, 64), ".")

	// i is time in full seconds as integer
	i, err := strconv.Atoi(a[0])
	if err != nil {
		panic(err.Error())
	}

	// hours
	h := i / 3600

	// decrease i by full hours
	i -= h * 3600

	// minutes
	m := i / 60

	// seconds
	s := i - m*60

	return fmt.Sprintf(format, h, m, s)
}

// EscapePattern escapes special characters in pattern strings for usage in
// filepath.Glob() or filepath.Match. See: https://godoc.org/path/filepath#Match
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
	return err == nil, err
}

// FileSuffix return the suffix of a file without the dot. If the file name
// contains no dot, an empty string is returned
func FileSuffix(f string) string {
	if len(path.Ext(f)) == 0 {
		return ""
	}
	return path.Ext(f)[1:]
}

// FindFiles traverses directory trees starting for a list of root directories,
// to find files and directories to fulfill a certain filter condition.
// This condition must be implemented in a function, which is passed to
// FindFiles as a parameter. numWorkers is the number of concurrent Go
// routines that FindFiles uses.
// FindFiles returns two string arrays: One contains the directories and one
// the files. Both lists contain the absolute paths.
func FindFiles(roots []string, filter func(string) bool, numWorkers int) (*[]*string, *[]*string) {
	var (
		dirs     []*string      // list of directories to be returned
		files    []*string      // list of files to be returned
		traverse func(string)   // func needs to be declared here since it calls itself recursively
		wg       sync.WaitGroup // waiting group for the traversal
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
	traverse = func(dir string) {
		defer wg.Done()

		// loop at the entries of dir
		for _, entr := range entries(dir) {
			// distinguish between dirs and files. Both need to fulfill the
			// filter condition. If they do, the entry is appended to the
			// corresponding array (either dirs or files)
			if entr.IsDir() {
				// filter and add entry to dirs
				subDir := filepath.Join(dir, entr.Name())
				if filter(subDir) {
					dirs = append(dirs, &subDir)
				}
				// traverse the next level
				wg.Add(1)
				go traverse(subDir)
			} else {
				// only regular files are relevant
				if !entr.Mode().IsRegular() {
					continue
				}
				// filter and add entry to files
				file := filepath.Join(dir, entr.Name())
				if filter(file) {
					files = append(files, &file)
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
		go traverse(root)
	}

	// wait for traversals to be done
	wg.Wait()

	return &dirs, &files
}

// PathRelCopy determines first a relative path that is lexically equivalent to
// path when joined to srcBasepath with an intervening separator. If this is
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
		return "", err
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

// UserOK print the message s followed by " (Y/n)?" on stdout and askes the
// user to press either Y (to continue) or n (to stop)
func UserOK(s string) bool {
	var input string

	for {
		fmt.Printf("\r%s (Y/n)? ", s)
		if _, err := fmt.Scan(&input); err != nil {
			return false
		}
		switch {
		case input == "Y":
			return true
		case input == "n":
			return false
		}
		fmt.Println()
	}
}
