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

## Lượt đối chiếu audio thật 22:07–22:08

Chỉ đọc log/archive, không thay code/config hoặc deploy. Connection `20261123T150712Z-2034208276`. Snapshot riêng tải về local để giải mã, không đưa raw audio/JSONL/WAV lên Git; file nguồn trên VPS giữ nguyên. Snapshot chưa có close marker (phiên chưa đóng), không coi bản chụp là toàn bộ phiên đã kết thúc.

- Snapshot: 823 mic + 203 speaker packets, giải mã đúng 960 mẫu/packet ở 16 kHz mono (60 ms), sequence liên tục không thiếu trong snapshot. Không thấy AUDIO_CAPTURE missing/writer failed trong log đã đọc; vì chưa close nên chưa có tổng dropped cuối phiên.
- Xuất timeline và compact WAV cả hai hướng. Các file phân tích private tại `/tmp/xiaozhi-echo-analysis-20261123/`; không tự xóa dữ liệu.
- Năm khoảng từ frame speaker đầu đến frame speaker cuối không có mic packet tới server. Khoảng giữa tts_start/tts_stop có 1–2 mic packet trước output bắt đầu, nên không được nhầm tts_start với lúc loa đã có tiếng.
- Sau tts_stop, mic packet đầu tới lại sau khoảng 201–232 ms. Không biết ADC/DAC thực tế; không kết luận mic phần cứng tắt chỉ vì server không nhận packet.

| Giờ Việt Nam | Dữ liệu quan sát |
| --- | --- |
| 22:07:28.535 kết thúc TTS | 22:07:29.665 endpoint voice=420 ms, ASR 3 ký tự lúc 29.711 |
| 22:07:35.284 kết thúc TTS | 22:07:36.073 endpoint voice=120 ms, ASR 1 ký tự lúc 36.116 |
| 22:07:40.197 kết thúc TTS | 22:07:41.799 endpoint voice=540 ms, ASR 3 ký tự lúc 41.860 |
| 22:07:46.558 kết thúc TTS | Chỉ 60 ms bị short_voice_rejected lúc 46.841, không tự tạo ASR ở đoạn này |

Có chuỗi lượt sau phát loa đáng nghi, nhưng không có chú thích thời điểm người dùng thực sự nói nên không gắn nhãn mọi lượt là false positive. ASR voice_ms=0 vẫn do hạn chế reset nêu trên; dùng endpoint.voice_ms.

So sánh sơ bộ waveform: giải mã compact để ánh xạ packet, so mic 1 giây sau tts_stop với 1,5 giây output cuối trước đó. Tương quan raw ngắn có thể tăng giả do DC/tần số thấp. Sau hai tầng high-pass 200 Hz/low-pass 3500 Hz đơn giản, max tương quan chuẩn hóa theo cửa sổ 180 ms ở năm lượt khoảng 0,19–0,38. Đây là phép dò sơ bộ, chưa có mẫu đối chứng/hiệu chuẩn phòng; không chứng minh nội dung mic chính là lời robot, cũng không loại trừ echo sau lọc/codec/reverb. Không dùng chỉ số này như ngưỡng tự động.

Ví dụ 22:07:46, frame đầu RMS khoảng 0,088 rồi frame kế 0,0087: có xung mở đầu ngắn nhưng chưa phân biệt được tiếng đuôi loa/động tác/AGC/đường mic chỉ từ dữ liệu này. Không kết luận firmware AEC bị tắt hoặc có lỗi ADC.

Bước tiếp có giá trị: A/B cùng phòng không ai nói, cùng câu robot, âm lượng loa bình thường so với thấp; ghi mốc thực nghiệm rõ ràng. Sau đó quyết định guard sau mở mic và kiểm tra AEC/reference trong firmware. Chưa sửa threshold, chưa thêm guard, chưa thay ASR/LLM/TTS.

## Người dùng xác nhận nhãn thực nghiệm

Sau khi xem báo cáo, người dùng xác nhận: chỉ nói câu đầu tiên “xin chào”; mọi lượt sau đều là robot tự nói/tự trả lời hoặc nhận nhầm tiếng động. Thông tin này thay thế điểm chưa rõ trước đó về việc người dùng có nói tiếp hay không.

Do đó có thể xác nhận false-positive hội thoại trong phiên 22:07: sau lượt đầu, các đoạn 420 ms/120 ms/540 ms sinh chữ rồi tạo thêm LLM/TTS không phải lời người dùng mới. Không chỉ là tương quan thời điểm đáng nghi. Chưa thể gắn nhãn mọi đoạn đó là echo giọng loa: nguyên nhân âm học vẫn có thể là tiếng động, transient khi mở mic, tiếng đuôi loa hoặc tổ hợp.

Ưu tiên bước sửa tiếp theo: giữ snapshot thời lượng theo turn trước reset (khắc phục voice_ms=0), thêm cơ chế bảo vệ lúc auto-listen mở lại sau playback với xử lý buffer/VAD rõ ràng và kiểm thử không mất câu nói thật, đo A/B âm lượng/AEC. Không coi chỉ bật noise-filter hiện tại hay chặn từ “đây” là giải pháp đầy đủ. Đây là hướng đề xuất, chưa triển khai/deploy trong lượt xác nhận này.
