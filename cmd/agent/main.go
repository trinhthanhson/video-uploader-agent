package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/kardianos/service"

	"video-uploader-agent/internal/backend"
	"video-uploader-agent/internal/config"
	"video-uploader-agent/internal/fileops"
	"video-uploader-agent/internal/logger"
	"video-uploader-agent/internal/scanner"
	"video-uploader-agent/internal/stabilizer"
	"video-uploader-agent/internal/uploader"
	"video-uploader-agent/internal/watcher"
)

const (
	metricsFile = "upload-metrics.jsonl"
	serviceName = "VideoUploaderAgent"
)

type UploadMetric struct {
	OrderID            string    `json:"order_id"`
	FileName           string    `json:"file_name"`
	FileSizeBytes      int64     `json:"file_size_bytes"`
	DetectedAt         time.Time `json:"detected_at"`
	UploadStartedAt    time.Time `json:"upload_started_at"`
	UploadCompletedAt  time.Time `json:"upload_completed_at"`
	UploadDurationSec  float64   `json:"upload_duration_seconds"`
	TotalProcessingSec float64   `json:"total_processing_seconds"`
	UploadSpeedMBps    float64   `json:"upload_speed_mbps"`
	Status             string    `json:"status"`
	ErrorMessage       string    `json:"error_message,omitempty"`
}

type program struct {
	stopCh chan struct{}
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: "Video Uploader Agent",
		Description: "Background service to watch folders and upload videos to R2.",
	}

	prg := &program{
		stopCh: make(chan struct{}),
	}

	s, err := service.New(prg, svcConfig)
	if err != nil {
		panic(err)
	}

	// Support manual commands if needed:
	// agent.exe install
	// agent.exe start
	// agent.exe stop
	// agent.exe uninstall
	if len(os.Args) > 1 {
		cmd := os.Args[1]
		switch cmd {
		case "install", "start", "stop", "restart", "uninstall":
			if err := service.Control(s, cmd); err != nil {
				panic(err)
			}
			return
		}
	}

	// Nếu đang chạy trong service context thì chạy service luôn.
	if service.Interactive() {
		// Người dùng double click / chạy trực tiếp.
		// Tự cài và tự start service, không cần PowerShell.
		autoInstallAndStart(s)
		return
	}

	// Chạy trong Windows Service context.
	if err := s.Run(); err != nil {
		panic(err)
	}
}

func autoInstallAndStart(s service.Service) {
	// thử install trước; nếu đã tồn tại thì bỏ qua
	_ = service.Control(s, "install")
	_ = service.Control(s, "start")
}

func (p *program) run() {
	cfgPath := resolveConfigPath()

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		panic(err)
	}

	logDir := cfg.LogDir
	if logDir == "" {
		logDir = filepath.Join(exeDir(), "logs")
	}

	appLogger, logFile, err := logger.New(logDir)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	appLogger.Println("agent starting...")
	appLogger.Println("using config:", cfgPath)
	appLogger.Println("watch dir:", cfg.WatchDir)

	u := uploader.NewR2Uploader(cfg)
	bc := backend.NewClient(cfg)

	w, err := watcher.New()
	if err != nil {
		appLogger.Println("watcher init error:", err)
	} else {
		defer w.Close()

		if err := w.AddRecursive(cfg.WatchDir); err != nil {
			appLogger.Println("watch add recursive error:", err)
		} else {
			appLogger.Println("watcher attached to:", cfg.WatchDir)
		}

		go func() {
			for {
				select {
				case event, ok := <-w.Events():
					if !ok {
						return
					}
					appLogger.Println("watch event:", event.Op.String(), event.Name)

				case err, ok := <-w.Errors():
					if !ok {
						return
					}
					appLogger.Println("watch error:", err)

				case <-p.stopCh:
					return
				}
			}
		}()
	}

	ticker := time.NewTicker(time.Duration(cfg.ScanIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		processOnce(cfg, u, bc, appLogger)

		select {
		case <-ticker.C:
		case <-p.stopCh:
			appLogger.Println("agent stopping...")
			return
		}
	}
}

func processOnce(
	cfg *config.Config,
	u *uploader.R2Uploader,
	bc *backend.Client,
	appLogger interface{ Println(v ...any) },
) {
	jobs, err := scanner.Scan(cfg.WatchDir, cfg.AllowedExtensions)
	if err != nil {
		appLogger.Println("scan error:", err)
		return
	}

	if len(jobs) > 0 {
		appLogger.Println("scan found files:", len(jobs))
	}

	for _, job := range jobs {
		detectedAt := time.Now()

		appLogger.Println("found file:", job.FilePath)

		err := stabilizer.WaitUntilStable(
			job.FilePath,
			5*time.Second,
			time.Duration(cfg.StabilizeSeconds)*time.Second,
		)
		if err != nil {
			appLogger.Println("stabilize error:", err)
			continue
		}

		info, statErr := os.Stat(job.FilePath)
		if statErr != nil {
			appLogger.Println("stat file error:", statErr)
			continue
		}

		fileSizeBytes := info.Size()

		appLogger.Println("uploading:", job.FilePath)
		uploadStartedAt := time.Now()

		objectKey, err := u.Upload(job.FilePath, job.OrderID)
		uploadCompletedAt := time.Now()

		if err != nil {
			appLogger.Println("upload failed:", err)

			_, moveErr := fileops.MoveToDir(job.FilePath, cfg.FailedDir, job.OrderID)
			if moveErr != nil {
				appLogger.Println("move failed error:", moveErr)
			}

			notifyErr := bc.NotifyUploadFailed(backend.UploadFailedRequest{
				OrderID:      job.OrderID,
				FileName:     job.FileName,
				Status:       "failed",
				ErrorMessage: err.Error(),
			})
			if notifyErr != nil {
				appLogger.Println("notify failed status error:", notifyErr)
			}

			metric := UploadMetric{
				OrderID:            job.OrderID,
				FileName:           job.FileName,
				FileSizeBytes:      fileSizeBytes,
				DetectedAt:         detectedAt,
				UploadStartedAt:    uploadStartedAt,
				UploadCompletedAt:  uploadCompletedAt,
				UploadDurationSec:  uploadCompletedAt.Sub(uploadStartedAt).Seconds(),
				TotalProcessingSec: uploadCompletedAt.Sub(detectedAt).Seconds(),
				UploadSpeedMBps:    calculateSpeedMBps(fileSizeBytes, uploadStartedAt, uploadCompletedAt),
				Status:             "failed",
				ErrorMessage:       err.Error(),
			}
			if writeErr := appendMetric(resolveMetricsPath(), metric); writeErr != nil {
				appLogger.Println("write metric failed:", writeErr)
			}

			continue
		}

		appLogger.Println("upload success:", objectKey)

		movedPath, err := fileops.MoveToDir(job.FilePath, cfg.UploadedDir, job.OrderID)
		if err != nil {
			appLogger.Println("move uploaded error:", err)
			continue
		}

		appLogger.Println("moved to:", movedPath)

		notifyErr := bc.NotifyUploadSuccess(backend.UploadSuccessRequest{
			OrderID:   job.OrderID,
			FileName:  job.FileName,
			ObjectKey: objectKey,
			FileSize:  fileSizeBytes,
			Status:    "uploaded",
		})
		if notifyErr != nil {
			appLogger.Println("notify upload success error:", notifyErr)
		}

		metric := UploadMetric{
			OrderID:            job.OrderID,
			FileName:           job.FileName,
			FileSizeBytes:      fileSizeBytes,
			DetectedAt:         detectedAt,
			UploadStartedAt:    uploadStartedAt,
			UploadCompletedAt:  uploadCompletedAt,
			UploadDurationSec:  uploadCompletedAt.Sub(uploadStartedAt).Seconds(),
			TotalProcessingSec: uploadCompletedAt.Sub(detectedAt).Seconds(),
			UploadSpeedMBps:    calculateSpeedMBps(fileSizeBytes, uploadStartedAt, uploadCompletedAt),
			Status:             "uploaded",
		}
		if writeErr := appendMetric(resolveMetricsPath(), metric); writeErr != nil {
			appLogger.Println("write metric failed:", writeErr)
		}

		appLogger.Println("metric written for:", filepath.Base(movedPath))
	}
}

func calculateSpeedMBps(fileSizeBytes int64, startedAt, completedAt time.Time) float64 {
	durationSec := completedAt.Sub(startedAt).Seconds()
	if durationSec <= 0 {
		return 0
	}

	fileSizeMB := float64(fileSizeBytes) / 1024.0 / 1024.0
	return fileSizeMB / durationSec
}

func appendMetric(path string, metric UploadMetric) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	encoded, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	if _, err := f.Write(append(encoded, '\n')); err != nil {
		return err
	}

	return nil
}

func resolveConfigPath() string {
	return filepath.Join(exeDir(), "config.yaml")
}

func resolveMetricsPath() string {
	return filepath.Join(exeDir(), metricsFile)
}

func exeDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exePath)
}
