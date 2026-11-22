# Kế hoạch chuyển ASR sang PhoWhisper

Ngày: 2026-11-22. Nhánh: `feat/phowhisper-asr`. Trạng thái: đã triển khai code và kiểm tra local; chưa deploy PhoWhisper hoặc đổi ASR mặc định trên VPS.

1. **Phương án model và giao thức**
   - Thử `vinai/PhoWhisper-base` trước trên VPS 6 vCPU/12 GB RAM, CPU không GPU; dùng faster-whisper/CTranslate2 INT8. Đo thêm PhoWhisper-small trên cùng bộ mẫu nếu base chưa đạt chất lượng; chỉ chọn small nếu đáp ứng cả độ trễ khi chạy cùng VieNeu Nano.
   - Chuyển model từ nguồn VinAI chính thức, ghim revision/model/tokenizer và phiên bản thư viện; tạo artifact INT8 có checksum. Kiểm tra chuyển đổi bằng âm thanh mẫu trước tích hợp. Không tải/chuyển model mỗi lần nhận câu nói.
   - Giữ Go và giao thức FunASR WebSocket hiện tại: PCM16 mono 16 kHz, kết thúc bằng is_speaking=false, trả text/is_final/mode/wav_name. ESP32 tiếp tục dùng Opus; backend giải mã trước ASR. PhoWhisper xử lý sau hết câu, chưa cung cấp nhận dạng streaming từng từ.

2. **Đo bản hiện tại trước khi đổi**
   - Kiểm tra CPU flags, CPU steal, RAM khả dụng và tài nguyên khi VieNeu đang sinh giọng nói trên VPS thật. VNNI/AVX-512 là khả năng tăng tốc, không đặt làm điều kiện bắt buộc; xác minh INT8 thực sự được runtime hỗ trợ.
   - Tạo bộ ít nhất 30 câu có bản chép chuẩn: Bắc/Trung/Nam, câu ngắn/dài, tên riêng, chữ số, tiếng ồn; thêm ít nhất 20 mẫu im lặng/tiếng động. Không đưa audio cá nhân lên repository.
   - Đo cùng bộ mẫu với Gipformer và PhoWhisper: WER/CER với cùng quy tắc chuẩn hóa, p50/p95 thời gian sau hết câu, queue_ms/inference_ms, RTF, CPU/RAM; tách khỏi thời gian VAD, LLM và TTS.

3. **Tích hợp tối thiểu**
   - Giữ nguyên cầu nối Gipformer để quay lui; thêm phowhisper_bridge.py, Dockerfile.phowhisper và service thử nghiệm asr-phowhisper chỉ trong mạng Docker.
   - Bắt đầu 1 worker, 2 CPU threads, giới hạn container 3 vCPU/3 GB RAM; đây là ngân sách thử nghiệm, điều chỉnh theo phép đo. Hàng đợi tối đa 4 yêu cầu, timeout rõ ràng, không tạo thêm model theo kết nối.
   - language=vi, task=transcribe, beam_size=1, condition_on_previous_text=false làm cấu hình thử ban đầu. Giữ VAD backend, kiểm tra tiêu chí no-speech trên bộ mẫu; không tự bật thêm VAD chồng nhau hoặc cắt chữ bằng ngưỡng chưa đo.
   - Health chỉ ready sau tải model và warm-up thành công. Kiểm tra PCM sai, đoạn rỗng, giới hạn 60 giây, quá tải, ngắt kết nối và timeout. Ghi metric, không mặc định ghi nội dung lời nói.
   - Tạo cấu hình ASR riêng trong Manager, giữ cấu hình Gipformer; thử một robot trước khi đổi mặc định. Xác minh cách áp dụng cấu hình và việc kết nối lại của phiên đang mở.

4. **GitHub Actions và triển khai thử**
   - Tuân thủ DEPLOYMENT_GUIDE.md: triển khai qua GitHub Actions, không SCP/build/restart bằng tay để bỏ qua pipeline.
   - Bổ sung workflow ASR riêng với paths cho bridge/Dockerfile/model manifest/compose và công cụ cấu hình liên quan. Workflow hiện tại chỉ thay binary Go, chưa copy/rebuild service ASR, nên phải bổ sung luồng artifact/image ASR và cấu hình compose.
   - Nhánh tính năng chỉ kiểm thử CI, không deploy VPS. Sau khi kiểm tra và merge production, chạy workflow từ production ở chế độ thử để chỉ dựng sidecar; không đổi ASR mặc định. Chế độ chuyển mặc định là bước riêng sau benchmark. Thêm kiểm tra ref production, khóa concurrency dùng chung với workflow backend và health/smoke test.
   - Build/ghim image và artifact model ở CI, tải model trước chuyển đổi, sao lưu cấu hình cần đổi; không đưa secrets/model weights vào Git. Chỉ cập nhật ASR và cấu hình liên quan, tránh restart Go/TTS nếu không cần. Cập nhật hướng dẫn và nhật ký.

5. **Điều kiện chọn và chuyển chính thức**
   - Mục tiêu ban đầu cần xác nhận bằng đo: câu 2–5 giây có ASR p95 <=1,5 giây sau hết câu ở tải đơn; RTF <1. Đo thêm 2 và 4 lượt nói đồng thời và ASR chạy cùng TTS để xác định công suất thực, chưa cam kết số robot đang nói cùng lúc.
   - WER/CER không kém Gipformer trên bộ mẫu; không bỏ sót câu ngắn quan trọng; không tạo chữ trên bộ 20 mẫu im lặng/tiếng động đã chọn. Theo dõi TTS không hụt âm thanh và host không swap/OOM.
   - Nếu base không đạt chất lượng, so sánh small; nếu small quá chậm, chưa chuyển mặc định và báo kết quả/tradeoff. Không mặc định PhoWhisper sẽ nhanh hoặc chính xác hơn trên robot thật.
   - Sau khi đạt, thử robot thật toàn luồng nhận dạng → LLM → VieNeu → Opus, theo dõi ít nhất 30 phút trước áp dụng rộng hơn. Cập nhật mặc định qua pipeline và kiểm tra robot kết nối lại.

6. **Quay lui**
   - Giữ Gipformer, cấu hình cũ và image/model revision cũ trong thời gian thử. Khi health/smoke test thất bại hoặc độ trễ/chất lượng không đạt, workflow khôi phục endpoint/cấu hình cũ, chỉ khởi động lại dịch vụ bắt buộc và kiểm tra lại ASR/WSS.
   - Không khôi phục toàn bộ SQLite cho rollback ASR để tránh ghi đè tài khoản/thiết bị mới. Dừng sidecar thử nghiệm sau khi không còn request đang xử lý nếu tài nguyên VPS bị thiếu.

Nguồn kỹ thuật:
- https://github.com/VinAIResearch/PhoWhisper
- https://github.com/SYSTRAN/faster-whisper
- https://opennmt.net/CTranslate2/hardware_support.html

## 7. Bàn giao cho agent tiếp theo

### Hiện trạng đã xác minh từ repository

- HEAD tại thời điểm lập kế hoạch: `f799f74`; nhánh `feat/phowhisper-asr` đã có trên fork `hungnguyenmb/xiaozhi-esp32-server-golang`.
- Mới có tài liệu kế hoạch; chưa viết bridge, workflow ASR, tải model hoặc thay đổi dịch vụ trên VPS cho PhoWhisper.
- `deployment/vps/asr_bridge.py`: Gipformer Sherpa-ONNX INT8, 1 executor worker, 2 inference threads, 16 kết nối tối đa, 4 slot chờ/xử lý, giới hạn audio 60 giây.
- `deployment/vps/compose.yaml`: Go dùng dịch vụ `asr:10096`; ASR và TTS cùng chia sẻ CPU host. Tổng quota container có thể vượt 6 vCPU vật lý được cấp, không suy ra công suất bằng cộng quota.
- `deployment/vps/prepare_runtime.py`: FunASR offline, sample_rate=16000, auto_end=false. Không chạy lại script khởi tạo này trên production chỉ để đổi ASR vì có thể ảnh hưởng cấu hình đang dùng.
- `deployment/vps/verify_asr.py`: mới kiểm tra 1 file WAV, tải 4 yêu cầu và JSON sai; chưa đủ làm benchmark chất lượng.
- `.github/workflows/deploy-backend.yml`: paths có compose nhưng hành động thực tế chỉ chuyển binary và rebuild backend. Chưa có pipeline ASR.
- Chưa khảo sát lại VPS trong task lập kế hoạch này. Số liệu tốc độ cũ trong nhật ký không thay thế baseline mới.

### Đọc trước khi bắt đầu

1. `AGENTS.md`, `DEPLOYMENT_GUIDE.md` và tài liệu này.
2. `deployment/vps/compose.yaml`, `asr_bridge.py`, `Dockerfile.asr`, `verify_asr.py`, `prepare_runtime.py`.
3. `.github/workflows/deploy-backend.yml`, `deployment/vps/HUONG_DAN_TRIEN_KHAI.md`, `DEPLOYMENT_LOG.md`, `BUG_LOG.md`.
4. Adapter FunASR Go và API/cơ chế lưu cấu hình Manager: tìm bằng `rg` trước khi quyết định cách chọn ASR cho từng robot. Chưa xác nhận khả năng chọn riêng từng robot; nếu hệ thống chỉ có ASR toàn cục, không tự đổi toàn cục để thử một robot. Ưu tiên harness/phiên thử cách ly, báo giới hạn trước bước chuyển.

### Danh sách file dự kiến

| File | Công việc |
| --- | --- |
| `deployment/vps/phowhisper_bridge.py` (mới) | Giữ contract WebSocket; model dùng chung, queue giới hạn, metric và hủy yêu cầu |
| `deployment/vps/Dockerfile.phowhisper` (mới) | Image CPU, thư viện ghim phiên bản, user không root |
| `deployment/vps/phowhisper-manifest.json` (mới) | Model ID/revision, tokenizer, conversion options, checksum và phiên bản runtime; không lưu weights |
| `deployment/vps/compose.phowhisper.yaml` (mới, ưu tiên) | Overlay sidecar thử, volume model read-only, resource limits, health, không publish port host |
| `deployment/vps/verify_phowhisper.py` (mới) | Kiểm tra giao thức, audio, lỗi và benchmark bằng đường dẫn mẫu được truyền vào |
| `deployment/vps/test_phowhisper_bridge.py` (mới) | Kiểm thử contract/queue độc lập với việc tải weights; kiểm thử tích hợp thật là bước riêng |
| `.github/workflows/deploy-asr.yml` (mới) | CI và chế độ thử/chuyển/quay lui, chỉ deploy từ production; triển khai image/model/overlay có version |
| `deployment/vps/configure_phowhisper.py` (nếu cần) | Đổi đúng trường endpoint/config ID qua API được hỗ trợ, lưu snapshot tối thiểu để rollback, idempotent |
| `DEPLOYMENT_GUIDE.md` | Mô tả workflow ASR, inputs, giới hạn, cách lấy artifact và kiểm tra rollback |
| `deployment/vps/task.md`, `DEPLOYMENT_LOG.md`, `BUG_LOG.md` | Tiến độ thực, bằng chứng đo, lỗi thực tế và giới hạn chưa xác minh |

Tên file mới là dự kiến, agent có thể điều chỉnh để khớp codebase. Không sửa Go/frontend nếu bridge tương thích đủ; nếu thực sự cần, ghi rõ nguyên nhân trước khi mở rộng phạm vi.

### Thứ tự thực hiện và điểm dừng

- [x] P0 — Tạo nhánh và lưu kế hoạch. Không có thay đổi runtime.
- [ ] P1 — Xác minh CPU/runtime, contract ASR và cơ chế chọn cấu hình. Thu baseline Gipformer và chuẩn bị bộ audio có quyền sử dụng. Ghi phần thiếu dữ liệu, không bịa kết quả.
- [ ] P2 — Chuyển PhoWhisper-base sang INT8; xác minh artifact thực sự load được, đúng tiếng Việt và compute type. Lưu revision/checksum và command tái tạo; so sánh một số mẫu trước/sau quantization.
- [ ] P3 — Viết bridge/image/overlay; kiểm tra đoạn rỗng, PCM sai, JSON sai, sample rate, quá dài, nhiều câu trên một socket, queue đầy, client đóng và worker lỗi.
- [ ] P4 — Thêm CI/workflow ASR; kiểm tra cú pháp, ref guard, concurrency, secrets và rollback. Model phải sẵn sàng trước khi đổi endpoint; không suy luận rằng HTTP 200 của Go chứng minh ASR tốt.
- [ ] P5 — Merge mã đã kiểm tra vào production theo phạm vi được người dùng cho phép; chạy Actions chế độ thử sidecar. Ghi run URL, commit SHA, image digest, model checksum, health và tài nguyên. Không tự merge/deploy chỉ vì có checklist này.
- [ ] P6 — Benchmark A/B, kể cả ASR đồng thời với VieNeu; nếu chưa đạt thì giữ Gipformer mặc định, báo số đo. Không tự nâng model để bỏ qua điều kiện tốc độ.
- [ ] P7 — Sau khi đủ bằng chứng và có phạm vi phát hành, Actions chuyển cấu hình, thử robot thật, theo dõi 30 phút; xác minh rollback trước khi coi là hoàn tất.

Bước tiếp theo: hoàn tất dữ liệu benchmark P1 và chạy kiểm tra Linux trong CI. Người dùng đã yêu cầu triển khai code; chưa merge/deploy production hoặc chuyển ASR mặc định trong task này. Các mốc cũ bên trên là checklist tổng, xem cập nhật thực tế bên dưới để tránh hiểu nhầm.

### Quy ước đo và tiêu chí nghiệm thu

- `queue_ms`: từ lúc kết thúc câu đến lúc bắt đầu suy luận; `inference_ms`: thời gian chạy model đến lúc đã duyệt hết generator kết quả; `asr_total_ms`: từ gửi hết câu đến nhận JSON cuối. Không chỉ đo thời gian tạo generator của faster-whisper.
- RTF = inference_seconds/audio_seconds. Warm-up tách khỏi lượt đo steady-state; ghi cả cold start, số mẫu và cách tính percentile.
- Chuẩn hóa WER/CER nhất quán, giữ dấu tiếng Việt, báo cả kết quả chuẩn hóa và lỗi tên riêng/chữ số. Bộ 30 câu là vòng sàng lọc, chưa chứng minh chất lượng với mọi giọng/vùng miền.
- Đạt khi contract đúng, chất lượng không kém baseline trên bộ mẫu, p95 tải đơn theo mục 5, không timeout/OOM, TTS không hụt tiếng trong bài thử và robot thật hoàn tất hội thoại.
- Đếm số kết nối online riêng với số yêu cầu suy luận đồng thời; báo công suất theo phép đo, không cam kết nhiều robot chỉ từ RAM/vCPU.
- Silence/noise phải không sinh chữ trong bộ mẫu đã chọn, nhưng không tuyên bố loại bỏ hoàn toàn hallucination.
- Ngưỡng queue/timeout và cấu hình no-speech được điều chỉnh theo số đo; kiểm tra tác vụ inference đã timeout không tiếp tục tích tụ trong executor. Không giải phóng slot sớm khi native inference vẫn đang chạy.

### Bàn giao và rollback cần ghi gì

- Báo cáo benchmark chỉ chứa ID mẫu và số đo; không mặc định lưu lời nói, secrets hay audio cá nhân trong Git/Actions logs.
- Mỗi lần triển khai lưu run URL, SHA nguồn, image digest/model revision, config ID/endpoint cũ và mới; snapshot nhạy cảm giữ trên VPS theo quyền hạn chế.
- Workflow thử không ghi đè compose gốc hoặc runtime config; dùng overlay rõ ràng. Rollback chỉ trả các trường cấu hình đã đổi và revision dịch vụ; không restore toàn bộ SQLite.
- Rollback phải kiểm tra nhận dạng tiếng Việt qua endpoint Gipformer cũ, OTA/WSS và TTS; không chỉ kiểm tra container đang chạy.
- Cập nhật checklist sau từng pha với kết quả thực. Khi dừng, ghi bước đang làm, file đã sửa, lệnh kiểm tra, lỗi còn tồn tại và bước kế tiếp.


## 8. Kết quả triển khai code — 2026-11-22

### Phần đã hoàn tất

- P1 một phần: đọc adapter Go và Model/Controller Manager, xác nhận ASR toàn cục, không có ASRConfigID riêng trên Agent. Khảo sát VPS chỉ đọc: 6 vCPU AMD EPYC, AVX2, không thấy VNNI/AVX-512, khoảng 10 GB RAM khả dụng, không swap; container ASR/Go/TTS healthy. Snapshot CPU steal 0% không đại diện cho mọi thời điểm.
- Baseline Gipformer trên VPS với một file speech công khai 5,08 giây: tải đơn 573 ms, bốn yêu cầu 172/376/596/769 ms; JSON sai bị từ chối. Chưa có bộ 30 speech/20 noise hoặc WER/CER.
- P2 một phần: chuyển thành công PhoWhisper-base revision `7ebdb9e88f5cc5271fb88f4d642c82ff9388650e` sang INT8 trên local ARM64. model.bin 76.396.677 byte; SHA-256 `8b039b26e4e640942900d93f1094ee83a2d71c0245e398bc83b02494e868e588`. Artifact/model cache ở /tmp ngoài repo. Đây là checksum artifact local, không hardcode để thay checksum artifact CI.
- Model thật qua WebSocket bridge trên local ARM64, 2 threads: mẫu 5,08 giây trả văn bản không rỗng sau 511,4 ms; bốn yêu cầu 526,89/1094,78/1613,64/2137,8 ms; PCM toàn zero trả text rỗng. Không so trực tiếp tốc độ local với VPS, không suy ra WER từ việc có text.
- P3 code hoàn tất: bridge, queue có giới hạn và deadline, model dùng chung, image runtime/converter, model manifest/checksum, Compose overlay; không sửa Go/frontend/Gipformer/firmware.
- P4 code hoàn tất: Actions test/build Linux, model WebSocket smoke, artifact, trial/benchmark/activate/rollback với production ref guard; concurrency dùng chung workflow backend. Có công cụ cấu hình typed API, rollback chọn lọc, smoke TLS/OTA/WSS.
- Kiểm tra local: 28 unittest PASS, Python compile, actionlint cho 2 workflow và Compose config PASS với env TTS giả rỗng trong thư mục tạm. Không dùng secrets thật để validate Compose.

### Phần chưa hoàn tất, agent phải giữ rõ

- Docker daemon local không chạy: chưa build/run image Linux tại máy này, chưa chạy Actions, chưa có workflow run URL/digest image Linux.
- Chưa deploy sidecar hoặc thay mặc định trên VPS, chưa tạo cấu hình PhoWhisper trong Manager. Không có model weights PhoWhisper trên VPS do task này.
- Chưa benchmark PhoWhisper trên CPU AMD VPS, chưa so sánh WER/CER, chưa đo tải cùng VieNeu bằng bộ mẫu nghiệm thu, chưa thử loa/micro robot thật hoặc theo dõi 30 phút.
- URL benchmark yêu cầu ở activate là dấu vết cho người review, không phải bộ máy tự chứng nhận chất lượng; vẫn phải rà soát số đo trước phát hành.
- Workflow mới cần xuất hiện trên default branch để GitHub cho dispatch; kiểm tra điều kiện này khi chuẩn bị phát hành. Nhánh tính năng chỉ CI, không bỏ qua guard production.
- Git pull --rebase đầu task bị từ chối do tài liệu kế hoạch chưa commit; fetch xác nhận HEAD và origin/feat/phowhisper-asr không lệch (0/0). Không stash/reset hoặc xóa tài liệu người dùng. Chưa commit/push các thay đổi của task code.

### Đi tiếp

1. Review code và lưu thay đổi lên nhánh tính năng theo yêu cầu người dùng; chạy CI Linux và xử lý lỗi thực nếu có.
2. Chuẩn bị bộ mẫu riêng trên VPS bằng quy trình dữ liệu được cho phép. Schema/lệnh và bốn chế độ Actions nằm ở DEPLOYMENT_GUIDE.md mục 6.
3. Khi có phạm vi phát hành, merge production, chạy trial rồi benchmark. Chỉ activate sau khi chất lượng, độ trễ, TTS và kế hoạch reconnect robot được kiểm chứng.
4. Cập nhật mốc còn thiếu bằng run URL/SHA/digest và số đo thực; không đánh dấu P5–P7 hoàn tất chỉ vì unit tests đã qua.

## 9. Commit/merge và CI Linux — 2026-11-23

- Người dùng đã yêu cầu commit và merge production. Commit triển khai 025bd16, sửa Docker context e3a026b; PR #1 đã merge tại 26078b2.
- CI Linux run 45084318304 thành công: 28 tests, Compose, build/chuyển model INT8, warm-up và WebSocket speech/silence với model thật. Đây là GitHub runner, chưa phải benchmark VPS AMD.
- CI phát hiện Docker context bị .dockerignore cũ loại file; đã thêm Dockerfile.phowhisper.dockerignore allowlist.
- Sửa thêm phạm vi concurrency sau khi thấy CI production bị hủy do tranh chỗ pending: khóa dùng chung chỉ áp dụng job deploy ASR, CI theo ref/event. Không dispatch trial/activate trong task commit/merge này.
