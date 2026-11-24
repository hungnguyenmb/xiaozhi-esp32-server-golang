import random
import unittest

from benchmark_audio_frontend import mix, score


class BenchmarkTest(unittest.TestCase):
    def test_missing_short_speech_counts_as_deletions(self):
        result = score([{"id": "a", "reference": "không"}], [{"ID": "a",
            "Accepted": 0, "Segments": [], "ProcessingMs": 1, "AudioMs": 100}])
        self.assertEqual(result["speech"]["missed_speech"], 1)
        self.assertEqual(result["speech"]["wer"], 1)

    def test_noise_acceptance_is_not_word_accuracy(self):
        result = score([{"id": "a", "reference": ""}], [{"ID": "a",
            "Accepted": 1, "Segments": [], "ProcessingMs": 1, "AudioMs": 100}])
        self.assertEqual(result["noise"]["accepted_noise"], 1)

    def test_joint_scaling_does_not_clip(self):
        pcm = [30000, -30000] * 800
        a = mix(pcm, 16000, 0, "fan", random.Random(1))
        b = mix(pcm, 16000, 0, "fan", random.Random(1))
        self.assertEqual(a, b)
        self.assertEqual(len(a), len(pcm))
        self.assertLessEqual(max(abs(x) for x in a), 32767)


if __name__ == "__main__":
    unittest.main()
