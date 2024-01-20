package ioutil_test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/ioutil"
	"github.com/andrescosta/goico/pkg/test"
)

// It grants read, write, and execute permissions to the owner and the group
const defaultPermission = 0o770

type node struct {
	name    string
	entries []*node // nil if the entry is a file
}

var rootTreeNoFiles = &node{
	"testdir",
	[]*node{
		{"d1", []*node{}},
		{
			"d2",
			[]*node{
				{"d3", []*node{}},
				{
					"d4",
					[]*node{},
				},
			},
		},
	},
}

var rootTreeNoDirs = &node{
	"f1", nil,
}

var rootTree = &node{
	"testdir",
	[]*node{
		{"f1t", nil},
		{"d1", []*node{}},
		{"f2t", nil},
		{
			"d2",
			[]*node{
				{"f3t", nil},
				{"d3", []*node{}},
				{
					"d4",
					[]*node{
						{"f4t", nil},
						{"f5t", nil},
					},
				},
			},
		},
	},
}

func TestWriteToRandomFile(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	preffix := "test"
	suffix := "ioutil"
	content := []byte("testing TestWriteToRandomFile")

	f, err := WriteToRandomFile(tempDir, preffix, suffix, content)
	test.Nil(t, err, fmt.Sprintf("WriteToRandomFile %q : %s", tempDir, err))
	if !strings.HasPrefix(filepath.Base(f), preffix) {
		t.Fatalf("The file name %s does not start with %s", f, preffix)
	}
	if !strings.HasSuffix(f, suffix) {
		t.Fatalf("The file name %s does not start with %s", f, preffix)
	}
	c, err := os.ReadFile(f)
	test.Nil(t, err, fmt.Sprintf("ReadFile: Error reading the file %s: %s", f, err))
	if !bytes.Equal(c, content) {
		t.Fatalf("Content is different %s %s", c, content)
	}
}

func TestFileExists(t *testing.T) {
	t.Parallel()
	fileName := tempRandomFileName(t)
	ok, err := FileExists(fileName)
	test.Nil(t, err)
	if ok {
		t.Fatalf("The file should not exists: %s ", fileName)
	}
	err = os.WriteFile(fileName, []byte("content"), os.ModeAppend)
	test.Nil(t, err)
	ok, err = FileExists(fileName)
	test.Nil(t, err)
	if !ok {
		t.Fatalf("The file does not exists: %s ", fileName)
	}
}

func TestTouch(t *testing.T) {
	t.Parallel()
	fileName := tempRandomFileName(t)
	err := Touch(fileName)
	test.Nil(t, err)
	c, err := os.ReadFile(fileName)
	test.Nil(t, err)
	if len(c) > 0 {
		t.Fatalf("The file is not empty: %s", c)
	}
}

func TestFiles(t *testing.T) {
	tempDir := t.TempDir()
	makeTestTree(t, rootTree, tempDir, nil)
	dir := filepath.Join(tempDir, rootTree.name)
	files, err := Files(dir)
	test.Nil(t, err)
	test.NotEmpty(t, files)
	for _, f := range files {
		if f.IsDir() {
			t.Fatalf("%s is a directory", f.Name())
		}
	}
	checkAllFilesReturned(t, rootTree, files, func(n *node, de []fs.DirEntry) bool {
		for _, des := range de {
			if n.name == des.Name() && !des.IsDir() {
				return true
			}
		}
		return false
	})
}

func TestDirs(t *testing.T) {
	tempDir := t.TempDir()
	makeTestTree(t, rootTree, tempDir, nil)
	files, err := Dirs(tempDir)
	test.Nil(t, err)
	test.NotEmpty(t, files)
	for _, f := range files {
		if !f.IsDir() {
			t.Fatalf("%s is a file", f.Name())
		}
	}
	checkAllDirsReturned(t, rootTree, files, func(n *node, de []fs.DirEntry) bool {
		for _, des := range de {
			if n.name == des.Name() && des.IsDir() {
				return true
			}
		}
		return false
	})
}

func TestNoDirs(t *testing.T) {
	tempDir := t.TempDir()
	makeTestTree(t, rootTreeNoDirs, tempDir, nil)
	files, err := Dirs(tempDir)
	test.Nil(t, err)
	test.Empty(t, files)
}

func TestNoFiles(t *testing.T) {
	tempDir := t.TempDir()
	makeTestTree(t, rootTreeNoFiles, tempDir, nil)
	dir := filepath.Join(tempDir, rootTree.name)
	files, err := Files(dir)
	test.Nil(t, err)
	test.Empty(t, files)
}

func TestReadOldestFile(t *testing.T) {
	tempDir := t.TempDir()
	makeTestTree(t, rootTree, tempDir, func() { time.Sleep(1 * time.Second) })
	n := getFirstFile(t, rootTree)
	d, f, err := ReadOldestFile(tempDir, "f", "t")
	name := filepath.Base(*f)
	test.Nil(t, err)
	if name != n.name {
		t.Fatalf("ioutil.Files: expecting %s getting %s", n.name, name)
	}
	if !bytes.Equal(d, []byte(n.name)) {
		t.Fatalf("ioutil.Files: content is different between %s and %s", n.name, name)
	}
}

func TestLastLinesNoLfNoEmpty(t *testing.T) {
	t.Parallel()
	f := tempRandomFileName(t)
	createFile(t, f, 10, "line", "\n", false)
	l, err := LastLines(f, 3, true, true)
	test.Nil(t, err)
	test.NotIsLen(t, l, 3)
	if l[0] != "line7" {
		t.Fatalf(`Expecting "line7" got %s`, l[0])
	}
	if l[1] != "line8" {
		t.Fatalf(`Expecting "line7" got %s`, l[1])
	}
	if l[2] != "line9" {
		t.Fatalf(`Expecting "line7" got %s`, l[2])
	}
}

func TestLastLinesNoCrLfNoEmpty(t *testing.T) {
	t.Parallel()
	f := tempRandomFileName(t)
	createFile(t, f, 10, "line", "\r\n", false)
	l, err := LastLines(f, 3, true, true)
	test.Nil(t, err)
	test.NotIsLen(t, l, 3)
	if l[0] != "line7" {
		t.Fatalf(`Expecting "line7" got %s`, l[0])
	}
	if l[1] != "line8" {
		t.Fatalf(`Expecting "line7" got %s`, l[1])
	}
	if l[2] != "line9" {
		t.Fatalf(`Expecting "line7" got %s`, l[2])
	}
}

func TestLastLinesSmallerEmpty(t *testing.T) {
	t.Parallel()
	f := tempRandomFileName(t)
	createFile(t, f, 3, "line", "\r\n", true)
	l, err := LastLines(f, 10, false, true)
	test.Nil(t, err)
	test.NotIsLen(t, l, 3)
	if l[0] != "" {
		t.Fatalf(`Expecting "" got %s`, l[0])
	}
	if l[1] != "line1" {
		t.Fatalf(`Expecting "line1" got %s`, l[1])
	}
	if l[2] != "" {
		t.Fatalf(`Expecting "" got %s`, l[2])
	}
}

func TestLastLinesLfNoEmpty(t *testing.T) {
	t.Parallel()
	f := tempRandomFileName(t)
	createFile(t, f, 10, "line", "\n", false)
	l, err := LastLines(f, 3, true, false)
	test.Nil(t, err)
	test.NotIsLen(t, l, 3)
	if l[0] != "line7\n" {
		t.Fatalf(`Expecting "line7" got %s`, l[0])
	}
	if l[1] != "line8\n" {
		t.Fatalf(`Expecting "line7" got %s`, l[1])
	}
	if l[2] != "line9\n" {
		t.Fatalf(`Expecting "line7" got %s`, l[2])
	}
}

func TestLastLinesCrLfNoEmpty(t *testing.T) {
	t.Parallel()
	f := tempRandomFileName(t)
	createFile(t, f, 10, "line", "\r\n", false)
	l, err := LastLines(f, 3, true, false)
	test.Nil(t, err)
	test.NotIsLen(t, l, 3)
	if l[0] != "line7\r\n" {
		t.Fatalf(`Expecting "line7" got %s`, l[0])
	}
	if l[1] != "line8\r\n" {
		t.Fatalf(`Expecting "line7" got %s`, l[1])
	}
	if l[2] != "line9\r\n" {
		t.Fatalf(`Expecting "line7" got %s`, l[2])
	}
}

func TestLastLinesSmallerNoEmpty(t *testing.T) {
	t.Parallel()
	f := tempRandomFileName(t)
	createFile(t, f, 2, "line", "\n", false)
	l, err := LastLines(f, 10, true, false)
	test.Nil(t, err)
	test.NotIsLen(t, l, 2)
	if l[0] != "line0\n" {
		t.Fatalf(`Expecting "line0" got %s`, l[0])
	}
	if l[1] != "line1\n" {
		t.Fatalf(`Expecting "line1" got %s`, l[1])
	}
}

func TestLastLinesCrLfEmpty(t *testing.T) {
	t.Parallel()
	f := tempRandomFileName(t)
	createFile(t, f, 10, "line", "\r\n", true)
	l, err := LastLines(f, 5, false, true)
	test.Nil(t, err)
	test.NotIsLen(t, l, 5)
	if l[0] != "line5" {
		t.Fatalf(`Expecting "line0" got %s`, l[0])
	}
	if l[1] != "" {
		t.Fatalf(`Expecting "line1" got %s`, l[1])
	}
	if l[2] != "line7" {
		t.Fatalf(`Expecting "line0" got %s`, l[2])
	}
	if l[3] != "" {
		t.Fatalf(`Expecting "line1" got %s`, l[3])
	}
	if l[4] != "line9" {
		t.Fatalf(`Expecting "line1" got %s`, l[4])
	}
}

func TestLastLinesLfEmpty(t *testing.T) {
	t.Parallel()
	f := tempRandomFileName(t)
	createFile(t, f, 10, "line", "\n", true)
	l, err := LastLines(f, 5, false, true)
	test.Nil(t, err)
	test.NotIsLen(t, l, 5)
	if l[0] != "line5" {
		t.Fatalf(`Expecting "line0" got %s`, l[0])
	}
	if l[1] != "" {
		t.Fatalf(`Expecting "line1" got %s`, l[1])
	}
	if l[2] != "line7" {
		t.Fatalf(`Expecting "line0" got %s`, l[2])
	}
	if l[3] != "" {
		t.Fatalf(`Expecting "line1" got %s`, l[3])
	}
	if l[4] != "line9" {
		t.Fatalf(`Expecting "line1" got %s`, l[4])
	}
}

func TestLastLinesNoCrLfEmpty(t *testing.T) {
	t.Parallel()
	f := tempRandomFileName(t)
	createFile(t, f, 10, "line", "\r\n", true)
	l, err := LastLines(f, 5, false, false)
	test.Nil(t, err)
	test.NotIsLen(t, l, 5)
	if l[0] != "line5\r\n" {
		t.Fatalf(`Expecting "line0" got %s`, l[0])
	}
	if l[1] != "\r\n" {
		t.Fatalf(`Expecting "" got %s`, l[1])
	}
	if l[2] != "line7\r\n" {
		t.Fatalf(`Expecting "line0" got %s`, l[2])
	}
	if l[3] != "\r\n" {
		t.Fatalf(`Expecting "" got %s`, l[3])
	}
	if l[4] != "line9\r\n" {
		t.Fatalf(`Expecting "line1" got %s`, l[4])
	}
}

func TestLastLinesNoLfEmpty(t *testing.T) {
	t.Parallel()
	f := tempRandomFileName(t)
	createFile(t, f, 10, "line", "\n", true)
	l, err := LastLines(f, 5, false, false)
	test.Nil(t, err)
	test.NotIsLen(t, l, 5)
	if l[0] != "line5\n" {
		t.Fatalf(`Expecting "line0" got %s`, l[0])
	}
	if l[1] != "\n" {
		t.Fatalf(`Expecting "" got %s`, l[1])
	}
	if l[2] != "line7\n" {
		t.Fatalf(`Expecting "line0" got %s`, l[2])
	}
	if l[3] != "\n" {
		t.Fatalf(`Expecting "" got %s`, l[3])
	}
	if l[4] != "line9\n" {
		t.Fatalf(`Expecting "line1" got %s`, l[4])
	}
}

func tempRandomFileName(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	return filepath.Join(tempDir, randomEncodedString(t))
}

func randomEncodedString(t *testing.T) string {
	t.Helper()
	rb := make([]byte, 5)
	_, err := rand.Read(rb)
	test.Nil(t, err)
	return base64.URLEncoding.EncodeToString(rb)
}

func makeTestTree(t *testing.T, n *node, baseDir string, waiter func()) {
	t.Helper()
	entryName := filepath.Join(baseDir, n.name)
	if n.entries == nil {
		err := os.WriteFile(entryName, []byte(n.name), os.ModeAppend)
		test.Nil(t, err)
		if waiter != nil {
			waiter()
		}
		return
	}
	err := os.Mkdir(entryName, defaultPermission)
	test.Nil(t, err)
	for _, nn := range n.entries {
		makeTestTree(t, nn, entryName, waiter)
	}
}

func checkAllFilesReturned(t *testing.T, n *node, entries []fs.DirEntry, checker func(*node, []fs.DirEntry) bool) {
	t.Helper()
	if n.entries == nil {
		found := checker(n, entries)
		if !found {
			t.Fatalf("File %s not found", n.name)
			return
		}
	}
	if n.entries != nil {
		for _, nn := range n.entries {
			checkAllFilesReturned(t, nn, entries, checker)
		}
	}
}

func checkAllDirsReturned(t *testing.T, n *node, entries []fs.DirEntry, checker func(*node, []fs.DirEntry) bool) {
	t.Helper()
	if n.entries != nil {
		found := checker(n, entries)
		if !found {
			t.Fatalf("Dir %s not found", n.name)
			return
		}
	}
	if n.entries != nil {
		for _, nn := range n.entries {
			checkAllDirsReturned(t, nn, entries, checker)
		}
	}
}

func getFirstFile(t *testing.T, n *node) *node {
	t.Helper()
	if n.entries == nil {
		return n
	}
	for _, nn := range n.entries {
		r := getFirstFile(t, nn)
		if r != nil {
			return r
		}
	}
	return nil
}

func createFile(t *testing.T, name string, lines int, preffix string, suffix string, empty bool) {
	t.Helper()
	var buffer bytes.Buffer
	for i := 0; i < lines; i++ {
		if empty && i%2 == 0 {
			buffer.Write([]byte(suffix))
			continue
		}
		buffer.Write([]byte(fmt.Sprintf("%s%s%s", preffix, strconv.Itoa(i), suffix)))
	}
	if err := os.WriteFile(name, buffer.Bytes(), os.ModeAppend); err != nil {
		t.Fatalf("os.WriteFile: %s", err)
	}
}
