# Chẩn đoán mic / VAD / ASR — 2026-11-23

## Trạng thái bàn giao

Đã code trên nhánh `feat/vad-noise-filter-271`, cùng phần tích hợp PR271 chưa commit/push. Chưa deploy hoặc bật trên VPS, chưa test robot thật. Giữ Gipformer và cấu hình nhận dạng hiện tại. Không tự nâng ngưỡng dựa trên dữ liệu giả lập.

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
