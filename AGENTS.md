# Hướng dẫn tiếp tục dự án Xiaozhi Go

- Đọc `DEPLOYMENT_GUIDE.md` trước mọi thay đổi liên quan triển khai. Người dùng yêu cầu deploy qua GitHub Actions; không deploy thủ công để bỏ qua pipeline. Deploy VPS từ nhánh `production`; nhánh tính năng dùng phát triển/CI.
- Công việc PhoWhisper đang ở nhánh `feat/phowhisper-asr`. Đọc `deployment/vps/PHOWHISPER_PLAN.md` để biết hiện trạng, file dự kiến, checklist, tiêu chí đo và rollback trước khi code.
- Kế hoạch là tài liệu bàn giao; không tự coi các bước chưa làm là đã được kiểm thử hoặc đã được phép phát hành production.
- Giữ Gipformer để quay lui. Không đổi firmware, LLM, TTS, dữ liệu thiết bị hoặc tài khoản ngoài phạm vi ASR.
- Cập nhật tiến độ thực và giới hạn kiểm chứng trong kế hoạch/task khi bàn giao cho agent khác. Không đưa secrets, model weights hoặc audio cá nhân lên Git.
- Quyết định ngày 2026-11-23: giữ Gipformer mặc định sau khi thử PhoWhisper-base/small. Đọc `deployment/vps/PHOWHISPER_EVALUATION_20261123.md` trước khi đề xuất activate; cả hai chưa tốt hơn trên VPS hiện tại.

- Chẩn đoán mic/VAD/ASR: đọc `deployment/vps/AUDIO_DIAGNOSTICS.md`. Log metadata bật qua chat.audio_diagnostics, mặc định mã nguồn tắt; trạng thái deploy/bật runtime ghi tại AUDIO_DIAGNOSTICS.md, chưa nghiệm thu robot thật.

- Ghi âm đầy đủ: đọc `deployment/vps/FULL_AUDIO_CAPTURE.md`. Người dùng cho phép lưu toàn bộ mic/output server nhận/gửi; không tự xóa recordings, không đưa audio/WAV vào Git. Kiểm tra trạng thái bật runtime trong tài liệu; không nhầm packet nhận được với ghi mic phần cứng 24/7.

- Ngày 2026-11-23 đã bật audio_capture trên VPS qua Actions 45199579444, probe packet/WAV đạt. Dữ liệu riêng nằm runtime/storage/audio-captures; không tự xóa/đưa lên Git. Đã có phân tích audio thật và xác nhận false positives sau câu đầu; đọc ECHO_DIAGNOSIS_20261123.md và FULL_AUDIO_CAPTURE.md.

- Kế hoạch ngắt lời bằng keyword cục bộ và sửa auto-listen: `deployment/vps/WAKE_WORD_INTERRUPT_PLAN.md`. Người dùng xác nhận firmware zero-one và chọn sửa backend trước; xem mục điều chỉnh phạm vi ở đầu kế hoạch, không tự sửa/flash firmware.

- Quyết định mới: giữ nguyên firmware zero-one, ưu tiên sửa Go backend vì người dùng xác nhận firmware chạy tốt với server Xiaozhi. Phần firmware trong WAKE_WORD_INTERRUPT_PLAN.md đã hoãn; đọc mục điều chỉnh phạm vi ở đầu kế hoạch. Không tự build/flash firmware cho đợt này.

- Backend ASR gate: đọc `deployment/vps/ASR_INPUT_GATE.md`. Đã deploy backend gate qua Actions 45226377103, giữ firmware; frame có epoch, auto-resume guard và snapshot trước reset. Chưa coi hiệu quả robot thật đã được nghiệm thu; trạng thái phát hành ghi trong tài liệu này.

- Backend gate hiện enabled=true/post_tts_guard_ms=300, capture/diagnostics giữ bật. CI/health/probe đạt, robot thật sau rollout còn chờ; đọc ASR_INPUT_GATE.md và không tự flash firmware/đổi model.

- Kế hoạch lọc nhiễu backend ngày2026-11-24: deployment/vps/BACKEND_NOISE_SUPPRESSION_PLAN.md. Chưa code/deploy; giữ câu ngắn/lời nhỏ và speech+noise là tiêu chí bắt buộc. Đọc trước khi điều chỉnh VAD hoặc thêm denoiser; không chốt threshold0.6 từ replay đơn giản.

- Đang triển khai nhánh feat/backend-noise-suppression: đọc deployment/vps/BACKEND_NOISE_SUPPRESSION.md (cấu hình off/shadow/enforce, native build, replay, số đo và phần chưa đạt) cùng kế hoạch. Chưa merge/deploy; không tự bật enforce hoặc đổi firmware.

- Code lọc nhiễu đã push feat/backend-noise-suppression (72e2cf03), CI Linux45315184231 đạt. Chưa merge/deploy; còn nghiệm thu câu ngắn/giọng nhỏ/robot thật và CPU VPS. Xem BACKEND_NOISE_SUPPRESSION.md trước tiếp tục; mặc định lọc vẫn off.

- Runtime mới nhất2026-11-24: PR17 merge production5d5cceb6, Actions45316196051 deploy thành công/bật audio_preprocess=enforce theo yêu cầu người dùng để thử robot. Gate300 ms/diagnostics/capture giữ bật; health/native/OTA/WSS/admin đạt. Chưa nghiệm thu mic thật. Đọc mục phát hành cuối BACKEND_NOISE_SUPPRESSION.md trước check log/tuning; các ghi chú off/chưa deploy phía trên là lịch sử.

- Kiểm tra mic thật sau enforce: deployment/vps/NOISE_ROBOT_TEST_20261124.md. Native chạy nhưng còn false positives; AlignToolMessages nhân đôi assistant khi2 tool calls gây OpenAI400, handler còn đọc lỗi kỹ thuật ra loa. Lượt check log chưa sửa code/config; ưu tiên xử lý hai lỗi riêng, không blacklist từ ngắn.

- Đang sửa nhánh fix/tool-history-error-speech: TOOL_HISTORY_ERROR_FIX.md ghi regression, contiguous tool-round normalization, UUID32 cho persistence và silent provider errors. Người dùng đã yêu cầu deploy qua Actions; không đổi noise frontend/firmware. Chờ trạng thái CI/deploy trong tài liệu trước kết luận đã phát hành.

- Mới nhất2026-11-24: PR19/production27e0b975, Actions45322232698 đã deploy sửa tool-history/UUID32/silent provider errors. CI45321517370/45321746615 đạt, health/native/TLS/OTA/WSS/admin đạt; config không đổi (noise=enforce/gate300 ms/capture/diagnostics). Đọc TOOL_HISTORY_ERROR_FIX.md trước tiếp tục; chờ test robot đa tool thật, false-positive audio vẫn chưa sửa.
