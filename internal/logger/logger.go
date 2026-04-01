package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

func New(logDir string) (*log.Logger, *os.File, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, nil, err
	}

	logPath := filepath.Join(logDir, "agent.log")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	mw := io.MultiWriter(os.Stdout, f)
	l := log.New(mw, "", log.Ldate|log.Ltime|log.Lshortfile)

	return l, f, nil
}
