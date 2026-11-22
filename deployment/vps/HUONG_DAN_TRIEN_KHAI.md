# Xiaozhi Go trên VPS 194.233.73.211

## Cấu hình đã cài

- Ubuntu 24.04 x86_64, 6 vCPU AMD EPYC ảo hóa, khoảng 12 GB RAM, AVX2.
- Backend Go upstream `hackers365/xiaozhi-esp32-server-golang`, release v0.6.5, mã nguồn tham chiếu `bbf9716`.
- Dùng binary phát hành chính thức; không viết lại hay biên dịch lại mã nguồn upstream.
- SQLite lưu quản trị/cấu hình; chưa cần MySQL hoặc Redis cho cấu hình hiện tại.
- Gipformer 1.5 68M ONNX INT8 chạy CPU qua sherpa-onnx. Cầu nối Python nhỏ cung cấp giao thức FunASR cho backend Go.
- ASR hiện nhận audio PCM16 16kHz, trả kết quả cuối sau khi kết thúc câu; không cung cấp bản chép tạm theo thời gian thực.
- Giới hạn ASR một worker, hai luồng tính toán và tối đa bốn tác vụ chờ/chạy; đây chưa phải cam kết tải thực tế.
- TTS mặc định Edge, giọng `vi-VN-HoaiMyNeural`; phụ thuộc dịch vụ bên ngoài.
- LLM chưa có API key, chờ người dùng cấu hình.
- Tắt MQTT/UDP, nhận diện người nói và dịch vụ ngoài không cần thiết.

## 1. Kết nối VPS bằng SSH key

Chạy trên máy Mac đang làm việc:

```bash
ssh -i ~/.ssh/antigravity_key root@194.233.73.211
```

Public key `antigravity-ai-agent` đã được thêm theo cách không trùng lặp vào `/root/.ssh/authorized_keys`.
Không thay đổi mật khẩu root hay tắt phương thức đăng nhập hiện có.

## 2. Mở trang quản trị

Mở trực tiếp `https://adminzeroone.hungnguyenaistem.online/` hoặc `/login` trên trình duyệt.
Tài khoản quản trị: `admin`. Mật khẩu trong file `~/.config/xiaozhi-go/194.233.73.211-admin.json` trên Mac hoặc `/root/.config/xiaozhi-go/admin.json` trên VPS, quyền 0600.
Không đưa file này vào Git hoặc chia sẻ công khai.

Không cần mở SSH tunnel để sử dụng trang quản trị qua tên miền. Khi cần truy cập nội bộ trực tiếp, vẫn có thể dùng:

```bash
ssh -i ~/.ssh/antigravity_key -N -L 18080:127.0.0.1:8080 root@194.233.73.211
```

Sau đó mở `http://127.0.0.1:18080`.

## 3. Thêm API LLM

1. Đăng nhập trang quản trị.
2. Mở quản lý cấu hình LLM, chọn cấu hình `Chờ cấu hình API LLM`.
3. Điền base URL, model và API key theo dịch vụ của bạn. Nếu dịch vụ có giao diện tương thích OpenAI, dùng provider OpenAI.
4. Bật và đặt cấu hình đó làm mặc định, lưu lại.
5. Dùng công cụ kiểm tra trong trang quản trị rồi thử hội thoại trên thiết bị.

Endpoint LLM ban đầu trỏ loopback cổng không phục vụ để tránh gửi audio/văn bản tới một dịch vụ chưa được chọn.
Chưa thể kiểm tra hội thoại đầy đủ trước khi có API LLM.

## 4. Kết nối ESP32

- OTA chính thức: `https://otazeroone.hungnguyenaistem.online/xiaozhi/ota/`
- WebSocket chính thức: `wss://otazeroone.hungnguyenaistem.online/xiaozhi/v1/`
- OTA trả kèm token WebSocket; firmware phải gửi `Authorization: Bearer <token>` khi kết nối.
- Đã bật kích hoạt thiết bị. Thiết bị mới cần được liên kết/kích hoạt trong trang quản trị trước khi sử dụng.
- Một thiết bị kiểm tra có ID `02:00:00:00:00:01` có thể xuất hiện ở trạng thái chờ kích hoạt.

Đã cấu hình Let’s Encrypt cho `otazeroone.hungnguyenaistem.online`; hỗ trợ TLS 1.2/1.3. HTTP cổng 80 chuyển sang HTTPS. OTA trong Manager và YAML đã trả URL WSS, giữ nguyên token.
Nếu đổi URL hoặc token OTA trong giao diện, cần cập nhật đồng bộ cấu hình gateway và URL local; không thay riêng một phía.

## 5. Vị trí dữ liệu và quản lý dịch vụ

Thư mục trên VPS: `/opt/docker/antigravity/xiaozhi-go`.

```bash
cd /opt/docker/antigravity/xiaozhi-go
docker compose ps
docker compose logs --tail=100 backend asr gateway
docker stats --no-stream
```

- `runtime/config/`: cấu hình Go, manager và secrets.
- `runtime/data/`: SQLite và audio lịch sử.
- `runtime/logs/`, `runtime/storage/`: log và file ứng dụng.
- `runtime/models/gipformer/`: model INT8 và thông tin revision/checksum.
- `releases/v0.6.5/`: ZIP gốc cùng checksum tải về; checksum này không phải chữ ký phát hành của tác giả.
- `/root/.config/xiaozhi-go/admin.json`: thông tin quản trị.

Container tự khởi động lại bằng `restart: unless-stopped`; Docker đã enable cùng hệ thống.
Log Docker xoay vòng 3 file x 10 MB mỗi container; vẫn cần quản lý dung lượng audio lịch sử.
Sau khi thay đổi mã triển khai, dùng `docker compose up -d --build`; thao tác này có thể làm gián đoạn phiên hội thoại.

## 6. Kiểm tra

```bash
cd /opt/docker/antigravity/xiaozhi-go
docker compose exec -T asr python - < verify_asr.py
python3 verify_backend.py
```

Các bài kiểm tra ASR dùng audio mẫu công khai từ repository tác giả Gipformer.
Kết quả thời gian là suy luận cho mẫu đã gửi, không bao gồm thời gian ghi âm, VAD, LLM, TTS hay mạng thiết bị.
Không chạy lại `prepare_runtime.py` hoặc `seed_console.py` trên dữ liệu đang sử dụng: các script phục vụ cài mới.

## 7. Cổng mạng

| Cổng | Phạm vi | Dịch vụ |
|---|---|---|
| 22/TCP | Công cộng | SSH |
| 80/TCP | Công cộng | Xác thực ACME và chuyển sang HTTPS |
| 443/TCP | Công cộng | HTTPS/WSS theo tên miền |
| 8989/TCP | Công cộng IPv4 | Gateway cũ; upstream của host Nginx |
| 8080/TCP | 127.0.0.1 | Trang quản trị/API |
| 10096/TCP | Chỉ mạng Docker | Gipformer ASR |

Các endpoint quản trị/inject/MCP API của upstream không được proxy ra cổng 8989.
Backend/ASR chạy UID 10001, dữ liệu được mount lâu dài và cấu hình mount chỉ đọc.

## 8. Backup

Cần sao lưu định kỳ database bằng SQLite backup API và cấu hình; giữ thêm bản sao ngoài VPS.
Thư mục snapshot của nhà cung cấp không thay thế việc kiểm tra phục hồi dữ liệu.
Đợt cài đặt ban đầu này không thay thế ứng dụng cũ và chưa cấu hình lịch backup bên ngoài.


## 9. SSL cho otazeroone.hungnguyenaistem.online

- Chứng chỉ: Let’s Encrypt, SAN đúng tên miền, khóa RSA 2048.
- Ngày hết hạn của chứng chỉ đầu tiên: 19/02/2027; tự thay đổi sau gia hạn.
- Nginx host: `/etc/nginx/sites-available/otazeroone.hungnguyenaistem.online.conf`, symlink tương ứng trong sites-enabled.
- Chứng chỉ: `/etc/letsencrypt/live/otazeroone.hungnguyenaistem.online/fullchain.pem`.
- Private key giữ trên VPS tại cùng thư mục, không đưa vào repository.
- Webroot ACME: `/var/www/letsencrypt`; giữ cổng 80 truy cập được cho gia hạn HTTP-01.
- Timer: `certbot.timer`; deploy hook `/etc/letsencrypt/renewal-hooks/deploy/xiaozhi-nginx-reload` kiểm tra rồi reload Nginx.
- `otazeroone` phục vụ OTA/WebSocket tại `/xiaozhi/`; trang quản trị/API đặt riêng ở `adminzeroone`. Cổng Manager 8080 vẫn chỉ bind localhost.

Kiểm tra trạng thái và gia hạn thử:

```bash
systemctl status certbot.timer --no-pager
certbot certificates
certbot renew --cert-name otazeroone.hungnguyenaistem.online --dry-run --run-deploy-hooks
cd /opt/docker/antigravity/xiaozhi-go
docker compose exec -T asr python - < verify_tls.py
```

`enable_tls_urls.py` cập nhật URL OTA trong Manager qua API và YAML, giữ nguyên token/LLM/TTS. Khi đổi tên miền cần cập nhật cả hai nơi; không sửa riêng dữ liệu trong DB.


## 10. Domain quản trị riêng

- Quản trị: `https://adminzeroone.hungnguyenaistem.online/`.
- OTA robot: `https://otazeroone.hungnguyenaistem.online/xiaozhi/ota/`.
- WebSocket robot: `wss://otazeroone.hungnguyenaistem.online/xiaozhi/v1/`.
- Tài khoản/mật khẩu giữ nguyên. Hai origin khác nhau nên cần đăng nhập lại trên domain admin.
- Root và /login của domain OTA chuyển người dùng sang domain admin. API quản trị không còn phục vụ trên domain OTA.
- Chứng chỉ admin riêng tại `/etc/letsencrypt/live/adminzeroone.hungnguyenaistem.online/`, cùng timer và deploy hook gia hạn đã cài.
- Nginx admin: `/etc/nginx/sites-available/adminzeroone.hungnguyenaistem.online.conf`.
- Cấu hình nguồn: `adminzeroone.nginx.conf` và `otazeroone.nginx.conf` trong thư mục triển khai.


## 11. Thêm/kích hoạt robot đúng Device-ID

1. Trong `设备管理` (Quản lý thiết bị), phân biệt `设备标识` (Device-ID/MAC) với `激活码` (mã kích hoạt).
2. Device-ID phải là MAC robot gửi lên, ví dụ `68:ee:8f:5b:33:00`; không điền mã 6 số vào trường này.
3. Mã 6 số chỉ điền vào trường mã kích hoạt. Nếu dùng mã để liên kết bản ghi sẵn có, phải giữ nguyên MAC của bản ghi robot.
4. Chọn người sở hữu và bật kích hoạt cho đúng dòng MAC. Việc bật kích hoạt ở một dòng có Device-ID sai không kích hoạt robot thật.
5. Chờ robot gọi OTA lại hoặc khởi động lại robot. Sau đó kiểm tra kết nối WebSocket và cấu hình LLM.
6. Hai bản ghi thử có Device-ID là mã 6 số không phải robot thật; chỉ xóa nếu người dùng chủ động xác nhận.

## 12. OpenAI và kết thúc câu tiếng Việt

1. Chọn provider `openai`, giữ API key trong trang quản trị, đặt Base URL là `https://api.openai.com/v1`. Adapter hiện tại tự nối `/chat/completions`; không điền `/responses` hoặc `/chat/completions` vào Base URL.
2. Model đang dùng là `gpt-5.4-mini`. Release v0.6.5 gửi tham số cũ `max_tokens`; đặt `max_tokens` bằng `0` trong JSON cấu hình để adapter bỏ tham số này. Giá trị này không đặt trần token đầu ra ở API; dùng lời nhắc yêu cầu trả lời ngắn. Muốn giới hạn cứng bằng `max_completion_tokens` cần cập nhật adapter.
3. Với bridge Gipformer offline đang triển khai, ASR `auto_end` phải là `false`: backend VAD phát hiện khoảng lặng rồi gửi kết thúc audio tới ASR. `true` chỉ phù hợp provider tự phát hiện kết thúc câu; bridge hiện tại không có khả năng đó.
4. Khi đổi cấu hình, phiên robot đang mở có thể giữ cấu hình cũ. Kết nối lại robot hoặc khởi động lại riêng backend để tạo phiên mới.
5. Đã thử WAV tiếng Việt mã hóa Opus 60 ms, gửi qua WSS công khai ở chế độ auto, chỉ thêm khoảng lặng và không gửi listen stop: nhận STT, câu trả lời OpenAI và 66.141 byte âm thanh TTS. Đây là thiết bị mô phỏng, chưa thay thế việc nghe thử loa/micro robot thật.

## 13. VieNeu Nano TTS trên CPU — 2026-11-22

Luồng hiện tại: VieNeu Nano tạo WAV PCM16 mono 24 kHz theo từng đoạn → backend Go mã hóa Opus → WSS → ESP32. Nano tổng hợp xong từng đoạn văn bản rồi phát đoạn đó, không sinh audio theo từng frame như Turbo; backend vẫn gửi Opus liên tục trong khi những đoạn sau đang tổng hợp.

- Cấu hình quản trị: `vieneu_tts`, provider `openai`, giọng `Mai Anh`, model `vieneu-v3-nano`, `response_format=wav`, `stream=true`.
- `api_url` là đầy đủ `http://tts:8000/v1/audio/speech` (khác Base URL của LLM).
- Container `tts` không publish cổng ra host. Có Bearer token riêng trong `runtime/vieneu.env` (0600); không dùng API key OpenAI để gọi VieNeu.
- Nguồn tại `vieneu-source/`, commit `6566468cf14974807050495036ab75cd041143f7`. Image `xiaozhi-vieneu-cpu:6566468-api`. Python dependencies theo uv.lock upstream.
- `patch_vieneu_api.py` áp dụng bản vá nhỏ cho API vốn chỉ phục vụ Turbo: chọn Nano, đúng sample rate, hai luồng CPU, giọng preset không cần aliases và chia tối đa 64 ký tự. Không sửa SDK sinh giọng hoặc backend Go.
- Giới hạn container: 4 vCPU, 4 GB RAM; model chạy hai luồng CPU, một lượt tổng hợp và hàng đợi bốn yêu cầu. Đây không phải cấu hình phục vụ nhiều robot nói cùng lúc với độ trễ thấp.
- Model cache ở `runtime/vieneu-cache`, chủ sở hữu UID 1000. Không chạy lại `prepare_runtime.py`/`seed_console.py` trên dữ liệu đang dùng.
- `vieneu-manifest.json` trên VPS ghi image ID và revision các model trong cache tại lúc cài. Model Nano đã tải revision `aba295eb96a6fa6003ebe417cc1f2802a7adc1dc`; không tự cập nhật nguồn/image khi khởi động lại.

Kiểm tra dịch vụ:

```bash
cd /opt/docker/antigravity/xiaozhi-go
docker compose ps
docker compose logs --tail 50 tts
docker compose exec -T tts python - < verify_vieneu.py
python3 verify_vieneu_opus.py
```

`verify_vieneu.py` đo thời điểm nhận PCM sau header WAV, không nhầm header được gửi sớm thành âm thanh đầu. `verify_vieneu_opus.py` kết nối thiết bị mô phỏng qua WSS, inject đúng văn bản vào thiết bị đó, giải mã các frame bằng libopus; không phát câu thử lên robot thật.

Quay về Edge TTS nếu cần:

```bash
cd /opt/docker/antigravity/xiaozhi-go
python3 configure_vieneu.py rollback
docker compose restart backend
```

Snapshot cấu hình TTS trước tích hợp ở `runtime/vieneu_previous_tts.json` (0600); backup SQLite/config/compose tại `/opt/docker/antigravity/backups/xiaozhi-vieneu-20261121T155823Z`. Edge vẫn được giữ lại, không có cơ chế tự fallback khi VieNeu quá tải.

Để dựng lại từ nguồn đã tải: `python3 patch_vieneu_api.py`, `docker compose build tts`, `docker compose up -d --no-deps tts`. Nếu clone lại, phải checkout đúng commit trước chạy patch. Script dừng nếu nguồn không khớp mẫu đã review.

Kết quả đo Nano: WAV âm thanh đầu 1,27–1,82 giây, RTF 0,54–0,59 ở hai câu thử. Kiểm tra WSS dài nhận Opus đầu 1,98 giây, server tổng hợp xong sau 5,24 giây, nhận 20.021 byte/10,92 giây audio. Đây là phép đo một luồng, không cam kết tốc độ khi có tải đồng thời. Turbo FP32 đã đo RTF 1,22–1,42 và không phù hợp tốc độ phát trên VPS này; người dùng chọn Nano ưu tiên mượt.

Lỗi ASR nhận sai âm thanh nền thành “ĐÂY” là vấn đề riêng, không được thay đổi bởi cấu hình TTS này.

## 14. Giao diện tiếng Việt cho admin và tài khoản cá nhân

1. Mở `https://adminzeroone.hungnguyenaistem.online/` và đăng nhập bằng tài khoản hiện tại. Người chưa có tài khoản chọn **Đăng ký**. Không cần tạo lại tài khoản/robot hoặc nhập lại API key.
2. Nếu tab cũ vẫn hiện giao diện trước đó, tải lại bằng Ctrl+Shift+R (Windows) hoặc Cmd+Shift+R (Mac). Trang tự chọn giao diện điện thoại khi màn hình nhỏ.
3. Admin có menu cấu hình/quản lý; tài khoản cá nhân có trợ lý AI, thiết bị, người nói, nhân bản giọng và kho kiến thức. Địa chỉ trang và quyền API giữ nguyên.
4. Tên và mô tả cấu hình đã lưu trước đây có thể vẫn bằng tiếng Trung: đây là dữ liệu được giữ nguyên. Bản dịch cũng không ghi đè tên robot, nội dung hội thoại, model, mã giọng hoặc nội dung do provider trả về.

Nguồn giao diện: `manager/frontend`. Catalog `src/locales/vi.json`, thuật ngữ hiệu đính `vi-overrides.json`, phản hồi server `server-vi.json`; `server.js` chỉ dịch thông báo hệ thống đã biết, giữ lỗi kỹ thuật chưa có bản dịch để tra cứu. `components.js` bổ sung locale thư viện.

Kiểm tra/build trước cập nhật:

```bash
cd manager/frontend
npm ci
npm test
npm run build
```

Script `scripts/localize-vi.mjs` kiểm kê các literal/nút chữ có thể dịch bằng AST, bỏ qua khóa/giá trị kỹ thuật, regex và console. `--apply` chỉ dùng khi catalog đã được review; không dịch JSON hoặc cấu hình runtime một cách hàng loạt. Bài kiểm tra đối chiếu ID với baseline trước Việt hóa trong `test/fixtures/localization-wire-values.json`; chỉ cập nhật baseline sau khi review thay đổi tính năng có chủ đích.

Trên VPS, static frontend ở `frontend/releases/`, symlink `frontend/current` trỏ bản đang chạy; `frontend/public/assets` giữ các chunk có hash của những lần phát hành. /api và /ws vẫn tới Go Manager localhost:8080, các đường dẫn nội bộ tiếp tục bị chặn. Không mở cổng mới.

Sau khi tải gói dist-only `/tmp/xiaozhi-ui-vi.tgz` lên VPS, chạy `python3 deploy_console_vi.py` trong thư mục triển khai. Script sao lưu SQLite/Nginx, tạo release, đổi symlink, kiểm tra và reload Nginx; không restart các dịch vụ âm thanh.

Quay hẳn về giao diện nhúng Go cũ nếu cần:

```bash
sudo cp /opt/docker/antigravity/backups/xiaozhi-console-vi-20261122T140831Z/admin-nginx.conf /etc/nginx/sites-available/adminzeroone.hungnguyenaistem.online.conf
sudo nginx -t
sudo systemctl reload nginx
```

Không cần khôi phục SQLite cho thao tác đổi giao diện. Muốn quay về một release static trước đó thì đổi symlink frontend/current về thư mục release đã giữ và giữ cấu hình Nginx static.

Đã kiểm tra 25 tuyến admin, 8 lượt điều hướng user, đăng ký/đăng nhập API, biểu mẫu chính và màn hình 390 px. Không gửi thật mọi biểu mẫu tích hợp ngoài hoặc thay đổi dữ liệu robot để thử bản dịch. Ảnh QA cục bộ nằm trong `manager/frontend/output/playwright/` và được gitignore; tài khoản QA không có robot hoặc quyền quản trị.
