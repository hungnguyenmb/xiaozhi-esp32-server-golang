# Đã bật thử lọc nhiễu trên VPS — 2026-11-24

Người dùng đã yêu cầu merge/deploy và bật lọc để thử robot. PR #17 merge production5d5cceb6; Actions45316196051 thành công, runtime chat.audio_preprocess.mode=enforce. Đây là thử nghiệm mic thật, không đồng nghĩa đã đạt toàn bộ tiêu chí câu ngắn/giọng nhỏ/false positives. Firmware/Gipformer/LLM/TTS và capture giữ nguyên. Xem mục phát hành mới nhất trong BACKEND_NOISE_SUPPRESSION.md; không áp dụng các trạng thái “chưa deploy/off” của lịch sử bên dưới cho runtime hiện tại.

# Bàn giao lọc nhiễu backend — 2026-11-24

## Hiện trạng

Nhánh feat/backend-noise-suppression từ production06ab4ed7; code đã commit/push, CI Linux cuối thành công. Người dùng yêu cầu triển khai code theo BACKEND_NOISE_SUPPRESSION_PLAN.md. Chưa merge/deploy, runtime VPS giữ nguyên. Đừng tự coi code đã đạt mọi tiêu chí chất lượng.

## Đường xử lý đã có

- TEN-VAD giữ phần dư qua packet, saturate float+1.0 đúng32767, reset xóa FIFO/score. Thay đổi này áp dụng cả preprocess=off; cần regression Linux/robot trước merge.
- RNNoise native v0.1 commit cdf196b1e9de2f8ff1003328ebf9a4316477429d, built-in weights/BSD license; không tải model lúc chạy. SpeexDSP quality5 resample16→48→16 kHz, output160 mẫu/10 ms; scale RNNoise float theo đơn vị int16. Thư viện riêng libxiaozhi_rnnoise_v01 tránh nạp nhầm ABI v0.2.
- Denoiser/frontend state thuộc một goroutine/epoch, không qua pool chung. Thư viện native/FFT tables giữ resident theo process; init chung được serialize, stream states đóng riêng. Unsupported OS/format hoặc thiếu lib: khởi tạo bypass có log; lỗi giữa enforce hủy/restart lượt ASR rồi bypass ở epoch mới, không nối raw vào giữa audio có delay.
- Frontend inference theo512 mẫu/32 ms. Xác nhận2 start hits trong3 hop hoặc1 hop strong; duy trì qua ngưỡng thấp hơn. Nhánh cũ loại dưới100 ms không áp dụng cho câu đã xác nhận bởi frontend. Ring pre-roll200 ms và current hop gửi đúng một lần; các hop yếu/im lặng trong câu vẫn vào ASR. Turn dùng threshold im lặng thực của client, so sánh strict>, giữ chờ first-text của realtime như cũ.
- Chuẩn hóa segmentation/chẩn đoán dựa trên32 ms thực, không tính cả60 ms packet khi chỉ xử lý32 ms. Pipeline cũ vẫn đếm theo packet; không khẳng định đã sửa mọi metric của tất cả provider.
- Gate/guard và abort điều khiển giữ nguyên; guard packet decode-only. Có kiểm tra epoch trước commit onset/forward. Speaker nhận PCM raw được ghép theo số mẫu; ASR/VAD nhận clean cùng timeline. Capture gốc không đổi.
- Bộ đệm đầu câu cũ được giới hạn để không giữ nền vô hạn. Cấu trúc ASR loop chỉ thêm vòng xử lý native hops, không viết lại các mode/speaker/LLM.

## Cấu hình thử nghiệm (mặc định tắt)

```yaml
chat:
  audio_preprocess:
    mode: off       # off | shadow | enforce
    denoise: true   # false chỉ dùng để A/B phân đoạn
    start: 0.55
    continue: 0.35
    strong: 0.85
```

Đây là config thử, chưa chốt ngưỡng production. Khi bật, frontend dùng ngưỡng trong chat.audio_preprocess thay cho ngưỡng VAD của provider DB; không tự sửa DB, chỉnh giao diện VAD cũ không đổi ngưỡng frontend. Chỉ16 kHz mono/TEN/server endpoint; manual/ASR AutoEnd/provider khác giữ đường cũ. Không có section hoặc mode=off thì không nạp DSP; bật/tắt qua restart cho phiên mới. Shadow chỉ log metadata và giữ PCM/quyết định cũ, không tạo thêm chat/LLM/TTS.

AUDIO_PREPROCESS setup/bypass/accepted/interval ghi mode/epoch/ngưỡng/delay native/RMS/processing time và counters; có processing_p95_ms (tối đa256 chunk đầu mỗi kỳ1s), max_queue input và rejected/candidate/accepted. Đây là thời gian xử lý DSP theo chunk, không phải p95 latency đầu-cuối. Không log transcript/audio. AUDIO_DIAG vẫn theo dõi VAD/endpoint/ASR. Native delay không bao gồm packet buffering, hop FIFO, xác nhận onset hoặc mạng. Chưa có adaptive noise floor, không ghi processed audio song song tự động.

## Build/test cục bộ

```bash
bash deployment/vps/build_rnnoise.sh /tmp/xiaozhi-rnnoise-lib
# Cài SpeexDSP riêng cho môi trường phát triển (macOS: brew install speexdsp).
# macOS:
export XIAOZHI_RNNOISE_LIBRARY=/tmp/xiaozhi-rnnoise-lib/libxiaozhi_rnnoise_v01.dylib
export XIAOZHI_SPEEXDSP_LIBRARY=/opt/homebrew/lib/libspeexdsp.dylib
export XIAOZHI_TEST_RNNOISE=1
# Linux dùng .so, SpeexDSP libspeexdsp.so.1 qua system loader.
# Dùng cấu hình CGO/ONNX/Go tương ứng ASR_INPUT_GATE.md cho package chat.
go test -tags nolibopusfile ./internal/domain/audio/denoise ./internal/domain/audio/frontend ./internal/domain/vad/ten_vad ./cmd/audio-frontend-replay ./internal/app/server/chat
```

Dockerfile.linux build/copy RNNoise+SpeexDSP+license vào bundle/ten-vad/lib/Linux/x64; loader production dùng LD_LIBRARY_PATH hiện hữu. Build script giữ source tạm để tra cứu, không đặt source/model/audio vào repo. Windows/!cgo không có engine native, giữ stub báo không hỗ trợ.

## Replay và chấm điểm (dữ liệu riêng ngoài Git)

```bash
umask 077
go build -tags nolibopusfile -o /tmp/audio-frontend-replay ./cmd/audio-frontend-replay
/tmp/audio-frontend-replay -suite /private/path/suite.json -denoise=true \
  -asr-url ws://127.0.0.1:19096/ -include-text > /private/path/replay.json
python3 deployment/vps/benchmark_audio_frontend.py score \
  --suite /private/path/suite.json --report /private/path/replay.json
python3 deployment/vps/benchmark_audio_frontend.py mix \
  --suite /private/path/suite.json --output /private/path/new-mixed-suite
```

Suite JSON như verify_phowhisper.py: id/wav/reference; empty reference=noise. mix tạo nhóm clean, fan/click tại SNR0/5/10/20 dB, scale chung để tránh clipping, giữ source_id để chia tập theo nguồn trước tune. Đó là nhiễu mô phỏng, không thay phòng/robot thật. File mới0600/thư mục0700, không ghi đè/xóa recordings.

Capture: `-input /private/capture-dir -windows /private/windows.json`. Windows JSON `[{"Start":"2026-11-24T08:44:35.843+07:00","End":"2026-11-24T08:44:41.201+07:00","guard_ms":0}]`. Phải lấy từ log đúng phiên; command không suy busy từ âm thanh. Decode priming trong guard, kiểm tra sequence/dropped/format; snapshot chưa đóng không chứng minh toàn phiên đầy đủ. Đây là replay frontend/turn trên cửa sổ đã định, không tái hiện pool/network wall-clock/gate lifecycle hoàn chỉnh; các phần đó cần integration/robot thật riêng.

Không -include-text thì không có transcript trong JSON; chấm WER cần bật và giữ output riêng. EOF còn câu ghi Open=true, không giả là endpoint. WAV có thể thêm `-tail-ms 600` để đẩy hết tail DSP/FIFO và mô phỏng im lặng; số mẫu padding báo riêng, không dùng để đo latency endpoint thật. Capture không cho thêm padding vào cửa sổ. Công cụ chỉ gọi ASR tùy chọn, không LLM/TTS/DB.

## Bằng chứng sơ bộ và giới hạn

- Local macOS ARM native tests full packages liên quan/race chọn lọc/vet/actionlint đạt ở thời điểm ghi; CI Linux lần đầu [45314376269](https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45314376269) đạt backend+frontend (commit5458b79f); CI cuối [45315184231](https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45315184231) commit72e2cf03 cũng thành công, gồm native self-check, mock health rollback, tests/vet/build và frontend. Docker daemon local không chạy nên chưa build Docker đầy đủ. Kiểm tra run CI trước merge.
- Suite cũ50 mẫu:30 FLEURS tiếng Việt +20 synthetic noise/silence, không phải50 câu nói sạch. Config thử .55/.35/.85+RNNoise mở đủ30speech, không mở20noise. WER4.28% (20/467) so raw Gipformer4.07% (19/467) đo lại qua SSH tunnel ASR hiện hữu. Raw-ASR-only không có VAD lọc20noise và trả text cả20; script verify vì vậy exit1 sau khi in JSON, đây là kết quả hồi quy tiếng nền chứ không phải benchmark bị mất dữ liệu. Phép so WER này chưa padded EOF, còn Open segments; không dùng như nghiệm thu mất đầu/cuối câu.
- Snapshot robot23 cửa sổ: frontend .55/.35/.85 chưa denoise mở13 lượt; có denoise mở6, gồm câu chào đầu. Thử .65/.40/.90 mở3. Vẫn còn nền nghi ngờ, nhãn “các cửa sổ sau đều noise” chưa được xác nhận độc lập cho phiên mới. Không suy FPR/giờ từ tập ngắn này và không chọn ngưỡng bằng chỉ1 câu chào.
- RTF DSP sơ bộ~0.026 trên Mac, không phảiCPU VPS. ASR gọi trên VPS nhưng DSP chạy local; chưa benchmark1/3/5 luồng DSP+ASR+TTS trên VPS, chưa có p95 latency tổng hợp.
- Chưa đủ200 câu ngắn/nhiều giọng/phòng, chưa gắn nhãn onset/offset, chưa kiểm tra keyword/robot thật trên code mới. Vì vậy chưa đạt tiêu chí enforce production. Benchmark speech+noise đã có kết quả bên dưới; vẫn thiếu dữ liệu ngắn/giọng nhỏ/nhãn robot thật.

## Phát hành/quay lui

Actions deploy-backend có input audio_preprocess=unchanged/off/shadow/enforce, defaultunchanged; migration chỉ thay mode, giữ config khác/owner/mode và backup/health rollback hiện hữu. Artifact binary/tool/libs/license được kiểm tra đủ trước đổi config/binary. Khi effective mode là shadow/enforce, health chạy /app/audio_frontend_replay -check-native trong container; lỗi nạp/chạy native kích hoạt rollback như health failure. Không gọi workflow ở nhánh tính năng để deploy (chỉproduction). Code mới không tự bật lọc trên VPS.

Sau benchmark/CI/robot đạt mới đề xuất shadow rồi enforce. Nếu mất câu/CPU tăng/lỗi native: Actions mode=off hoặc rollback binary/image nếu crash; giữ gate/capture/firmware/Gipformer. Thư viện mới có tên riêng/pinned, bản cũ không nạp RNNoise. Không xóa audio để quay lui.

## Việc tiếp theo

1. Hoàn tất mixed-noise A/B, đối chiếu baseline và ghi chất lượng theo từng SNR; không trộn calibration/held-out.
2. CI Linux đã đạt; build Docker/release bundle đầy đủ và thêm robot integration/nhãn câu ngắn thật; chưa thể chứng minh từ test synthetic.
3. Xử lý các cửa sổ nền còn lọt mà không hy sinh lời nhỏ/câu ngắn; nếu RNNoise chưa đạt, đánh giá mức lọc/DeepFilterNet theo kế hoạch, không tự cài cả hai trên production.
4. Đo trên VPS dưới tải thực, xem p95 và queue/drop; chỉ công bố số robot đồng thời đã đo.
5. Chỉ merge/deploy khi được yêu cầu sau review; nghiệm thu và lưu run IDs/kết quả thực. Không thay firmware.

## A/B speech+noise hoàn tất — 2026-11-24

Bộ hiệu chỉnh/replay290 mẫu:30 FLEURS gốc,240 bản trộn cùng30 câu với fan/click ở0/5/10/20 dB SNR,20 noise/silence tổng hợp. Denoise và bypass dùng cùng frontend/onset .55/.35/.85, thêm600 ms silence ở EOF để drain DSP; padding báo riêng. ASR Gipformer trên VPS qua SSH tunnel, DSP trên Mac, không gọi LLM/TTS. Chạy hai nhánh benchmark đồng thời nên không dùng ASR wall-time để kết luận latency production.

| Nhóm (30 câu/nhóm) | WER bypass denoise | WER RNNoise | Bỏ sót cả câu RNNoise |
|---|---:|---:|---:|
| Clean |3.64%|4.28%|0/30|
| Fan0 dB |8.57%|9.42%|0/30|
| Fan5 dB |5.14%|5.78%|0/30|
| Fan10 dB |4.28%|4.93%|0/30|
| Fan20 dB |3.85%|4.50%|0/30|
| Click0 dB |7.28%|6.42%|0/30|
| Click5 dB |5.35%|5.57%|0/30|
| Click10 dB |5.57%|4.71%|0/30|
| Click20 dB |3.64%|4.28%|0/30|

20 mẫu không có lời: bypass denoise mở1/20, RNNoise0/20. Điều này chỉ xác nhận tập synthetic ngắn này; không phải tỉ lệ false accept/giờ của môi trường thật. Native/resampler delay15 ms, không tính FIFO32 ms/onset/packet/mạng; DSP RTF gộp~0.029 trên Mac. Không tuyên bố nhiều robot/VPS CPU đã đạt.

Kết luận: RNNoise có lợi trên một số tiếng va chạm và nền, nhưng không luôn cải thiện ASR; mọi nhóm giữ đủ câu, WER tăng nhiều nhất~0.86 điểm phần trăm. Đây là tập calibration có chung30 câu, chưa là kiểm định độc lập, chưa có200 câu ngắn. Không đổi default threshold lên.65 từ chỉ một bản ghi. Giữ off/shadow trước enforce.

Dữ liệu riêng tại /tmp/xiaozhi-noise-mixed-20261124-v2 và /tmp/xiaozhi-gate-audio-20261124 trên máy làm việc; không có transcript/audio trong Git. Bộ v1 bị lỗi scale PCM int16 trong generator, không dùng kết quả; đã sửa scale theo32767 và thêm test kiểm chứng SNR/amplitude, giữ bản cũ để tra cứu thay vì xóa. Khi máy dọn /tmp cần lấy lại capture từ VPS và tạo suite mới; không coi đường dẫn tạm là bản lưu trữ dài hạn.

## Kết thúc lượt code

- Code commits5458b79f và72e2cf03 đã push origin/feat/backend-noise-suppression; chưa PR/merge production/deploy.
- CI cuối45315184231 backend/frontend thành công; native tests/race chọn lọc, benchmark generator/config migration/full-script native-failure rollback, vet/build/self-check đã qua Linux. Các giới hạn vet/race cũ của ASR_INPUT_GATE.md vẫn áp dụng, không tuyên bố race toàn repo sạch.
- VPS kiểm tra read-only vẫn healthy; SHA256 binary10ce67289f12961eda3f85bbb47e6feb4dbd8b800ec0bc4b4bb2546192ac717e không đổi. Chỉ gọi ASR qua tunnel để benchmark, không cài runtime/code mới lên VPS; tunnel đã đóng.
- Người dùng đã được đề nghị cung cấp thời điểm/các câu thử ngắn; chưa có câu trả lời trong lượt này. Chưa gắn nhãn thêm audio hoặc xác nhận nghiệm thu mic thật. Giữ bộ lọc tắt cho đến bước kiểm chứng tiếp theo.

## Phát hành thử robot — 2026-11-24, 10:03–10:05 UTC+7

- Người dùng chủ động yêu cầu merge production/deploy/bật lọc để thử, thay cho chờ đủ bộ nghiệm thu trước enforce. Không thay ngưỡng mặc định thử .55/.35/.85, RNNoise=true; các giá trị này lấy từ code vì runtime chỉ có mode=enforce.
- [PR #17](https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/pull/17) đã merge production5d5cceb62b1a01641a6454fad5255cf9796278b8.
- Actions push45316167741 bị hủy ở Build Linux Binary via Docker trước SCP/deploy. [Actions45316196051](https://github.com/hungnguyenmb/xiaozhi-esp32-server-golang/actions/runs/45316196051) workflow_dispatch thành công, audio_preprocess=enforce, các input audio_diagnostics/audio_capture/asr_input_gate=unchanged.
- Build Docker Linux/native bundle thực tế thành công. Deploy HTTP health và RNNoise/SpeexDSP/TEN native self-check trong container đều PASS. Binary SHA256 mới9ec6839f5edc5b3f98c89bec5d414ead71c997b1f6055a14babd6c5445db86fe; image sha256:d07d60d8d352dc4627a03b399b72b1b230f0199dc189cf811b8c3cdb0b31aa87.
- Runtime enforce; gate enabled/guard300 ms, diagnostics enabled/interval1000 ms, capture enabled/directory như trước. So với backup main_config.yaml.diagnostics-20261124T030315Z.bak: mọi config ngoài audio_preprocess giống hệt, UID/GID10001:10001/mode0600 giữ nguyên.
- Backend healthy/restarts0; ASR/Gipformer, PhoWhisper container dự phòng, TTS và gateway không restart. Admin HTTPS200; verify_tls.py kiểm tra certificate/TLS, OTA HTTPS, WSS hello, auth401, internal route404, redirect308 đều PASS.
- Log probe chỉ thấy setup TEN/auto/asr_auto_end=false; chưa có audio robot thật hoặc AUDIO_PREPROCESS accepted sau phát hành tại thời điểm chốt. Native self-check xác minh thư viện chạy được, không thay thế nghiệm thu VAD/ASR/câu ngắn qua mic vật lý.
- Bản trước: binary10ce67289f12961eda3f85bbb47e6feb4dbd8b800ec0bc4b4bb2546192ac717e, image sha256:e7025db14b55174504542672b83786716aa179346f3d3d880df6711e924082a5. Nếu thử mất giọng/chậm/tự nói nhiều: Actions production audio_preprocess=off (inputs khác unchanged) hoặc rollback code theo DEPLOYMENT_GUIDE.md khi cần; không xóa recordings, không sửa firmware.

Bước tiếp theo: một câu chào rồi im lặng20–30 giây; thử câu ngắn “dừng/tắt/dạ/không”, giọng nhỏ, tiếng quạt/nhiễu và keyword giữa TTS. Ghi timestamp/câu thật để phân loại các accepted/rejected và xem processing_p95_ms/max_queue. Không suy số robot hỗ trợ từ health probe.
