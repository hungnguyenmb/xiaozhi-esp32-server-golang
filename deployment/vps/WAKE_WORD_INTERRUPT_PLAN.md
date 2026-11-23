# Điều chỉnh phạm vi: backend trước, giữ firmware — 2026-11-23

Người dùng xác nhận cùng firmware zero-one hoạt động tốt với server Xiaozhi và muốn ưu tiên xử lý Go backend. Đây là ràng buộc hiện hành: **không sửa/build/flash firmware trong đợt xử lý trước mắt**. Các đề xuất firmware ở phần kế hoạch ban đầu bên dưới được hoãn, không còn là điều kiện bắt buộc để triển khai backend.

Rà soát nguồn firmware chỉ cho thấy các điểm cần cân nhắc, không chứng minh phải thay firmware. Luồng hiện có nhận keyword tại thiết bị rồi gửi abort có thể được backend xử lý bằng giao thức hiện hữu. Chưa có trace đối chứng server Xiaozhi nên không khẳng định cơ chế nội bộ của server đó.

Kế hoạch backend đã điều chỉnh:

1. Giữ firmware/model keyword hiện tại; dùng listen/start, abort reason wake_word_detected và tts start/stop hiện có. Xác minh backend hủy generation/LLM/TTS, clear queue và trả stop đúng trên đường keyword abort; không để frame cũ/stop muộn làm hỏng lượt nghe mới.
2. Đối chiếu thời lượng audio, pacing gửi và thời điểm tts_stop. Chỉ sửa nếu có bằng chứng sai thứ tự/hoàn tất quá sớm; không tự tăng độ trễ TTS cho mọi lượt. Server không có phép đo DAC vật lý nếu firmware chưa cung cấp.
3. Phân biệt auto-resume sau TTS với đánh thức/chủ động mở nghe, dựa trên lịch sử sự kiện. Thử gate ngắn trên backend chỉ cho audio auto-resume; dữ liệu capture vẫn giữ đầy đủ. Không áp dụng delay mù cho keyword abort hoặc câu đầu tiên. Không phát lại buffer vừa bị gate bỏ; reset VAD đúng vòng đời.
4. Sửa snapshot voice_ms theo lượt trước reset, gắn với đúng kết quả offline. Kiểm tra các điều kiện chấp nhận đoạn âm thanh đầu lượt bằng dữ liệu thật; không chỉ lọc mọi từ ngắn hoặc cấm “đây”. Đánh giá nguy cơ cắt đầu câu khi dùng guard.
5. Tests/CI và deploy Go qua Actions; thử cùng firmware: chỉ “xin chào” rồi im lặng, nhiễu, keyword khi đang nói, từ ngắn/nói ngay sau robot. Đo false positives, thành công ngắt lời, mất đầu câu và độ trễ.
6. Chỉ quay lại firmware/AEC khi có bằng chứng backend không đủ, giải thích giới hạn và trao đổi phạm vi mới với người dùng trước sửa. Không bảo đảm tuyệt đối rằng backend có thể thay thế AEC phần cứng.

Trạng thái lượt này: chỉ rà soát và chỉnh kế hoạch, chưa code/deploy bản chống false positive.

---

# Kế hoạch ngắt lời bằng từ khóa cục bộ — 2026-11-23

Trạng thái: người dùng yêu cầu kế hoạch, chưa thực hiện code/flash/deploy cho phương án này. Giữ mic phục vụ wake word trên ESP32 khi robot phát loa, không đưa âm thanh thường vào ASR trong giai đoạn phát. Giữ Gipformer/LLM/TTS, tài khoản/thiết bị/WiFi và cơ chế ghi audio hiện hành. Không coi việc đọc firmware workspace là xác nhận firmware đã nạp.

## Bằng chứng đầu vào

Đọc FULL_AUDIO_CAPTURE.md và ECHO_DIAGNOSIS_20261123.md: người dùng chỉ nói “xin chào” ban đầu, các lượt tiếp là false positives. Capture thật cho thấy mic chỉ trở lại sau playback, không có mic packet trong khoảng frame output của năm lượt. Chưa xác định phần nào do vọng/phần nào do noise. ASR voice_ms=0 vì reset trước kết quả offline; phải sửa snapshot turn.

Backend có HandleAbortMessage, HandleListenStart/Detect và generation của hàng đợi TTS để tận dụng. Mã `projects/zero-one/main/application.cc` có OnWakeWordDetected khi Speaking và nhánh bật AFE wake word khi auto; `zero-one-3d` cũng có nhánh tương tự nhưng không mặc định dùng thay thế. Build đang nạp/board/pin mic-reference/model keyword phải kiểm chứng trước code firmware.

## Hành vi đích

- Listening: mic gửi audio cho lượt ASR hợp lệ.
- Processing/Speaking: không gửi mic thường cho ASR; mic vẫn chạy nhận từ khóa cục bộ nếu phần cứng/AFE hỗ trợ.
- Keyword detected trong lúc nói: dừng loa ngay tại thiết bị, bỏ audio phát đang chờ, gửi abort hiện có; server hủy generation cũ, LLM/TTS và không nhận frame cũ trở lại hàng đợi.
- Listening sau interrupt: chỉ nhận command mới, xóa buffer cũ; tránh mất đầu câu nói ngay sau từ khóa. Không áp dụng mù cùng delay 300 ms cho đường đánh thức chủ động.
- TTS kết thúc bình thường: chờ output queue/DAC thực sự hết, xử lý buffer mic/decoder/VAD rồi mới mở lượt nghe auto. Thử guard 300 ms ban đầu, hiệu chỉnh theo dữ liệu; cần đánh giá mất đầu câu và tiếng beep chuyển trạng thái. Server tts_stop đơn thuần không chứng minh loa đã hết tiếng.
- Không hứa ngắt lời bằng mọi câu nói. Từ khóa phải được model wake word trên thiết bị hỗ trợ; nhận một từ bất kỳ bằng ASR server là phạm vi khác.

## Các bước triển khai

### 1. Khóa đúng firmware và đo baseline

Xác minh project/commit/binary/version đang nạp, board/codec, listening mode, wake-word implementation AFE/custom, keyword thực tế và AEC/reference path. Đọc AGENTS/README của repo firmware trước sửa. Ghi log serial về state, keyword callback, output queue/playback done và các lệnh abort/listen; không log secrets. Nếu cần flash mới, thực hiện theo skill đúng thiết bị và giữ WiFi/NVS; chưa flash khi chưa xác định đúng bản/board.

### 2. Sửa backend theo lượt

Tạo nhánh tính năng từ production trong repo Go. Giữ snapshot voice duration/timestamp trước OnVoiceSilence reset, gắn đúng lượt/kết quả offline. Tận dụng cancel/generation hiện có cho abort, xóa input buffer cũ/reset VAD đúng vòng đời; không cho late TTS/frame cũ làm lượt mới. Log turn/reason để chứng minh. Chỉ dùng audio trong cửa sổ nghe hợp lệ cho ASR; capture tiếp tục ghi dữ liệu transport dù bị gate bỏ qua. Không bật bộ lọc để che lỗi thời lượng.

### 3. Hoàn thiện firmware keyword interrupt

Tạo nhánh tương ứng trong repo firmware đã xác minh. Giữ wake-word engine khi Speaking, tắt upload mic thường ở trạng thái này. Khi phát hiện keyword: local stop/clear output ngay, gửi abort, chuyển sang Listening bằng luồng hiện có. Giữ buffer ngắn của command sau keyword nếu cần trong lúc đồng bộ, nhưng không tái gửi wake audio/tiếng loa cũ vào command. Kiểm tra không nhân đôi listen/start và frame generation cũ. Không tự thêm giao thức stop-mic nếu luồng hiện tại đã đáp ứng; chỉ bổ sung xác nhận playback nếu thiếu và có bằng chứng.

### 4. Sửa chuyển Speaking→Listening tự động

Xác định mốc audio queue/DAC hết bằng firmware, không dựa riêng server stop. Bỏ buffer tồn và xử lý VAD/audio processing trước mở lượt; guard ban đầu 300 ms chỉ cho auto-resume, giữ nguyên dữ liệu capture để đánh giá phần bị bỏ. Xem cả tiếng thông báo/beep lúc mở mic. Tách đường keyword interrupt để tránh làm chậm hoặc cắt command. AEC chỉ bật khi reference path thật sự hỗ trợ và được kiểm tra; không chỉ đổi cờ build để tuyên bố đã khử vọng.

### 5. Kiểm thử và nghiệm thu

Go tests: snapshot sống qua reset và không lẫn turn; abort hủy generation, frame muộn bị loại, restart/close không kẹt; capture vẫn hoạt động trước gate. Race test phần thay đổi, vet/build, CI Linux. Firmware build đúng target/config và kiểm tra state/queue trên thiết bị.

Mục tiêu thử robot (ghi rõ điều kiện, không coi đây là kết quả đã đạt):

- Một câu “xin chào”, sau đó im lặng: ít nhất 30 chu kỳ phát/kết thúc thử, không phát sinh thêm lượt ASR/LLM ngoài chủ ý.
- Nhiễu phòng và hai mức âm lượng loa: không tự kích hoạt; kiểm thử cả câu TTS có chứa từ khóa để phát hiện tự đánh thức.
- Keyword khi Speaking: ít nhất 19/20 lần thành công trong khoảng cách/âm lượng đã thống nhất; đo local stop từ callback keyword, mục tiêu p95 dưới 300 ms (không tính từ đầu người dùng bắt đầu đọc keyword).
- Nói ngay sau keyword và ngay sau robot: không mất đầu câu, giữ “dừng”, “không”, “vâng”. Nếu guard làm mất lời thật thì điều chỉnh trước phát hành rộng.
- Lặp abort nhanh, mất mạng/reconnect, output đến muộn, hai robot độc lập; không treo state hoặc phát lại câu cũ.

### 6. Phát hành, rollback và bàn giao

Backend chỉ deploy qua GitHub Actions từ production sau CI. Firmware phát hành trên đúng repo/board theo quy trình riêng, ghi checksum/config/model keyword; thử trên robot hiện tại trước mở rộng, giữ bản binary cũ để quay lui, không xóa WiFi/NVS ngoài yêu cầu. Kiểm tra tương thích thứ tự cập nhật hai phía, ưu tiên tận dụng abort/listen hiện có. Rollback backend bằng Actions và firmware bằng bản đã lưu theo quy trình tương ứng; không xóa recordings. Ghi commit/run/firmware hash, số đo đạt/chưa đạt vào log, task và kế hoạch.

## Checklist bàn giao

- [x] Lưu kế hoạch và liên kết trong AGENTS/implementation_plan.
- [ ] Xác minh firmware thực tế và baseline serial.
- [ ] Snapshot/abort/gating backend và tests.
- [ ] Firmware wake-word interrupt + playback transition.
- [ ] CI/build, test robot A/B và số đo nghiệm thu.
- [ ] Deploy/flash có kiểm soát, kiểm tra lại và tài liệu rollback.

## Xác nhận firmware zero-one và rà soát nguồn

Người dùng xác nhận robot đã nạp firmware của project zero-one. Đọc nguồn hiện tại trong projects/zero-one (chưa đối chiếu checksum binary thực tế):

- sdkconfig bật CONFIG_USE_AFE_WAKE_WORD và CONFIG_USE_AUDIO_PROCESSOR. Speaking ở auto mode đã tắt voice processing và bật wake word; OnWakeWordDetected khi Speaking đã gọi AbortSpeaking với reason wake_word_detected. Không cần viết lại bộ nhận keyword.
- AbortSpeaking chỉ đặt aborted_=true và gửi abort; không dừng/clear loa cục bộ. OnIncomingAudio chỉ kiểm tra state Speaking, không kiểm tra aborted_, nên vẫn có thể nhận thêm frame trong lúc đợi server. Cần local stop/clear queue và bỏ frame cũ trong giai đoạn abort, sau đó chuyển nghe có kiểm soát.
- Khi nhận tts stop, callback chuyển Speaking→Listening ngay. EnableVoiceProcessing(true) gọi ResetDecoder để xóa hàng đợi output, nhưng không phải xác nhận playback tự nhiên đã phát hết. Cần phân biệt kết thúc bình thường (drain đúng) với ngắt chủ động (clear ngay).
- Input có warm-up vTaskDelay 120 ms; đây là chờ, không đọc/bỏ PCM/DMA tồn hoặc đảm bảo không có tiếng đuôi loa. Không được nói firmware hoàn toàn chưa có warm-up; cần thay/hoàn thiện flush/guard dựa vào mốc playback thực.
- MAIN_EVENT_SEND_AUDIO lấy mọi packet trong send queue và gửi nếu có protocol, không gate theo state. Cần đánh giá encode đang chạy/packet tồn sau chuyển trạng thái; bảo đảm không gửi frame phiên cũ.
- AudioService::IsIdle chỉ kiểm tra bốn queue, không có cờ output đang xử lý; cần xác định output in-flight/DAC drain nếu dùng làm mốc hoàn tất, không chỉ chờ queue.empty().

Kết luận: cần sửa firmware tối thiểu cho abort cục bộ và chuyển phát→nghe, đồng thời sửa snapshot/gating ở Go. Chưa thay code zero-one, chưa build/flash/deploy trong lượt rà soát này. Giữ WiFi/NVS khi phát hành sau.
