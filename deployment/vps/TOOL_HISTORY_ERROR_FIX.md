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
