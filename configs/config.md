# config.md

## Mục đích

File `config.yaml` dùng để cấu hình service upload video:

* theo dõi folder
* upload video lên R2
* gửi trạng thái về backend
* quản lý file sau khi xử lý

---

## Cấu hình chính

### 1. Thư mục

* `watch_dir`: nơi nhân viên copy video vào
  👉 `D:/video-drop/<ORDER_ID>/file.mp4`

* `uploaded_dir`: chứa file upload thành công

* `failed_dir`: chứa file upload lỗi

---

### 2. Quy tắc file

* `allowed_extensions`: định dạng file được xử lý (vd: `.mp4`, `.mov`)

---

### 3. Thời gian xử lý

* `stabilize_seconds`: thời gian chờ file copy xong
  👉 file không đổi size trong N giây mới upload

* `scan_interval_seconds`: thời gian quét folder

* `max_retry`: số lần retry upload nếu lỗi

---

### 4. Cloudflare R2

* `access_key_id`, `secret_access_key`: key upload
* `bucket`: nơi lưu video
* `endpoint`: URL R2

---

### 5. Backend

* `base_url`: API server
* `api_key`: key xác thực (nếu có)
* `timeout_seconds`: timeout gọi API

---

## Luồng hoạt động

1. Copy file vào:
   `watch_dir/<ORDER_ID>/`

2. Service:

   * detect file
   * chờ ổn định
   * upload lên R2

3. Sau đó:

   * gọi backend
   * move file sang:

     * `uploaded_dir` (thành công)
     * `failed_dir` (lỗi)

---

## Lưu ý

* Folder phải đúng format: `<ORDER_ID>`
* Không đặt file trực tiếp trong `watch_dir`
* Không commit key thật vào repo

---
