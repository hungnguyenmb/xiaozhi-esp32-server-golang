# Backend ASR input gate — 2026-11-24

## Phạm vi và trạng thái

Người dùng đồng ý triển khai phương án backend, giữ nguyên firmware zero-one. Code từ `fix/asr-post-playback-gate` đã merge qua PR #14/#15, production `bad3c0d9`; đã deploy và bật qua Actions 45226377103 thành công. Giữ Gipformer/LLM/TTS, keyword tại ESP32 và audio_capture/audio_diagnostics hiện hữu. Đọc WAKE_WORD_INTERRUPT_PLAN.md và ECHO_DIAGNOSIS_20261123.md để biết baseline: người dùng chỉ nói “xin chào” ban đầu, các lượt sau là false positives.

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
- Workflow qua actionlint/bash -n, mock deploy success/health rollback giữ nguyên YAML còn lại/owner/mode. CI Linux 45226007130 đã qua, gồm bản sửa empty retry; robot thật sau rollout còn chờ.

Có dò offline dữ liệu cũ bằng native TEN ở Mac (hop512/threshold0.4) và PCM đã decode, nhưng mô phỏng chưa tái hiện đầy đủ vòng đời pool/endpoint/log thực. Guard đơn thuần không phải bộ phân loại tiếng động hoàn chỉnh; không coi phép dò đó là benchmark đạt hoặc bằng chứng đã hết false positives. Chưa đổi ngưỡng VAD, chưa bật noise filter runtime để che lỗi. Cần đo robot thật sau deploy, nhất là câu nói ngay sau robot để kiểm tra mất đầu câu.

## Log và nghiệm thu

`ASR_INPUT_GATE listen` ghi epoch/guarded/mode; `settling` đánh dấu packet đầu guard; `explicit_abort` ghi keyword boolean. Đối chiếu AUDIO_DIAG endpoint/asr: voice_ms không còn mặc định mất về 0 trên lượt offline có endpoint. Capture giữ dữ liệu mic/output đầy đủ như trước, không tự xóa.

Sau CI/release: kiểm tra config enabled/guard và các mục khác không đổi, container/HTTPS/OTA/WSS, capture hoạt động. Robot thật cần một câu rồi im lặng, nhiễu phòng, wake keyword giữa TTS, câu ngắn và câu ngay sau TTS; ghi thời điểm để đối chiếu. Đây là bước nghiệm thu chưa hoàn tất, không khẳng định đã loại mọi tiếng vọng.

Bổ sung rà soát: bắt đầu epoch mới sau khi đã đồng bộ stop output cũ. Sau speaker/hook có kiểm tra context/token trước side effects; chuyển kết quả vào LLM giữ source context/input token và publish dưới stopSpeakingMu, tránh gắn câu cũ vào SessionCtx mới. Test context hủy và token bị thay đều không tạo item LLM; release ownership kiểm tra trong cùng resourceMu với slot provider.

Rà soát vòng đời phiên: Configure ở phiên mới vô hiệu epoch/trạng thái nghe cũ; Close/ResetToSilentState đóng gate. Full tests/race chọn lọc sau cập nhật đạt, có test reconfigure không chấp nhận audio từ phiên trước.

Rà soát trước phát hành phát hiện ASR trả rỗng không có TTS/client listen-start để mở gate. Đã dừng build trước deploy và thêm restartASRInputForRetry: auto retry tự tạo epoch/đợi provider/Ready, không chờ playback; lỗi restart giữ gate đóng. Test empty retry không TTS và failure path là bắt buộc. Không thay model hoặc firmware.

## Lượt phát hành 2026-11-24

- PR #14: gate/epoch/snapshot và handoff theo source context. CI Linux 45224625088 đạt.
- PR #15: tự mở lại auto ASR sau kết quả rỗng/transient, không chờ TTS; CI Linux 45226007130 đạt.
- Build 45225225393 bị hủy trước deploy khi rà soát empty retry; binary VPS cũ vẫn SHA256 `6807ceaac08192dba7bb53f9740020c138bbe76c4590623771af3505400810ad`, image `sha256:d4055b867d2df6a5a8d478f7cfb67de11f68a4c7aad4c61dbec66836cfafe30f`, healthy/restarts=0. Không coi PR #14 riêng lẻ là bản đã nghiệm thu.
- Release hiện hành: Actions 45226377103, ref production bad3c0d9, asr_input_gate=enabled/post_tts_guard_ms=300, capture/diagnostics unchanged. VPS đã xác minh thành công như mục bên dưới.

## Xác minh VPS sau phát hành

- Actions [45226377103](https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45226377103) thành công; code production bad3c0d9 (gồm PR14 và PR15).
- Core SHA256: `10ce67289f12961eda3f85bbb47e6feb4dbd8b800ec0bc4b4bb2546192ac717e`.
- Runtime `asr_input_gate={enabled: true, post_tts_guard_ms: 300}`; so YAML với backup `main_config.yaml.diagnostics-20261123T172431Z.bak` xác nhận mọi mục ngoài gate không đổi. Config vẫn owner 10001:10001/mode0600. Capture và diagnostics giữ enabled=true.
- Backend healthy/restarts=0; ASR Gipformer/TTS/gateway không restart. Admin HTTPS 200; TLS/OTA/WSS hello, route protection và HTTP redirect qua probe PASS.
- Probe `20261123T172523Z-2969557647` gửi synthetic silence trên thiết bị test chưa activated: archive có 100 mic+47 speaker packets, closed=true/dropped=0. `ASR_INPUT_GATE explicit_abort ... wake_word=true` xuất hiện. Probe này xác minh wire/control/capture và health, không xác minh VAD/ASR của robot activated hay khả năng nhận keyword vật lý. Trường device trong log abort dùng msg.DeviceID nên có thể trống nếu client không gửi; đối chiếu thêm session ở log command/capture.
- Chưa có lượt robot thật sau deploy để kết luận đã hết tự nói. Người dùng cần thử một câu rồi im lặng, keyword giữa TTS, câu ngắn và nói ngay sau robot; lấy timestamp đối chiếu log/capture. Guard chỉ xử lý vùng chuyển lượt, không thay thế khử vọng/phân loại tiếng động hoàn chỉnh.

## Kiểm tra robot thật 2026-11-24, 08:44–08:48 (UTC+7)

Người dùng báo robot vẫn tự trả lời. Đã đối chiếu Docker logs, chat_messages read-only và snapshot audio riêng; không sửa runtime/model/firmware trong lần kiểm tra này. Gate enabled/guard300 ms, backend healthy/restarts=0. Capture snapshot có 2254 records, 2208 audio packets, không thấy sequence gap trong phần đã lấy; không suy ra toàn phiên đã đóng hoặc mic phần cứng được ghi 24/7. Không đưa audio/transcript cá nhân lên Git.

- Lượt đầu endpoint 08:44:41.201: voice1080 ms/idle540 ms, kết quả ASR 08:44:42.184, asr983 ms. Snapshot duration đã hoạt động, không còn voice_ms=0 sau reset.
- TTS stop08:44:51.165 → listen08:44:51.243 guarded=true → settling08:44:51.396. Endpoint kế tiếp08:45:01.058, voice480 ms, ASR272 ms. Nhận nhầm xuất hiện gần10 giây sau stop, ngoài vùng guard. Nhiều lượt sau cũng có guarded=true rồi nhận nền thành từ ngắn và tạo phản hồi.
- Câu chào có VAD max~0.9169/RMS~0.163 ở cửa sổ lời nói. Các cửa sổ nền điển hình max~0.47–0.55/RMS~0.01–0.016, nhưng threshold runtime0.4 vẫn chấp nhận. Đây là dấu hiệu false positive ở VAD→ASR sau auto-resume; chưa đủ chứng cứ quy toàn bộ thành echo loa.
- Dò offline bằng native TEN macOS/hop512 trên Opus capture: ở cửa sổ đầu, threshold0.4 cho1140 ms positive (log1080), threshold0.6 cho720 ms/max liên tiếp660. Hai cửa sổ nền đầu giảm780/2820 ms về0 ở0.6. Một số cửa sổ khác vẫn còn60–120 ms. Replay khác runtime ở reset/endpoint/pool/timing và không chạy đầy đủ quyết định endpoint; các số này không phải tỉ lệ lọc thành công hay benchmark end-to-end.

Kết luận: gate xử lý chuyển lượt nhưng chưa giải quyết tiếng động bị nhận là lời nói sau khi nghe lại. Không chỉ tăng guard và không chặn cứng các từ ngắn vì có thể là lệnh thật. Bước tiếp theo là thử điều chỉnh VAD có kiểm soát (điểm bắt đầu/kết thúc và điều kiện tiếng nói ổn định), replay đầy đủ trước khi deploy qua Actions, rồi nghiệm thu câu ngắn/lời nhỏ/xa mic/nền/keyword ngắt lời. Chưa đổi threshold và chưa bật noise filter runtime. Hiệu quả chống tự trả lời: chưa đạt.
