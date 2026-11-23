# Lưu đầy đủ âm thanh mic/loa — 2026-11-23

## Yêu cầu và phạm vi

Người dùng đã yêu cầu lưu đầy đủ để phân tích sau này, thay cho chỉ lưu đoạn VAD hoặc tự xóa sau 24 giờ. Không có lịch tự xóa/ghi đè. PR #10 đã merge production f3c804b7, CI Linux 45199001796 thành công. Đã phát hành thành công bằng Actions 45199579444 và bật audio_capture trên VPS.

Bộ ghi lưu **mọi gói âm thanh server nhận được** từ transport và mọi gói gửi thành công xuống robot (TTS/media), trước gate kích hoạt/listen/VAD/ASR ở chat. Có gói mic thì ghi kể cả khi client VoiceStop hoặc TTS đang chạy. Không thể lưu âm thanh mà firmware không thu/không gửi, cũng không khôi phục gói mất trước server hoặc bị drain/bỏ trong transport trước RecvAudio. Đây không phải ghi âm mic phần cứng liên tục 24/7. Không đổi ASR/LLM/TTS, firmware, ngưỡng hoặc thuật toán chống vọng.

## Định dạng và vị trí

Mặc định mã nguồn tắt. Cấu hình runtime:

```yaml
chat:
  audio_capture:
    enabled: true
    directory: /app/storage/audio-captures
```

Trên VPS tương ứng `runtime/storage/audio-captures/` bên trong thư mục dự án. Mỗi kết nối transport có thư mục UTC-random riêng, file `000001.jsonl`, `000002.jsonl`… tối đa xấp xỉ 8 MiB/segment (một record lớn hơn giới hạn có thể vượt; gói Opus bình thường nhỏ). Chỉ chia file, không xóa file cũ. Đây là thư mục riêng, không có endpoint tải xuống public; truy cập bằng SSH hoặc docker exec. File 0600, thư mục mới 0700. Gitignore chặn audio-captures; không commit bản ghi, WAV xuất ra hoặc raw logs cá nhân.

Mỗi JSONL chứa header/device, audio mic hoặc speaker với dữ liệu Opus base64, format sample_rate/channels/frame_duration, timestamp UTC lúc nhận/gửi ở server và sequence. Marker tts_start/tts_stop chỉ ghi khi lệnh gửi thành công; không lưu arbitrary commands/API key. speaker_send_error đánh dấu lỗi gửi, không coi dữ liệu gửi lỗi là đã phát. Close ghi tổng dropped nếu đóng bình thường.

Timestamp server không phải thời gian ADC/DAC vật lý; gói gửi thành công không chứng minh loa đã phát hết. Session ID không được thêm vào packet; đối chiếu bằng device và timestamp với AUDIO_DIAG, thư mục là ID kết nối. Khoảng không có mic packet có thể là robot ngừng truyền, không kết luận môi trường im lặng.

## Độ đầy đủ và vận hành

Writer bất đồng bộ, hàng đợi 1024 record/kết nối; không đợi ổ đĩa trên luồng audio. Nếu queue đầy hoặc writer lỗi, ghi AUDIO_CAPTURE missing/writer failed và sequence có thể có khoảng trống. Không được tuyên bố lossless khi có lỗi. Flush mỗi giây/close; crash/kill có thể mất phần chưa flush hoặc để record cuối dở; không fsync mỗi packet. Khi writer lỗi, ngừng ghi kết nối đó; các kết nối sau sẽ thử tạo bộ ghi mới. Không làm hỏng luồng hội thoại vì lỗi ghi.

Linux/macOS kiểm tra dung lượng khả dụng khi mở segment; dưới 2 GiB thì dừng bộ ghi và báo lỗi, không xóa dữ liệu cũ. Đây là kiểm tra định kỳ theo segment, không phải quota toàn hệ thống. Nền tảng khác chưa có kiểm tra free-space. Theo dõi `du -sh runtime/storage/audio-captures` và `df -h`; không tự dọn recordings.

Bật/tắt qua GitHub Actions **Deploy Backend & Core Services**, ref **production**, input **audio_capture=enabled/disabled**. Input audio_diagnostics=unchanged giữ log metadata hiện hữu. Input mặc định unchanged không đổi runtime capture. Script giữ các trường YAML khác, uid/gid/mode và sao lưu cấu hình; health failure rollback core binary/config. Không deploy thủ công. Restart backend làm các robot kết nối lại; tắt chỉ ngừng các phiên sau restart, không xóa file.

## Xuất WAV để nghe/phân tích

Pipeline cài công cụ `/app/audio_capture_export` trong backend image, tương ứng `bundle/audio_capture_export` trên host. Đóng phiên robot trước khi export để có close marker (kết nối đóng mới flush hết; đoạn đang ghi có thể dở). Ví dụ trong container, thay CONNECTION bằng thư mục thực:

```bash
docker compose exec -T backend ./audio_capture_export -input /app/storage/audio-captures/CONNECTION -direction mic -output /app/storage/audio-captures/CONNECTION/mic.wav
docker compose exec -T backend ./audio_capture_export -input /app/storage/audio-captures/CONNECTION -direction speaker -output /app/storage/audio-captures/CONNECTION/speaker.wav
```

Công cụ đọc lần lượt segments, giải mã Opus thành WAV PCM16 đúng sample rate/channels khai báo; không đổi codec robot. Không ghi đè output đã tồn tại, chọn tên mới nếu thử lại. Mặc định timeline=true, dùng anchor header chung để giữ tương đối mốc mic/loa và chèn khoảng trống server lớn hơn 100 ms. Đây là xấp xỉ theo thời gian nhận/gửi, chịu network jitter; không dùng WAV timeline để đo chính xác độ trễ AEC. `-timeline=false` nối packet liên tục để nghe nội dung mà không giữ khoảng ngừng. Raw JSONL mới là nguồn đối chiếu thời gian chính.

Nếu format thay đổi giữa kết nối, codec không phải Opus, data hỏng hoặc close ghi dropped>0, exporter báo lỗi; WAV đã ghi có thể chỉ là phần trước lỗi, không coi đầy đủ. Thiếu close marker báo cảnh báo (phiên đang mở/crash). Giới hạn WAV RIFF 4 GiB; export kết nối ngắn hơn nếu vượt. Có thể chạy local: `go run -tags nolibopusfile ./cmd/audio-capture-export ...` với libopus dev/cgo, không đưa input lên Git.

## Bàn giao kỹ thuật và kiểm tra

- `internal/domain/audio/capture`: writer độc lập, không giải mã trên luồng hội thoại.
- `server_transport.go`: mic tại RecvAudio, speaker sau SendAudio thành công, marker TTS và Close.
- `chat.go`: cập nhật định dạng mic/loa sau hello và cập nhật output config.
- `cmd/audio-capture-export`: xuất WAV riêng hai hướng, giữ timestamp gốc trong JSONL.
- `docker/Dockerfile.linux` và deploy-backend.yml: build/cài exporter, tùy chọn capture runtime, bảo toàn owner.
- Tests: sao chép packet gốc, marker/hướng/format; ghi trước gate VoiceStop, output lỗi không tính đã gửi; queue overflow; writer failure; concurrent close; segment rotation không mất packet; Opus→WAV và timeline/compact; không ghi đè, báo archive bị cắt. Race tests capture/exporter và test chat chọn lọc đạt trên Mac, full chat tests/vet/build cũng đạt. CI Linux đã qua (45199001796); deploy và kiểm tra file/export VPS đã qua.
- Script deploy đã qua bash -n/actionlint và giả lập success/health rollback, giữ owner/mode và tất cả YAML ngoài audio_capture.

Sau phát hành: kiểm tra audio_capture enabled, owner config, container healthy, file thực có mic/speaker, thử export WAV từ phiên giả lập có audio tổng hợp. Robot thật cần thử im lặng/nhiễu/loa phát/lời nói; đối chiếu ECHO_DIAGNOSIS_20261123.md. Không chặn tiếng vọng bằng xóa dữ liệu hoặc lọc mọi từ ngắn.

## Kết quả phát hành

- PR #10, production code `f3c804b7`, deploy run [45199579444](https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45199579444) thành công.
- Runtime enabled=true, directory=/app/storage/audio-captures. So sánh YAML với backup mới nhất: mọi nội dung ngoài audio_capture không đổi; audio_diagnostics giữ enabled=true. Owner/mode main_config: 10001:10001/0600.
- Backend healthy, restart count=0; admin HTTPS 200, verify_tls.py OTA/WSS/HTTP redirect/route protection PASS.
- Probe synthetic silence (không phải giọng người dùng) tại kết nối `20261123T150140Z-4163951130`: 100 mic packets, 46 speaker packets từ thông báo kích hoạt bình thường, close=true, dropped=0, archive JSONL 37.688 bytes. Mic được ghi dù thiết bị test chưa activated, chứng minh capture đặt trước gate chat.
- Export bằng utility trong container: probe-mic.wav 16 kHz mono PCM16/6.000 s; probe-speaker.wav 16 kHz mono PCM16/4.686 s (gồm khoảng trước output theo timeline). File giữ trong thư mục kết nối riêng trên VPS, không commit hoặc công khai.
- SHA256 core: `6807ceaac08192dba7bb53f9740020c138bbe76c4590623771af3505400810ad`; exporter: `223ffc66d6f5213bd37a8bde3abd8e52556fb501963b14828a7e462eb9e5dcf3`.
- Chưa phân tích âm thanh echo thật sau triển khai. Bước tiếp: người dùng thử robot, ghi thời điểm; lấy thư mục của robot theo header/device, export mic/speaker, đối chiếu timestamps và AUDIO_DIAG. Không nhầm dữ liệu probe với tiếng động phòng thật. Không tự xóa dữ liệu.

## Lượt đọc log/audio tiếp theo

Đã có snapshot audio robot thật lúc 22:07–22:08, 823 mic/203 speaker packets và xuất WAV. Phân tích sơ bộ chưa chứng minh nội dung mic trùng lời loa; có tái kích hoạt đáng nghi sau mở mic. Chi tiết và giới hạn ở ECHO_DIAGNOSIS_20261123.md. Không đổi cấu hình/deploy trong lượt kiểm tra này.
