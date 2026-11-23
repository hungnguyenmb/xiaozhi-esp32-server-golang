# Chẩn đoán mic / VAD / ASR — 2026-11-23

## Trạng thái bàn giao

Đã merge PR #6 và bản sửa script PR #7/#8 vào production. CI Linux tests/vet/build và frontend đạt tại Actions run 45191130321. Đã deploy và bật chẩn đoán qua Actions run 45194051848 thành công; chưa nghiệm thu robot thật. Giữ Gipformer và cấu hình nhận dạng hiện tại. Không tự nâng ngưỡng dựa trên dữ liệu giả lập.

## Bật và tắt

Trong cấu hình runtime backend (không phải cấu hình frontend), bổ sung chọn lọc:

```yaml
chat:
  audio_diagnostics:
    enabled: true
    interval_ms: 1000
```

Giữ mọi trường `chat` hiện hữu, không thay toàn bộ YAML. Triển khai/code/config bằng quy trình GitHub Actions trong DEPLOYMENT_GUIDE.md; restart backend để áp dụng. Mặc định false, interval mặc định 1000 ms và giới hạn 500–10000 ms. Bật trước một buổi thử có kiểm soát; tắt lại bằng enabled=false sau khi thu đủ dữ liệu. Không thêm audio/transcript vào Git. Nhật ký dùng mức INFO, tìm chuỗi `AUDIO_DIAG`; log cũ của hệ thống có thể vẫn chứa transcript ở DEBUG, tính năng mới không loại bỏ log cũ.

## Dữ liệu ghi

- `setup`: provider, need_vad, listen_mode, asr_auto_end và realtime_mode để biết VAD có bị bỏ qua hay không.
- `window`: device/session/provider, wall_ms (thời gian thực), audio_ms (tổng thời lượng frame), frames, voice_frames (quyết định cuối luồng, có thể từ client/bypass), vad_calls, power_skips (tiền lọc công suất Silero).
- RMS trên mẫu PCM float đã giải mã Opus và peak trị tuyệt đối; không phải dB SPL đã hiệu chuẩn. So sánh trên cùng mic/gain. Giá trị gần 1 có thể chỉ ra tín hiệu mạnh; cần audio thực để kết luận clipping.
- score_samples/min/mean/max: xác suất của **cửa sổ suy luận cuối mỗi lần gọi** TEN/Silero, không phải mọi cửa sổ và không phải confidence ASR. score_samples=0 nghĩa là không có dữ liệu, không có nghĩa xác suất bằng 0. Provider khác có thể không hỗ trợ. Không dùng min/max này như quyết định cuối cho cả chunk vì chunk có thể chứa nhiều cửa sổ.
- `endpoint`: silence_end khi backend quyết định kết thúc theo im lặng, short_voice_rejected khi đoạn có tiếng nói dưới 100 ms bị logic hiện hữu loại; voice_ms/idle_ms là số liệu trạng thái tại thời điểm đó.
- `asr`: result (kể cả rỗng), text_chars (số Unicode rune), voice_ms, asr_ms theo GetAsrDuration hiện hữu, idle_ms; sau hook có hook_stop/error. asr_ms không phải phép đo latency thuần của riêng model, cũng không phải tổng thời gian đến âm thanh trả lời.

Cửa sổ flush khi đủ thời gian **và có frame mới**, lúc endpoint hoặc audio processing thoát; không có timer riêng khi mic không gửi. Một frame có thể chứa phần trước lời nói; RMS tính từ frame giải mã gốc. Không ghi từng frame, không ghi audio/raw PCM, transcript, API key. Định danh device/session vẫn là metadata thiết bị nên giới hạn người truy cập và áp dụng rotation/retention của log container hiện hữu.

## Buổi đo và cách suy luận

Ghi chú thời gian/robot cho từng tình huống khoảng 20–30 giây:

1. Phòng yên tĩnh, không nói.
2. Quạt, gõ bàn, di chuyển đồ vật, không nói.
3. Robot phát loa, người dùng im lặng.
4. Nói thật ở khoảng cách sử dụng: câu đầy đủ và “dừng”, “không”, “vâng”; nghỉ giữa câu rồi nói tiếp.

Đối chiếu device/session và timestamp (không có turn ID mới). RMS cao chưa chắc là lời nói. Nếu không nói mà voice_frames/score tăng và ASR sinh chữ, nghi VAD nhận nhầm; thử ngưỡng bằng dữ liệu nhiều tình huống, không chặn mọi từ ngắn. Nếu chỉ sai khi loa phát, kiểm tra AEC/đường âm thanh firmware. Nếu VAD ít tiếng nói nhưng ASR vẫn có chữ, kiểm tra chế độ bỏ qua VAD/client/provider auto-end và đầu vào ASR. silence_end quá sớm so với câu thật thì xem ngưỡng im lặng/hysteresis; không giảm timeout để che độ trễ model. Dữ liệu này chỉ hỗ trợ khoanh vùng, chưa đủ phân biệt chính xác tiếng gõ với giọng người.

## File và kiểm chứng

- `internal/app/server/chat/audio_diagnostics.go`: thống kê theo goroutine xử lý audio, có gate enabled.
- `asr.go`: móc sau VAD/trước cập nhật trạng thái, endpoint và nhận kết quả/hook.
- `ten_vad/vad.go`, `silero_vad/vad.go`: accessor có mutex cho xác suất gần nhất; không thay quyết định VAD.
- `audio_diagnostics_test.go`: RMS/peak/counters/reset, disabled, provider không có score.
- `config/config.yaml`: ví dụ tắt mặc định.

Go tests chat/TEN/Silero/hooks đã qua trên macOS native theo môi trường PR271_INTEGRATION.md. CI Linux/robot thật và hiệu quả giảm nhiễu chưa được xác minh. Không có test frontend mới vì không đổi UI. Bước tiếp: review diff, commit/push chạy CI Linux; deploy qua Actions khi có yêu cầu, bật log thử, thu kết quả bốn tình huống, ghi nhận số đo rồi mới hiệu chỉnh. Việc thu audio ngắn chưa triển khai trong lượt này.

Kiểm tra race toàn package chat phát hiện race ở test MCP hiện hữu: `waitForMcpInitState` (mcp_reinit_test.go:124) đọc trạng thái trong khi `finishMcpInit` (chat.go:589) ghi. Không sửa ngoài phạm vi; không được báo toàn package chat đã sạch race. Race tests TEN/Silero/hooks qua; race test riêng `TestAudioDiagnostics` đã qua. Vet các package thay đổi và build cmd/server cũng đã qua (macOS; có cảnh báo linker native hiện hữu).

## Phát hành 2026-11-23

- Production code: `c63cc55e` (PR #6); sửa shell deploy: `0300aba0` (PR #7), giữ owner config: `2e685d3a` (PR #8).
- Frontend: Actions 45191741004 thành công.
- Backend bật chẩn đoán: Actions 45194051848 thành công, code production 2e685d3a. Backend healthy, restart count=0; admin HTTPS 200, TLS/OTA/WSS hello và bảo vệ route qua verify_tls.py đều PASS.
- Lượt 45191781529 thất bại vì thiếu fi trước khi thay binary/config; bản sửa đã qua bash -n và actionlint. Script cập nhật YAML bật/tắt đã thử với dữ liệu giả, giữ nguyên mọi trường ngoài audio_diagnostics.
- Backend cũ trước phát hành: SHA256 `2fffce649a031dd6413be4c4822561b9cfb84a10e5772b8166cd236dce32037b`, Docker image `sha256:d4b73de66d6e3be379c1a1e5a14c15259822d4d4acef6bb94a3462f6cf31e41d`. Không prune image cũ trước nghiệm thu.
- Chỉ bật audio_diagnostics, không tự bật asr_noise_filter trên YAML cũ. Ngưỡng kết thúc câu thực tế trước phát hành là 200 ms, realtime_mode=4; không đổi các giá trị này.
- Tắt log qua **Deploy Backend & Core Services**, ref production, input audio_diagnostics=disabled. Mặc định unchanged không đổi trạng thái runtime. Pipeline chỉ chạy từ production, sao lưu main_config.yaml trước thay; health check thất bại khôi phục binary và phần cấu hình đã sao lưu. Lưu ý lỗi build/up sớm cần kiểm tra thủ công trạng thái trước chạy lại; không tự coi health rollback bao phủ mọi lỗi shell.

Run 45192967450 health check thất bại do file YAML mới thuộc root; rollback trả backend cũ healthy. PR #8 giữ owner UID/GID 10001 và quyền file trước atomic replace. Đã thử cả script thành công/health rollback bằng commands giả lập trên thư mục tạm (không thay VPS).

### Xác minh sau deploy

- Binary mới SHA256: `44a75d2f3eaf965dfddd21305172e9bf56825368bb01a681f200979aa199c0b6`.
- Runtime `audio_diagnostics={enabled: true, interval_ms: 1000}`, owner config `10001:10001`, mode `0600`.
- So sánh YAML với `main_config.yaml.diagnostics-20261123T142924Z.bak`: mọi trường ngoài audio_diagnostics giống hệt; silence=200 ms và realtime_mode=4 giữ nguyên. Không thay DB/ASR/LLM/TTS.
- Log `AUDIO_DIAG setup` đã xuất hiện qua thiết bị smoke test, TEN VAD, need_vad=true, auto_end=false.
- Chưa có kết quả đo mic/loa thật. Người dùng cần thử bốn tình huống đã liệt kê trước khi chỉnh ngưỡng; giữ log tạm thời rồi tắt qua Actions.

Smoke probe gửi Opus im lặng tổng hợp qua kết nối test và xác nhận setup/HTTPS/WSS; chưa quan sát được window/endpoint trong phiên giả lập này. Không coi phép thử đó là xác minh đủ luồng mic/ASR: cần robot thật vào phiên nghe để thu AUDIO_DIAG window/asr. Không lưu audio cá nhân.
