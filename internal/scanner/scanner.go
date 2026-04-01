package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

type FileJob struct {
	OrderID  string
	FilePath string
	FileName string
}

// Scan sẽ quét toàn bộ watch_dir
func Scan(watchDir string, allowedExts []string) ([]FileJob, error) {
	var jobs []FileJob

	// đọc tất cả folder con (ORDER_ID)
	orderDirs, err := os.ReadDir(watchDir)
	if err != nil {
		return nil, err
	}

	for _, dir := range orderDirs {
		if !dir.IsDir() {
			continue
		}

		orderID := dir.Name()
		orderPath := filepath.Join(watchDir, orderID)

		files, err := os.ReadDir(orderPath)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			fileName := file.Name()
			ext := strings.ToLower(filepath.Ext(fileName))

			if !isAllowed(ext, allowedExts) {
				continue
			}

			fullPath := filepath.Join(orderPath, fileName)

			job := FileJob{
				OrderID:  orderID,
				FilePath: fullPath,
				FileName: fileName,
			}

			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

func isAllowed(ext string, allowed []string) bool {
	for _, a := range allowed {
		if ext == strings.ToLower(a) {
			return true
		}
	}
	return false
}
