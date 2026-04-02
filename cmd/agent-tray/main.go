package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/getlantern/systray"
	"github.com/kardianos/service"
	"github.com/sqweek/dialog"
	"gopkg.in/yaml.v3"

	"video-uploader-agent/internal/config"
)

const (
	serviceName    = "VideoUploaderAgent"
	serviceExeName = "agent.exe"
	configFileName = "config.yaml"
)

var (
	mStatus         *systray.MenuItem
	mStart          *systray.MenuItem
	mStop           *systray.MenuItem
	mRestart        *systray.MenuItem
	mInstall        *systray.MenuItem
	mUninstall      *systray.MenuItem
	mOpenConfig     *systray.MenuItem
	mOpenLogs       *systray.MenuItem
	mOpenUploaded   *systray.MenuItem
	mOpenFailed     *systray.MenuItem
	mOpenWatch      *systray.MenuItem
	mChooseWatch    *systray.MenuItem
	mChooseUploaded *systray.MenuItem
	mChooseFailed   *systray.MenuItem
	mOpenApp        *systray.MenuItem
	mQuit           *systray.MenuItem
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	setTrayIconStopped()
	systray.SetTitle("Video Agent")
	systray.SetTooltip("Video Upload Agent")

	mStatus = systray.AddMenuItem("Status: Checking...", "Current service status")
	mStatus.Disable()

	systray.AddSeparator()

	mStart = systray.AddMenuItem("Start Service", "Start background service")
	mStop = systray.AddMenuItem("Stop Service", "Stop background service")
	mRestart = systray.AddMenuItem("Restart Service", "Restart background service")

	systray.AddSeparator()

	mInstall = systray.AddMenuItem("Install Service", "Install Windows service")
	mUninstall = systray.AddMenuItem("Uninstall Service", "Uninstall Windows service")

	systray.AddSeparator()

	mChooseWatch = systray.AddMenuItem("Choose Watch Folder", "Select watch folder")
	mChooseUploaded = systray.AddMenuItem("Choose Uploaded Folder", "Select uploaded folder")
	mChooseFailed = systray.AddMenuItem("Choose Failed Folder", "Select failed folder")

	systray.AddSeparator()

	mOpenWatch = systray.AddMenuItem("Open Watch Folder", "Open watch folder")
	mOpenUploaded = systray.AddMenuItem("Open Uploaded Folder", "Open uploaded folder")
	mOpenFailed = systray.AddMenuItem("Open Failed Folder", "Open failed folder")
	mOpenLogs = systray.AddMenuItem("Open Logs Folder", "Open logs folder")
	mOpenConfig = systray.AddMenuItem("Open Config", "Open config.yaml")
	mOpenApp = systray.AddMenuItem("Open App Folder", "Open app folder")

	systray.AddSeparator()

	mQuit = systray.AddMenuItem("Exit Tray", "Exit tray application")

	go handleMenu()
	go autoRefreshStatus()
}

func onExit() {}

func handleMenu() {
	for {
		select {
		case <-mInstall.ClickedCh:
			if err := runServiceCommand("install"); err != nil {
				updateStatus("Status: Install failed")
			} else {
				updateStatus("Status: Installed")
			}

		case <-mUninstall.ClickedCh:
			if err := runServiceCommand("uninstall"); err != nil {
				updateStatus("Status: Uninstall failed")
			} else {
				updateStatus("Status: Uninstalled")
			}

		case <-mStart.ClickedCh:
			if err := runServiceCommand("start"); err != nil {
				updateStatus("Status: Start failed")
			} else {
				updateStatus("Status: Running")
			}

		case <-mStop.ClickedCh:
			if err := runServiceCommand("stop"); err != nil {
				updateStatus("Status: Stop failed")
			} else {
				updateStatus("Status: Stopped")
			}

		case <-mRestart.ClickedCh:
			if err := runServiceCommand("restart"); err != nil {
				updateStatus("Status: Restart failed")
			} else {
				updateStatus("Status: Running")
			}

		case <-mChooseWatch.ClickedCh:
			if err := chooseAndSaveFolder("watch"); err != nil {
				updateStatus("Status: Watch folder save failed")
			} else {
				updateStatus("Status: Watch folder updated")
			}

		case <-mChooseUploaded.ClickedCh:
			if err := chooseAndSaveFolder("uploaded"); err != nil {
				updateStatus("Status: Uploaded folder save failed")
			} else {
				updateStatus("Status: Uploaded folder updated")
			}

		case <-mChooseFailed.ClickedCh:
			if err := chooseAndSaveFolder("failed"); err != nil {
				updateStatus("Status: Failed folder save failed")
			} else {
				updateStatus("Status: Failed folder updated")
			}

		case <-mOpenWatch.ClickedCh:
			_ = openFolder(watchDir())

		case <-mOpenUploaded.ClickedCh:
			_ = openFolder(uploadedDir())

		case <-mOpenFailed.ClickedCh:
			_ = openFolder(failedDir())

		case <-mOpenLogs.ClickedCh:
			_ = openFolder(logsDir())

		case <-mOpenConfig.ClickedCh:
			_ = openFile(configPath())

		case <-mOpenApp.ClickedCh:
			_ = openFolder(appDir())

		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func chooseAndSaveFolder(target string) error {
	cfg, err := loadRawConfig()
	if err != nil {
		return err
	}

	current := ""
	switch target {
	case "watch":
		current = cfg.WatchDir
	case "uploaded":
		current = cfg.UploadedDir
	case "failed":
		current = cfg.FailedDir
	default:
		return fmt.Errorf("unknown target: %s", target)
	}
	fmt.Println("Thư mục hiện tại là:", current)
	selected, err := dialog.Directory().Title("Choose folder").Browse()
	if err != nil {
		return err
	}

	selected = filepath.Clean(selected)

	switch target {
	case "watch":
		cfg.WatchDir = selected
	case "uploaded":
		cfg.UploadedDir = selected
	case "failed":
		cfg.FailedDir = selected
	}

	if err := os.MkdirAll(selected, 0755); err != nil {
		return err
	}

	return saveRawConfig(cfg)
}

func autoRefreshStatus() {
	for {
		refreshStatus()
		time.Sleep(5 * time.Second)
	}
}

func refreshStatus() {
	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: "Video Uploader Agent",
		Description: "Background service to watch folders and upload videos to R2.",
	}

	prg := &dummyProgram{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		updateStatus("Status: Service error")
		return
	}

	st, err := s.Status()
	if err != nil {
		updateStatus("Status: Not installed")
		setTrayIconStopped()
		enableForStopped()
		return
	}

	switch st {
	case service.StatusRunning:
		updateStatus("Status: Running")
		setTrayIconRunning()
		enableForRunning()
	case service.StatusStopped:
		updateStatus("Status: Stopped")
		setTrayIconStopped()
		enableForStopped()
	default:
		updateStatus(fmt.Sprintf("Status: %v", st))
		setTrayIconStopped()
	}
}

func updateStatus(text string) {
	mStatus.SetTitle(text)
}

func enableForRunning() {
	mStart.Disable()
	mStop.Enable()
	mRestart.Enable()
	mInstall.Disable()
	mUninstall.Enable()
}

func enableForStopped() {
	mStart.Enable()
	mStop.Disable()
	mRestart.Disable()
	mInstall.Enable()
	mUninstall.Enable()
}

func runServiceCommand(cmd string) error {
	exe := serviceExePath()
	c := exec.Command(exe, cmd)
	c.Dir = appDir()
	output, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w - %s", cmd, err, string(output))
	}
	return nil
}

type dummyProgram struct{}

func (p *dummyProgram) Start(s service.Service) error { return nil }
func (p *dummyProgram) Stop(s service.Service) error  { return nil }

func loadRawConfig() (*config.Config, error) {
	return config.LoadConfig(configPath())
}

func saveRawConfig(cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

func serviceExePath() string {
	return filepath.Join(appDir(), serviceExeName)
}

func configPath() string {
	return filepath.Join(appDir(), configFileName)
}

func logsDir() string {
	cfg, err := loadRawConfig()
	if err == nil && cfg.LogDir != "" {
		return cfg.LogDir
	}
	return filepath.Join(appDir(), "logs")
}

func watchDir() string {
	cfg, err := loadRawConfig()
	if err == nil && cfg.WatchDir != "" {
		return cfg.WatchDir
	}
	return filepath.Join(appDir(), "video-drop")
}

func uploadedDir() string {
	cfg, err := loadRawConfig()
	if err == nil && cfg.UploadedDir != "" {
		return cfg.UploadedDir
	}
	return filepath.Join(appDir(), "video-uploaded")
}

func failedDir() string {
	cfg, err := loadRawConfig()
	if err == nil && cfg.FailedDir != "" {
		return cfg.FailedDir
	}
	return filepath.Join(appDir(), "video-failed")
}

func appDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func openFile(path string) error {
	return exec.Command("cmd", "/C", "start", "", path).Start()
}

func openFolder(path string) error {
	_ = os.MkdirAll(path, 0755)
	return exec.Command("explorer", path).Start()
}

func setTrayIconRunning() {
	// 1. Thử đọc file bên ngoài trước
	iconData, err := os.ReadFile(filepath.Join(appDir(), "icon.ico"))
	if err == nil {
		systray.SetIcon(iconData)
	} else {
		// 2. Nếu không có file, dùng icon mặc định đã khai báo ở cuối code
		systray.SetIcon(iconRunning)
	}
	systray.SetTooltip("Video Upload Agent - Running")
}

func setTrayIconStopped() {
	systray.SetIcon(iconStopped)
	systray.SetTooltip("Video Upload Agent - Stopped")
}

var iconRunning = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0xf3, 0xff,
	0x61, 0x00, 0x00, 0x00, 0x1d, 0x49, 0x44, 0x41,
	0x54, 0x38, 0x8d, 0x63, 0xfc, 0xcf, 0x40, 0x1a,
	0x60, 0x22, 0x51, 0xfd, 0xa8, 0x86, 0x51, 0x0d,
	0x43, 0x35, 0x0c, 0xd5, 0x30, 0x54, 0xc3, 0x50,
	0x0d, 0x03, 0x00, 0x48, 0x0c, 0x02, 0x1e, 0x58,
	0x58, 0x5a, 0x73, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

var iconStopped = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0xf3, 0xff,
	0x61, 0x00, 0x00, 0x00, 0x1d, 0x49, 0x44, 0x41,
	0x54, 0x38, 0x8d, 0x63, 0xfc, 0xcf, 0xc0, 0xc0,
	0xc0, 0x00, 0x46, 0x35, 0x0c, 0xd5, 0x30, 0x54,
	0xc3, 0x50, 0x0d, 0x43, 0x35, 0x0c, 0xd5, 0x30,
	0x00, 0x00, 0x31, 0x4f, 0x01, 0x1b, 0xdf, 0xf7,
	0xf6, 0x54, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45,
	0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}
