<template>
  <div class="mqtt-server-config">
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      class="config-form"
      v-loading="loading"
    >
      <div class="config-layout">
        <el-card class="config-card config-card-main" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">MQTT Server</p>
                <h3>Giám sát và truy cập</h3>
                <p class="card-description">định cấu hình địa chỉ nghe và trạng thái bật của Máy chủ MQTT tích hợp để truy cập thiết bị và chương trình chính.</p>
              </div>
              <el-tag :type="serverReady ? 'success' : 'warning'" effect="plain" round>
                {{ serverReady ? 'Thông số dịch vụ hoàn chỉnh của' : 'được thêm vào' }}
              </el-tag>
            </div>
          </template>

          <div class="field-grid">
            <el-form-item label="Trạng thái kích hoạt" prop="enable">
              <div class="switch-field">
                <div>
                  <div class="switch-title">kích hoạt Máy chủ MQTT tích hợp</div>
                  <div class="field-help">sẽ không còn nghe các kết nối MQTT phía thiết bị sau khi đóng.</div>
                </div>
                <el-switch v-model="form.enable" />
              </div>
            </el-form-item>

            <el-form-item label="Máy chủ nghe" prop="listen_host">
              <el-input v-model="form.listen_host" placeholder="Ví dụ: 0.0.0.0" />
            </el-form-item>

            <el-form-item label="Cổng nghe" prop="listen_port">
              <el-input-number v-model="form.listen_port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
            </el-form-item>
          </div>
        </el-card>

        <div class="side-stack">
          <el-card class="config-card" shadow="never">
            <template #header>
              <div class="card-head">
                <div>
                  <p class="card-kicker">Authentication</p>
                  <h3>Chứng nhận và chữ ký</h3>
                  <p class="card-description">Nếu xác thực được bật, vui lòng điền vào tài khoản quản trị viên được chương trình chính sử dụng để kết nối với Máy chủ MQTT và giữ khóa ký nhất quán với OTA.</p>
                </div>
                <el-tag :type="form.enable_auth ? 'warning' : 'info'" effect="plain" round>
                  {{ form.enable_auth ? 'Đã bật xác thực' : 'Truy cập ẩn danh' }}
                </el-tag>
              </div>
            </template>

            <div class="field-stack">
              <el-form-item label="kích hoạt xác thực" prop="enable_auth">
                <div class="switch-field">
                  <div>
                    <div class="switch-title">Xác minh tên người dùng và mật khẩu MQTT</div>
                    <div class="field-help">Khi được bật, tên người dùng và mật khẩu sẽ được xác minh khi máy khách kết nối.</div>
                  </div>
                  <el-switch v-model="form.enable_auth" />
                </div>
              </el-form-item>

              <el-form-item label="Người dùng quản trị viên" prop="username">
                <el-input v-model="form.username" placeholder="Xác thực  chưa được bật và có thể tạm thời để trống." />
              </el-form-item>

              <el-form-item label="Mật khẩu quản trị viên" prop="password">
                <el-input v-model="form.password" type="password" placeholder="Xác thực  chưa được bật và có thể tạm thời để trống." show-password />
              </el-form-item>

              <el-form-item label="Khóa ký" prop="signature_key">
                <el-input v-model="form.signature_key" placeholder="Vui lòng nhập key ký phù hợp với OTA" />
                <div class="field-help">Khóa này cần nhất quán với khóa ký trên trang cấu hình OTA.</div>
              </el-form-item>
            </div>
          </el-card>

          <el-card class="config-card" shadow="never">
            <template #header>
              <div class="card-head">
                <div>
                  <p class="card-kicker">MQTTS</p>
                  <h3>Cấu hình  TLS</h3>
                  <p class="card-description">Khi thiết bị cần được kết nối thông qua MQTTS, hãy bật TLS và hoàn thành đường dẫn tệp chứng chỉ.</p>
                </div>
                <el-tag :type="form.tls.enable ? 'success' : 'info'" effect="plain" round>
                  {{ form.tls.enable ? 'Đã bật  TLS' : 'TLS chưa được kích hoạt' }}
                </el-tag>
              </div>
            </template>

            <div class="field-stack">
              <el-form-item label="Kích hoạt TLS" prop="tls.enable">
                <div class="switch-field">
                  <div>
                    <div class="switch-title">cho phép thiết bị kết nối qua MQTTS</div>
                    <div class="field-help">Sau khi  được bật, vui lòng điền cổng TLS, tệp chứng chỉ và tệp khóa cùng lúc.</div>
                  </div>
                  <el-switch v-model="form.tls.enable" />
                </div>
              </el-form-item>

              <el-form-item label="Cổng TLS" prop="tls.port">
                <el-input-number v-model="form.tls.port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
              </el-form-item>

              <el-form-item label="Tệp chứng chỉ" prop="tls.pem">
                <el-input v-model="form.tls.pem" placeholder="Ví dụ: certs/server.pem" />
              </el-form-item>

              <el-form-item label="Tệp khóa" prop="tls.key">
                <el-input v-model="form.tls.key" placeholder="Ví dụ: certs/server.key" />
              </el-form-item>
            </div>
          </el-card>
        </div>
      </div>

      <div class="footer-bar">
        <p class="footer-note">
          sẽ cập nhật cấu hình MQTT Server mặc định sau khi lưu; nếu phân phối MQTT của OTA cũng được bật, vui lòng xác nhận rằng các khóa ký là nhất quán.
        </p>
        <div class="footer-actions">
          <el-button plain :loading="loading" @click="loadConfig">đặt lại cấu hình hiện tại</el-button>
          <el-button type="primary" :loading="saving" @click="handleSave">Lưu cấu hình</el-button>
        </div>
      </div>
    </el-form>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'

const loading = ref(false)
const saving = ref(false)
const configId = ref(null)
const formRef = ref(null)

const createDefaultFormState = () => ({
  enable: true,
  listen_host: '0.0.0.0',
  listen_port: 1883,
  username: '',
  password: '',
  signature_key: 'xiaozhi_ota_signature_key',
  enable_auth: false,
  tls: {
    enable: false,
    port: 8883,
    pem: '',
    key: ''
  }
})

const form = reactive(createDefaultFormState())

const validateUsername = (_, value, callback) => {
  if (form.enable_auth && !String(value || '').trim()) {
    callback(new Error('Tên người dùng quản trị viên không được để trống khi xác thực được bật.'))
    return
  }
  callback()
}

const validatePassword = (_, value, callback) => {
  if (form.enable_auth && !String(value || '').trim()) {
    callback(new Error('Mật khẩu quản trị viên không thể để trống khi xác thực được bật.'))
    return
  }
  callback()
}

const validateTlsPort = (_, value, callback) => {
  if (form.tls.enable && (!value || value < 1 || value > 65535)) {
    callback(new Error('Khi bật TLS, số cổng phải nằm trong khoảng 1-65535'))
    return
  }
  callback()
}

const validateTlsPem = (_, value, callback) => {
  if (form.tls.enable && !String(value || '').trim()) {
    callback(new Error('Đường dẫn tệp chứng chỉ không thể trống khi bật TLS'))
    return
  }
  callback()
}

const validateTlsKey = (_, value, callback) => {
  if (form.tls.enable && !String(value || '').trim()) {
    callback(new Error('Đường dẫn tệp chính không thể trống khi bật TLS'))
    return
  }
  callback()
}

const rules = {
  listen_host: [{ required: true, message: 'Vui lòng nhập địa chỉ máy chủ nghe', trigger: 'blur' }],
  listen_port: [
    { required: true, message: 'Vui lòng nhập số cổng nghe', trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: 'Số cổng  phải nằm trong khoảng 1-65535', trigger: 'blur' }
  ],
  username: [{ validator: validateUsername, trigger: 'blur' }],
  password: [{ validator: validatePassword, trigger: 'blur' }],
  signature_key: [{ required: true, message: 'Vui lòng nhập key ký', trigger: 'blur' }],
  'tls.port': [{ validator: validateTlsPort, trigger: 'blur' }],
  'tls.pem': [{ validator: validateTlsPem, trigger: 'blur' }],
  'tls.key': [{ validator: validateTlsKey, trigger: 'blur' }]
}

const serverReady = computed(() => {
  return Boolean(String(form.listen_host || '').trim() && Number(form.listen_port))
})

const resetForm = () => {
  Object.assign(form, createDefaultFormState())
}

const loadConfig = async () => {
  try {
    loading.value = true
    const response = await api.get('/admin/mqtt-server-configs')
    const configs = response.data?.data || []

    if (configs.length > 0) {
      const config = configs[0]
      configId.value = config.id

      let configData = {}
      try {
        configData = JSON.parse(config.json_data || '{}')
      } catch (error) {
        ElMessage.warning('Định dạng cấu hình máy chủ  MQTT không bình thường và đã được khôi phục về giá trị mặc định')
        configData = {}
      }

      form.enable = configData.enable !== undefined ? configData.enable : true
      form.listen_host = String(configData.listen_host || '0.0.0.0')
      form.listen_port = Number(configData.listen_port) > 0 ? Number(configData.listen_port) : 1883
      form.username = String(configData.username || '')
      form.password = String(configData.password || '')
      form.signature_key = String(configData.signature_key || 'xiaozhi_ota_signature_key')
      form.enable_auth = configData.enable_auth !== undefined ? configData.enable_auth : false
      form.tls.enable = configData.tls?.enable !== undefined ? configData.tls.enable : false
      form.tls.port = Number(configData.tls?.port) > 0 ? Number(configData.tls.port) : 8883
      form.tls.pem = String(configData.tls?.pem || '')
      form.tls.key = String(configData.tls?.key || '')
    } else {
      configId.value = null
      resetForm()
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.message || 'Không tải được cấu hình Máy chủ MQTT')
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    const payload = {
      name: 'MQTT Server配置',
      config_id: 'mqtt_server_mqtt_server_config',
      provider: 'mqtt_server',
      json_data: JSON.stringify({
        enable: !!form.enable,
        listen_host: String(form.listen_host || '').trim(),
        listen_port: Number(form.listen_port),
        username: String(form.username || '').trim(),
        password: String(form.password || ''),
        signature_key: String(form.signature_key || '').trim(),
        enable_auth: !!form.enable_auth,
        tls: {
          enable: !!form.tls.enable,
          port: Number(form.tls.port),
          pem: String(form.tls.pem || '').trim(),
          key: String(form.tls.key || '').trim()
        }
      }),
      enabled: true,
      is_default: true
    }

    if (configId.value) {
      await api.put(`/admin/mqtt-server-configs/${configId.value}`, payload)
      ElMessage.success('Cấu hình máy chủ  MQTT đã được cập nhật')
    } else {
      const response = await api.post('/admin/mqtt-server-configs', payload)
      configId.value = response.data?.data?.id || configId.value
      ElMessage.success('Đã lưu cấu hình máy chủ  MQTT')
    }

    await loadConfig()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || 'Không lưu được cấu hình Máy chủ MQTT')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.mqtt-server-config {
  padding: 0 24px 32px;
}

.config-form {
  display: grid;
  gap: 24px;
}

.config-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(340px, 0.95fr);
  gap: 24px;
}

.side-stack {
  display: grid;
  gap: 24px;
}

.config-card {
  border: 1px solid rgba(255, 255, 255, 0.88);
  background: rgba(255, 255, 255, 0.88);
  box-shadow: var(--apple-shadow-md);
}

.card-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.card-kicker {
  display: block;
  margin: 0;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--apple-text-tertiary);
}

.card-head h3 {
  margin: 8px 0 0;
  font-size: 22px;
  line-height: 1.15;
  letter-spacing: -0.03em;
  color: var(--apple-text);
}

.card-description,
.field-help,
.footer-note {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--apple-text-secondary);
}

.field-grid {
  display: grid;
  gap: 20px 18px;
}

.field-stack {
  display: grid;
  gap: 20px;
}

.switch-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px 18px;
  align-items: center;
}

.switch-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--apple-text);
}

.footer-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 0 4px;
}

.footer-note {
  max-width: 640px;
  margin: 0;
}

.footer-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 12px;
}

:deep(.el-card__header) {
  padding: 24px 24px 0;
  border-bottom: none;
  background: transparent;
}

:deep(.el-card__body) {
  padding: 24px;
}

:deep(.el-form-item) {
  margin-bottom: 0;
}

:deep(.el-form-item__label) {
  font-size: 14px;
  font-weight: 600;
  color: var(--apple-text);
}

@media (max-width: 1180px) {
  .config-layout {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .mqtt-server-config {
    padding: 0 16px 24px;
  }

  :deep(.el-card__body) {
    padding: 20px;
  }

  :deep(.el-card__header) {
    padding: 20px 20px 0;
  }

  .footer-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .footer-actions {
    justify-content: stretch;
  }

  .footer-actions :deep(.el-button) {
    flex: 1;
  }
}
</style>
