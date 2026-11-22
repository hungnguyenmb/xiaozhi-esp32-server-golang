<template>
  <div class="config-page">
    <el-button type="primary" @click="openCreate">bổ sung cấu hình Qwen Realtime</el-button>
    <el-table :data="configs" v-loading="loading" style="width: 100%; margin-top: 16px">
      <el-table-column prop="name" label="Tên" />
      <el-table-column prop="config_id" label="ID cấu hình" />
      <el-table-column label="Trạng thái" width="100">
        <template #default="{ row }">{{ row.enabled ? 'Bật' : 'bị vô hiệu hóa' }}</template>
      </el-table-column>
      <el-table-column label="Mặc định" width="80">
        <template #default="{ row }">{{ row.is_default ? 'Có' : 'Không' }}</template>
      </el-table-column>
      <el-table-column label="Thao tác" width="180">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">Chỉnh sửa</el-button>
          <el-button size="small" type="danger" :disabled="row.is_default" @click="remove(row)">Xóa</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="visible" :title="editing ? 'Cấu hình thời gian thực Qwen của  Editor' : 'bổ sung cấu hình Qwen Realtime'" width="min(680px, calc(100vw - 32px))">
      <el-form label-width="130px">
        <el-form-item label="Tên" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="ID cấu hình" required><el-input v-model="form.config_id" :disabled="!!editing" /></el-form-item>
        <el-form-item label="API Key" required><el-input v-model="form.data.api_key" type="password" show-password autocomplete="new-password" /></el-form-item>
        <el-form-item label="Địa chỉ WebSocket"><el-input v-model="form.data.base_url" /></el-form-item>
        <el-form-item label="Workspace"><el-input v-model="form.data.workspace" placeholder="tùy chọn" /></el-form-item>
        <el-form-item label="Mẫu"><el-input v-model="form.data.model" /></el-form-item>
        <el-form-item label="Giọng nói"><el-input v-model="form.data.voice" /></el-form-item>
        <el-form-item label="Phát hiện vòng">
          <el-select v-model="form.data.turn_detection" style="width: 100%">
            <el-option label="smart_turn" value="smart_turn" />
            <el-option label="server_vad" value="server_vad" />
            <el-option label="Gửi hướng dẫn sử dụng" value="null" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.data.turn_detection === 'server_vad'" label="Thời lượng tắt tiếng của  (ms)">
          <el-input-number v-model="form.data.silence_duration_ms" :min="200" :max="6000" :step="100" />
        </el-form-item>
        <el-form-item label="Lời nhắc hệ thống"><el-input v-model="form.data.instructions" type="textarea" :rows="3" placeholder="Sử dụng phần giới thiệu vai trò trợ lý AI khi để trống" /></el-form-item>
        <el-form-item label="Bật"><el-switch v-model="form.enabled" /></el-form-item>
        <el-form-item label="được đặt làm mặc định"><el-switch v-model="form.is_default" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">Hủy</el-button>
        <el-button type="primary" :loading="saving" @click="save">Lưu</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../../utils/api'

const defaults = () => ({
  name: '', config_id: '', enabled: true, is_default: false,
  data: {
    api_key: '', base_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/realtime',
    workspace: '', model: 'qwen-audio-3.1-realtime-plus', voice: 'longanqian_v3.1',
    turn_detection: 'smart_turn', silence_duration_ms: 800, instructions: ''
  }
})
const configs = ref([])
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const editing = ref(null)
const form = reactive(defaults())

const load = async () => {
  loading.value = true
  try {
    configs.value = (await api.get('/admin/realtime-configs')).data?.data || []
  } finally {
    loading.value = false
  }
}
const openCreate = () => {
  Object.assign(form, defaults())
  editing.value = null
  visible.value = true
}
const openEdit = (row) => {
  let data = {}
  try { data = JSON.parse(row.json_data || '{}') } catch (_) { data = {} }
  Object.assign(form, { ...defaults(), ...row, data: { ...defaults().data, ...data } })
  editing.value = row
  visible.value = true
}
const save = async () => {
  if (!form.name.trim() || !form.config_id.trim() || !form.data.api_key.trim()) {
    ElMessage.warning('Tên , ID cấu hình và Khóa API không được để trống')
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.name.trim(), config_id: form.config_id.trim(),
      provider: 'aliyun_qwen_realtime', json_data: JSON.stringify(form.data),
      enabled: form.enabled, is_default: form.is_default
    }
    if (editing.value) await api.put(`/admin/realtime-configs/${editing.value.id}`, payload)
    else await api.post('/admin/realtime-configs', payload)
    visible.value = false
    ElMessage.success('Đã lưu')
    await load()
  } catch (err) {
    ElMessage.error(err.response?.data?.error || 'Không thể lưu')
  } finally {
    saving.value = false
  }
}
const remove = async (row) => {
  try {
    await ElMessageBox.confirm(`Xóa cấu hình"${row.name}”？`, 'Xác nhận xóa', { type: 'warning' })
    await api.delete(`/admin/realtime-configs/${row.id}`)
    await load()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error(err.response?.data?.error || 'Không thể xóa')
  }
}
onMounted(load)
</script>
