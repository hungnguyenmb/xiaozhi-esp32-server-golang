<template>
  <el-button
    :type="isPlaying ? 'warning' : 'primary'"
    :loading="loading"
    size="default"
    :disabled="!voice"
    @click="handleTogglePlay"
  >
    <el-icon v-if="!loading" style="margin-right: 4px">
      <VideoPause v-if="isPlaying" />
      <VideoPlay v-else />
    </el-icon>
    <span>{{ isPlaying ? 'Tạm dừng' : 'Nghe thử' }}</span>
  </el-button>
</template>

<script setup>
import { computed, ref, onBeforeUnmount, watch } from 'vue'
import { VideoPlay, VideoPause } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '@/utils/api'
import { translateServerMessage } from '@/locales/server'
import { extractPreviewError, previewPlaybackIdentity } from './voicePreview'

const props = defineProps({
  provider: { type: String, default: 'aliyun_qwen' },
  configId: { type: String, default: '' },
  model: { type: String, default: '' },
  voice: { type: String, default: '' },
  instruction: { type: String, default: '' },
  languageType: { type: String, default: '' },
  apiKey: { type: String, default: '' },
  apiUrl: { type: String, default: '' },
  region: { type: String, default: '' },
  text: { type: String, default: '' },
  isAdmin: { type: Boolean, default: false }
})

const loading = ref(false)
const isPlaying = ref(false)
let audioObj = null
let audioObjectUrl = ''
let requestVersion = 0
let requestController = null

const stopAudio = () => {
  if (audioObj) {
    audioObj.pause()
    audioObj.currentTime = 0
    audioObj = null
  }
  if (audioObjectUrl) {
    URL.revokeObjectURL(audioObjectUrl)
    audioObjectUrl = ''
  }
  isPlaying.value = false
}

const playbackIdentity = computed(() => previewPlaybackIdentity(props))
const invalidatePlayback = () => {
  requestVersion += 1
  if (requestController) {
    requestController.abort()
    requestController = null
  }
  loading.value = false
  stopAudio()
}
watch(playbackIdentity, invalidatePlayback)

const handleTogglePlay = async () => {
  if (isPlaying.value) {
    stopAudio()
    return
  }

  if (!props.voice) {
    ElMessage.warning('Vui lòng chọn hoặc nhập giọng đọc trước')
    return
  }

  loading.value = true
  const currentRequestVersion = ++requestVersion
  const currentController = new AbortController()
  requestController = currentController
  try {
    const endpoint = props.isAdmin ? '/admin/tts/preview' : '/user/tts/preview'
    const payload = {
      provider: props.provider,
      config_id: props.configId,
      model: props.model,
      voice: props.voice,
      instruction: props.instruction,
      language_type: props.languageType,
      text: props.text || 'Xin chào, mình là Xiaozhi. Đây là giọng đọc bạn đang nghe thử.'
    }
    if (props.isAdmin) {
      payload.api_key = props.apiKey
      payload.api_url = props.apiUrl
      payload.region = props.region
    }
    const response = await api.post(
      endpoint,
      payload,
      {
        responseType: 'blob',
        timeout: 35000,
        signal: currentController.signal
      }
    )
    if (currentRequestVersion !== requestVersion) return

    const blob = new Blob([response.data], { type: response.headers['content-type'] || 'audio/mpeg' })
    const audioUrl = URL.createObjectURL(blob)

    stopAudio()
    audioObjectUrl = audioUrl
    const currentAudio = new Audio(audioUrl)
    audioObj = currentAudio
    audioObj.onended = () => {
      isPlaying.value = false
      stopAudio()
    }
    audioObj.onerror = () => {
      ElMessage.error('Không thể phát âm thanh nghe thử')
      isPlaying.value = false
      stopAudio()
    }

    await audioObj.play()
    if (currentRequestVersion !== requestVersion || audioObj !== currentAudio) {
      stopAudio()
      return
    }
    isPlaying.value = true
  } catch (err) {
    if (currentRequestVersion !== requestVersion) return
    stopAudio()
    const msg = await extractPreviewError(err)
    ElMessage.error(translateServerMessage(msg))
  } finally {
    if (currentRequestVersion === requestVersion) {
      requestController = null
      loading.value = false
    }
  }
}

onBeforeUnmount(() => {
  invalidatePlayback()
})
</script>
