package main

import (
	"github.com/ddkwork/golibrary/mylog"
	"github.com/ddkwork/golibrary/safeType"
	"go/format"
	"os"
	"path/filepath"
)

func WriteGoFile[T safeType.Type](name string, data T) (ok bool) {
	s := safeType.New(data)
	source, err := format.Source(s.Bytes())
	if !mylog.Error(err) {
		return write(name, false, s.Bytes())
	}
	return write(name, false, source)
}

func WriteAppend[T safeType.Type](name string, data T) bool     { return write(name, true, data) }
func WriteTruncate[T safeType.Type](name string, data T) bool   { return write(name, false, data) }
func WriteBinaryFile[T safeType.Type](name string, data T) bool { return write(name, false, data) }

func write[T safeType.Type](name string, isAppend bool, data T) (ok bool) {
	if !CreatDirectory(filepath.Dir(name)) {
		return
	}
	fnCreateFile := func() (*os.File, error) {
		if isAppend {
			return os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
		}
		return os.Create(name) //== os.Truncate(name, 0)
	}
	f, err := fnCreateFile()
	if !mylog.Error(err) {
		switch {
		case os.IsExist(err):
			return write(name, isAppend, data)
		case os.IsNotExist(err):
			return write(name, isAppend, data)
		case os.IsPermission(err):
			return
		}
	}
	s := safeType.New(data)
	if !mylog.Error2(f.Write(s.Bytes())) {
		return
	}
	return mylog.Error(f.Close())
}

func CreatDirectory(dir string) bool {
	fnMakeDir := func() bool { return mylog.Error(os.MkdirAll(dir, os.ModePerm)) }
	info, err := os.Stat(dir)
	switch {
	case os.IsExist(err):
		return true
	case os.IsNotExist(err):
		return fnMakeDir()
	case err == nil:
		return info.IsDir()
	default:
		return mylog.Error(err)
	}
}
