# Đã bật thử lọc nhiễu trên VPS — 2026-11-24

Người dùng đã yêu cầu merge/deploy và bật lọc để thử robot. PR #17 merge production5d5cceb6; Actions45316196051 thành công, runtime chat.audio_preprocess.mode=enforce. Đây là thử nghiệm mic thật, không đồng nghĩa đã đạt toàn bộ tiêu chí câu ngắn/giọng nhỏ/false positives. Firmware/Gipformer/LLM/TTS và capture giữ nguyên. Xem mục phát hành mới nhất trong BACKEND_NOISE_SUPPRESSION.md; không áp dụng các trạng thái “chưa deploy/off” của lịch sử bên dưới cho runtime hiện tại.

# Code lọc nhiễu đã kiểm thử — 2026-11-24

Nhánh feat/backend-noise-suppression, code72e2cf03 đã push. CI Linux45315184231 backend/frontend thành công, local tests/race/vet/actionlint và mock rollback đạt. A/B290 mẫu hoàn tất, số đo theo nhóm tại BACKEND_NOISE_SUPPRESSION.md. Chưa merge/deploy; còn câu ngắn/giọng nhỏ/nhiễu phòng thật và tải DSP trên VPS, chưa đạt nghiệm thu enforce. Agent tiếp theo đọc báo cáo và kế hoạch trước thay đổi hoặc phát hành; giữ firmware và Gipformer.

# Bàn giao code lọc nhiễu — 2026-11-24

Nhánh feat/backend-noise-suppression, code/core+replay+native packaging đã có. A/B290 mẫu hoàn tất, không bỏ cả câu ở270 mẫu có lời nhưng chưa có đủ câu ngắn/giọng nhỏ; noise thật còn lọt. CI Linux đầu45314376269 đạt; đang chạy lại cho bổ sung diagnostics/self-check deploy/test generator. Chưa merge/deploy, giữ bộ lọc off. Xem BACKEND_NOISE_SUPPRESSION.md (số đo/commands/phần chưa đạt) và kế hoạch để tiếp tục. Không tự bật enforce chỉ vì test synthetic đạt.

# Đang triển khai lọc nhiễu — 2026-11-24

Nhánh feat/backend-noise-suppression từ production06ab4ed7. Đã có FIFO TEN, RNNoise/SpeexDSP, frontend onset/pre-roll, off/shadow/enforce, diagnostics, replay và benchmark harness. Test local đạt; đang A/B mixed speech+noise và chuẩn bị CI Linux. Chưa merge/deploy; chưa đạt nghiệm thu câu ngắn/robot thật. Đọc BACKEND_NOISE_SUPPRESSION.md và BACKEND_NOISE_SUPPRESSION_PLAN.md để tiếp tục; không tự bật enforce từ benchmark sơ bộ.

# Lọc nhiễu backend — kế hoạch 2026-11-24

Kế hoạch hiện hành: [BACKEND_NOISE_SUPPRESSION_PLAN.md](BACKEND_NOISE_SUPPRESSION_PLAN.md). Chỉ lập tài liệu, chưa code/deploy. Ưu tiên bảo toàn mẫu/FIFO, benchmark denoise CPU, xác nhận đầu câu có pre-roll và kiểm chứng câu ngắn/lời lẫn nhiễu; giữ firmware, Gipformer, gate và capture.

# Kiểm tra robot tự trả lời — 2026-11-24

- [x] Đối chiếu log/capture sau deploy gate: guard hoạt động, duration snapshot đúng; vẫn có false positives ngoài guard.
- [x] Dò threshold offline, ghi giới hạn kiểm chứng trong ASR_INPUT_GATE.md; không đổi runtime.
- [ ] Điều chỉnh VAD và kiểm chứng end-to-end, nhất là lời nói nhỏ/câu ngắn; phát hành qua Actions khi có thay đổi.
- [ ] Nghiệm thu lại robot: hiện chưa hết tự trả lời.

# Trạng thái ASR input gate — 2026-11-24

- [x] Code gate/epoch/guard/snapshot và ưu tiên abort, giữ firmware.
- [x] Test local/full chat, race chọn lọc, vet theo phạm vi ghi rõ, CI Linux đạt.
- [x] Bổ sung empty retry trước phát hành; PR14/15 merge production.
- [x] Actions 45226377103 deploy và bật guard 300 ms; config/health/TLS/OTA/WSS/capture qua probe.
- [ ] Robot thật: một câu rồi im lặng, keyword ngắt lời, câu ngắn/nói ngay sau TTS; xác nhận không cắt đầu câu và đo false positives.

Bàn giao: ASR_INPUT_GATE.md. Các mục phía dưới là lịch sử.

# Trạng thái lưu toàn bộ audio — 2026-11-23

- [x] Capture mic/output trước gate chat, archive/timestamps, exporter WAV và tests.
- [x] CI Linux đạt; PR #10 merge production, Actions 45199579444 deploy/bật capture thành công.
- [x] File thực có đủ 100 mic+46 speaker packets ở probe, dropped=0, WAV xuất được; health/HTTPS/OTA/WSS đạt.
- [ ] Thu lượt robot thật sau deploy và đối chiếu mic/loa để phân tích echo.

Đọc FULL_AUDIO_CAPTURE.md. Các trạng thái phía dưới là lịch sử trước phát hành.

# Trạng thái phát hành chẩn đoán — 2026-11-23

- [x] Commit/push và merge PR #6/#7/#8 vào production.
- [x] CI Linux tests/vet/build và frontend đạt.
- [x] Deploy qua Actions, bật log tạm thời, kiểm tra health/HTTPS/OTA/WSS và cấu hình.
- [ ] Thử robot thật trong im lặng/nhiễu/loa phát/lời nói ngắn, ghi số đo.
- [ ] Hiệu chỉnh dựa trên số đo rồi tắt chẩn đoán qua Actions.

Bàn giao hiện hành: AUDIO_DIAGNOSTICS.md. Các mục dưới là lịch sử trước phát hành.

# PR #271 — 2026-11-23

- [x] Tích hợp upstream trên nhánh thử; giữ tiếng Việt và Gipformer.
- [x] Bảo vệ câu ngắn tiếng Việt, tương thích ngưỡng cũ, reset native TEN.
- [x] Go tests/race/vet/build; frontend 10 tests/build; biểu mẫu qua API giả lập.
- [x] Thêm CI Linux và tài liệu [PR271_INTEGRATION.md](PR271_INTEGRATION.md).
- [ ] Commit/push và CI Linux thực tế.
- [ ] Review/merge, deploy qua Actions, thử robot thật và nhiều thiết bị.

---

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

### PhoWhisper — thử thực tế và quyết định 2026-11-23
- [x] CI Linux, trial base/small qua Actions, health và TLS/OTA/WSS.
- [x] 30 speech FLEURS +20 synthetic nonspeech; base đầy đủ tải 1/2/4, small tải đơn.
- [x] Đánh giá phiên robot thật 08:04–08:07 và xác minh ASR mặc định vẫn vps_asr/Gipformer.
- [x] Theo lựa chọn người dùng: giữ Gipformer; small vẫn kém về WER/độ trễ, không activate.
- [x] Dừng tải đồng thời small còn lại, lưu report/giới hạn trong PHOWHISPER_EVALUATION_20261123.md.
- Không đánh dấu P7 chuyển mặc định là hoàn tất: chuyển đổi không diễn ra vì kết quả không đạt và người dùng chọn giữ model cũ.

## Chẩn đoán âm thanh 2026-11-23

Đã thêm log bật/tắt, thống kê RMS/peak/score VAD và ASR/endpoint, unit tests và hướng dẫn AUDIO_DIAGNOSTICS.md. Chưa lưu audio, chưa deploy, chưa có số đo robot thật.

Xác minh chẩn đoán: test chat/TEN/Silero/hooks, race riêng chẩn đoán và TEN/Silero/hooks, vet và build server đạt. Race toàn chat còn lỗi MCP đã ghi BUG_LOG; Linux/robot thật chưa kiểm tra.

## Lưu toàn bộ âm thanh

Code capture/exporter và tests đã có trên feat/full-audio-capture. Còn CI Linux, deploy Actions bật capture và xác minh file/export trên VPS; đọc FULL_AUDIO_CAPTURE.md trước tiếp tục.

## Backend ASR gate — 2026-11-24

Đã code và test local gate/snapshot/keyword abort, không sửa firmware. Còn CI Linux và phát hành/robot thật; bàn giao ASR_INPUT_GATE.md.
