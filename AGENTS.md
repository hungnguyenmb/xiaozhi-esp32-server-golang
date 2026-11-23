# Hướng dẫn tiếp tục dự án Xiaozhi Go

- Đọc `DEPLOYMENT_GUIDE.md` trước mọi thay đổi liên quan triển khai. Người dùng yêu cầu deploy qua GitHub Actions; không deploy thủ công để bỏ qua pipeline. Deploy VPS từ nhánh `production`; nhánh tính năng dùng phát triển/CI.
- Công việc PhoWhisper đang ở nhánh `feat/phowhisper-asr`. Đọc `deployment/vps/PHOWHISPER_PLAN.md` để biết hiện trạng, file dự kiến, checklist, tiêu chí đo và rollback trước khi code.
- Kế hoạch là tài liệu bàn giao; không tự coi các bước chưa làm là đã được kiểm thử hoặc đã được phép phát hành production.
- Giữ Gipformer để quay lui. Không đổi firmware, LLM, TTS, dữ liệu thiết bị hoặc tài khoản ngoài phạm vi ASR.
- Cập nhật tiến độ thực và giới hạn kiểm chứng trong kế hoạch/task khi bàn giao cho agent khác. Không đưa secrets, model weights hoặc audio cá nhân lên Git.
- Quyết định ngày 2026-11-23: giữ Gipformer mặc định sau khi thử PhoWhisper-base/small. Đọc `deployment/vps/PHOWHISPER_EVALUATION_20261123.md` trước khi đề xuất activate; cả hai chưa tốt hơn trên VPS hiện tại.
