<template>
  <div class="mqtt-config">
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      class="config-form"
      v-loading="loading"
    >
      <div class="config-layout">
        <el-card class="config-card" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">Connection</p>
                <h3>Thông số kết nối</h3>
                <p class="card-description">Trước tiên hãy hoàn thành địa chỉ, giao thức và danh tính khách hàng để đảm bảo chương trình chính có thể kết nối ổn định với Nhà môi giới mục tiêu.</p>
              </div>
              <el-tag :type="isCoreFieldsComplete ? 'success' : 'warning'" effect="plain" round>
                {{ isCoreFieldsComplete ? 'Thông số đầy đủ của' : 'được thêm vào' }}
              </el-tag>
            </div>
          </template>

          <div class="field-grid">
            <el-form-item label="Tên cấu hình" prop="name">
              <el-input v-model="form.name" placeholder="Ví dụ: Kết nối MQTT mặc định" />
            </el-form-item>

            <el-form-item label="Địa chỉ môi giới" prop="broker">
              <el-input v-model="form.broker" placeholder="Ví dụ: mqtt://127.0.0.1 hoặc broker.example.com" />
            </el-form-item>

            <el-form-item label="Kiểu kết nối" prop="type">
              <el-select v-model="form.type" placeholder="Vui lòng chọn loại kết nối" style="width: 100%">
                <el-option
                  v-for="option in connectionTypeOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
            </el-form-item>

            <el-form-item label="Cổng" prop="port">
              <el-input-number
                v-model="form.port"
                :min="1"
                :max="65535"
                controls-position="right"
                style="width: 100%"
              />
            </el-form-item>

            <el-form-item label="ID khách hàng" prop="client_id" class="field-span-full">
              <el-input
                v-model="form.client_id"
                placeholder="Vui lòng nhập ID khách hàng được chương trình chính sử dụng để kết nối với Nhà môi giới"
              />
              <div class="field-help">
                Bạn nên sử dụng ID khách hàng ổn định và có thể nhận dạng để tạo điều kiện thuận lợi cho việc định vị nguồn kết nối ở phía Nhà môi giới.
              </div>
            </el-form-item>
          </div>
        </el-card>

        <el-card class="config-card config-card-side" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">Authentication</p>
                <h3>Thông tin chứng nhận</h3>
                <p class="card-description">Nếu Nhà môi giới bật xác thực tài khoản và mật khẩu, vui lòng điền thông tin xác thực có quyền đăng ký.</p>
              </div>
              <el-tag :type="hasCredentials ? 'success' : 'info'" effect="plain" round>
                {{ hasCredentials ? 'Đã điền voucher' : 'có thể để trống' }}
              </el-tag>
            </div>
          </template>

          <div class="field-stack">
            <el-form-item label="Tên đăng nhập" prop="username">
              <el-input v-model="form.username" placeholder="Xác thực  chưa được bật và có thể để trống." />
            </el-form-item>

            <el-form-item label="Mật khẩu" prop="password">
              <el-input
                v-model="form.password"
                type="password"
                placeholder="Xác thực  chưa được bật và có thể để trống."
                show-password
              />
            </el-form-item>
          </div>

          <div class="helper-panel">
            <div class="helper-item">
              <span>Hướng dẫn xác thực</span>
              <p>Nếu chỉ điền tên người dùng hoặc mật khẩu, nó cũng sẽ được lưu cùng nhau; nên duy trì nó theo cặp theo chính sách thực tế của Nhà môi giới.</p>
            </div>
            <div class="helper-item">
              <span>Nhắc nhở giao thức</span>
              <p>Các kết hợp phổ biến của  là TCP/1883, SSL/TLS/8883, WebSocket/8083, điều này cuối cùng phụ thuộc vào cài đặt Nhà môi giới của bạn.</p>
            </div>
          </div>
        </el-card>
      </div>

      <div class="footer-bar">
        <p class="footer-note">
          Sau khi  được lưu, cấu hình Máy khách MQTT mặc định sẽ được cập nhật cho chương trình chính để kết nối với Nhà môi giới sau đó.
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
import api from '@/utils/api'

const loading = ref(false)
const saving = ref(false)
const configId = ref(null)
const formRef = ref()

const createDefaultFormState = () => ({
  name: 'MQTT配置',
  is_default: true,
  enable: true,
  broker: '',
  type: 'tcp',
  port: 1883,
  client_id: '',
  username: '',
  password: ''
})

const form = reactive(createDefaultFormState())

const connectionTypeOptions = [
  { label: 'TCP', value: 'tcp' },
  { label: 'WebSocket', value: 'websocket' },
  { label: 'SSL/TLS', value: 'ssl' }
]

const rules = {
  name: [{ required: true, message: '请输入配置名称', trigger: 'blur' }],
  broker: [{ required: true, message: 'Vui lòng nhập địa chỉ MQTT Broker', trigger: 'blur' }],
  type: [{ required: true, message: '请选择连接类型', trigger: 'change' }],
  port: [
    { required: true, message: 'Vui lòng nhập số cổng', trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: 'Số cổng  phải nằm trong khoảng 1-65535', trigger: 'blur' }
  ],
  client_id: [{ required: true, message: 'Vui lòng nhập ID khách hàng', trigger: 'blur' }]
}

const hasCredentials = computed(() => {
  return Boolean(String(form.username || '').trim() || String(form.password || '').trim())
})

const isCoreFieldsComplete = computed(() => {
  return Boolean(
    String(form.broker || '').trim() &&
    String(form.client_id || '').trim() &&
    form.type &&
    Number(form.port)
  )
})

const resetForm = () => {
  Object.assign(form, createDefaultFormState())
}

const generateConfig = () => {
  return {
    enable: form.enable,
    broker: String(form.broker || '').trim(),
    type: form.type,
    port: Number(form.port),
    client_id: String(form.client_id || '').trim(),
    username: String(form.username || '').trim(),
    password: String(form.password || '')
  }
}

const applyLoadedConfig = (config) => {
  configId.value = config?.id || null
  form.name = config?.name || 'Cấu hình  MQTT'
  form.is_default = config?.is_default ?? true

  let configData = {}
  try {
    configData = JSON.parse(config?.json_data || '{}')
  } catch (error) {
    ElMessage.warning('Định dạng cấu hình MQTT không bình thường và đã được trả về giá trị mặc định.')
    configData = {}
  }

  form.enable = typeof configData.enable === 'boolean' ? configData.enable : true
  form.broker = String(configData.broker || '')
  form.type = String(configData.type || 'tcp')
  form.port = Number(configData.port) > 0 ? Number(configData.port) : 1883
  form.client_id = String(configData.client_id || '')
  form.username = String(configData.username || '')
  form.password = String(configData.password || '')
}

const loadConfig = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/mqtt-configs')
    const configs = response.data?.data || []

    if (configs.length > 0) {
      applyLoadedConfig(configs[0])
    } else {
      configId.value = null
      resetForm()
    }
  } catch (error) {
    ElMessage.error('Không tải được cấu hình MQTT')
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
    const generatedConfigId = `mqtt_${String(form.name || '')
      .replace(/[^a-zA-Z0-9]/g, '_')
      .toLowerCase()}`

    const nextConfigPayload = generateConfig()
    let configData
    let isUpdate = false

    if (configId.value) {
      const response = await api.get('/admin/mqtt-configs')
      const configs = response.data?.data || []
      const existingConfig = configs.find(item => item.id === configId.value)

      if (existingConfig) {
        let existingData = {}
        try {
          existingData = JSON.parse(existingConfig.json_data || '{}')
        } catch {
          existingData = {}
        }

        configData = {
          name: form.name,
          config_id: generatedConfigId,
          is_default: true,
          json_data: JSON.stringify({
            ...existingData,
            ...nextConfigPayload
          })
        }
        isUpdate = true
      }
    }

    if (!configData) {
      configData = {
        name: form.name,
        config_id: generatedConfigId,
        is_default: true,
        json_data: JSON.stringify(nextConfigPayload)
      }
    }

    if (isUpdate) {
      await api.put(`/admin/mqtt-configs/${configId.value}`, configData)
      ElMessage.success('Đã cập nhật cấu hình  MQTT')
    } else {
      const response = await api.post('/admin/mqtt-configs', configData)
      configId.value = response.data?.data?.id || configId.value
      ElMessage.success('Đã lưu cấu hình  MQTT')
    }

    await loadConfig()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || 'Không lưu được cấu hình MQTT')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.mqtt-config {
  display: grid;
  padding: 0 24px 32px;
}

.footer-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 12px;
}

.card-kicker {
  display: block;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.card-kicker {
  color: var(--apple-text-tertiary);
}

.card-description,
.field-help,
.helper-item p,
.footer-note {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--apple-text-secondary);
}

.config-form {
  display: grid;
  gap: 24px;
}

.config-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.95fr);
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

.card-head h3 {
  margin: 8px 0 0;
  font-size: 22px;
  line-height: 1.15;
  letter-spacing: -0.03em;
  color: var(--apple-text);
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px 18px;
}

.field-span-full {
  grid-column: 1 / -1;
}

.field-stack {
  display: grid;
  gap: 20px;
}

.helper-panel {
  display: grid;
  gap: 14px;
  margin-top: 8px;
  padding-top: 18px;
  border-top: 1px solid rgba(229, 229, 234, 0.72);
}

.helper-item span {
  display: block;
  font-size: 13px;
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
  max-width: 620px;
  margin: 0;
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
  .mqtt-config {
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

  .field-grid {
    grid-template-columns: 1fr;
  }

  .footer-actions {
    justify-content: stretch;
  }

  .footer-actions :deep(.el-button) {
    flex: 1;
  }
}
</style>
