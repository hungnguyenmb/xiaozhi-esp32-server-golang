# Tích hợp PR #271 — 2026-11-23

Trạng thái: PR #6 đã merge production từ nhánh feat/vad-noise-filter-271; CI Linux đạt (45191130321). Đã deploy qua Actions 45194051848; kết quả và trạng thái bật log tại AUDIO_DIAGNOSTICS.md. Giữ Gipformer/LLM/TTS và ngưỡng runtime.

## Nguồn và thay đổi

Nguồn: https://github.com/hackers365/xiaozhi-esp32-server-golang/pull/271 — bốn commit `e8a1a917`, `638224b1`, `3222664d`, `bdd5be37` (head `bdd5be37ef0e2e5dc6d3fe2ed5480b829631ed22`). Áp dụng diff từ merge-base, giải quyết ba xung đột Vue để giữ bản Việt hóa hiện tại.

- Hook lọc kết quả ASR chỉ chứa dấu câu/từ đệm; khởi động lại nhận dạng realtime sau kết quả bị bỏ. Mode 2 chỉ ngắt phát sau khi kết quả qua hook.
- VAD dùng ngưỡng bắt đầu/kết thúc riêng (hysteresis), giữ trạng thái giữa frame; Silero có tiền lọc RMS trước khi phát hiện tiếng nói.
- Giữ các câu ngắn tiếng Việt: “dừng”, “tắt”, “vâng”, “dạ”, “có”, “không”… Chỉ bỏ `hm/hmm/hmmm/uh/um` khi biết thời lượng tiếng nói dưới 300 ms; không bỏ theo độ ngắn của mọi từ tiếng Việt. Log lý do bỏ không ghi transcript.
- Cấu hình cũ thiếu `threshold_low` dùng chính `threshold` ở backend và giao diện, tránh âm thầm đổi hành vi. Reset TEN tạo lại native handle để không mang trạng thái phiên trước sang robot khác trong pool.
- Thêm nhãn/validation tiếng Việt và kiểm thử cấu hình cũ. ID/provider/model giữ nguyên.

## Xác minh

Đã qua trên macOS ARM64:

- Go test sáu package: `internal/domain/vad/{hysteresis,power,ten_vad,silero_vad}`, `internal/domain/chat/hooks`, `internal/app/server/chat`.
- Race test hooks/hysteresis/TEN; vet sáu package và `cmd/server`; build server với `-tags nolibopusfile`.
- Frontend 10 kiểm thử và production build. Cảnh báo bundle lớn hiện có vẫn còn.
- Playwright dùng API giả lập tại localhost: cấu hình cũ high=0.8 tải thành high=low=0.8 ở cả danh sách VAD và wizard; low>high hiện lỗi tiếng Việt. Không gọi API production.

Môi trường Go local dùng `GOTOOLCHAIN=go1.24.4`, `TMPDIR=/private/tmp`, `CGO_CFLAGS=-I/tmp/xiaozhi-ort-test/onnxruntime-linux-x64-1.21.0/include` và `CGO_LDFLAGS='-L/tmp/xiaozhi-ort-test/lib -Wl,-rpath,/tmp/xiaozhi-ort-test/lib'`. Runtime ONNX macOS có sẵn là 1.32.0, khác Linux production 1.21.0. Go 1.26 hệ thống không tương thích dependency sonic hiện tại. Vet toàn cây VAD bị chặn bởi ví dụ/test upstream có main trùng và NewWebRTCVADPool không tồn tại; không sửa ngoài phạm vi.

Workflow `.github/workflows/ci-vad-noise-filter.yml` kiểm tra native Linux ONNX 1.21.0, Go 1.24.11, tests/race/vet/build và frontend; không có bước deploy. Đã chạy thành công trên GitHub (45191130321).

## Trước khi phát hành

1. Push nhánh và yêu cầu CI Linux thành công; review diff trước merge production.
2. Deploy theo GitHub Actions trong DEPLOYMENT_GUIDE.md. Không cài trực tiếp trên VPS.
3. Cấu hình mẫu bật noise filter nhưng mặc định Go tắt nếu thiếu cấu hình. Binary mới không tự bật bộ lọc trên YAML runtime cũ. Khi triển khai phải bổ sung chọn lọc cấu hình `chat_hooks.plugins` theo mẫu trong `config/config.yaml`, giữ các plugin/secrets hiện hữu, không chép đè toàn bộ YAML. Xác minh tên plugin và enabled sau khi nạp cấu hình.
4. Giữ threshold_low=threshold để bắt đầu với hành vi cũ; chỉ hạ ngưỡng kết thúc sau khi đo trên robot thật.
5. Test robot trong phòng yên tĩnh/có nhiễu: câu ngắn tiếng Việt, nói bình thường, ngắt lời mode 2, kết quả bị lọc rồi nói tiếp, nhiều robot đồng thời và pool VAD. Các kiểm thử này chưa thực hiện; hook test chưa thay thế e2e realtime restart.
6. So sánh tỷ lệ tự nói, bỏ sót lời nói, độ trễ ASR và tải CPU. Không coi thay đổi này là tối ưu độ trễ LLM/TTS. Nếu lỗi, rollback bằng quy trình Actions và khôi phục phần cấu hình đã thay.
