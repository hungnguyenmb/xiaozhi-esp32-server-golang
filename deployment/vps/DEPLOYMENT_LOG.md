# 2026-11-23 — đã bật lưu toàn bộ audio

PR #10 deploy thành công qua Actions 45199579444, production f3c804b7. audio_capture enabled=true; cấu hình còn lại không đổi. Backend healthy/restarts=0, admin/OTA/WSS PASS. Probe lưu 100 mic+46 speaker packets, closed=true/dropped=0, xuất WAV cả hai hướng thành công. Không tự xóa recordings; đường dẫn và số đo/handoff tại FULL_AUDIO_CAPTURE.md. Chưa kết luận echo từ audio thật; cần lượt robot sau triển khai.

# 2026-11-23 — deploy PR271 và bật chẩn đoán

PR #6/#7/#8 đã merge production 2e685d3a. CI Linux 45191130321 và frontend deploy 45191741004 đạt; backend deploy bật chẩn đoán 45194051848 đạt. Backend healthy/restarts=0, admin 200, TLS/OTA/WSS smoke PASS. YAML ngoài audio_diagnostics không đổi, giữ Gipformer và silence=200 ms. Hai lỗi pipeline (fi, owner config) cùng rollback đã ghi BUG_LOG; chi tiết backup/hash/handoff ở AUDIO_DIAGNOSTICS.md. Chưa nghiệm thu mic/loa và nhiều robot thật.

# 2026-11-23 — PR #271, chỉ thay đổi cục bộ

Đã tích hợp bộ lọc ASR/VAD trên feat/vad-noise-filter-271; bổ sung bảo vệ tiếng Việt và tương thích cấu hình cũ. Go tests/race/vet/build và frontend tests/build đạt trên Mac; biểu mẫu kiểm tra với API giả lập. Chưa deploy, chưa thay runtime VPS, chưa kiểm chứng Linux/robot thật. Chi tiết: [PR271_INTEGRATION.md](PR271_INTEGRATION.md).

# Nhật ký triển khai 2026-11-21

## VPS 194.233.73.211

- Đã đăng nhập root, thêm public key `antigravity-ai-agent` không trùng lặp và kiểm tra kết nối bằng key.
- Cài Docker Engine/Compose, enable Docker khi boot; không khởi động lại VPS.
- Backend Go: binary upstream v0.6.5; source tham chiếu tag v0.6.5 tại commit bbf9716.
- Model Gipformer: revision `dd9227dcd8705c13f33bdbe59728d546ab94480f`; đối chiếu SHA256 với LFS metadata cho các file ONNX.
- Triển khai ba container: backend Go/manager, ASR nội bộ, Nginx gateway.
- SQLite và dữ liệu được mount; secrets ngẫu nhiên, admin riêng, quản trị localhost.
- Cổng công khai 8989, OTA + WebSocket có token. Chặn endpoint nội bộ tại proxy.
- LLM để chờ cấu hình theo yêu cầu người dùng; không sử dụng API key có sẵn của dự án khác.

## Kết quả kiểm tra thực tế

| Kiểm tra | Kết quả |
|---|---|
| SSH bằng key | Thành công |
| Đăng nhập admin và profile | Thành công |
| VAD TEN | Thành công |
| ASR audio mẫu 5,08 giây | 0,202 giây, có văn bản tiếng Việt |
| Bốn yêu cầu ASR đồng thời | 0,208 / 0,403 / 0,564 / 0,719 giây |
| ASR JSON sai kiểu | Đóng kết nối mã 1007 |
| OTA | Trả URL IP public, token và yêu cầu kích hoạt |
| WebSocket qua gateway | Nhận hello hợp lệ |
| TTS câu tiếng Việt qua Go tới WebSocket | 9.965 byte Opus; audio đầu sau khoảng 887 ms |
| Health từ Internet | HTTP 200 |
| WebSocket thiếu token từ Internet | HTTP 401 |
| Endpoint inject nội bộ từ Internet | HTTP 404 |

Timings là một phép thử mẫu, không phải SLA hoặc benchmark tải dài hạn. Chưa kiểm tra thiết bị ESP32 thật, hội thoại LLM đầy đủ, HTTPS/WSS và khả năng phục hồi sau reboot thực tế.

## Điểm cần hoàn tất bởi người dùng

1. Thêm API LLM trong trang quản trị.
2. Đã có tên miền OTA và HTTPS/WSS; quản trị vẫn truy cập qua SSH tunnel.
3. Kích hoạt thiết bị thật và kiểm tra audio trong môi trường sử dụng.

Xem BUG_LOG.md để biết hạn chế của công cụ test upstream với cấu hình và giọng Việt.

## Kiểm tra lại sau hơn 5 phút chạy backend/ASR

- Backend và ASR vẫn healthy; gateway hoạt động, không có restart sau sửa map_hash.
- RAM quan sát: backend khoảng 66 MiB; ASR khoảng 179 MiB; gateway khoảng 6 MiB (thời điểm kiểm tra ít tải).
- SQLite integrity_check: ok.
- Backup ban đầu: /opt/docker/antigravity/backups/xiaozhi-go-initial-20261121T143854Z.
- Lần kiểm tra lại TTS: 9.971 byte Opus, audio đầu sau khoảng 445 ms. Không coi kết quả này là độ trễ cố định.

## SSL — 2026-11-21

- DNS A otazeroone.hungnguyenaistem.online trả 194.233.73.211, không có AAAA tại lúc kiểm tra.
- Backup trước SSL: /opt/docker/antigravity/backups/xiaozhi-go-pre-ssl-20261121T144301Z.
- Cài host Nginx 1.24 và Certbot 2.9, dùng HTTP-01 webroot.
- Cấp chứng chỉ Let's Encrypt RSA 2048, SAN đúng domain; hết hạn ban đầu 19/02/2027.
- Host Nginx cổng 443 proxy qua gateway 8989, cổng 80 chuyển 308 sang HTTPS; quản trị vẫn localhost.
- Cập nhật URL OTA trong Manager và YAML thành wss://otazeroone.hungnguyenaistem.online/xiaozhi/v1/.
- TLS 1.2 và 1.3, HTTPS health 200, xác minh chứng chỉ thành công từ máy local.
- Kiểm tra OTA HTTPS, hello WSS public, thiếu token trả 401, endpoint nội bộ trả 404, HTTP redirect 308: PASS.
- Đã bật certbot.timer cùng deploy hook kiểm tra/reload Nginx.
- Chạy `certbot renew --dry-run --run-deploy-hooks --no-random-sleep-on-renew`: PASS; deploy hook kiểm tra cấu hình và reload Nginx thành công.

## Trang quản trị qua tên miền — 2026-11-21

- Mở frontend và API quản trị trên https://otazeroone.hungnguyenaistem.online/.
- Host Nginx root proxy tới Manager 127.0.0.1:8080; /xiaozhi/ tiếp tục qua gateway.
- Chặn /api/internal/, /api/configs, /api/system/configs và /admin/inject_msg từ Internet.
- Không thay tài khoản, token thiết bị, chứng chỉ hay container.
- Backup Nginx: /opt/docker/antigravity/backups/xiaozhi-admin-domain-20261121T145404Z.
- Frontend tự chuyển tới /login, đã quan sát bằng trình duyệt.
- Kiểm tra JS/CSS, đăng nhập admin HTTPS, profile/dashboard: PASS. API admin chưa đăng nhập trả 401; API nội bộ trả 404. OTA HTTPS và hello WSS tiếp tục PASS.

## Tách domain quản trị — 2026-11-21

- Admin: https://adminzeroone.hungnguyenaistem.online/, DNS A trỏ đúng 194.233.73.211.
- Cấp chứng chỉ Let's Encrypt RSA 2048 riêng, hết hạn ban đầu 19/02/2027.
- Nginx admin proxy Manager localhost; domain OTA giữ /xiaozhi/, root và /login chuyển sang admin.
- Backup trước thay đổi: /opt/docker/antigravity/backups/xiaozhi-split-admin-20261121T145843Z.
- Không thay token robot, URL OTA/WSS, tài khoản hay database.
- Kiểm tra bên ngoài: HTTPS admin 200, TLS verify=0; root OTA 302 tới admin; API quản trị trên OTA trả 404.
- Đăng nhập admin HTTPS, profile, dashboard, JS/CSS và kiểm tra quyền API: PASS. OTA HTTPS/hello WSS: PASS. Gia hạn thử chứng chỉ admin cùng deploy hook: PASS.

## Khôi phục kích hoạt robot — 2026-11-21

- Robot thật báo MAC `68:ee:8f:5b:33:00`, firmware User-Agent esp32-s3n16r8-emoji/2.0.1.
- Các dòng do người dùng nhập trước đó có Device-ID bằng mã 6 số, nên không khớp MAC.
- Sao lưu SQLite: /opt/docker/antigravity/backups/xiaozhi-device-activation-20261121T151906Z.
- Cập nhật qua API admin bản ghi MAC thật ID 4: user_id=1, activated=true, nickname test zero one. Không xóa dòng khác.
- Internal check-activation xác nhận true. Request OTA kiểm tra trả WSS và không có activation challenge.
- Robot thật gọi POST /xiaozhi/ota/ lúc 22:20:02 (Asia/Ho_Chi_Minh), HTTP 200, response 256 byte; log backend xác nhận xử lý đúng MAC/client UUID của robot.
- Chưa xác minh được phiên WebSocket/audio thực; cần người dùng đánh thức robot để thử.
- API LLM vẫn để trống và chưa có agent; chưa thể kiểm tra hội thoại đầy đủ.
- Xem BUG_LOG.md về lỗi định dạng activation payload upstream; không sửa firmware hoặc bỏ kiểm tra bảo mật.
## Sửa hội thoại OpenAI/ASR — 2026-11-21 22:36–22:41 +07

- Sao lưu SQLite/config tại `/opt/docker/antigravity/backups/xiaozhi-conversation-fix-20261121T153635Z`.
- Giữ model gpt-5.4-mini/API key, sửa Base URL về https://api.openai.com/v1 và bỏ tham số max_tokens không tương thích bằng cấu hình max_tokens=0. Giới hạn: không áp trần token đầu ra ở API.
- Sửa auto_end=false qua API quản trị, YAML runtime và mặc định prepare_runtime.py; bridge offline dùng backend VAD kết thúc câu.
- Adapter Go gọi OpenAI thành công (first_packet_ms=1577), VAD thành công. WSS + Opus 60 ms + VAD + Gipformer STT + OpenAI + TTS tiếng Việt thành công (66.141 byte, audio đầu sau STT khoảng 1.556 ms).
- Dòng thiết bị mô phỏng được khôi phục trạng thái sau test. Khởi động lại riêng backend để bỏ phiên robot cũ; ASR/gateway giữ nguyên.
- Test trên thiết bị mô phỏng; chưa xác nhận người dùng nghe được câu trả lời trên robot thật.
- Sau restart, backend healthy; robot thật MAC 68:ee:8f:5b:33:00 tự kết nối lại lúc 22:40:18 +07, gửi listen start và lấy được MCP tools. Kiểm tra lại audio mô phỏng sau restart vẫn PASS. TLS 1.3, OTA HTTPS, WSS hello, chặn WS thiếu token và chặn route nội bộ đều PASS.

## VieNeu Nano CPU — 2026-11-21/22

- Backup trước thay đổi: `/opt/docker/antigravity/backups/xiaozhi-vieneu-20261121T155823Z` (SQLite, config, compose).
- Ghim upstream VieNeu commit `6566468cf14974807050495036ab75cd041143f7`, Docker CPU với uv.lock; cache bền vững, API chỉ nội bộ và Bearer riêng.
- Turbo FP32 RTF 1,22–1,42 trên các phép đo luồng CPU; không đủ nhanh hơn tốc độ phát. Người dùng chọn Nano ưu tiên mượt.
- API upstream chỉ có Turbo nên thêm patch tối thiểu trong script riêng để dùng Nano native 24 kHz, presets không có aliases, 2 threads và max_chars=64. Model/SDK/Go không thay đổi.
- Nano WAV chunked: câu ngắn TTFA PCM 1271 ms, RTF 0,591; câu dài TTFA PCM 1815 ms, RTF 0,541. Thiếu token trả 401.
- Tạo config riêng `vieneu_tts`, provider openai, voice Mai Anh, WAV; giữ Edge và snapshot rollback. Chọn mặc định sau khi Nano qua bài kiểm tra trực tiếp.
- WSS thật qua domain + thiết bị mô phỏng: 20.021 byte Opus, 10,92 giây audio, frame 60 ms giải mã libopus thành công; frame đầu 1978 ms. Server TTS kết thúc sinh sau 5,24 giây nên xác minh gửi trước khi sinh xong toàn câu.
- Restart riêng backend để bỏ cấu hình TTS cũ trong phiên robot; không thay đổi LLM hoặc ASR/VAD.
- Kiểm tra chạy TTS và ASR đồng thời: TTS RTF 0,51–0,56, ASR single 0,437 giây và bốn yêu cầu 0,181–0,647 giây. Nano dùng khoảng 458 MB RAM trong lúc thử, CPU gần hai lõi.
- Xác minh container không publish cổng host, secret/snapshot mode 0600; ghi source/image/model revisions vào `vieneu-manifest.json` trên VPS.
- Sau restart, thử toàn luồng ASR→OpenAI→Nano→Opus WSS: nhận STT tiếng Việt, ba đoạn trả lời, 21.455 byte audio; STT sau 6206 ms, audio đầu sau 9130 ms từ đầu gửi WAV thử. Thiết bị mô phỏng khôi phục trạng thái kích hoạt trong finally. Chưa có xác nhận loa robot thật ở lần bàn giao này.
- Theo dõi hơn 5 phút sau tạo container Nano cuối: TTS/backend/ASR healthy, TTS restart count=0; không có lỗi HTTP 500/429 trong các lượt kiểm tra. Robot thật chưa xuất hiện kết nối lại trong log sau restart backend.
## Việt hóa console — 2026-11-22

- Nhánh nguồn `feat/vietnamese-console`; không commit/push. Kiểm kê ban đầu: 3.422 vị trí/1.835 chuỗi trong 66 file frontend. Bảng dịch và thuật ngữ tập trung tại manager/frontend/src/locales; dịch sơ bộ các chuỗi giao diện công khai upstream rồi hiệu đính luồng chính. Không gửi code, secrets hay dữ liệu người dùng đến dịch vụ dịch.
- Việt hóa đăng nhập/đăng ký/setup, desktop/mobile, các trang admin/user, metadata router, biểu mẫu, chú thích sử dụng; vi-VN cho ngày giờ và locale thư viện. Bổ sung nhãn trợ năng tiếng Việt còn thiếu của Element Plus.
- Thông báo backend: 734 mẫu phản hồi có sẵn trong catalog và các thông báo dùng chung. Dịch ở lớp hiển thị lỗi/thông báo API, lỗi preview Blob và lỗi SSE; không dịch đệ quy dữ liệu, không sửa API backend.
- Giữ nguyên protocol IDs/provider/model/voice và option values. Ví dụ JSON giữ nguyên cấu trúc/khóa. Không tự chạy diagnose.js trên trang vì script chẩn đoán cũ gọi localhost:8080 và in localStorage ra console.
- npm test: 7/7 PASS; production build PASS. Cảnh báo kích thước bundle >500 kB vẫn có; không nâng thư viện hoặc refactor ngoài phạm vi.
- Playwright: kiểm tra đăng nhập/đăng ký, biểu mẫu thêm robot/trợ lý, trang user/admin trên desktop và viewport 390 px. Tạo một tài khoản cá nhân `vi_qa_...` qua register để thử quyền; không liên kết robot, không xóa bản ghi.
- Audit domain thật: 25 tuyến admin và 8 lượt điều hướng user render thành công, không có pageerror. User vào /admin/users bị chuyển về /agents; API admin trả 403 cho user, 200 cho admin. Hai màn hình admin có tên/mô tả cấu hình cũ bằng Trung được giữ nguyên dữ liệu.
- Giao diện tách khỏi Go binary, Nginx phục vụ dist qua frontend/current; /api và /ws vẫn proxy Manager. Chunk có hash lưu dùng chung, asset cũ của Go có fallback. Không restart backend/ASR/TTS.
- Release cuối: `/opt/docker/antigravity/xiaozhi-go/frontend/releases/vi-20261122T141908Z`.
- Backup đầu trước tách UI: `/opt/docker/antigravity/backups/xiaozhi-console-vi-20261122T140831Z`; backup trước bản hiệu đính cuối: `/opt/docker/antigravity/backups/xiaozhi-console-vi-20261122T141400Z` (SQLite và Nginx).
- Nginx -t/reload PASS. Public login/SPA/bundle 200, profile thiếu token 401, internal API 404. TLS/OTA HTTPS/WSS hello/WS thiếu token 401/HTTP redirect PASS. Container giữ healthy, không có thay đổi cấu hình hội thoại.

- Hiệu chỉnh cuối trên điện thoại: nút Đổi câu hỏi không bị co/xuống dòng, rút gọn placeholder mật khẩu; build PASS, kiểm tra ảnh trên domain thật ở 390×844. Nginx -t PASS sau cập nhật.

## PhoWhisper ASR — triển khai code, 2026-11-22

- Nhánh feat/phowhisper-asr: thêm sidecar FunASR offline tương thích Go, faster-whisper 1.2.1/CTranslate2 4.6.0 INT8, VinAI base revision cố định; không đổi Go/firmware/Gipformer/LLM/TTS.
- Queue một worker, 4 chỗ chờ; deadline không tạo inference native thứ hai khi phép gọi cũ chưa kết thúc. Validate PCM/control, health sau checksum/warm-up, metric không ghi transcript.
- Thêm manifest/chuyển model/checksum, overlay nội bộ, benchmark WER/CER và rollout chọn lọc qua typed API. Actions bốn chế độ trial/benchmark/activate/rollback; production-only deployment, image ID và artifact hash, khóa concurrency chung backend.
- Khảo sát VPS chỉ đọc: AMD AVX2, chưa thấy VNNI/AVX-512, khoảng 10 GB RAM khả dụng. Baseline Gipformer 5,08 giây audio: đơn 573 ms, 4 yêu cầu 172–769 ms. Chưa có WER/CER.
- Model thật local ARM64 INT8, 2 threads: 511,4 ms tải đơn và 526,89–2137,8 ms/4 yêu cầu cho cùng mẫu speech. Silence zero rỗng; không phải benchmark VPS hoặc đánh giá noise/độ chính xác đầy đủ.
- 28 unit tests PASS; Python compile/actionlint/Compose config PASS. Docker daemon local chưa chạy, chưa build image Linux/chạy CI hoặc deploy PhoWhisper trên VPS. Không đổi ASR mặc định, chưa xác minh robot thật.
- Kế hoạch/bàn giao và lệnh đầy đủ: PHOWHISPER_PLAN.md mục 8 và DEPLOYMENT_GUIDE.md mục 6.

## PhoWhisper — đánh giá trên VPS 2026-11-23

- Trial base/small và benchmark qua Actions; sửa OTA probe GET thiếu Device-Id ở PR #3, small profile ở PR #4. Không deploy thủ công dịch vụ.
- Bộ 30 speech FLEURS CC BY 4.0 +20 nonspeech tổng hợp: Gipformer WER 4,07%, base 10,49%, small 7,07%. p95 tải đơn 50 mẫu lần lượt khoảng 0,28–0,31 / 3,53 / 5,91 giây. Small tải đồng thời được chủ động dừng sau tải đơn không đạt.
- User chọn giữ Gipformer; ASR mặc định xác minh vps_asr, chưa từng chạy activate. Sidecar small healthy nhưng không làm mặc định. Dừng đúng benchmark client để trả CPU, giữ backend/ASR/TTS.
- Chi tiết số đo/nguồn/giới hạn/run URL: PHOWHISPER_EVALUATION_20261123.md.

## 2026-11-23 — thêm chẩn đoán cục bộ, không deploy

Bổ sung AUDIO_DIAG, cấu hình tắt mặc định, accessor xác suất TEN/Silero và unit tests. Không thay threshold/ASR/LLM/TTS. Bàn giao và cách bật/đọc: AUDIO_DIAGNOSTICS.md.

## 2026-11-23 — phát triển lưu toàn bộ audio

Người dùng yêu cầu lưu đầy đủ, không chỉ VAD. Thêm capture Opus theo kết nối tại transport, markers/timestamps, writer async có báo thiếu/lỗi, rotation giữ dữ liệu, free-space guard Linux/macOS, exporter WAV và toggle Actions. Kiểm thử và trạng thái phát hành trong FULL_AUDIO_CAPTURE.md.
