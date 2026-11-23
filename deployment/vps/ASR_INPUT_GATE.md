# Backend ASR input gate — 2026-11-24

## Phạm vi và trạng thái

Người dùng đồng ý triển khai phương án backend, giữ nguyên firmware zero-one. Code trên `fix/asr-post-playback-gate`; chưa deploy ở thời điểm viết mục này. Giữ Gipformer/LLM/TTS, keyword tại ESP32 và audio_capture/audio_diagnostics hiện hữu. Đọc WAKE_WORD_INTERRUPT_PLAN.md và ECHO_DIAGNOSIS_20261123.md để biết baseline: người dùng chỉ nói “xin chào” ban đầu, các lượt sau là false positives.

## Hành vi

- Auto: chốt tiếng nói/đưa yêu cầu vào LLM hoặc enqueue TTS thì chặn audio vào VAD/ASR. Các gói vẫn được archive trước gate ở ServerTransport; lệnh điều khiển không bị chặn.
- Mỗi listen có epoch; frame queue gắn epoch, begin-listen drain frame cũ, worker kiểm tra lại trước VAD và trước ghi ASR. Ghi ASR cuối cùng tuyến tính hóa với Block/BeginListen. ASR khởi tạo chưa xong thì giữ frame trong queue/đợi Ready; tránh làm rơi câu đầu chỉ vì provider chưa sẵn sàng.
- Sau TTS kết thúc bình thường, nếu client auto-listen lại trong 2 giây thì đánh dấu guard cho các frame đầu. Guard mặc định 300 ms tính từ packet đầu tới ở lượt mới; chỉ khởi động khi packet đó tới trong 1 giây từ listen/start, tránh cắt câu nói thật rất muộn nếu client không gửi nền liên tục.
- Gói trong guard vẫn được giải mã để prime trạng thái Opus, nhưng không chạy VAD/ASR hoặc vào lịch sử nhận dạng. Khi qua guard, VAD mới lấy/reset theo epoch; không replay gói đã bỏ vào ASR. Decoder cũng được tạo lại khi epoch đổi để tránh mang lịch sử lượt cũ.
- Abort từ keyword hoặc nút bấm là chủ động: bỏ guard, dùng luồng hủy LLM/TTS/generation sẵn có. Tín hiệu listen/detect cũng đánh dấu chủ động. Dấu ưu tiên được giữ tối đa 2 giây, dùng một lần khi begin-listen. Không thay protocol firmware.
- Manual/realtime không dùng busy/settling policy mới; epoch vẫn ngăn frame auto cũ lọt khi đổi mode. Zero value gate trong test/client cũ giữ hành vi legacy.
- Pacing TTS hiện đã đợi playbackTail và grace trước stop. Không đổi timing này vì chưa có bằng chứng cần tăng delay TTS cho mọi lượt.

## Snapshot và concurrency

ASRInputTurn riêng cho mỗi recognition stream, lưu voice_ms/idle_ms/end timestamp trước OnVoiceSilence gọi VAD.Reset. Snapshot chỉ chốt một lần; final offline và hook dùng snapshot cũ, không đọc lại counter đã reset hoặc counter lượt sau. Log result.asr_ms tính từ snapshot end tới final, tránh lấy trước MarkAsrFinalTextAt rồi thành 0. Provider auto-end có thể không có endpoint riêng, lúc đó end lấy tại final nên asr_ms=0 là hợp lệ, không phải tổng inference.

ASR loop giữ ownership token; kết quả sau khi context hủy/token đổi không tiếp tục xử lý, loop cũ không release resource của stream mới. Các restart nội bộ cập nhật token sở hữu. Cờ IsTtsStart đổi sang atomic.Bool qua accessor hiện có để tránh race SendTtsStart/abort debugState mà test mới phát hiện.

## Cấu hình và rollback

```yaml
chat:
  asr_input_gate:
    enabled: true
    post_tts_guard_ms: 300
```

Khi thiếu cấu hình: mặc định enabled=true, guard=300 ms. Giá trị guard đọc lúc tạo ChatSession, giới hạn 0..1000 ms; 0 tắt settling nhưng giữ busy/epoch gate. Restart backend để áp dụng chắc chắn cho mọi phiên.

Actions **Deploy Backend & Core Services**, ref production:

- `asr_input_gate=enabled`, `post_tts_guard_ms=300`: bật và ghi giá trị guard được chọn vào runtime YAML.
- `asr_input_gate=disabled`: tắt policy mới; không thay các mục audio/ASR khác, snapshot/concurrency fixes vẫn còn trong binary.
- `asr_input_gate=unchanged` (mặc định): không đổi runtime config; guard input bị bỏ qua.

Input audio_diagnostics/audio_capture để unchanged khi chỉ điều chỉnh gate. Script backup YAML, giữ uid/gid/mode; health failure rollback binary/config. Không deploy qua SSH để bỏ pipeline. Nếu cần rollback toàn bộ code thì dùng bản phát hành trước theo quy trình Actions, không xóa recordings.

## Kiểm thử và giới hạn

Đã qua local macOS native theo môi trường PR271_INTEGRATION.md:

- Full tests client/chat/hooks/capture; race chọn lọc AudioInputGate, ASRInputTurn, AudioCapture, AudioDiagnostics và keyword-abort TTS integration.
- Gate: lượt đầu, busy, natural auto-resume, giữ packet decode-only trong guard, bypass wake, stale Ready/frame/queue, đợi ASR init, Block giải phóng waiter, manual/realtime/disabled, câu tới muộn, concurrent lifecycle.
- Snapshot: giữ 420 ms voice/540 ms idle qua reset, không bị lần seal zero ghi đè, turn mới độc lập.
- Keyword abort thực sự dừng active TTS và đổi generation; kể cả một stop muộn vẫn không guard command mới. Capture test chứng minh audio busy vẫn archive nhưng không vào queue ASR.
- Vet chat/cmd đạt; client vet với lostcancel=false đạt. Vet đầy đủ client còn cảnh báo InitLlm context leak có sẵn, không sửa trong phạm vi này. Race toàn chat còn MCP reinit đã ghi từ trước, không tuyên bố toàn repo sạch race.
- Workflow qua actionlint/bash -n, mock deploy success/health rollback giữ nguyên YAML còn lại/owner/mode. CI Linux và robot thật còn chờ.

Có dò offline dữ liệu cũ bằng native TEN ở Mac (hop512/threshold0.4) và PCM đã decode, nhưng mô phỏng chưa tái hiện đầy đủ vòng đời pool/endpoint/log thực. Guard đơn thuần không phải bộ phân loại tiếng động hoàn chỉnh; không coi phép dò đó là benchmark đạt hoặc bằng chứng đã hết false positives. Chưa đổi ngưỡng VAD, chưa bật noise filter runtime để che lỗi. Cần đo robot thật sau deploy, nhất là câu nói ngay sau robot để kiểm tra mất đầu câu.

## Log và nghiệm thu

`ASR_INPUT_GATE listen` ghi epoch/guarded/mode; `settling` đánh dấu packet đầu guard; `explicit_abort` ghi keyword boolean. Đối chiếu AUDIO_DIAG endpoint/asr: voice_ms không còn mặc định mất về 0 trên lượt offline có endpoint. Capture giữ dữ liệu mic/output đầy đủ như trước, không tự xóa.

Sau CI/release: kiểm tra config enabled/guard và các mục khác không đổi, container/HTTPS/OTA/WSS, capture hoạt động. Robot thật cần một câu rồi im lặng, nhiễu phòng, wake keyword giữa TTS, câu ngắn và câu ngay sau TTS; ghi thời điểm để đối chiếu. Đây là bước nghiệm thu chưa hoàn tất, không khẳng định đã loại mọi tiếng vọng.

Bổ sung rà soát: bắt đầu epoch mới sau khi đã đồng bộ stop output cũ. Sau speaker/hook có kiểm tra context/token trước side effects; chuyển kết quả vào LLM giữ source context/input token và publish dưới stopSpeakingMu, tránh gắn câu cũ vào SessionCtx mới. Test context hủy và token bị thay đều không tạo item LLM; release ownership kiểm tra trong cùng resourceMu với slot provider.

Rà soát vòng đời phiên: Configure ở phiên mới vô hiệu epoch/trạng thái nghe cũ; Close/ResetToSilentState đóng gate. Full tests/race chọn lọc sau cập nhật đạt, có test reconfigure không chấp nhận audio từ phiên trước.

Rà soát trước phát hành phát hiện ASR trả rỗng không có TTS/client listen-start để mở gate. Đã dừng build trước deploy và thêm restartASRInputForRetry: auto retry tự tạo epoch/đợi provider/Ready, không chờ playback; lỗi restart giữ gate đóng. Test empty retry không TTS và failure path là bắt buộc. Không thay model hoặc firmware.
