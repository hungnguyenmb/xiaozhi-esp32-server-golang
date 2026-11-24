# Đã bật thử lọc nhiễu trên VPS — 2026-11-24

Người dùng đã yêu cầu merge/deploy và bật lọc để thử robot. PR #17 merge production5d5cceb6; Actions45316196051 thành công, runtime chat.audio_preprocess.mode=enforce. Đây là thử nghiệm mic thật, không đồng nghĩa đã đạt toàn bộ tiêu chí câu ngắn/giọng nhỏ/false positives. Firmware/Gipformer/LLM/TTS và capture giữ nguyên. Xem mục phát hành mới nhất trong BACKEND_NOISE_SUPPRESSION.md; không áp dụng các trạng thái “chưa deploy/off” của lịch sử bên dưới cho runtime hiện tại.

# Kế hoạch lọc nhiễu backend, bảo toàn tiếng Việt ngắn — 2026-11-24

## Trạng thái và phạm vi

Người dùng đã đồng ý triển khai ngày2026-11-24. Đang code/kiểm thử trên feat/backend-noise-suppression; chưa merge production hoặc deploy. Xem BACKEND_NOISE_SUPPRESSION.md để tiếp quản hiện trạng và giới hạn kiểm chứng. Tiếp tục tại repo xiaozhi-server-go, tạo nhánh feat/backend-noise-suppression từ production khi bắt đầu code; kiểm tra working tree và giữ nguyên các sửa tài liệu chưa commit. Các mục thiết kế/tiêu chí bên dưới là mục tiêu; không tự coi là đã nghiệm thu.

Mục tiêu: giảm tiếng nền bị nhận thành lời nói, giữ lời người khi có tiếng động và bảo vệ câu ngắn. Không thể cam kết không bỏ sót tuyệt đối khi nhiễu che mất giọng; phải chứng minh bằng tập kiểm thử phân nhóm. Giữ firmware zero-one, Opus/protocol, ASR Gipformer, LLM/TTS và dữ liệu tài khoản/thiết bị. Giữ gate đang bật và audio capture đầy đủ. Không dùng blacklist “đây/dạ/không…” để che false positives.

Đọc trước: AGENTS.md, DEPLOYMENT_GUIDE.md, ASR_INPUT_GATE.md, AUDIO_DIAGNOSTICS.md, FULL_AUDIO_CAPTURE.md, ECHO_DIAGNOSIS_20261123.md. Lần triển khai sau chỉ phát hành qua GitHub Actions từ production.

## Bằng chứng và điểm cần kiểm tra

- Gate300 ms hoạt động nhưng log08:44–08:48 ngày24/11 vẫn có nhận nhầm nhiều giây sau TTS. Ngưỡng TEN runtime0.4, không có threshold_low riêng; tăng guard không giải quyết nhận nền sau auto-resume.
- Dò offline0.6 giảm mạnh positive trên nền, vẫn giữ câu chào; chưa chạy đầy đủ endpoint/ASR và chưa có đủ câu ngắn được gắn nhãn. Không áp dụng0.6 đại trà từ bằng chứng này.
- internal/app/server/chat/asr.go là luồng decode/VAD/forward ASR. Có trimFirstSpeechAudio giữ tối đa200 ms trước tiếng nói; cần kiểm tra việc xóa buffer và gửi lại để tránh mất/nhân đôi mẫu.
- internal/domain/vad/ten_vad/vad.go chia hop512 và bỏ phần dư mỗi lần gọi. Với frame960 mẫu ở16 kHz, phần448 mẫu chưa được TEN xử lý; cần ghép mẫu dư qua các lần gọi. Đây là phát hiện mã nguồn, chưa chứng minh là nguyên nhân duy nhất của lỗi thực tế.
- Nhánh short_voice_rejected hiện so sánh dưới100 ms dù thông điệp cũ ghi300 ms. Không thay bằng thời lượng tối thiểu dài hơn để lọc nhiễu.
- Hook internal/domain/chat/hooks/asr_noise_filter.go là lọc văn bản, không phải khử nhiễu PCM; runtime chưa bật. Danh sách giữ câu ngắn không cứu được âm thanh đã bị loại trước ASR. Không giả định Gipformer có confidence/no_speech_prob nếu provider chưa trả trường đó.

## Thiết kế tối thiểu dự kiến

Luồng: lưu Opus gốc → gate/epoch hiện hữu → decode → chuẩn hóa PCM → khử nhiễu streaming → VAD và xác nhận đầu câu → pre-roll + toàn bộ đoạn câu → Gipformer → hook hiện hữu → LLM.

### 1. Bảo toàn dữ liệu và trạng thái

- Giữ archive mic/output nguyên bản trước mọi lọc, không ghi đè và không tự xóa. Audio sau lọc chỉ xuất riêng ở chế độ chẩn đoán đã bật, có liên kết timestamp/epoch và quyền riêng như capture hiện hữu; không mặc định nhân đôi toàn bộ lưu trữ production.
- Khớp sample rate/channels thực tế của decoder; không giả định mọi thiết bị luôn16 kHz/60 ms. Xác định mono, scale float/int16 và clipping chính xác ở từng native boundary.
- Giữ FIFO cho phần dư TEN và bộ khử nhiễu/resampler; số mẫu và timestamp không phụ thuộc ranh giới packet. Tính voice/idle theo mẫu thực, không đếm nguyên60 ms cho một hop32 ms nếu điều đó làm sai endpoint.
- State DSP/FIFO/noise floor độc lập theo phiên/epoch. Khi abort, reset phiên, đổi format hoặc trả pool: loại dữ liệu cũ, đóng tài nguyên đúng một lần. Không rò pre-roll/âm thanh giữa robot. Không reset model mỗi packet.
- Packet trong settling vẫn tuân thủ decode-only hiện hữu, không thành lời nói/pre-roll hợp lệ. Xác định và đo thời gian warmup DSP khi bắt đầu epoch; không mở đường bypass gate hoặc tự thêm guard dài.

### 2. Khử nhiễu streaming trên CPU

- Thử RNNoise native C với wrapper Go nhỏ trong tiến trình, không khởi tạo FFmpeg/Python process cho từng packet. Chốt commit/model/license và build reproducible trước tích hợp Docker/CI.
- RNNoise dùng đầu vào48 kHz; nếu PCM đang16 kHz cần resample streaming16→48→16, kiểm tra frame size qua API và scale mẫu theo header/demo. Upsample không phục hồi dải âm đã mất. Đo cả delay của resampler/model và warmup.
- So sánh bypass với RNNoise, và nếu cần mức trộn giảm nhiễu nhẹ hơn. Chỉ thêm tham số mức lọc nếu phép đo chứng minh cần thiết; không thêm nhiều engine production ngay từ đầu.
- DeepFilterNet là phương án benchmark dự phòng nếu RNNoise làm mất âm tiết/ngắn hoặc không đủ hiệu quả; chưa mặc định cài thêm service/model. Chọn theo WER/recall/CPU/độ trễ thực tế, không chỉ nghe âm thanh “sạch”.
- Đầu ra dùng cho cả VAD và ASR phải cùng timeline; không ghép VAD đã lọc với ASR raw một cách tùy tiện. Chạy A/B để phát hiện denoiser tạo artefact hoặc làm ASR kém đi.
- Lỗi DSP phải có log/counter và chuyển rõ sang bypass ở ranh giới epoch sạch, không treo ASR hay trộn PCM khác delay. Production bật/tắt theo phiên mới để tránh đổi timeline giữa câu.

Nguồn đối chiếu: [RNNoise README](https://github.com/xiph/rnnoise/blob/main/README), [API native](https://github.com/xiph/rnnoise/blob/main/include/rnnoise.h), [DeepFilterNet](https://github.com/Rikorose/DeepFilterNet). Các nguồn mô tả công nghệ, không chứng minh hiệu năng trên VPS này.

### 3. Xác nhận đầu câu nhưng giữ câu ngắn

- Dùng trạng thái chờ → nghi có lời → đang nói → chờ kết thúc. Tận dụng hysteresis hiện có: ngưỡng bắt đầu cao hơn ngưỡng duy trì, tránh cắt âm yếu giữa câu. Thử lưới ngưỡng bằng dữ liệu, không chốt0.6 trước benchmark.
- Thay quyết định mở câu từ một packet bằng bằng chứng theo cửa sổ ngắn, có dung sai gián đoạn. Dải40–100 ms là ứng viên thử, không phải hard minimum mặc định; đánh giá cả câu thật60–200 ms. Không yêu cầu tiếng nói liên tục300–500 ms.
- Ring pre-roll200 ms hiện có là điểm xuất phát; khi xác nhận phải phát lại từ trước onset và gửi đúng một lần. Chỉ tăng nếu nhãn onset chứng minh thiếu đầu câu. Giới hạn bộ nhớ theo thời gian, không tích lũy nền vô hạn.
- Khi câu đã mở, gửi liên tục cả âm yếu và khoảng nghỉ ngắn tới ASR; không chỉ gửi những hop VAD=true. Dùng hangover/kết thúc theo khoảng im lặng, không cắt theo biến động năng lượng từng khung.
- Không dùng thời lượng câu hoặc số ký tự làm điều kiện loại duy nhất. Rà lại nhánh dưới100 ms: phân biệt xung nhiễu và câu ngắn bằng bằng chứng âm học; nếu chưa tách được, ưu tiên giữ câu và ghi chẩn đoán thay vì chặn mù.
- Ước lượng nền từ các đoạn đủ chắc không có lời nói, cập nhật chậm và đóng băng khi nghi nói/đang nói. Mức vượt nền là tín hiệu hỗ trợ; không bắt buộc mọi giọng phải vượt một ngưỡng RMS/SNR cố định. Khởi tạo khi mở mic gặp người đang nói không được học luôn giọng đó thành nền. Chỉ bổ sung lớp thích nghi nếu A/B cho thấy tăng ích so với denoise+hysteresis.
- Wake-word abort/detect ưu tiên theo protocol hiện hữu. Không lấy keyword làm bằng chứng rằng mọi audio theo sau đều là lời người và không để bộ lọc chặn điều khiển abort.

### 4. Chẩn đoán và replay có thể tái lập

- Công cụ replay dùng cùng đường xử lý với production: Opus state, gate epoch/guard, FIFO native, reset/pool, pre-roll, endpoint và Gipformer. Không dùng bảng VAD offline đơn giản làm chứng cứ end-to-end.
- Log metadata theo epoch/turn: chế độ bypass/shadow/enforce, raw/clean RMS, noise floor nếu có, VAD high/low và phân bố score theo hop, candidate/accepted/rejected reason, pre-roll ms, voice/idle ms, DSP queue/processing/algorithmic delay, ASR duration và lỗi/reset/drop.
- Shadow không thay quyết định production; chỉ so sánh VAD/segmentation mới. Replay riêng để đo ASR/LLM giả lập; không gửi hai lượt thật vào LLM/TTS và không ghi nhầm chat_messages shadow.
- VAD score không phải confidence ASR. Mặc định log không chứa transcript/audio; giữ audio riêng theo FULL_AUDIO_CAPTURE.md. Log tổng hợp có giới hạn tần suất.

## Dữ liệu và tiêu chí nghiệm thu đề xuất

Chuẩn bị nhãn speech onset/offset và transcript; tách tập chỉnh tham số và tập kiểm định theo buổi thu/người/phòng. Không chia ngẫu nhiên các mảnh cùng một recording vào cả hai tập. Raw capture hiện có là tập hồi quy lỗi; chỉ một câu chào chưa đủ nghiệm thu.

| Nhóm | Dữ liệu tối thiểu đề xuất | Tiêu chí phát hành |
|---|---|---|
| Không có người nói | Mỗi loại ít nhất10 phút: phòng yên, quạt/ù, gõ/va chạm; thêm sau TTS | 0 lượt LLM/TTS tự phát trên tập lỗi cũ; tổng false accepts mới giảm≥90% so baseline và≤1 lượt/giờ gộp; báo rõ thời lượng |
| Câu ngắn thật | Ít nhất200 lượt, nhiều người; “dừng/tắt/bật/vâng/dạ/không/có/ừ/đây”, cả giọng nhỏ/xa | Recall nhận có lời≥98% gộp,≥95% mỗi nhóm; không hồi quy so baseline, báo riêng đúng lệnh/transcript |
| Lời nói và nhiễu đồng thời | Câu ngắn+câu dài, thu thật và trộn có nhãn SNR0/5/10/20 dB | Báo recall/WER theo từng mức; mục tiêu không tăng WER quá1 điểm phần trăm tuyệt đối và không tăng bỏ sót ở nhóm baseline còn nghe được; mức bị che mất giọng ghi riêng |
| Giọng sạch | Ít nhất100 câu có phụ âm đầu/cuối, âm nhỏ và khoảng nghỉ | WER không tăng quá1 điểm phần trăm tuyệt đối, không có lỗi mất/nhân đôi mẫu, không mất đầu/cuối câu trong kiểm tra có nhãn |
| Sau TTS/ngắt lời | Ít nhất30 lần mỗi tình huống: keyword giữa TTS, nói ngay sau TTS, auto-resume | Không tạo vòng tự trả lời; keyword vẫn dừng output; lời sau keyword không bị gate/DSP cắt |
| Hiệu năng VPS | Chạy đồng thời ASR+TTS thực, thử1/3/5 luồng mic hoạt động | Mục tiêu thêm latency p95≤100 ms; DSP p95 xử lý nhanh hơn độ dài chunk, queue không tăng, không drop, còn≥25% CPU headroom ở mức tải công bố |

Các con số là mục tiêu đề xuất, chưa đạt/chưa benchmark. Báo riêng tỷ lệ phát hiện câu và WER; không thể suy câu đúng chỉ từ VAD. Không tuyên bố hỗ trợ nhiều robot dựa trên số kết nối idle. Ghi số lượt/sai số và giới hạn tập mẫu; một giờ không false accept không chứng minh mọi môi trường an toàn. Nếu không đồng thời đạt lọc nhiễu và giữ câu ngắn thì chưa bật enforce đại trà, không giảm tiêu chí để phát hành.

## Thứ tự thực hiện và file dự kiến

1. **Baseline/replay:** thu nhãn, tái hiện vòng tự trả lời bằng capture, đo pipeline cũ. Tạo công cụ trong deployment/vps hoặc cmd theo cấu trúc exporter hiện có; audio/manifest riêng tư nằm ngoài Git. Ghi version/config/hash để lặp lại.
2. **Sửa tính liên tục mẫu:** TEN FIFO/hop và thời lượng, pre-roll/endpoint regression. So sánh riêng trước khi thêm denoiser; tránh gộp nguyên nhân.
3. **Thử denoise:** wrapper native nhỏ dự kiến internal/domain/audio/denoise (xác nhận cấu trúc audio có sẵn trước tạo), integration tại internal/app/server/chat/asr.go; thêm dependency build tối thiểu Docker/CI. Bypass mặc định, test scale/frame/reset/error/close.
4. **Xác nhận đầu câu:** state theo epoch, hysteresis/pre-roll/hangover; rà short_voice_rejected. Dự kiến helper/test cạnh internal/app/server/chat/asr.go; không refactor toàn session hoặc thêm pipeline framework.
5. **Chẩn đoán/A-B:** mở rộng audio_diagnostics.go và tests, xuất báo cáo theo nhóm. Config tối thiểu dự kiến chat.audio_preprocess.mode=off|shadow|enforce, engine chỉ engine đã chọn; các ngưỡng chỉ expose khi cần vận hành. Đây là schema dự kiến, chưa tồn tại; bảo toàn cấu hình provider DB, tránh hai nơi cùng sở hữu một threshold.
6. **Kiểm thử/phát hành:** unit + integration + race chọn lọc, CI Linux native; update main_config example và .github/workflows/deploy-backend.yml có migration chọn lọc/backup/rollback nếu cần. Hoàn thiện báo cáo trước bật thử trên robot hiện tại, sau đó mới mở rộng.

Tests bắt buộc: ranh giới packet20/40/60 ms và chunk khác nhau cho kết quả mẫu tương đương; remainder không mất; một âm tiết ngắn; lời nhỏ chồng tiếng quạt/gõ; pause không chặt câu; clip/+1.0 conversion; guard/abort/reset/close/retry/error; không gửi epoch cũ vào ASR; hai robot không chia sẻ DSP; processed audio alignment và pre-roll không trùng. DSP failure không biến thành deadlock. Test synthetic bổ sung regression, không thay thu thật. Giữ các giới hạn vet/race có sẵn đã ghi trong ASR_INPUT_GATE.md, không tuyên bố toàn repo sạch nếu chưa chạy.

## Phát hành và quay lui

- Nhánh mới/PR, CI Linux thành công, review diff và bản benchmark. Merge production/deploy chỉ khi được yêu cầu trong bước triển khai; lần lập kế hoạch này không phát hành.
- Theo DEPLOYMENT_GUIDE.md: Actions build/deploy, backup binary/config, giữ owner/mode và secrets. Không đổi firmware/model, không SSH chép binary hay sửa config để bỏ pipeline.
- Bật shadow trước trong khoảng quan sát có giới hạn; đo CPU overhead do chạy thêm nhánh, không coi latency shadow bằng enforce. Sau đạt benchmark bật enforce thử cho robot hiện tại qua cơ chế config đơn giản của dự án, không xây hệ thống rollout phức tạp.
- Sau bật: kiểm tra container/HTTPS/OTA/WSS, capture, câu ngắn/ngay sau TTS/keyword, false accepts và latency. Chỉ công bố mức tải đồng thời đã đo.
- Rollback khi mất câu ngắn, méo đầu/cuối, queue tăng, lỗi native hoặc tự trả lời tăng: tắt preprocess/enforce qua Actions và dùng pipeline cũ, giữ gate/ASR Gipformer/capture; nếu native làm crash thì rollback binary/image trước đó. Không xóa recordings hoặc các dữ liệu đã có.

## Bàn giao cho agent tiếp theo

- [x] Đọc luồng mã nguồn và bằng chứng log, lưu kế hoạch; nguồn ứng viên đã đối chiếu.
- [ ] Thu/tách nhãn đủ câu ngắn và lời lẫn nhiễu; baseline/replay end-to-end.
- [x] Sửa FIFO/saturation có regression; duration32 ms ở pipeline mới, pipeline cũ vẫn theo packet.
- [x] Benchmark RNNoise v0.1 pinned trên Mac, mixed290 mẫu; giữ thử nghiệm, chưa chốt enforce production.
- [x] Tích hợp gate/candidate/pre-roll/endpoint, diagnostics và off/shadow/enforce.
- [x] Native/race local và CI Linux cuối45315184231 đạt (code72e2cf03).
- [ ] Benchmark DSP trên VPS dưới tải ASR+TTS, câu ngắn/giọng nhỏ và nghiệm thu robot thật.
- [ ] Shadow/enforce thử, nghiệm thu robot thật; deploy qua Actions khi có yêu cầu.

Cập nhật file này cùng task.md/implementation_plan.md; sau code ghi BUG_LOG.md và DEPLOYMENT_LOG.md theo kết quả thực, ghi run ID/config/version và việc chưa xác minh. Không đưa secrets, audio, transcript cá nhân hoặc model weights lên Git. Không tự coi checklist dự kiến là công việc đã hoàn thành.

## Tiến độ code — 2026-11-24

Đã tạo nhánh từ production06ab4ed7 và triển khai bản thử: TEN FIFO/saturation, RNNoise v0.1/SpeexDSP native streaming, frontend hop512/onset/pre-roll, off/shadow/enforce, reset/epoch/fallback và replay WAV/capture + ASR tùy chọn. Bypass mặc định, giữ speaker PCM raw và archive raw. Code giới hạn 16kHz mono/TEN với endpoint server; manual/AutoEnd/provider khác bypass có log. Noise floor thích nghi và DeepFilterNet chưa thêm vì chưa có bằng chứng cần thiết sau A/B.

Đã qua local tests packages liên quan, race chọn lọc, vet và actionlint; CI Linux cuối45315184231 đã thành công. Đã A/B50 mẫu gồm30speech+20noise tổng hợp; tất cả speech mở lượt, noise không mở lượt với config thử .55/.35/.85. WER sơ bộ4.28% so raw Gipformer4.07%; chưa có đủ câu ngắn/người nói nhỏ. Capture còn cửa sổ nền mở lượt, vì vậy chưa đạt điều kiện enforce. Đọc báo cáo/usage hiện hành, không tự bật production từ các số sơ bộ này.

Cập nhật A/B:290 mẫu gồm30clean+240speech/noise mix+20noise, RNNoise không bỏ cả câu ở270 mẫu có lời,0/20noise mở lượt; WER theo nhóm ghi trong BACKEND_NOISE_SUPPRESSION.md. Còn câu ngắn/giọng nhỏ/người/phòng mới, các cửa sổ robot nhận nền và CPU/latency VPS; chưa đủ để phát hành enforce. CI đầu45314376269 và cuối45315184231 thành công, chưa deploy.
