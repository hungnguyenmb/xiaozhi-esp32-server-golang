# Bàn giao lọc nhiễu backend — 2026-11-24

## Hiện trạng

Nhánh feat/backend-noise-suppression từ production06ab4ed7, đang kiểm thử. Người dùng yêu cầu triển khai code theo BACKEND_NOISE_SUPPRESSION_PLAN.md. Chưa merge/deploy, runtime VPS giữ nguyên. Đừng tự coi code đã đạt mọi tiêu chí chất lượng.

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

Đây là config thử, chưa chốt ngưỡng production. Chỉ16 kHz mono/TEN/server endpoint; manual/ASR AutoEnd/provider khác giữ đường cũ. Không có section hoặc mode=off thì không nạp DSP; bật/tắt qua restart cho phiên mới. Shadow chỉ log metadata và giữ PCM/quyết định cũ, không tạo thêm chat/LLM/TTS.

AUDIO_PREPROCESS setup/bypass/accepted/interval ghi mode/epoch/ngưỡng/delay native/RMS/processing time và counters; không log transcript/audio. AUDIO_DIAG vẫn theo dõi VAD/endpoint/ASR. Native delay không bao gồm packet buffering, hop FIFO, xác nhận onset hoặc mạng. Chưa có adaptive noise floor, không ghi processed audio song song tự động.

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

- Local macOS ARM native tests full packages liên quan/race chọn lọc/vet/actionlint đạt ở thời điểm ghi; CI Linux còn chờ. Docker daemon local không chạy nên chưa build Docker đầy đủ. Kiểm tra run CI trước merge.
- Suite cũ50 mẫu:30 FLEURS tiếng Việt +20 synthetic noise/silence, không phải50 câu nói sạch. Config thử .55/.35/.85+RNNoise mở đủ30speech, không mở20noise. WER4.28% (20/467) so raw Gipformer4.07% (19/467) đo lại qua SSH tunnel ASR hiện hữu. Raw-ASR-only không có VAD lọc20noise và trả text cả20; script verify vì vậy exit1 sau khi in JSON, đây là kết quả hồi quy tiếng nền chứ không phải benchmark bị mất dữ liệu. Phép so WER này chưa padded EOF, còn Open segments; không dùng như nghiệm thu mất đầu/cuối câu.
- Snapshot robot23 cửa sổ: frontend .55/.35/.85 chưa denoise mở13 lượt; có denoise mở6, gồm câu chào đầu. Thử .65/.40/.90 mở3. Vẫn còn nền nghi ngờ, nhãn “các cửa sổ sau đều noise” chưa được xác nhận độc lập cho phiên mới. Không suy FPR/giờ từ tập ngắn này và không chọn ngưỡng bằng chỉ1 câu chào.
- RTF DSP sơ bộ~0.026 trên Mac, không phảiCPU VPS. ASR gọi trên VPS nhưng DSP chạy local; chưa benchmark1/3/5 luồng DSP+ASR+TTS trên VPS, chưa có p95 latency tổng hợp.
- Chưa đủ200 câu ngắn/nhiều giọng/phòng, chưa gắn nhãn onset/offset, chưa kiểm tra keyword/robot thật trên code mới. Vì vậy chưa đạt tiêu chí enforce production. Benchmark speech+noise đang tiếp tục; cập nhật số cuối trước bàn giao.

## Phát hành/quay lui

Actions deploy-backend có input audio_preprocess=unchanged/off/shadow/enforce, defaultunchanged; migration chỉ thay mode, giữ config khác/owner/mode và backup/health rollback hiện hữu. Artifact libs và license được kiểm tra đủ trước đổi config/binary. Không gọi workflow ở nhánh tính năng để deploy (chỉproduction). Code mới không tự bật lọc trên VPS.

Sau benchmark/CI/robot đạt mới đề xuất shadow rồi enforce. Nếu mất câu/CPU tăng/lỗi native: Actions mode=off hoặc rollback binary/image nếu crash; giữ gate/capture/firmware/Gipformer. Thư viện mới có tên riêng/pinned, bản cũ không nạp RNNoise. Không xóa audio để quay lui.

## Việc tiếp theo

1. Hoàn tất mixed-noise A/B, đối chiếu baseline và ghi chất lượng theo từng SNR; không trộn calibration/held-out.
2. Kiểm tra CI Linux và đóng gói native. Thêm robot integration/nhãn câu ngắn thật; chưa thể chứng minh từ test synthetic.
3. Xử lý các cửa sổ nền còn lọt mà không hy sinh lời nhỏ/câu ngắn; nếu RNNoise chưa đạt, đánh giá mức lọc/DeepFilterNet theo kế hoạch, không tự cài cả hai trên production.
4. Đo trên VPS dưới tải thực, xem p95 và queue/drop; chỉ công bố số robot đồng thời đã đo.
5. Chỉ merge/deploy khi được yêu cầu sau review; nghiệm thu và lưu run IDs/kết quả thực. Không thay firmware.
