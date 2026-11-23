# Đánh giá robot và PhoWhisper trên VPS — 2026-11-23

## Quyết định hiện tại

Người dùng đã chọn **giữ Gipformer, thử PhoWhisper-small** sau khi xem số đo base. Small đã thử xong tải đơn nhưng chưa tốt hơn Gipformer; không activate PhoWhisper. ASR mặc định xác minh lại qua Manager là `vps_asr` (Gipformer). Sidecar PhoWhisper-small còn sẵn để chẩn đoán, không được chọn làm mặc định.

## Phiên robot thật khoảng 08:04–08:07, giờ Việt Nam

- Đọc được 7 lượt ASR, 6 lượt có metric âm thanh đầu; một lượt bị gián đoạn kết nối và hủy gọi công cụ.
- Gipformer xử lý khoảng 68–262 ms sau khi nhận đủ audio, riêng lượt đầu 912 ms. Thời gian asr_first_text 1–4,7 giây còn chứa thời gian robot thu âm; không coi toàn bộ là inference.
- Từ VAD báo hết câu đến frame TTS đầu: 1,968–7,326 giây trên các lượt hoàn tất, trung vị khoảng 4,208 giây. LLM token đầu 0,633–6,203 giây; TTS frame đầu 0,879–2,934 giây.
- LLM và khởi tạo đoạn TTS chiếm phần lớn độ trễ; đổi ASR không tự giải quyết toàn bộ độ trễ hội thoại. Một số metric llm_total/tts_total còn chịu cách pipeline đợi phát sinh audio, không suy diễn chỉ từ tổng này rằng API LLM tính toán toàn thời gian đó.
- Có WebSocket unexpected EOF lúc 08:05:08, sau đó tool transport context canceled và broken pipe. Có MCP heartbeat timeout sau phiên. Log chưa chứng minh nguyên nhân là Wi-Fi, tắt/khởi động lại robot hay lỗi mạng khác.
- Các đoạn TTS trong cửa sổ quan sát trả HTTP 200, phần lớn RTF dưới 1; các đoạn rất ngắn vẫn chịu chi phí gói đầu. Chưa có bản chép chuẩn của lời người dùng nên không tính WER từ log robot.

## Cách đo ASR

- VPS guest 6 vCPU/12 GB RAM, báo AMD EPYC/AVX2; không thấy AVX-512/VNNI được expose. Không suy ra chính xác CPU host từ tên CPU ảo.
- 30 câu từ Google FLEURS `vi_vn`, split validation, giấy phép CC BY 4.0; tất cả 5 câu trong khoảng 2–5 giây và 25 câu được chọn xác định trong khoảng 5–8 giây, seed 20261123. Đây là mẫu sàng lọc nhỏ, không đại diện mọi giọng/phòng/micro của robot.
- Thêm 20 mẫu tổng hợp zero/white noise/50 Hz hum/impulse. Đây là kiểm tra decoder nhận trực tiếp audio, **không qua VAD của robot**.
- Cùng audio, PCM16 mono 16 kHz; hai PhoWhisper dùng CTranslate2 INT8, 2 threads, 1 worker, quota container 3 vCPU/3 GB. Model nguồn và runtime được ghim trong artifact.
- WER/CER chỉ tính 30 câu có transcript, dùng NFC/chữ thường/bỏ dấu câu/giữ dấu tiếng Việt; CER có khoảng trắng sau chuẩn hóa. p95 bảng dưới tính thời gian từ gửi hết câu đến JSON cuối trên cả 50 mẫu, không phải tổng thời gian hội thoại.

| Model | WER 30 câu | CER 30 câu | p95 tải đơn, 50 mẫu | p95 riêng 5 câu 2–5s | p95 tải 4 yêu cầu |
| --- | --- | --- | --- | --- | --- |
| Gipformer | 4,07% | 1,61% | 0,278–0,313s | 0,282s ở lượt so base | 0,980–1,037s |
| PhoWhisper-base | 10,49% | 5,19% | 3,531s | 1,958s | 9,506s |
| PhoWhisper-small | 7,07% | 3,86% | 5,910s | 4,868s | Chưa đo xong; dừng sau tải đơn |

- Small tốt hơn base về WER nhưng vẫn kém Gipformer, đồng thời chậm hơn rõ. Snapshot RAM khi có tải: Gipformer khoảng 292 MiB, base khoảng 208 MiB, small khoảng 637 MiB; đây không phải phép đo peak chuẩn hóa. PhoWhisper dùng gần 2 lõi CPU trong thử nghiệm.
- Decoder sinh chữ trên 20/20 mẫu nonspeech ở Gipformer và 15/20 ở cả hai PhoWhisper. Không suy ra tỷ lệ robot tự nói tương ứng vì production có VAD phía trước; cũng không coi việc zero-audio trả rỗng là chứng minh chống mọi hallucination. Cần giữ VAD và đo noise thực từ micro nếu điều tra tự nói.
- Bộ kiểm tra đánh dấu đỏ khi nonspeech sinh chữ, dù vẫn lưu report WER/latency. Không gọi đây là crash/hỏng dịch vụ. Lượt tải 2/4 của small được chủ động dừng sau khi tải đơn đã kém ở cả chất lượng và độ trễ; không có số đo hoàn chỉnh để công bố cho hai lượt đó.

## Dấu vết triển khai và báo cáo

- Base trial: https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45092837122 . Helper cài và smoke thành công, nhưng bước probe cuối dùng GET OTA thiếu Device-Id nhận 400; sửa ở PR #3 thành POST thiết bị hợp lệ.
- Base benchmark: https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45093346390 . Report trên VPS: `runtime/asr-benchmark/reports/1795396468772072665/`.
- Small manifest revision `a86b604c346caf7148c37512eafe783a16420adb`, merge PR #4 tại `32ba6f6`.
- Small trial **SUCCESS**: https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45093997463 . Image ID `sha256:732d8bec0c69f8052081ab3050aea6640f8eaec3d4671b2b1463902637147188`; model/speech/silence/TLS/OTA/WSS đều qua.
- Small benchmark: https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45094536526 . Report tải đơn đầy đủ trên VPS: `runtime/asr-benchmark/reports/1795397222100484090/phowhisper-1.json`.
- Chủ động cancel workflow small benchmark và dừng đúng container client benchmark thuộc thử nghiệm; không dừng ASR/TTS/backend. Không dispatch activate hoặc rollback cấu hình; mặc định chưa từng đổi.
- Dataset/audio/transcript/provenance ở `runtime/asr-benchmark/` trên VPS quyền thư mục hạn chế, không đưa audio hoặc weights vào Git. Nguồn: https://huggingface.co/datasets/google/fleurs . Dữ liệu cache local ở /tmp ngoài repository.

## Khuyến nghị bàn giao

Giữ Gipformer trên VPS này. Nếu mục tiêu là giảm lag, bước tiếp theo nên phân tích LLM token đầu và cách chia/đệm TTS; không tự thay provider/model LLM hoặc TTS trong task ASR. Nếu muốn phát hành PhoWhisper sau này, cần bằng chứng mới về chất lượng và tốc độ phù hợp, rồi mới activate bằng GitHub Actions theo quyết định người dùng. Không coi việc trial healthy là đã đạt nghiệm thu.
