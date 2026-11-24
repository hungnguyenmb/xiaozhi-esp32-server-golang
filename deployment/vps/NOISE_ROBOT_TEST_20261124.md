# Chẩn đoán robot sau bật lọc — 2026-11-24

Người dùng báo log lỗi và robot nhận tiếng động thành “Đây”. Lượt này chỉ đọc logs/DB/capture và source, không sửa runtime/code sản phẩm/deploy, không xóa lịch sử. Runtime enforce phát hành bởi Actions45316196051 vẫn healthy.

## Bằng chứng robot thật (UTC+7)

- Phiên bắt đầu10:23:58, TEN/auto/server endpoint, RNNoise=true/start0.55/continue0.35/strong0.85/native delay15 ms. Có AUDIO_PREPROCESS setup trên mic thật; không có bypass/native failure trong phần log đã lấy.
-10:24:02 ASR câu chào: voice1248 ms/idle512 ms, ASR150 ms; điểm VAD cực đại cửa sổ lời nói~0.973.
-10:25:16.154 frontend chấp nhận một đoạn với score0.7708; endpoint10:25:17.591 voice608 ms/idle512 ms, ASR134 ms; DB lưu một từ3 ký tự. Đây là nhận nhầm theo báo cáo người dùng, ngoài vùng guard nhiều giây.
-10:25:55.055 nhận đoạn score0.6114; endpoint10:25:55.593 voice96 ms/idle512 ms, ASR104 ms; DB lưu từ người dùng báo bị nhận nhầm. Lượt tiếp theo10:26:47 cũng lưu cùng từ theo DB read-only.
- DSP p95 các kỳ thường~2–3 ms/chunk, thỉnh thoảng~10–15 ms; max_queue thường0–1, đầu phiên2. Không có bằng chứng nghẽn DSP là nguyên nhân chính. Các kỳ ngay sau đổi epoch còn cộng thống kê kỳ trước, không so trực tiếp RMS raw/clean của những kỳ đó để suy gain hoặc echo.
- RNNoise giảm rõ nhiều nền (ví dụ RMS raw~0.01 thành clean~0.0003–0.002), nhiều candidate bị reject. Nhưng TEN vẫn có đỉnh cao trên nhiễu và hai hit đủ mở câu. Bảo vệ câu ngắn hiện cho96 ms đã xác nhận đi ASR; không thể sửa bằng tăng minimum duration hoặc chặn mọi từ ngắn.

## Lỗi LLM riêng biệt, xác định từ log và source

Ngay sau câu chào, model gọi đồng thời2 MCP tools về persona/emotions. Cả hai tool đã trả kết quả; request LLM tiếp theo lỗi400: assistant tool_calls thiếu tool messages đúng thứ tự. Cùng hai call IDs tái xuất hiện ở các lượt sau, làm lỗi lặp lại; không phải thiếu API key hoặc RNNoise lỗi.

Nguồn cần sửa: internal/data/client/client.go, AlignToolMessages. Nhánh assistant có ToolCalls duyệt từng call và append toàn bộ msg vào output cho mỗi ID có phản hồi. Với2 calls hợp lệ, một assistant bị nhân đôi thành assistant(A,B), assistant(A,B), tool(A), tool(B). Assistant đầu không được theo ngay bởi tool responses, nên vi phạm chuỗi API. Map toàn cục used/valid cũng chưa đảm bảo chronology/group completeness; cần test multi-call, partial/missing/out-of-order/orphan và history truncation, không chỉ bỏ một append mù.

internal/app/server/chat/llm.go, nhánh IsLLMErrorMessage chuyển errMsg vào pushRawText; lỗi kỹ thuật được đọc nguyên văn bằng TTS rồi lưu assistant khoảng311 ký tự. Các khoảng stop→listen dài~37–39 giây khớp việc đọc thông báo lỗi, không phảiASR134 ms chậm. Không đưa call IDs/error nguyên văn/transcript cá nhân vào Git.

Các log context canceled/WS normal-close của thiết bị probe lúc10:04 là lúc đóng smoke connection, tách khỏi lỗi400 của robot thật.

## Audio/replay riêng tư

Snapshot capture của phiên robot đã lấy về /tmp/xiaozhi-noise-robot-20261124, file0600/thư mục0700. Raw trên VPS vẫn giữ nguyên theo FULL_AUDIO_CAPTURE.md, không đưa audio/transcript vào Git. windows.json có3 listen windows có endpoint trong snapshot log ban đầu; DB đã có lượt thứ4 muộn hơn nhưng chưa có đủ metadata trong snapshot đó.

Replay cùng frontend config trên3 windows vẫn nhận được nền ở windows2/3; window3 tái hiện96 ms voice. Window2 offline tách2 segments trong khi runtime có1 endpoint ở cuối cửa sổ, nên không coi replay trùng toàn bộ lifecycle/endpoint thực. Giữ snapshot/timestamps để kiểm chứng bản sửa sau; không dùng số offline làm FPR/giờ.

## Thứ tự xử lý đề xuất, chưa áp dụng

1. Sửa AlignToolMessages theo nhóm assistant/tool liền kề, không nhân đôi assistant; giữ đủ phản hồi cho từng tool_call_id, xử lý orphan/partial/truncated mà không chạy lại tool có side effect. Test multi-tool regression trước sửa, giữ history gốc và sửa request view để phục hồi các phiên bị lỗi khi phù hợp.
2. Không chuyển nguyên văn lỗi provider/call IDs thành câu trả lời TTS/history; log kỹ thuật riêng và phản hồi ngắn phù hợp khi thật sự cần. Không chỉ tắt MCP để che sai chuỗi lịch sử.
3. Replay capture mới, đo bằng chứng speech/noise trước onset và sau denoise, đánh giá ngưỡng/điều kiện xác nhận có so với mức nền. Có nhiễu score0.77 nên nâng đơn thuần0.55→0.65 không chắc giải quyết. Giữ câu ngắn/giọng nhỏ; không blacklist từ bị nhận nhầm.
4. Lượt96 ms ngay sau resume có thể liên quan âm đuôi/echo; đây là giả thuyết cần đối chiếu speaker/mic, không kết luận toàn bộ là tiếng vọng. Lượt608 ms muộn sau resume cho thấy chỉ kéo guard dài hơn không đủ.
5. Code/CI/benchmark và deploy qua Actions production theo DEPLOYMENT_GUIDE.md khi thực hiện bản sửa; ghi run ID/kết quả robot thật. Hiện lỗi400 và false positives chưa được sửa.
