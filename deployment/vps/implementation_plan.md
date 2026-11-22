# Công việc hiện tại — PhoWhisper ASR, 2026-11-22

Kế hoạch triển khai và checklist bàn giao: [PHOWHISPER_PLAN.md](PHOWHISPER_PLAN.md).
Người dùng đã yêu cầu triển khai code trên nhánh `feat/phowhisper-asr`. Phạm vi: bridge CPU INT8, image/overlay, kiểm thử, benchmark harness, API activate/rollback và GitHub Actions. Chưa chuyển ASR mặc định hoặc deploy VPS trong task code này. Các mục dưới là lịch sử triển khai; quy trình phát hành hiện tại tuân thủ `../../DEPLOYMENT_GUIDE.md`.

# Kế hoạch cài Xiaozhi Go trên VPS 194.233.73.211

## Việt hóa quản trị và tài khoản cá nhân — 2026-11-22

Phạm vi: toàn bộ tuyến Vue desktop/mobile, đăng ký/đăng nhập/setup, biểu mẫu và thành phần dùng chung, quản trị, trợ lý/thiết bị/kiến thức/giọng nói của người dùng, lỗi API và nội dung trợ giúp. Không đổi giao thức, ID, model, mã giọng, phân quyền hoặc dữ liệu người dùng.

1. Kiểm kê chuỗi theo AST để phân biệt nội dung hiển thị với khóa/giá trị kỹ thuật. Lưu bảng dịch tập trung, hiệu đính các thuật ngữ và luồng chính.
2. Thay nội dung giao diện nguồn, locale Element Plus/Vant, lang=vi, tiêu đề và định dạng ngày giờ/số. Giữ zh-CN trong mã giọng/ngôn ngữ của provider.
3. Việt hóa thông báo nghiệp vụ từ API ở lớp hiển thị, giữ nguyên nội dung dữ liệu và chi tiết kỹ thuật. Không duyệt dịch đệ quy toàn bộ JSON trả về.
4. Kiểm tra build, kiểm thử frontend, rà soát chữ Trung còn sót có phân loại; kiểm tra đăng ký/đăng nhập, admin và tài khoản cá nhân trên desktop/mobile.
5. Sao lưu và triển khai frontend riêng qua Nginx admin cùng origin API; giữ Go binary và các container âm thanh. Kiểm tra route SPA/API/WSS và khả năng quay lui.
6. Báo rõ phạm vi đã kiểm tra, nội dung nguyên bản từ provider được giữ, và cập nhật hướng dẫn/nhật ký/task. Không commit/push hoặc gửi thông báo ngoài chat.

## VieNeu-TTS CPU — 2026-11-21/22

1. Sao lưu SQLite/config/compose và ghim commit nguồn VieNeu.
2. Dùng container CPU ONNX riêng, một luồng, cổng 8000 chỉ Docker; model cache bền vững và secret riêng. Turbo FP32 đo RTF >1; người dùng đã chọn Nano ưu tiên phát mượt. API có patch nhỏ hỗ trợ Nano 24 kHz, hai luồng CPU và chia văn bản tối đa 64 ký tự.
3. Kiểm tra HTTP WAV chunked, thời gian gói đầu/RTF, CPU/RAM trước đổi mặc định.
4. Tích hợp provider openai TTS bằng api_url nội bộ, response_format=wav; xác minh backend chuyển từng phần sang Opus 60 ms qua WSS và giải mã được.
5. Chuyển mặc định khi kiểm tra đạt; khởi động lại riêng backend nếu phiên robot còn giữ cấu hình cũ. Giữ Edge TTS để quay lại.
6. Theo dõi dịch vụ, cập nhật tài liệu và nêu rõ kết quả đo, giới hạn CPU và phần chưa nghe thử trên loa thật. Không sửa ASR/VAD trong yêu cầu này.

1. Thêm public key antigravity-ai-agent; kiểm tra SSH bằng key.
2. Kiểm tra hệ điều hành, CPU, RAM, ổ đĩa và cổng đang dùng.
3. Dùng release upstream v0.6.5, Docker và SQLite; không thay mã nguồn upstream.
4. Nối Gipformer 1.5 ONNX INT8 qua dịch vụ FunASR WebSocket nội bộ.
5. Mở cổng 8989 qua Nginx, giới hạn đường dẫn; admin 8080 chỉ localhost.
6. Tạo secrets ngẫu nhiên, tài khoản admin riêng; bật kích hoạt thiết bị.
7. Kiểm tra health, đăng nhập, OTA, WebSocket, ASR và tài nguyên.
8. Để người dùng cấu hình API LLM sau khi cài; báo rõ phần chưa kiểm tra.

Tiêu chí: SSH key hoạt động; container healthy; ASR trả văn bản tiếng Việt;
OTA trả URL public; WebSocket bắt tay đúng; endpoint quản trị không lộ public.
Chưa có tên miền: chỉ bật HTTP/WS cho thử nghiệm, HTTPS/WSS cấu hình sau.
Dữ liệu và secrets giữ trên VPS và thư mục được gitignore.
Không commit/push hoặc gửi thông báo ngoài chat.

## 8. Quy Tắc Định Tuyến & Ngữ Cảnh (Project Routing & Context)

Để tránh việc User phải nhắc lại các quy tắc con, Agent **BẮT BUỘC** tuân thủ cơ chế sau:

1.  **Tự Động Nhận Diện:** Khi User yêu cầu một tác vụ (VD: "Viết bài", "Deploy web", "Sửa API"), Agent phải tự động kiểm tra xem tác vụ đó thuộc dự án con nào trong thư mục `projects/`.
2.  **Ưu Tiên Quy Tắc Cục Bộ (Local Rules Override):**
    *   Trước khi làm việc, **PHẢI** kiểm tra xem trong thư mục dự án con đó có file `RULES.md`, `CONTENT_RULES.md` hoặc `README.md` không.
    *   Nếu có, các quy tắc trong đó có quyền **ƯU TIÊN CAO HƠN** quy tắc chung này.

**Ví dụ cụ thể:**
*   Khi User nói: *"Viết bài"* hoặc *"Auto-Post"* -> **TỰ ĐỘNG** đọc `projects/content-automation/CONTENT_RULES.md` để lấy văn phong và cấu trúc.
*   Khi User nói: *"Sửa Landing Page"* -> **TỰ ĐỘNG** đọc `projects/thammyvien-landingpage/README.md` (nếu có).

---


## CORE PROTOCOL
WORKING_PRINCIPLES.md hiện không có mục CORE PROTOCOL; áp dụng phạm vi đọc, thay đổi tối thiểu, không xóa dữ liệu và xác minh kết quả.

## Bổ sung SSL — 2026-11-21

1. Kiểm tra DNS, cổng 80/443 và dịch vụ hiện tại.
2. Sao lưu cấu hình Nginx, cấu hình Go và SQLite trước cập nhật.
3. Cài host Nginx + Certbot; dùng HTTP-01 webroot cấp chứng chỉ cho otazeroone.hungnguyenaistem.online.
4. Cấu hình HTTPS/redirect HTTP và proxy WebSocket tới gateway hiện tại; giữ admin localhost.
5. Cập nhật OTA/vision URL trong YAML và Manager qua API, giữ nguyên token.
6. Bật timer gia hạn cùng deploy hook reload Nginx; chạy renew dry-run.
7. Kiểm tra HTTPS, chuỗi chứng chỉ, OTA và bắt tay WSS; cập nhật hướng dẫn.

## Công khai trang quản trị qua tên miền — 2026-11-21

1. Dùng domain hiện có otazeroone.hungnguyenaistem.online, root proxy tới Manager localhost:8080.
2. Giữ /xiaozhi/ qua gateway, không thay cấu hình robot hay token.
3. Chặn các API nội bộ trên domain; xác thực quản trị bằng tài khoản/JWT hiện tại.
4. Sao lưu cấu hình Nginx, chạy nginx -t và reload không dừng container.
5. Kiểm tra giao diện, tài nguyên JS/CSS, đăng nhập admin HTTPS, quyền truy cập API và WSS.

## Tách domain quản trị — 2026-11-21

1. Xác nhận adminzeroone.hungnguyenaistem.online trỏ VPS 194.233.73.211.
2. Sao lưu Nginx; cấp chứng chỉ riêng bằng HTTP-01, giữ timer gia hạn hiện có.
3. Domain admin proxy Manager localhost:8080; domain OTA chỉ proxy /xiaozhi/, root chuyển người dùng sang admin.
4. Kiểm tra SSL, đăng nhập admin, tài nguyên frontend, API nội bộ và OTA/WSS; cập nhật tài liệu.

## Khôi phục kích hoạt robot — 2026-11-21

1. Đối chiếu log OTA với MAC thật và các bản ghi quản trị.
2. Sao lưu SQLite, chỉ cập nhật bản ghi MAC thật qua API admin; không xóa các dòng khác.
3. Gắn user admin và activated=true theo thao tác kích hoạt mà người dùng đã yêu cầu.
4. Theo dõi OTA/WebSocket từ robot thật, phân biệt kích hoạt với cấu hình LLM còn thiếu.
5. Ghi lại lỗi 400 của luồng activation payload upstream và hướng dẫn điền Device-ID đúng.
## Sửa luồng hội thoại OpenAI/ASR — 2026-11-21
Giữ model/API key người dùng đã chọn. Sao lưu SQLite và cấu hình, sửa base_url OpenAI thành https://api.openai.com/v1 vì adapter dùng Chat Completions, đặt ASR auto_end=false để backend VAD kết thúc câu cho bridge offline. Đồng bộ giá trị mặc định triển khai. Xác minh LLM bằng chính adapter Go, VAD/ASR và âm thanh TTS; ghi rõ giới hạn thử robot thật.

