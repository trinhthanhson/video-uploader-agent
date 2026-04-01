package fileops

import (
	"fmt"
	"os"
	"path/filepath"
)

func MoveToDir(srcPath, baseTargetDir, orderID string) (string, error) {
	fileName := filepath.Base(srcPath)
	targetDir := filepath.Join(baseTargetDir, orderID)

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("create target dir: %w", err)
	}

	dstPath := filepath.Join(targetDir, fileName)

	if err := os.Rename(srcPath, dstPath); err != nil {
		return "", fmt.Errorf("move file: %w", err)
	}

	return dstPath, nil
}
