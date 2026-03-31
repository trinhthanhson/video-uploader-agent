# Video Uploader Agent - Development Roadmap

Dự án này là một Agent chạy nền để tự động phát hiện, kiểm tra tính ổn định, upload video lên Cloudflare R2 và thông báo kết quả cho Backend.

---

## 1. Cấu trúc thư mục (Project Structure)

```text
video-uploader-agent/
 ├── cmd/
 │    └── agent/
 │         └── main.go          # Điểm khởi chạy chương trình
 ├── internal/
 │    ├── config/
 │    │    └── config.go        # Đọc và validate cấu hình YAML
 │    ├── watcher/
 │    │    └── watcher.go       # Lắng nghe sự kiện file mới (FSNotify)
 │    ├── scanner/
 │    │    └── scanner.go       # Quét file định kỳ trong folder
 │    ├── stabilizer/
 │    │    └── stabilizer.go    # Kiểm tra file đã copy xong chưa
 │    ├── uploader/
 │    │    └── r2.go            # Logic upload lên Cloudflare R2
 │    ├── backend/
 │    │    └── client.go        # Gọi API callback về Backend chính
 │    ├── fileops/
 │    │    └── fileops.go       # Di chuyển file sau khi xử lý (Success/Fail)
 │    └── logger/
 │         └── logger.go        # Quản lý Log ứng dụng
 ├── configs/
 │    └── config.yaml           # File cấu hình môi trường
 └── go.mod                     # Quản lý dependencies