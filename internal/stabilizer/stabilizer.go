package stabilizer

import (
	"os"
	"time"
)

// WaitUntilStable kiểm tra file có ổn định không.
// Rule: size file không đổi trong stableFor duration.
func WaitUntilStable(filePath string, checkInterval time.Duration, stableFor time.Duration) error {
	var stableSince time.Time
	var lastSize int64 = -1

	for {
		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}

		currentSize := info.Size()

		if currentSize != lastSize {
			lastSize = currentSize
			stableSince = time.Now()
		} else {
			if time.Since(stableSince) >= stableFor {
				return nil
			}
		}

		time.Sleep(checkInterval)
	}
}
