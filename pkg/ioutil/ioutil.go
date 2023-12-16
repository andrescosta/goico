package ioutil

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

func WriteToRandomFile(path, preffix, suffix string, data []byte) error {
	if err := CreateDirsIfNotExist(path); err != nil {
		return err
	}
	fn, err := randomFileName(preffix, suffix)
	if err != nil {
		return err
	}
	fullpath := BuildPathWithFile(path, fn)
	return Write(fullpath, data)
}

func FileExists(fullPath string) (bool, error) {
	_, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func CreateEmptyIfNotExists(fullpath string) error {
	e, err := FileExists(fullpath)

	if err != nil {
		return err
	}

	if !e {
		return Write(fullpath, []byte(""))
	}

	return nil
}

func Write(file string, data []byte) error {
	f, err := os.Create(file)

	if err != nil {
		return errors.Join(errors.New("error creating file"), err)
	}

	defer func() {
		f.Close()
	}()

	w := bufio.NewWriter(f)

	if _, err := w.Write(data); err != nil {
		return errors.Join(errors.New("error writing file"), err)
	}

	if err := w.Flush(); err != nil {
		return errors.Join(errors.New("error flushing data"), err)
	}

	return nil
}

func Subdirs(path string) ([]string, error) {
	dires, err := os.ReadDir(path)

	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0)

	for _, d := range dires {
		dirs = append(dirs, d.Name())
	}

	return dirs, nil
}

func filesSorted(path string, preffix, suffix string) ([]os.DirEntry, error) {
	var files []fs.DirEntry

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return files, nil
	}

	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(d.Name(), preffix) && strings.HasSuffix(d.Name(), suffix) {
			files = append(files, d)
		}

		return nil
	})

	if err != nil {
		return nil, errors.Join(errors.New("error getting files"), err)
	}

	sort.Slice(files, func(i, j int) bool {
		in1, _ := files[i].Info()

		in2, _ := files[j].Info()

		return in1.ModTime().Unix() < in2.ModTime().Unix()
	})

	return files, nil
}

func Files(path string) ([]os.DirEntry, error) {
	return files(path, func(d fs.DirEntry) bool {
		return !d.IsDir()
	})
}

func Dirs(path string) ([]os.DirEntry, error) {
	f, err := files(path, func(d fs.DirEntry) bool {
		return d.IsDir()
	})

	if err != nil {
		return nil, err
	}

	if len(f) == 0 {
		return f, nil
	}
	return f[1:], nil
}

func files(path string, filter func(fs.DirEntry) bool) ([]os.DirEntry, error) {
	var files []fs.DirEntry

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return files, nil
	}

	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if filter(d) {
			files = append(files, d)
		}

		return nil
	})

	if err != nil {
		return nil, errors.Join(errors.New("error getting files"), err)
	}

	sort.Slice(files, func(i, j int) bool {
		in1, _ := files[i].Info()

		in2, _ := files[j].Info()

		return in1.ModTime().Unix() < in2.ModTime().Unix()
	})

	return files, nil
}

func topNfilesSorted(path, preffix, suffix string, n int) ([]os.DirEntry, error) {
	files, err := filesSorted(path, preffix, suffix)
	if err != nil {
		return nil, err
	}
	if len(files) < n {
		return files, nil
	}
	return files[:n], nil
}

func OldestFile(path, preffix, suffix string) ([]byte, *string, error) {
	files, err := topNfilesSorted(path, preffix, suffix, 1)

	if err != nil {
		return nil, nil, err
	}

	if len(files) == 0 {
		return nil, nil, nil
	}

	filename := BuildFullPathWithFile([]string{path}, files[0].Name())

	data, err := Read(filename)

	if err != nil {
		return nil, nil, err
	}

	return data, &filename, nil
}

func Read(file string) ([]byte, error) {
	return os.ReadFile(file)
}

func Remove(file string) error {
	return os.Remove(file)
}

func Rename(file, newFile string) error {
	return os.Rename(file, newFile)
}

func randomFileName(preffix, suffix string) (string, error) {
	size := 20
	rb := make([]byte, size)
	_, err := rand.Read(rb)
	if err != nil {
		return "", err
	}
	rs := base64.URLEncoding.EncodeToString(rb)
	return preffix + rs + suffix, nil
}

func CreateDirsIfNotExist(directory string) error {
	return os.MkdirAll(directory, os.ModeExclusive)
}

func BuildFullPath(directories []string) string {
	return filepath.Join(directories...)
}

func BuildFullPathWithFile(directories []string, file string) string {
	return filepath.Join(BuildFullPath(directories), file)
}

func BuildPathWithFile(path, file string) string {
	return filepath.Join(path, file)
}

func LastLines(file string, nlines int, skipEmpty bool, noincludecrlf bool) ([]string, error) {
	bufferSize := int64(4096)

	fileHandle, err := os.Open(file)

	if err != nil {
		return nil, err
	}

	defer fileHandle.Close()

	stat, err := fileHandle.Stat()

	if err != nil {
		return nil, err
	}

	filesize := stat.Size()

	if bufferSize > filesize {
		bufferSize = filesize
	}

	cursor := -bufferSize

	buffer := make([]byte, bufferSize)

	accLines := make([]string, 0)

	currLine := ""

loop:

	for {
		newOffset, err := fileHandle.Seek(cursor, io.SeekEnd)

		if err != nil {
			return nil, err
		}

		bytesRead, err := fileHandle.Read(buffer)

		if err != nil {
			return nil, err
		}

		for i := bytesRead - 1; i >= 0; i-- {
			if buffer[i] == '\n' || buffer[i] == '\r' {
				// not a new line because line break is CRLF(\n\r),

				if currLine != "\n" {
					appendLine := true

					if skipEmpty {
						if strings.TrimSpace(currLine) == "" {
							appendLine = false
						}
					}

					if appendLine {
						currLine = removeCRLF(noincludecrlf, currLine)

						accLines = append(accLines, currLine)

						if nlines == len(accLines) {
							break loop
						}
					}

					currLine = ""
				}
			}

			currLine = string(buffer[i]) + currLine
		}

		if newOffset == 0 {
			if currLine != "" {
				currLine = removeCRLF(noincludecrlf, currLine)

				accLines = append(accLines, currLine)
			}

			break loop
		}

		if newOffset > bufferSize {
			cursor -= int64(bytesRead)
		} else {
			// We are at the beginning of the file. We cannot move beyond that.

			cursor -= newOffset

			buffer = make([]byte, newOffset)
		}
	}

	slices.Reverse(accLines)

	return accLines, nil
}

func removeCRLF(noincludecrlf bool, currLine string) string {
	if noincludecrlf {
		currLine = strings.TrimSpace(currLine)
	}

	return currLine
}
