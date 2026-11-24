# Đã phát hành sửa lịch sử tool/error speech — 2026-11-24

PR19 đã merge production27e0b975, Actions45322232698 deploy thành công. Backend healthy/restarts0; HTTP/native/TLS/OTA/WSS/admin PASS. Cấu hình SHA672a0db8… không đổi, noise=enforce/gate300 ms/capture/diagnostics giữ nguyên. CI Linux push45321517370 và PR45321746615 đạt. Chờ người dùng thử robot thật; false positives âm thanh chưa thuộc bản sửa này. Đọc TOOL_HISTORY_ERROR_FIX.md để tiếp tục và xem giới hạn kiểm chứng; các trạng thái chờ phát hành phía dưới là lịch sử.

# Sửa lịch sử tool và chặn lỗi kỹ thuật ra loa — 2026-11-24

Người dùng đã yêu cầu sửa, merge/deploy qua GitHub Actions để thử lại. Nhánh fix/tool-history-error-speech từ production9e7f3ee9, mang theo báo cáo NOISE_ROBOT_TEST_20261124.md của nhánh tài liệu. Không đổi noise frontend/ngưỡng/enforce, firmware, model hoặc dữ liệu thiết bị; không xóa lịch sử/ghi âm.

## Kế hoạch và tiêu chí kiểm chứng

1. Viết regression trước sửa: assistant2 calls chỉ xuất hiện1 lần; lịch sử cũ có adjacent duplicate được phục hồi; partial/orphan/truncated/duplicate/out-of-order tool results không tạo request sai. Chuẩn hóa theo nhóm liền kề, không lấy kết quả qua user/round khác, không mutate message/slice gốc, idempotent. Không chạy lại tool có side effect.
2. Chặn provider Error message ở boundary trước text/hook/TTS. Log kỹ thuật riêng; kết thúc bằng empty end để giải phóng pipeline/turn, giữ phần lời hợp lệ trước lỗi nếu có. Không phát raw error hoặc lưu thành assistant. Kiểm thử bằng HTTP provider giả lập400 trong cảstream/nonstream, TTS queue/history trống và lượt sau vẫn trả bình thường.
3. Kiểm tra ID lưu lịch sử: role+timestamp millisecond có thể trùng khi lưu2 tool cùng vòng; nếu regression xác nhận, chuyển ID mới sang UUID32 ký tự, giữ tương thích hai giai đoạn/audio và không sửa ID bản ghi cũ.
4. Chạy client/chat tests, race chọn lọc, vet theo giới hạn đã biết, CI Linux. Cập nhật task/BUG_LOG/DEPLOYMENT_LOG và hướng dẫn.
5. Merge production sau CI, deploy Actions với mọi audio input unchanged; xác minh SHA/config (enforce/capture/diagnostics/gate không đổi), native/health/TLS/OTA/WSS. Phân biệt nghiệm thu backend và test robot thật. Log run IDs/rollback để agent khác tiếp tục.

Trạng thái: code/regression local đã có, chuẩn bị CI Linux; chưa merge/deploy trong đợt này. False positive tiếng động thành từ ngắn vẫn là công việc kế tiếp, không tuyên bố đã giải quyết bằng bản sửa tool.

## Đã triển khai và xác minh cục bộ

- Regression trước sửa tái hiện assistant bị nhân đôi, partial/orphan sai thứ tự, lỗi400 đi TTS ở cảstream/nonstream, và20 kết quả tool chỉ có1 ID duy nhất khi ghi cùng millisecond.
- AlignToolMessages ghép từng round assistant/tool liền kề; collapse adjacent legacy duplicate có cùng nội dung/calls, giữ mỗi ID đã có kết quả1 lần và xếp responses theo calls. Drop call thiếu phản hồi/orphan trong view, giữ prose hợp lệ, không mượn qua user turn và không chạy tool lại. Không mutate bản ghi/slice đầu vào; test idempotence/InitMessages/truncation đạt.
- GetMessages áp dụng view cho cả lịch sử short và temporary tool-round của memory none. Không xóa DB/Redis/audio; dữ liệu mất do ID collision cũ không tự khôi phục, nhưng request mới bỏ call không có kết quả để không tiếp tục400.
- AddMessage tạo UUID32 hex, giữ dạng/độ dài ID và liên kết hai giai đoạn audio hiện hữu; không migration ID cũ.
- Provider Error chỉ vào log kỹ thuật; pipeline nhận empty end trực tiếp, không đưa raw error qua text hooks/TTS/history. Không thêm câu thông báo thay thế. Lượt sau vẫn hoạt động; partial normal text trước lỗi stream vẫn được flush.
- Full client/chat tests đạt; race chọn lọc history/error/provider/ID và gate/snapshot đạt. Vet chat/cmd đạt, client dùng lostcancel=false theo giới hạn cũ. actionlint đạt.
- Thử mở rộng test package internal/domain/llm/eino_llm thấy test cũ không build vì gọi Response/ResponseWithFunctions đã không tồn tại; file/provider source không thay trong đợt này. Không sửa test API cũ ngoài phạm vi và không tuyên bố toàn repo test sạch. Adapter Eino thực được kiểm thử gián tiếp bằng fake HTTP/OpenAI qua pool trong chat tests.

## Cách tiếp tục/test sau phát hành

Đọc run CI/deploy và trạng thái mới nhất dưới đây trước khi thay đổi. Actions deploy-backend từproduction với audio_preprocess/audio_capture/audio_diagnostics/asr_input_gate=unchanged để giữ enforce+gate300 ms/capture. Xác minh health/native/TLS/OTA/WSS và SHA/config sau deploy.

Robot thật: khởi động lại/kết nối lại, nói một câu chào để phát sinh persona/emotion tools; kiểm tra không còn400 lặp và không đọc raw error/call IDs. Thử một câu tiếp để xác nhận session không bị kẹt. Theo dõi đa tool có đủtool responses/ID riêng trong DB mà không in transcript/call payload/secrets lên Git. Không coi health probe là nghiệm thu MCP trên firmware thật.

False positive tiếng động thành từ ngắn vẫn chưa sửa trong scope này. Giữ capture/metadata cho agent tiếp tục theo BACKEND_NOISE_SUPPRESSION.md và NOISE_ROBOT_TEST_20261124.md. Rollback code qua Actions theo DEPLOYMENT_GUIDE.md nếu có regression; không xóa history/audio và không đổi firmware.

## Phát hành và xác minh VPS — 2026-11-24, khoảng10:54–10:55 UTC+7

- Codec26f3a65445cc977c42ca0b4f4a728a6abc93ab4; [CI push45321517370](https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45321517370) và [CI PR45321746615](https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45321746615) đều thành công backend/frontend. Có full client/chat, các regression/race mới, vet/build/native check.
- [PR19](https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/pull/19) merge production27e0b975f672236238656782d32a41ee894c4d85; [Actions45322232698](https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45322232698) push production thành công, mọi audio input mặc địnhunchanged. Không deploy thủ công qua SSH.
- Binary SHA256 mới d3b12bc701b14775c40716e7e8c4d5fe8064d9f77d04dddb4bee49f55ec80980; image sha256:0ea92434dd89d1c29bd6ecc8a143c74cd47b4640b4af7601ce970b586d3c4e31.
- Config SHA256 trước/sau giống hệt672a0db82d68ce864ec150facbcbf4c970932a09fbba7deb3f51e6072482b45f; UID/GID10001:10001/mode0600. Runtime audio_preprocess.mode=enforce, gate enabled/guard300 ms, diagnostics enabled/interval1000 ms và capture enabled giữ nguyên. Không thay ASR/LLM/TTS provider/firmware hay lịch sử.
- Actions HTTP health/native RNNoise/SpeexDSP/TEN self-check PASS. Docker ban đầu starting theo chu kỳ healthcheck, sau đó healthy/restarts0 với exit0. ASR/TTS/gateway không restart; admin HTTPS200, verify_tls.py PASS TLS/OTA/WSS/auth401/internal404/redirect308.
- Không thấy lỗi tool_calls-order/panic/preprocess-bypass trong log container mới ở thời điểm chốt. Tuy nhiên probe chỉ hello, chưa có cuộc gọi đa MCP tool của robot thật sau deploy; không coi việc chưa thấy400 là nghiệm thu firmware/tool-round thật. Người dùng cần nói lại câu chào/câu tiếp, cung cấp thời điểm nếu lỗi để đối chiếu.
- Bản trước: binary9ec6839f5edc5b3f98c89bec5d414ead71c997b1f6055a14babd6c5445db86fe, image sha256:d07d60d8d352dc4627a03b399b72b1b230f0199dc189cf811b8c3cdb0b31aa87. Rollback theo Actions/DEPLOYMENT_GUIDE.md nếu regression, giữ config/audio/history. Không tự xóa tool records hoặc chạy lại action robot để khôi phục những kết quả cũ bị ID collision.

Bàn giao: lỗi thứ tự multi-tool/ID và provider error ra loa đã có code/regression/deploy; kiểm chứng mic/MCP thật còn chờ. Bước tiếp theo về tiếng động thành từ ngắn vẫn theo NOISE_ROBOT_TEST_20261124.md và BACKEND_NOISE_SUPPRESSION.md, không nâng ngưỡng hoặc blacklist từ chỉ dựa trên bản sửa này.
