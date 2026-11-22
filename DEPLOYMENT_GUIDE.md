# HƯỚNG DẪN TRIỂN KHAI VÀ SỬ DỤNG GITHUB ACTIONS (CI/CD)
> **Dành cho các AI Agent và Developer tiếp quản dự án `xiaozhi-esp32-server-golang`**

---

## 1. Tổng quan kiến trúc triển khai

Dự án này là hệ thống Backend Voice AI cho phần cứng nhúng ESP32 (Xiaozhi/Robot DeskEmoji), bao gồm 2 phần chính:
1. **Frontend (Bảng điều khiển Quản trị)**: Vue 3 + Vite, được Nginx trên Host VPS phục vụ dạng tệp tĩnh từ đường dẫn `/opt/docker/antigravity/xiaozhi-go/frontend/current`.
2. **Backend (Go Core Server + API Manager)**: Ứng dụng Golang chạy dạng Docker container (`backend`) kết nối nội bộ với ASR (Gipformer) và TTS (VieNeu/EdgeTTS), phục vụ giao thức WebSocket `/xiaozhi/v1/` và REST API `:8080`.

### Thông số máy chủ VPS Production:
- **Địa chỉ IP**: `194.233.73.211` (Ubuntu 24.04 x86_64, 6 vCPU, 12GB RAM)
- **Thư mục dự án trên VPS**: `/opt/docker/antigravity/xiaozhi-go/`
- **Domain Quản trị (Admin)**: `https://adminzeroone.hungnguyenaistem.online/`
- **Domain OTA & WebSocket**: `https://otazeroone.hungnguyenaistem.online/`
- **Tài khoản SSH kết nối**: `root` qua SSH Private Key (`~/.ssh/antigravity_key`).

---

## 2. Quy tắc Nhánh (Branch Strategy) kích hoạt CI/CD

Để đảm bảo an toàn tuyệt đối và không ảnh hưởng đến người dùng thiết bị thực tế:
- **Nhánh kích hoạt duy nhất**: **`production`**.
- Mọi nhánh tính năng khác (`main`, `dev`, `feat/vietnamese-console`, `fix/*`): Có thể commit và push tự do để lưu trữ hoặc test local mà **KHÔNG BAO GIỜ** tự động kích hoạt deploy lên VPS.
- **Để deploy**: Sau khi hoàn tất và kiểm tra code trên nhánh tính năng, Agent/Developer tạo Pull Request hoặc Merge nhánh đó vào **`production`**.

---

## 3. Cơ chế Selective Deployment (Thay đổi phần nào deploy phần đó)

Hệ thống CI/CD được tách thành 2 workflow độc lập trong thư mục `.github/workflows/`:

```
.github/workflows/
├── deploy-frontend.yml    # Chỉ kích hoạt khi thay đổi Frontend (Trang quản trị)
└── deploy-backend.yml     # Chỉ kích hoạt khi thay đổi Backend (Go Server)
```

### 3.1. Workflow 1: `deploy-frontend.yml` (Zero-Downtime Deployment)
- **Đường dẫn giám sát (`paths`)**:
  - `manager/frontend/**`
  - `deployment/vps/deploy_console_vi.py`
  - `.github/workflows/deploy-frontend.yml`
- **Cơ chế hoạt động**:
  1. GitHub Runner tải source, cài Node.js 18, chạy `npm ci` và `npm run build` để sinh thư mục `dist`.
  2. Nén thành `xiaozhi-ui-vi.tgz` và dùng SCP đẩy lên thư mục `/tmp/` của VPS.
  3. Kích hoạt script nguyên tử `python3 deploy_console_vi.py` trên VPS:
     - Tạo thư mục release phiên bản mới: `frontend/releases/vi-<timestamp>/`.
     - Giữ lại các chunk assets cũ trong `frontend/public/assets/` để người đang mở tab web không bị lỗi 404.
     - Tráo symlink nguyên tử: `ln -sfn ... frontend/current`.
     - Chạy `nginx -t && systemctl reload nginx`.
  4. **ƯU ĐIỂM TỐI THƯỢNG**:
     - **Backend Go KHÔNG hề khởi động lại**.
     - Các kết nối WebSocket của Robot ESP32 **KHÔNG bị ngắt**, cuộc trò chuyện âm thanh vẫn tiếp tục bình thường 100%.

### 3.2. Workflow 2: `deploy-backend.yml` (Quick Hot-Swap & Self-Healing)
- **Đường dẫn giám sát (`paths`)**:
  - `cmd/**`
  - `internal/**`
  - `manager/backend/**`
  - `pkg/**`, `logger/**`, `constants/**`, `config/**`
  - `deployment/vps/compose.yaml`, `deployment/vps/Dockerfile`
  - `docker/**`, `go.mod`, `go.sum`
  - `.github/workflows/deploy-backend.yml`
- **Cơ chế hoạt động**:
  1. GitHub Runner build mã nguồn Go (sử dụng Docker multi-stage `docker/Dockerfile.linux` với đầy đủ CGO: Opus, Ten-VAD, Sherpa-ONNX).
  2. Trích xuất file nhị phân `xiaozhi_server` (~12MB) và SCP lên `/tmp/` trên VPS.
  3. Trên VPS:
     - Tự động sao lưu binary hiện tại thành `bundle/xiaozhi_server.bak`.
     - Thay thế `bundle/xiaozhi_server` mới và phân quyền thực thi `755`.
     - Rebuild nhẹ container backend (`docker compose build backend`) và restart riêng backend (`docker compose up -d --no-deps backend`). Các container ASR và Gateway giữ nguyên.
     - **Vòng lặp tự động kiểm tra sức khỏe (Health Check)**: Gọi `/api/setup/status` trong 12 lần (tối đa 36s).
     - **Cơ chế tự phục hồi (Self-Healing Rollback)**: Nếu backend mới bị crash lúc khởi động, script trên VPS sẽ tự động khôi phục lại `xiaozhi_server.bak`, build lại và đưa hệ thống về trạng thái ổn định ngay lập tức.

---

## 4. Quản lý Secrets trên GitHub

Toàn bộ thông tin nhạy cảm đã được cấu hình trong `Repository Settings -> Secrets and variables -> Actions`:

| Secret Name | Mô tả | Giá trị sử dụng |
| :--- | :--- | :--- |
| `VPS_HOST` | Địa chỉ IP của máy chủ VPS | `194.233.73.211` |
| `VPS_USERNAME` | Tài khoản SSH trên VPS | `root` |
| `VPS_SSH_KEY` | SSH Private Key xác thực | Khóa đối ứng với `antigravity-ai-agent` trên VPS |

> ⚠️ **Quy tắc an toàn**: Không bao giờ hardcode IP, Password hay SSH Key vào mã nguồn hoặc file workflow. Luôn gọi qua cú pháp `${{ secrets.TÊN_SECRET }}`.

---

## 5. Hướng dẫn dành cho Agent khi thực hiện công việc

Khi một Agent nhận task phát triển hoặc bảo trì dự án này:

1. **Khi chỉnh sửa Giao diện Quản trị / Dịch thuật**:
   - Làm việc bên trong thư mục `manager/frontend/`.
   - Trước khi commit, **BẮT BUỘC** test build thử trên máy:
     ```bash
     cd projects/xiaozhi-server-go/manager/frontend
     npm run build
     ```
   - Commit và push lên nhánh làm việc.
   - Khi sẵn sàng phát hành, merge sang nhánh `production`.

2. **Khi chỉnh sửa Backend Go / Thêm tính năng AI / Fix Bug**:
   - Làm việc bên trong `internal/`, `cmd/`, `manager/backend/`.
   - Kiểm tra định dạng code và dependency:
     ```bash
     cd projects/xiaozhi-server-go
     go vet ./cmd/... ./internal/...
     ```
   - Commit và push lên nhánh làm việc.
   - Khi sẵn sàng phát hành, merge sang nhánh `production`.

3. **Kích hoạt Deploy thủ công (Manual Run)**:
   - Truy cập giao diện GitHub: `Actions` $\rightarrow$ Chọn workflow cần chạy (`Deploy Frontend` hoặc `Deploy Backend`) $\rightarrow$ Bấm **Run workflow** $\rightarrow$ Chọn nhánh `production`.

4. **Kiểm tra sau Deploy**:
   - Kiểm tra Web Admin: `curl -I https://adminzeroone.hungnguyenaistem.online/`
   - Kiểm tra OTA: `curl -I https://otazeroone.hungnguyenaistem.online/xiaozhi/ota/`
   - Kiểm tra container trên VPS:
     ```bash
     ssh -i ~/.ssh/antigravity_key root@194.233.73.211 "docker compose -f /opt/docker/antigravity/xiaozhi-go/compose.yaml ps"
     ```
