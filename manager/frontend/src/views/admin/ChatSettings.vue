<template>
  <div class="config-page">
    <div class="page-actions">
      <el-button @click="loadSettings" :loading="loading">Tải lại</el-button>
      <el-button type="primary" @click="saveSettings" :loading="saving">Lưu cài đặt</el-button>
    </div>

    <el-card v-loading="loading">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="180px" style="max-width: 720px;">
        <el-divider content-position="left">Xác thực</el-divider>
        <el-form-item label="Kích hoạt xác minh kích hoạt thiết bị" prop="auth.enable">
          <el-switch v-model="form.auth.enable" />
        </el-form-item>
        <el-form-item label="Đăng nhập xác minh kỹ thuật số" prop="auth.login_captcha_enabled">
          <el-switch
            v-model="form.auth.login_captcha_enabled"
            active-text="bật"
            inactive-text="Đóng"
          />
          <div class="form-help">
            Sau khi bật trang đăng nhập, bạn cần hoàn thành các câu hỏi số học; sau khi tắt, trang đăng nhập chỉ xác minh tên người dùng và mật khẩu. Được bật theo mặc định.
          </div>
        </el-form-item>

        <el-divider content-position="left">Thông số trò chuyện</el-divider>
        <el-form-item label="Thời gian nhàn rỗi tối đa của phiên  (ms)" prop="chat.max_idle_duration">
          <el-input-number v-model="form.chat.max_idle_duration" :min="0" :step="1000" style="width: 100%;" />
          <div class="form-help">
            đơn vị mili giây. Đặt nó thành 0 nghĩa là không có giới hạn về thời gian nhàn rỗi của phiên (nó sẽ không tự động bị ngắt kết nối do thời gian nhàn rỗi). Giá trị đề xuất: 30000 ~ 120000.
          </div>
        </el-form-item>
        <el-form-item label="Ngưỡng im lặng cuối câu (ms)" prop="chat.chat_max_silence_duration">
          <el-input-number v-model="form.chat.chat_max_silence_duration" :min="0" :step="10" style="width: 100%;" />
          <div class="form-help">
            được sử dụng để xác định phần cuối của câu: sau khi quá trình chuyển từ "âm thanh" sang "im lặng" tiếp tục đạt đến ngưỡng này, câu được coi là kết thúc và quá trình xử lý tiếp theo được kích hoạt. Mặc định 400ms. Ngưỡng càng nhỏ thì phản hồi càng nhanh nhưng càng dễ bị cắt. Ngưỡng càng lớn thì phản hồi càng ổn định nhưng càng chậm. Khuyến nghị 300 ~ 600ms.
          </div>
        </el-form-item>
        <el-form-item label="Chế độ ngắt thời gian thực" prop="chat.realtime_mode">
          <el-select v-model="form.chat.realtime_mode" style="width: 100%;">
            <el-option :value="1" label="1 - chế độ ngắt vad" />
            <el-option :value="2" label="2 - chế độ ngắt asr" />
            <el-option :value="3" label="3 - asr bị gián đoạn khi nhận diện người nói" />
            <el-option :value="4" label="4 - kết quả asr bị gián đoạn" />
          </el-select>
        </el-form-item>
        <el-form-item label="Mô tả lời nhắc hệ thống dùng chung" prop="chat.global_system_prompt">
          <el-input
            v-model="form.chat.global_system_prompt"
            type="textarea"
            :rows="6"
            maxlength="8000"
            show-word-limit
            placeholder="Nội dung này sẽ được ghép ở phía trước dấu nhắc hệ thống. Bạn nên điền vào các ràng buộc ở cấp độ nền tảng và cài đặt nhận dạng."
          />
          <div class="form-help">
            Thứ tự hiệu quả: Mô tả lời nhắc hệ thống dùng chung → Lời nhắc vai trò/thiết bị → thông tin thời gian chạy như thời gian/bộ nhớ.
          </div>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'

const loading = ref(false)
const saving = ref(false)
const formRef = ref()

const form = reactive({
  auth: {
    enable: false,
    login_captcha_enabled: true
  },
  chat: {
    max_idle_duration: 30000,
    chat_max_silence_duration: 400,
    realtime_mode: 4,
    global_system_prompt: ''
  }
})

const rules = {
  'chat.max_idle_duration': [
    { required: true, message: 'Vui lòng nhập thời gian rảnh tối đa của phiên', trigger: 'blur' }
  ],
  'chat.chat_max_silence_duration': [
    { required: true, message: 'Vui lòng nhập ngưỡng im lặng cuối câu', trigger: 'blur' }
  ],
  'chat.realtime_mode': [
    { required: true, message: 'Vui lòng chọn chế độ ngắt thời gian thực', trigger: 'change' }
  ],
  'chat.global_system_prompt': [
    { max: 8000, message: 'Mô tả lời nhắc hệ thống dùng chung không thể vượt quá 8000 ký tự.', trigger: 'blur' }
  ]
}

const loadSettings = async () => {
  loading.value = true
  try {
    const res = await api.get('/admin/chat-settings')
    const data = res.data?.data || {}
    form.auth.enable = !!data.auth?.enable
    form.auth.login_captcha_enabled = data.auth?.login_captcha_enabled !== false
    form.chat.max_idle_duration = Number(data.chat?.max_idle_duration ?? 30000)
    form.chat.chat_max_silence_duration = Number(data.chat?.chat_max_silence_duration ?? 400)
    form.chat.realtime_mode = Number(data.chat?.realtime_mode ?? 4)
    form.chat.global_system_prompt = String(data.chat?.global_system_prompt ?? '')
  } catch (error) {
    ElMessage.error('Không tải được cài đặt trò chuyện')
    console.error(error)
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    await api.put('/admin/chat-settings', {
      auth: {
        enable: !!form.auth.enable,
        login_captcha_enabled: form.auth.login_captcha_enabled !== false
      },
      chat: {
        max_idle_duration: Number(form.chat.max_idle_duration),
        chat_max_silence_duration: Number(form.chat.chat_max_silence_duration),
        realtime_mode: Number(form.chat.realtime_mode),
        global_system_prompt: String(form.chat.global_system_prompt || '')
      }
    })
    ElMessage.success('Cài đặt trò chuyện  đã được lưu thành công')
  } catch (error) {
    ElMessage.error('Không lưu được cài đặt trò chuyện')
    console.error(error)
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.config-page {
  padding: 20px;
}

.page-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.form-help {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
}
</style>
