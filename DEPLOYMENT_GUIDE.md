## Sửa lịch sử tool và lỗi đọc ra loa

Xem [kế hoạch, kiểm thử và bàn giao](deployment/vps/TOOL_HISTORY_ERROR_FIX.md). Bản sửa không cần migration hoặc thay cấu hình; khi deploy dùng mọi audio input=unchanged để giữ runtime lọc nhiễu đang thử. Trạng thái phát hành ghi tại tài liệu này; không xóa history/recordings để xử lý tool-call400.

## Lọc nhiễu backend (thử robot từ 2026-11-24)

Xem [hướng dẫn, số đo và phát hành](deployment/vps/BACKEND_NOISE_SUPPRESSION.md). PR17/Actions45316196051 đã bật runtime audio_preprocess=enforce theo yêu cầu người dùng. Input Actions audio_preprocess hỗ trợ unchanged/off/shadow/enforce; chỉ từ production, inputs audio khác để unchanged nếu không cần đổi. Workflow đóng gói native libs/license và chạy native self-check khi bật; lỗi health/native sẽ rollback binary/config. Code mặc định off khi không có cấu hình, nhưng runtime hiện đang enforce để nghiệm thu mic thật.

## Tích hợp VAD/ASR PR #271

Xem [kế hoạch kiểm tra và phát hành](deployment/vps/PR271_INTEGRATION.md). Đã triển khai ngày 2026-11-23 qua GitHub Actions; xem kết quả trong AUDIO_DIAGNOSTICS.md. YAML runtime cũ cần migration chọn lọc để bật noise filter, không chép đè secrets hoặc đổi Gipformer.

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


## 6. PhoWhisper ASR — quy trình thử và chuyển đổi

Nhánh phát triển: `feat/phowhisper-asr`. Kế hoạch/bàn giao chi tiết: `deployment/vps/PHOWHISPER_PLAN.md`.
Workflow mới: `.github/workflows/deploy-asr.yml`, tên **PhoWhisper ASR (CI and explicit deployment)**.

### 6.1. Kiểm tra mã trước phát hành

Push nhánh tính năng hoặc mở PR chạy contract tests, kiểm tra Compose, chuyển model VinAI revision cố định sang CTranslate2 INT8 và warm-up image Linux. CI tạo artifact `phowhisper-release` lưu 3 ngày. Các sự kiện push/PR không deploy VPS, kể cả push production: deploy ASR luôn là thao tác chọn rõ trong Actions.

Kiểm tra local bằng Python 3.12 trong môi trường riêng:

```bash
python3.12 -m venv /tmp/phowhisper-tests
/tmp/phowhisper-tests/bin/pip install aiohttp==3.13.2 numpy==2.2.6
/tmp/phowhisper-tests/bin/python -m unittest discover -s deployment/vps -p 'test_*phowhisper*.py' -v
```

Build và chuyển model trong Docker Linux (cần Docker daemon):

```bash
mkdir -p phowhisper-build/model
docker build --target converter -f deployment/vps/Dockerfile.phowhisper -t phowhisper-converter deployment/vps
docker run --rm -v "$PWD/phowhisper-build/model:/output" phowhisper-converter /output/converted
docker build --target runtime -f deployment/vps/Dockerfile.phowhisper -t xiaozhi-phowhisper:local deployment/vps
```

Không commit model weights, cache hoặc audio cá nhân. Manifest nguồn chứa model ID/revision và phiên bản trực tiếp; manifest artifact có SHA-256 từng file. Release lưu Docker image ID và SHA-256 các gói để kiểm tra trước khi chạy.

### 6.2. Deploy thử qua Actions

1. Sau khi kiểm tra mã và merge theo phạm vi phát hành được cho phép, vào Actions, chọn workflow PhoWhisper và **Run workflow**, nhánh **production**, action **trial**.
2. Pipeline build/chuyển model ở CI, chuyển artifact đến VPS và dựng riêng `asr-phowhisper`; không đổi compose gốc, endpoint Gipformer hay restart Go/TTS.
3. Service không publish cổng host. Go vẫn dùng ASR cũ. Workflow kiểm tra health, nhận dạng speech, silence và TLS/OTA/WSS bằng thiết bị smoke test hiện có.
4. Release đặt tại `runtime/phowhisper-releases/<sha>-<run>-<attempt>`; con trỏ ở `runtime/phowhisper-current.json`. Chỉ thay con trỏ sau kiểm tra thành công. Nếu thử thất bại, dựng lại release sidecar trước đó hoặc dừng sidecar mới nếu chưa có bản trước.
5. Trial bị từ chối nếu PhoWhisper đang là ASR mặc định, để không vô tình thay model đang phục vụ robot. Khôi phục ASR cũ trước khi thử release tiếp.

GitHub yêu cầu workflow có trên default branch để hiện nút chạy thủ công; nếu chưa thấy workflow, bổ sung định nghĩa workflow vào default branch theo quy trình repository, rồi chọn ref production khi dispatch. Không đổi sang deploy từ nhánh tính năng để bỏ qua quy tắc này.

### 6.3. Chuẩn bị và chạy benchmark

Trên VPS, bộ mẫu riêng đặt tại `runtime/asr-benchmark/`, gồm WAV mono PCM16 16 kHz và `suite.json`. Dùng audio được phép sử dụng; chuyển/tạo bộ mẫu qua quy trình dữ liệu riêng, không gửi recordings vào Actions artifact hoặc Git. Ví dụ schema (chưa phải bộ mẫu nghiệm thu):

```json
[
  {"id":"speech-001", "wav":"speech-001.wav", "reference":"xin chào việt nam"},
  {"id":"noise-001", "wav":"noise-001.wav", "reference":""}
]
```

Cần ít nhất 30 mẫu speech và 20 mẫu silence/noise. ID mẫu không chứa thông tin cá nhân. Chọn action **benchmark** từ production: chạy cùng bộ mẫu trên endpoint ASR cũ và PhoWhisper với concurrency 1/2/4, không đổi mặc định. Báo cáo lưu riêng trên VPS trong `runtime/asr-benchmark/reports/` và log Actions chỉ có số đo tổng hợp, không có bản chép. Silence/noise sinh chữ làm workflow báo lỗi sau khi lưu các báo cáo có thể thu được. Timeout/kết nối lỗi có thể khiến lượt đo không hoàn tất, phải xử lý trước nghiệm thu.

WER/CER sử dụng chuẩn hóa NFC, chữ thường, bỏ dấu câu, giữ dấu tiếng Việt; CER tính cả khoảng trắng sau chuẩn hóa. Báo cáo phân biệt queue, inference và thời gian ASR tổng khi server có metric; Gipformer hiện chỉ đo được thời gian tổng ở client. Benchmark không tự tạo tải VieNeu: cần thử thêm khi TTS đang hoạt động và xác nhận nghe không hụt tiếng. Không suy ra số robot phục vụ đồng thời từ số kết nối online.

### 6.4. Chuyển mặc định và quay lui

- **activate**: cần nhập `benchmark_run` là URL HTTPS trỏ tới bằng chứng benchmark đã rà soát. URL là dấu vết bàn giao, không tự chứng minh các chỉ số đạt. Chỉ chạy khi chất lượng/độ trễ/TTS đạt tiêu chí trong kế hoạch và đã có phạm vi phát hành.
- Script tạo cấu hình qua endpoint typed `/api/admin/asr-configs` (tránh lỗi tự đặt mặc định của endpoint generic), lưu ID cấu hình cũ trong `runtime/phowhisper-asr-state.json` quyền 0600, rồi chuyển mặc định qua API. Không sửa SQLite, không chép lại toàn bộ runtime/config.
- Manager hiện chọn ASR toàn cục: model Agent có `ASRSpeed` nhưng không có `ASRConfigID`. Vì vậy activate ảnh hưởng các phiên nhận cấu hình mới; chưa có canary riêng một robot. Các phiên giữ cấu hình cũ có thể cần kết nối lại. Chưa nghe robot thật thì chưa coi là nghiệm thu.
- **rollback**: chọn action rollback từ production. Trả mặc định về ID đã lưu, giữ các nội dung cấu hình được chỉnh về sau, xác minh ASR cũ và TLS/OTA/WSS. Từ chối ghi đè nếu quản trị viên đã chọn một mặc định khác ngoài rollout. Sidecar giữ lại để chẩn đoán; chưa tự xóa image/model/release.
- Health/smoke thất bại sau activate thì script tự trả cấu hình mặc định cũ. Rollback của sidecar không thay thế rollback cấu hình; cả hai cơ chế đều không restore toàn bộ SQLite.
- Workflow ASR và backend dùng chung concurrency group `xiaozhi-production-services`; không thay Compose đồng thời.

Workflow sử dụng các SSH secrets sẵn có; tài khoản Manager lấy từ file riêng `/root/.config/xiaozhi-go/admin.json` trên VPS, không cần đưa mật khẩu/token vào Actions.

## Log chẩn đoán âm thanh (đã deploy, đang bật tạm thời)

Xem [AUDIO_DIAGNOSTICS.md](deployment/vps/AUDIO_DIAGNOSTICS.md) để bật/tắt metadata RMS/VAD/ASR và thực hiện buổi đo. Mặc định tắt, không lưu audio hoặc transcript mới. Deploy qua GitHub Actions như quy trình hiện hữu.

Ngày 2026-11-23: Actions backend 45194051848 thành công, audio_diagnostics enabled=true/interval_ms=1000. Tắt bằng workflow Deploy Backend & Core Services, ref production, input audio_diagnostics=disabled; unchanged giữ trạng thái hiện hữu. Không tự bật noise filter runtime. Xem kết quả và giới hạn robot thật trong tài liệu chẩn đoán.

## Lưu đầy đủ audio

Xem [FULL_AUDIO_CAPTURE.md](deployment/vps/FULL_AUDIO_CAPTURE.md). Deploy Backend có input audio_capture=enabled/disabled/unchanged độc lập audio_diagnostics. Không tự xóa bản ghi; thư mục nằm trong runtime/storage riêng, không công khai.

Trạng thái ngày 2026-11-23: audio_capture đã bật qua Actions 45199579444, archive và xuất WAV đã kiểm tra trên VPS. Đọc FULL_AUDIO_CAPTURE.md trước thu/phân tích dữ liệu; không tự xóa recordings.

## ASR input gate (backend, giữ firmware)

Đọc [ASR_INPUT_GATE.md](deployment/vps/ASR_INPUT_GATE.md). Workflow backend có input asr_input_gate=enabled/disabled/unchanged và post_tts_guard_ms. Khi chỉ chỉnh gate, giữ audio_capture/audio_diagnostics unchanged. Thiếu YAML mặc định bật guard 300 ms trong binary mới; phải đọc trạng thái rollout/nghiệm thu trước kết luận đã sửa hết nhận nhầm.

Ngày 2026-11-24: ASR gate đã bật 300 ms qua Actions 45226377103; snapshot và empty retry đã sửa, health/probe đạt. Đọc ASR_INPUT_GATE.md để kiểm tra robot thật, điều chỉnh/tắt gate bằng Actions. Không đổi firmware hoặc tự xóa recordings.
