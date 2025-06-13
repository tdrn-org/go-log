// file.go
//
// Copyright (C) 2023-2025 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license. See the LICENSE file for details.

package log

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const fileWriterOpenFileFlags int = os.O_WRONLY | os.O_CREATE | os.O_APPEND
const fileWriterOpenFileMode os.FileMode = 0660

type fileWriter struct {
	fileName      string
	fileSizeLimit int64
	mutex         sync.Mutex
	file          *os.File
	lastErr       error
}

func (w *fileWriter) Write(b []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	// open file, if needed
	err := w.openIfNeeded("")
	if err != nil {
		if w.recordLastErrIfNeeded(err) {
			w.log(slog.LevelWarn, "failed to open log file", slog.String("file", w.fileName), slog.Any("err", err))
		}
		return os.Stderr.Write(b)
	}
	// rotate, if needed
	err = w.rotateIfNeeded()
	if err != nil {
		if w.recordLastErrIfNeeded(err) {
			w.log(slog.LevelWarn, "failed to rotate log file", slog.String("file", w.fileName), slog.Any("err", err))
		}
		return os.Stderr.Write(b)
	}
	// write (and close if failing, to retry on next write)
	n, err := w.file.Write(b)
	if err != nil {
		_ = w.file.Close()
		w.file = nil
		if w.recordLastErrIfNeeded(err) {
			w.log(slog.LevelWarn, "failed to write to log file", slog.String("file", w.fileName), slog.Any("err", err))
		}
		return os.Stderr.Write(b)
	}
	return n, err
}

func (w *fileWriter) recordLastErrIfNeeded(err error) bool {
	if w.lastErr == nil {
		w.lastErr = err
		return true
	}
	lastPathErr, ok1 := w.lastErr.(*fs.PathError)
	pathErr, ok2 := err.(*fs.PathError)
	if ok1 && ok2 && *lastPathErr == *pathErr {
		return false
	}
	if !errors.Is(w.lastErr, err) {
		w.lastErr = err
		return true
	}
	return false
}

func (w *fileWriter) openIfNeeded(fileName string) error {
	if w.file != nil {
		return nil
	}
	rotateFileName := fileName
	if rotateFileName == "" {
		rotateFileName = w.rotateFileName()
	}
	file, err := os.OpenFile(rotateFileName, fileWriterOpenFileFlags, fileWriterOpenFileMode)
	if err != nil {
		return err
	}
	w.file = file
	return nil
}

func (w *fileWriter) rotateIfNeeded() error {
	fileInfo, err := w.file.Stat()
	if err != nil {
		return err
	}
	if w.fileSizeLimit <= 0 || fileInfo.Size() < w.fileSizeLimit {
		return nil
	}
	rotateFileName := w.rotateFileName()
	if w.file.Name() == rotateFileName {
		return nil
	}
	_ = w.file.Close()
	w.file = nil
	err = w.openIfNeeded(rotateFileName)
	if err != nil {
		return err
	}
	return nil
}

func (w *fileWriter) rotateFileName() string {
	if w.fileSizeLimit <= 0 {
		return w.fileName
	}
	splitLen := len(w.fileName) - len(filepath.Ext(w.fileName))
	timestamp := time.Now().Format("20060102")
	for i := 1; ; i++ {
		fileName := fmt.Sprintf("%s-%s-%d%s", w.fileName[:splitLen], timestamp, i, w.fileName[splitLen:])
		_, err := os.Stat(fileName)
		if os.IsNotExist(err) {
			return fileName
		}
	}
}

func (w *fileWriter) log(level slog.Level, msg string, args ...any) {
	go func() {
		slog.Log(context.Background(), level, msg, args...)
	}()
}
