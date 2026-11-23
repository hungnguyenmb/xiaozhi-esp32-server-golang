# Chẩn đoán nghi thu lại tiếng loa — 2026-11-23

Phạm vi: chỉ đọc log VPS và đối chiếu mã nguồn; không đổi cấu hình, không deploy. Dữ liệu robot thật khoảng 21:38–21:39, giờ Việt Nam. Không chép raw log/transcript/audio cá nhân vào Git.

## Bằng chứng

Log mới đã có window/endpoint/asr trên robot thật (khác smoke test chưa kích hoạt trước đó). Provider TEN VAD, auto listen, asr_auto_end=false, realtime_mode=4.

| Backend hoàn tất gửi TTS stop | Client mở lại listen | Khoảng cách | Đoạn kế tiếp |
| --- | --- | --- | --- |
| 21:38:15.542 | 21:38:15.622 | 80 ms | endpoint 21:38:16.378, voice=120 ms; ASR 8 ký tự, hook không chặn |
| 21:39:26.864 | 21:39:26.953 | 89 ms | endpoint 21:39:27.800, voice=180 ms; ASR 3 ký tự, hook không chặn |
| 21:39:46.374 | 21:39:46.444 | 70 ms | endpoint 21:39:47.161, voice=120 ms; ASR 1 ký tự, hook không chặn |

Trong 9 chu kỳ quan sát, TTS stop→listen start khoảng 63–101 ms. Một số frame đầu sau mở mic có score TEN khoảng 0.49–0.54; sau đó VAD báo đoạn tiếng nói ngắn và ASR sinh chữ, tạo thêm phản hồi. Các đoạn 60 ms đã có short_voice_rejected, nhưng 120–180 ms vẫn qua. Idle tại nhiều endpoint là 540 ms; đây là số đo, không suy ra ngưỡng runtime chỉ từ số này.

Kết luận: có dấu hiệu mạnh của tái kích hoạt ngay sau phát loa, phù hợp tiếng đuôi loa/vọng âm hoặc audio buffer còn sót. Chưa đủ bằng chứng xác nhận âm thanh ASR chính là câu robot vừa phát, hoặc kết luận AEC firmware bị tắt: không có audio/reference playback để đối chiếu. Backend gửi TTS stop không phải phép đo chính xác thời điểm DAC/loa vật lý hết phát.

## Hạn chế phát hiện trong instrumentation

`OnVoiceSilence` (internal/data/client/client.go) gọi `state.Vad.Reset()`, xóa VoiceDurationInSession trước khi nhận kết quả offline. Vì vậy AUDIO_DIAG asr.voice_ms=0 không có nghĩa mic không có tiếng; phải dùng endpoint.voice_ms. ASRNoiseFilter nhận cùng giá trị 0 nên điều kiện lọc filler dựa trên thời lượng cũng thiếu dữ liệu. Log result.asr_ms được lấy trước MarkAsrFinalTextAt nên thường 0; số sau hook/statistic plugin hữu ích hơn. Cần giữ snapshot duration/time theo từng lượt trước reset, tránh dùng biến toàn cục trộn lượt.

Runtime chưa bật asr_noise_filter; đây là chủ ý giữ baseline chẩn đoán, không phải lỗi deploy. Dù bật filter hiện tại, nó không phải bộ khử vọng và không đủ chặn mọi từ thật bị ASR nhận nhầm.

## Việc tiếp theo

1. Đo A/B: cùng phòng không ai nói, giảm âm lượng loa và lặp lại; đối chiếu số lượt phát sinh ngay sau playback. Kiểm tra firmware AEC và đường reference loa/mic; không khẳng định AEC bật/tắt dựa trên log server.
2. Với auto mode hiện tại, thử khoảng bảo vệ sau playback khoảng 300–500 ms, nhưng phải xử lý rõ mốc client báo phát xong và buffer audio/VAD; không chỉ sleep rồi phát lại audio tồn. Đánh giá nguy cơ mất đầu câu người nói ngay sau robot. Đây là đề xuất, chưa triển khai.
3. Sửa snapshot thời lượng/metric theo turn và bổ sung kiểm thử trước dùng số liệu để lọc. Không chặn toàn bộ câu ngắn hoặc tăng ngưỡng toàn cục thiếu đo lường.
4. Nếu cần ngắt lời khi robot đang nói, ưu tiên AEC thực sự; khoảng chặn mic chỉ là đánh đổi cho chế độ luân phiên nói/nghe.
5. Mọi thay đổi code triển khai tiếp tục qua nhánh/CI/production Actions như DEPLOYMENT_GUIDE.md; giữ Gipformer/LLM/TTS.
