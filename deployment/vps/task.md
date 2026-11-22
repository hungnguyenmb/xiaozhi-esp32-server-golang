# Kế hoạch cài Xiaozhi Go trên VPS 194.233.73.211

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

## Tiến độ
- [x] SSH key và khảo sát VPS
- [x] Cài Docker và tải release
- [x] Triển khai backend và ASR
- [x] Kiểm tra SSH, admin, VAD, ASR, OTA, WebSocket và TTS tiếng Việt
- [x] Theo dõi lại backend/ASR sau hơn 5 phút, backup SQLite/cấu hình và chuẩn bị hướng dẫn bàn giao

## SSL — 2026-11-21
- [x] DNS trỏ đúng, cổng 80/443 trống
- [x] Sao lưu và cấp chứng chỉ
- [x] Cập nhật HTTPS/WSS và kiểm tra
- [x] Gia hạn tự động và tài liệu

## Trang quản trị tên miền — 2026-11-21
- [x] Kiểm tra frontend tích hợp và đường dẫn API
- [x] Cập nhật Nginx và kiểm tra đăng nhập HTTPS
- [x] Kiểm tra OTA/WSS, tài liệu và bàn giao

## Tách domain quản trị — 2026-11-21
- [x] Kiểm tra DNS và hiện trạng
- [x] Cấp SSL và cấu hình hai virtual host
- [x] Kiểm tra đăng nhập, OTA/WSS và gia hạn
- [x] Cập nhật tài liệu và bàn giao

## Khôi phục kích hoạt robot — 2026-11-21
- [x] Đối chiếu MAC/log/bản ghi
- [x] Sao lưu và gắn đúng bản ghi MAC
- [x] Theo dõi OTA từ robot thật, ghi nhận giới hạn WebSocket/audio và LLM còn thiếu
## Sửa luồng hội thoại OpenAI/ASR — 2026-11-21
- [x] Sao lưu và sửa base_url/auto_end qua quản trị
- [x] Xác minh LLM thật, VAD và luồng âm thanh WSS mô phỏng; thêm max_tokens=0 do giới hạn adapter GPT-5
- [x] Đồng bộ tài liệu và ghi nhật ký lỗi; chưa xác minh loa/micro robot thật

## VieNeu-TTS CPU — 2026-11-21/22
- [x] Sao lưu, ghim nguồn và dựng container
- [x] Kiểm tra WAV streaming và tài nguyên; người dùng chọn Nano sau benchmark Turbo
- [x] Tích hợp, kiểm tra Opus qua WSS, chuyển mặc định
- [x] Theo dõi hơn 5 phút, healthy không restart; tài liệu/rollback đầy đủ. Chưa xác minh loa robot thật vì chưa có kết nối lại/xác nhận người dùng.

## Việt hóa toàn bộ console — 2026-11-22
- [x] Kiểm kê 1.835 chuỗi, bảng thuật ngữ và danh mục thông báo backend
- [x] Việt hóa desktop/mobile, admin và người dùng; hiệu đính luồng chính
- [x] Thông báo API/Blob/SSE lỗi, locale Element Plus/Vant, tài liệu giao diện
- [x] Build và 7 kiểm thử PASS; rà soát AST không còn chuỗi Trung chưa phân loại trong nhóm nội dung dịch
- [x] Triển khai; 25 tuyến admin, 8 lượt điều hướng user, quyền 200/403, HTTPS/OTA/WSS PASS; có bản quay lui
- Giới hạn: tên/nội dung cấu hình cũ và dữ liệu provider được giữ nguyên. Chưa thao tác gửi thật mọi biểu mẫu dịch vụ ngoài, upload/nhân bản giọng hoặc xóa dữ liệu người dùng; chỉ mở các biểu mẫu chính và kiểm tra tuyến.


## PhoWhisper — kế hoạch bàn giao 2026-11-22

Tài liệu chính: `PHOWHISPER_PLAN.md`. Chỉ dẫn agent: `../../AGENTS.md`.
- [x] Nhánh `feat/phowhisper-asr`, kế hoạch triển khai và checklist bàn giao.
- [ ] P1: Baseline, CPU/runtime và cơ chế chọn ASR.
- [ ] P2–P4: Model INT8, bridge và CI/workflow ASR.
- [ ] P5–P7: Deploy thử qua Actions/production, benchmark, chuyển cấu hình và kiểm tra rollback theo phạm vi được cho phép.
Chưa viết code hoặc đổi VPS cho PhoWhisper. Agent tiếp theo bắt đầu từ P1 sau khi có yêu cầu thực hiện.

### PhoWhisper — code đã triển khai
- [x] Khảo sát CPU/RAM/container, baseline Gipformer một mẫu; xác minh ASR toàn cục.
- [x] Chuyển model chính thức sang INT8 và kiểm tra WebSocket với model thật trên local ARM64.
- [x] Bridge, image/converter, overlay, manifest/checksum, benchmark và API activate/rollback.
- [x] Workflow CI/build/trial/benchmark/activate/rollback, guard production và khóa concurrency backend.
- [x] 28 unittest PASS, Python compile, actionlint và Compose config PASS.
- [ ] Chạy Docker/Linux CI, lưu workflow run URL/image digest.
- [ ] Bộ mẫu nghiệm thu 30 speech/20 noise, benchmark trên VPS cùng VieNeu.
- [ ] Trial/activate/rollback trên VPS qua Actions và xác nhận robot thật.
Chi tiết số đo/giới hạn: PHOWHISPER_PLAN.md mục 8. Chưa deploy VPS hoặc đổi mặc định.
