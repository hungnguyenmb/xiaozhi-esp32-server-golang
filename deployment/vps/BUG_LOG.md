# 2026-11-23 — Rà soát tương thích khi tích hợp #271

- Default threshold_low cố định có thể làm cấu hình Silero cũ đổi ngưỡng: bỏ default cố định, fallback theo threshold; thêm test.
- Form sửa cấu hình cũ có thể tự thêm ngưỡng thấp khác threshold: dùng helper merge chung cho VADConfig/wizard; test cả low=0.
- Reset hysteresis chưa xóa lịch sử native TEN: tạo lại native handle khi trả pool; test so sánh instance mới và không mở lại handle đã đóng.
- Bộ lọc từ ngắn cần giữ câu trả lời tiếng Việt: thêm keep phrases và giới hạn filler Latin theo thời lượng; test hook vẫn cho câu tiếp theo đi qua.

# Ghi chú triển khai

## 2026-11-21: Nginx không dựng được map token

- Hiện tượng: gateway thoát lúc khởi động, không mở được cổng 8989.
- Nguyên nhân: chuỗi `Bearer` cộng token ngẫu nhiên dài hơn kích thước bucket mặc định.
- Sửa: thêm `map_hash_bucket_size 128` trước các `map`; kiểm tra `nginx -t` trước khởi động.
- Bài học: kiểm tra cấu hình proxy với secret có độ dài thực tế.

## 2026-11-21: Công cụ test upstream báo không có cấu hình dù DB có dữ liệu

- Hiện tượng: API `/api/admin/configs/test` trả `_none` khi không chỉ định config_ids.
- Điều tra: dữ liệu trong DB có enabled/default; mã upstream dùng type assertion map cho dữ liệu kiểu `gin.H`.
- Cách kiểm tra: chỉ định rõ `config_ids` cho từng provider để dùng nhánh đọc DB. Không thay mã nguồn upstream.
- Phạm vi: hạn chế công cụ test, không kết luận provider hội thoại mất cấu hình.

## 2026-11-21: Test TTS mặc định báo lỗi với giọng Việt

- Hiện tượng: test cấu hình TTS trả audio rỗng với `vi-VN-HoaiMyNeural`.
- Nguyên nhân: bài test upstream gửi cố định câu tiếng Trung `配置测试`.
- Xác minh: inject câu tiếng Việt qua backend Go tới WebSocket giả lập, nhận 9.965 byte audio Opus; audio đầu tiên sau khoảng 887 ms ở lần đo này.
- Không thay provider hoặc mã nguồn upstream; kiểm tra TTS bằng nội dung đúng ngôn ngữ.

## 2026-11-21: Nhập mã kích hoạt vào Device-ID làm robot không được nhận diện

- Robot thật báo Device-ID/MAC `68:ee:8f:5b:33:00`; hai bản ghi admin đã kích hoạt mang Device-ID `572439` và `124009`.
- Log OTA trước đó liên kết hai mã này với chính MAC trên. Khi Device-ID đổi thành mã số, backend không tìm được robot theo MAC và tạo một bản ghi pending mới.
- Khắc phục dữ liệu: backup SQLite rồi cập nhật qua API admin bản ghi thực ID 4 có MAC đúng, gán user_id=1 và activated=true. Không xóa/đổi các bản ghi người dùng đã thêm trước đó.
- Backup: `/opt/docker/antigravity/backups/xiaozhi-device-activation-20261121T151906Z`.
- Xác minh ban đầu: API check-activation trả true; request OTA kiểm tra trả WSS và không còn activation challenge. Đây là request kiểm tra, không phải bằng chứng robot thật đã hoàn tất kết nối.
- Phát hiện thêm: `/xiaozhi/ota/activate` upstream yêu cầu Payload.algorithm=hmac-sha256 trong khi firmware có thể gửi JSON phẳng hoặc `{}` khi chưa provision serial/HMAC, dẫn tới HTTP 400. Chưa thay đổi giao thức hay nới kiểm tra HMAC; với bản ghi MAC được admin kích hoạt, lần OTA kế tiếp không yêu cầu nhánh này.
- LLM vẫn chưa có API key, chưa có agent. Đây là bước cấu hình hội thoại riêng sau kích hoạt.
- 2026-11-21 — Hội thoại bị kẹt và OpenAI báo lỗi:
  - Log robot thật xác nhận đã kết nối WebSocket và gửi audio, nhưng chờ 60 giây rồi lỗi 404 `/v1/responses/chat/completions`.
  - Base URL bị nhập thành endpoint Responses API, trong khi adapter Go sử dụng Chat Completions. Sửa URL gốc thành `https://api.openai.com/v1`.
  - Kiểm tra tiếp phát hiện HTTP 400 vì adapter mặc định gửi `max_tokens=500`, model GPT-5.4 mini yêu cầu `max_completion_tokens`. Cấu hình `max_tokens=0` khiến adapter bỏ tham số cũ; không đổi model/key. Đổi lại không có trần token đầu ra ở API. Adapter chưa được nâng cấp để hỗ trợ trần mới.
  - Lỗi cấu hình do lần triển khai ban đầu: đặt ASR `auto_end=true` cho bridge offline làm backend bỏ qua VAD, nhưng bridge chỉ suy luận khi nhận kết thúc audio. Sửa DB quản trị, YAML runtime và prepare_runtime.py thành false.
  - Backup trước sửa: `/opt/docker/antigravity/backups/xiaozhi-conversation-fix-20261121T153635Z`.
  - LLM test qua adapter Go PASS, gói đầu 1.577 ms; VAD PASS. Test audio qua WSS ở chế độ auto nhận STT sau 6.289 ms và audio đầu sau 7.845 ms tính từ đầu phát sample, tổng 66.141 byte; không gửi listen stop nên xác minh được VAD tự kết thúc câu.
  - Kích hoạt tạm đúng dòng thiết bị mô phỏng có sẵn 02:00:00:00:00:01 để kiểm tra, khôi phục trạng thái kích hoạt trong finally; không sửa dòng robot thật. Khởi động lại riêng backend để phiên robot thật không giữ cấu hình cũ.
  - Bài học: kiểm tra đúng giao thức API và khả năng kết thúc câu của từng provider; cần thử toàn luồng âm thanh, không chỉ test ASR/TTS riêng lẻ.

- 2026-11-22 — Tích hợp VieNeu CPU:
  - Turbo FP32 trên VPS EPYC hiện tại chậm hơn real-time (RTF >1) dù thử 1/2/4 threads. Không suy diễn benchmark Core i5 của tác giả thành cam kết VPS. Người dùng đã chọn Nano; Nano đạt RTF khoảng 0,54–0,69 ở các bài thử một luồng.
  - API upstream hardcode Turbo/48 kHz và gọi `_voice_aliases`, `resolve_voice_name` mà Nano không có. Patch riêng expose mode/threads, dùng native 24 kHz và kiểm preset cho Nano; không giả header 48 kHz cho PCM 24 kHz. Startup Nano và WAV→Opus WSS đã kiểm tra.
  - POST config upstream v0.6.5 tự ép is_default=true khi không có dòng memory dù đang tạo TTS. prepare đã khôi phục Edge ngay sau tạo; script sửa để giữ mặc định cũ trong bước prepare. Kích hoạt VieNeu sau khi kiểm tra đạt.
  - Nano stream theo đoạn hoàn tất, không phải từng frame. Phải dùng đoạn đầu ngắn và đo PCM/Opus thật, không dùng thời điểm header WAV để báo TTFA.


## Kiểm tra PhoWhisper local — 2026-11-22

- Compose config trên checkout thiếu runtime/vieneu.env riêng của VPS. Không đưa secret vào repo: CI tạo env rỗng chỉ cho validation, local kiểm tra với project directory tạm. --no-env-resolution vẫn có thể kiểm tra sự tồn tại file nên không đủ nếu thiếu file.
- PyTorch wheel từ CPU index có thể có suffix +cpu. Kiểm tra phiên bản converter so phần trước dấu +, vẫn yêu cầu đúng 2.8.0; tránh từ chối wheel CPU đúng phiên bản.
- Lượt đầu tải model Hugging Face gặp TLS handshake timeout mặc định; chạy lại với HF_HUB_ETAG_TIMEOUT=60 và HF_HUB_DOWNLOAD_TIMEOUT=120 thành công, không tắt xác minh TLS.

## PhoWhisper Linux CI — 2026-11-23

- Run 45084209503: Docker COPY không thấy requirements/bridge vì deployment/vps/.dockerignore cũ chỉ cho phép bundle backend. Thêm Dockerfile.phowhisper.dockerignore với allowlist 4 file đúng image mới, không mở rộng context để đưa runtime secrets/model cache vào image.

- Sau merge PR #1: CI production run 45084561511 bị cancelled vì khóa chung đặt ở cấp toàn workflow; GitHub chỉ giữ một lượt pending cho một concurrency group. Chuyển khóa chung xuống job deploy của ASR, tách nhóm CI theo ref/event. Backend và thao tác ASR trên VPS vẫn tuần tự.

## PhoWhisper trial public probe — 2026-11-23

- Run 45092837122 đã cài sidecar, health/speech/silence/TLS/OTA/WSS đều qua trong helper; bước curl cuối báo đỏ do GET OTA thiếu Device-Id nhận 400. Sửa probe thành POST JSON kèm Device-Id/Client-Id smoke test sẵn có. Không sửa API hoặc bỏ kiểm tra trạng thái.

## 2026-11-23 — race phát hiện khi kiểm tra chẩn đoán

Race test toàn chat thất bại ở MCP reinit: waitForMcpInitState đọc trạng thái trong khi finishMcpInit ghi (mcp_reinit_test.go:124/chat.go:589). Không thuộc thay đổi chẩn đoán, chưa sửa. Xem AUDIO_DIAGNOSTICS.md để tiếp tục.

## 2026-11-23 — script bật chẩn đoán thiếu fi

Actions 45191781529 build thành công, bước SSH báo unexpected end of file do if cấu hình thiếu fi. Backend cũ vẫn healthy, SHA binary chưa đổi. Đã bổ sung fi và kiểm tra bash -n trên script trích xuất từ YAML trước chạy lại.

## 2026-11-23 — atomic config write đổi owner

Run 45192967450: backend mới không đọc được YAML do file thay thế thuộc root thay vì UID/GID 10001. Health rollback đã trả binary/config cũ. Sửa giữ st_uid/st_gid cùng mode trước replace; không chown sang root. Kiểm tra cấu hình trên VPS sau phát hành.

## 2026-11-24 — snapshot ASR và race TTS khi abort

OnVoiceSilence reset VAD trước final offline làm voice_ms=0. Đã giữ snapshot theo stream, chốt một lần trước reset, dùng cho log/hook; loop cũ không release resource của stream mới. Test keyword-abort race phát hiện Get/SetTtsStart không đồng bộ; đổi cờ sang atomic qua accessor. Vet đầy đủ client còn lostcancel ở InitLlm có sẵn; không sửa LLM ngoài phạm vi, kiểm tra các analyzer khác bằng lostcancel=false.

## 2026-11-24 — gate sau ASR rỗng

Phát hiện khi rà soát release: endpoint đóng gate nhưng ASR rỗng không tạo TTS/listen-start nên retry phải tự mở lại. Đã dừng build 45225225393 trước deploy, bổ sung helper tái khởi tạo gate cùng provider và test ASR rỗng/failure; không để robot bị kẹt sau tiếng động không ra chữ.

## 2026-11-24 — mẫu dư TEN và nhận nền sau gate (nhánh thử)

TEN từng bỏ phần mẫu không đủ hop512 ở cuối mỗi packet; +1.0 float có thể wrap thành -32768. Đã thêm FIFO/reset/saturation và tests độc lập kích thước packet. Không coi đây là nguyên nhân duy nhất của robot tự trả lời: noise ngoài guard vẫn có thể vào ASR. Đã thêm preprocessing/onset thử nghiệm, mặc định tắt; cần benchmark/robot theo BACKEND_NOISE_SUPPRESSION.md trước phát hành.
