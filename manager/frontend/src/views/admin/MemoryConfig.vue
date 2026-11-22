<template>
  <div class="config-page">
    <div class="page-actions">
      <el-button type="primary" @click="handleAddConfig">
        <el-icon><Plus /></el-icon>
        Thêm cấu hình
      </el-button>
    </div>

    <el-table :data="safeConfigs" style="width: 100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="Tên cấu hình" />
      <el-table-column prop="config_id" label="ID cấu hình" width="150" />
      <el-table-column prop="provider" label="Nhà cung cấp" width="120">
        <template #default="scope">
          <el-tag :type="getProviderTagType(scope.row.provider)">
            {{ scope.row.provider }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="enabled" label="Trạng thái kích hoạt" width="80" align="center">
        <template #default="scope">
          <el-switch 
            v-model="scope.row.enabled" 
            @change="toggleEnable(scope.row)"
          />
        </template>
      </el-table-column>
      <el-table-column prop="is_default" label="Cấu hình mặc định" width="80" align="center">
        <template #default="scope">
          <el-switch 
            v-model="scope.row.is_default" 
            @change="toggleDefault(scope.row)"
          />
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="Thời gian tạo" width="180">
        <template #default="scope">
          {{ formatDate(scope.row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="Thao tác" width="180">
        <template #default="scope">
          <el-button size="small" @click="editConfig(scope.row)">Chỉnh sửa</el-button>
          <el-button
            size="small"
            type="danger"
            @click="deleteConfig(scope.row.id)"
          >
            Xóa
          </el-button>
        </template>
      </el-table-column>
      
      <!-- 空状态插槽 -->
      <template #empty>
        <div class="empty-state">
          <el-icon size="64" color="#C0C4CC" class="empty-icon">
            <Box />
          </el-icon>
          <div class="empty-text">Chưa có cấu hình bộ nhớ</div>
          <div class="empty-description">Nhấp vào nút "Thêm cấu hình" ở trên để tạo cấu hình Bộ nhớ đầu tiên của bạn</div>
          <el-button type="primary" @click="handleAddConfig" class="empty-action">
            <el-icon><Plus /></el-icon>
            Thêm cấu hình
          </el-button>
        </div>
      </template>
    </el-table>

    <!-- 添加/编辑配置弹窗 -->
    <el-dialog
      v-model="showDialog"
      :title="editingConfig ? 'Chỉnh sửa cấu hình bộ nhớ' : 'Thêm cấu hình bộ nhớ'"
      width="600px"
      @close="handleDialogClose"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item label="Nhà cung cấp" prop="provider">
          <el-select v-model="form.provider" placeholder="Vui lòng chọn nhà cung cấp" style="width: 100%" @change="handleProviderChange">
            <el-option label="Memobase" value="memobase" />
            <el-option label="Mem0" value="mem0" />
            <el-option label="MemOS" value="memos" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="Tên cấu hình" prop="name">
          <el-input v-model="form.name" placeholder="Vui lòng nhập tên cấu hình" />
        </el-form-item>
        
        <el-form-item label="ID cấu hình" prop="config_id">
          <el-input v-model="form.config_id" placeholder="Vui lòng nhập ID cấu hình duy nhất" />
        </el-form-item>
        
        <!-- Memobase配置字段 -->
        <template v-if="form.provider === 'memobase'">
          <el-form-item label="Khóa API" prop="api_key">
            <el-input v-model="form.api_key" type="password" placeholder="Vui lòng nhập key API Memobase" show-password />
          </el-form-item>
          
          <el-form-item label="URL cơ sở" prop="base_url">
            <el-input v-model="form.base_url" placeholder="Vui lòng nhập URL cơ sở Memobase" />
          </el-form-item>
          
          <el-form-item label="Kích hoạt tìm kiếm" prop="enable_search">
            <el-switch v-model="form.enable_search" />
          </el-form-item>
          
          <el-form-item label="Ngưỡng tìm kiếm" prop="search_threshold">
            <el-input-number v-model="form.search_threshold" :min="0" :max="1" :step="0.1" :precision="1" style="width: 100%" />
          </el-form-item>
          
          <el-form-item label="Tìm kiếm TopK" prop="search_top_k">
            <el-input-number v-model="form.search_top_k" :min="1" :step="1" style="width: 100%" />
          </el-form-item>
        </template>
        
        <!-- Mem0配置字段 -->
        <template v-if="form.provider === 'mem0' || form.provider === 'memos'">
          <el-form-item label="Khóa API" prop="api_key">
            <el-input v-model="form.api_key" type="password" :placeholder="form.provider === 'memos' ? 'Vui lòng nhập khóa API tương thích MemOS' : 'Vui lòng nhập key API Mem0'" show-password />
          </el-form-item>
          
          <el-form-item label="URL cơ sở" prop="base_url">
            <el-input v-model="form.base_url" :placeholder="form.provider === 'memos' ? 'Vui lòng nhập URL cơ sở dịch vụ MemOS' : 'Vui lòng nhập URL cơ sở Mem0'" />
          </el-form-item>

          

          <el-form-item label="Kích hoạt tìm kiếm" prop="enable_search">
            <el-switch v-model="form.enable_search" />
          </el-form-item>
          
          <el-form-item label="Ngưỡng tìm kiếm" prop="search_threshold">
            <el-input-number v-model="form.search_threshold" :min="0" :max="1" :step="0.1" :precision="1" style="width: 100%" />
          </el-form-item>
          
          <el-form-item label="Tìm kiếm TopK" prop="search_top_k">
            <el-input-number v-model="form.search_top_k" :min="1" :step="1" style="width: 100%" />
          </el-form-item>
        </template>
      </el-form>
      
      <template #footer>
        <el-button @click="handleDialogClose">Hủy</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">
          Lưu
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Box } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { resolveMemoryProvider } from './forms/configProviderUtils'

const configs = ref([])
const loading = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const editingConfig = ref(null)
const formRef = ref()

// 确保configs始终是一个数组
const safeConfigs = computed(() => {
  return Array.isArray(configs.value) ? configs.value : []
})

const form = reactive({
  name: '',
  config_id: '',
  provider: 'memobase',
  is_default: false,
  enabled: true,
  api_key: '',
  base_url: '',
  enable_search: true,
  search_threshold: 0.5,
  search_top_k: 3,
  timeout_ms: 10000
})

// 默认URL配置
const defaultUrls = {
  memobase: 'https://api.memobase.dev',
  mem0: 'https://api.mem0.ai',
  memos: 'https://memos.memtensor.cn/api/openmem/v1'
}


const getProviderTagType = (provider) => {
  if (provider === 'memobase') return 'primary'
  if (provider === 'memos') return 'warning'
  return 'success'
}

const handleProviderChange = (value) => {
  // 清空表单字段
  form.api_key = ''
  form.base_url = defaultUrls[value] || ''
  form.enable_search = true
  form.search_threshold = 0.5
  form.search_top_k = 3
  form.timeout_ms = 10000
}

// 生成配置JSON字符串
const generateConfig = () => {
  const config = {
    api_key: form.api_key,
    base_url: form.base_url,
    enable_search: form.enable_search,
    search_threshold: form.search_threshold,
    search_top_k: form.search_top_k
  }

  if (form.provider === 'memos') {
    config.timeout_ms = form.timeout_ms
  }

  return JSON.stringify(config)
}

// 解析配置JSON字符串
const parseConfig = (jsonData) => {
  try {
    const config = JSON.parse(jsonData)
    form.api_key = config.api_key || ''
    form.base_url = config.base_url || defaultUrls[form.provider] || ''
    form.enable_search = config.enable_search !== undefined ? config.enable_search : true
    form.search_threshold = config.search_threshold !== undefined ? config.search_threshold : 0.5
    form.search_top_k = config.search_top_k !== undefined ? config.search_top_k : 3
    form.timeout_ms = config.timeout_ms !== undefined ? config.timeout_ms : 10000
  } catch (error) {
    console.error('解析配置失败:', error)
  }
}

const rules = {
  name: [
    { required: true, message: '请输入配置名称', trigger: 'blur' }
  ],
  config_id: [
    { required: true, message: 'Vui lòng nhập ID cấu hình', trigger: 'blur' }
  ],
  provider: [
    { required: true, message: '请选择提供商', trigger: 'change' }
  ],
  api_key: [
    { required: true, message: 'Vui lòng nhập khóa API', trigger: 'blur' }
  ],
  base_url: [
    { required: true, message: 'Vui lòng nhập URL cơ sở', trigger: 'blur' }
  ]
}

const formatDate = (dateString) => {
  return new Date(dateString).toLocaleString('vi-VN')
}

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/memory-configs')
    console.log('API response:', response)
    
    // 使用nextTick确保响应式更新的安全性
    await nextTick()
    
    // The backend returns { data: configs }, so we need to access response.data.data
    if (response && response.data && response.data.data && Array.isArray(response.data.data)) {
      // 使用Object.freeze防止意外修改，然后创建新数组
      const newConfigs = response.data.data.map(normalizeMemoryConfigRow)
      configs.value = newConfigs
    } else if (response && response.data && response.data.data) {
      // If response.data.data exists but is not an array, wrap it in an array
      configs.value = [normalizeMemoryConfigRow(response.data.data)]
    } else {
      // If no valid data, set to empty array
      configs.value = []
    }
    console.log('Loaded configs:', configs.value)
  } catch (error) {
    console.error('加载配置失败:', error)
    ElMessage.error('Không tải được cấu hình: ' + (error.message || 'Lỗi không xác định'))
    // Ensure configs is always an array to prevent render errors
    configs.value = []
  } finally {
    loading.value = false
  }
}

function parseJsonData(jsonData) {
  if (!jsonData || typeof jsonData !== 'string') return {}
  try {
    return JSON.parse(jsonData) || {}
  } catch {
    return {}
  }
}

function normalizeMemoryConfigRow(row) {
  const data = parseJsonData(row?.json_data)
  return {
    ...row,
    provider: resolveMemoryProvider(row?.provider, row?.config_id, data)
  }
}

const handleSave = async () => {
  if (!formRef.value) return
  
  try {
    await formRef.value.validate()
  } catch (error) {
    return
  }
  
  saving.value = true
  try {
    const configData = {
      name: form.name,
      config_id: form.config_id,
      provider: form.provider,
      enabled: form.enabled,
      is_default: form.is_default,
      json_data: generateConfig()
    }
    
    if (editingConfig.value) {
      await api.put(`/admin/memory-configs/${editingConfig.value.id}`, configData)
      ElMessage.success('Cập nhật cấu hình  thành công')
    } else {
      await api.post('/admin/memory-configs', configData)
      ElMessage.success('Cấu hình  được tạo thành công')
    }
    
    showDialog.value = false
    await loadConfigs()
  } catch (error) {
    ElMessage.error('Lưu không thành công: ' + error.message)
  } finally {
    saving.value = false
  }
}

const editConfig = (config) => {
  config = normalizeMemoryConfigRow(config)
  editingConfig.value = config
  form.name = config.name
  form.config_id = config.config_id
  form.provider = config.provider
  form.enabled = config.enabled
  form.is_default = config.is_default
  
  if (config.json_data) {
    parseConfig(config.json_data)
  }
  
  showDialog.value = true
}

const deleteConfig = async (id) => {
  try {
    await ElMessageBox.confirm('Bạn có chắc chắn muốn xóa cấu hình này không?', 'Xác nhận xóa', {
      type: 'warning'
    })
    
    await api.delete(`/admin/memory-configs/${id}`)
    ElMessage.success('Đã xóa')
    await loadConfigs()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('Xóa  không thành công: ' + error.message)
    }
  }
}

const toggleEnable = async (config) => {
  try {
    await api.put(`/admin/memory-configs/${config.id}`, {
      ...config,
      enabled: config.enabled
    })
    ElMessage.success(config.enabled ? 'Đã bật' : 'bị vô hiệu hóa')
  } catch (error) {
    config.enabled = !config.enabled
    ElMessage.error('Hoạt động không thành công: ' + error.message)
  }
}

const toggleDefault = async (config) => {
  try {
    if (config.is_default) {
      await api.post(`/admin/memory-configs/${config.id}/set-default`)
      ElMessage.success('đã được đặt làm cấu hình mặc định')
      await loadConfigs()
    } else {
      await api.put(`/admin/memory-configs/${config.id}`, {
        name: config.name,
        config_id: config.config_id,
        provider: config.provider,
        enabled: config.enabled,
        is_default: false,
        json_data: config.json_data || ''
      })
      ElMessage.success('đã hủy cấu hình mặc định (không kích hoạt bộ nhớ dài)')
      await loadConfigs()
    }
  } catch (error) {
    config.is_default = !config.is_default
    ElMessage.error('Hoạt động không thành công: ' + error.message)
  }
}

const handleAddConfig = () => {
  // 重置表单并设置默认值
  Object.assign(form, {
    name: '',
    config_id: '',
    provider: 'memobase',
    is_default: false,
    enabled: true,
    api_key: '',
    base_url: defaultUrls['memobase'], // 设置默认URL
    enable_search: true,
    search_threshold: 0.5,
    search_top_k: 3,
    timeout_ms: 10000,
  })
  
  editingConfig.value = null
  showDialog.value = true
}

const handleDialogClose = () => {
  showDialog.value = false
  editingConfig.value = null
  
  // 重置表单
  Object.assign(form, {
    name: '',
    config_id: '',
    provider: 'memobase',
    is_default: false,
    enabled: true,
    api_key: '',
    base_url: '',
    enable_search: true,
    search_threshold: 0.5,
    search_top_k: 3,
    timeout_ms: 10000,
  })
  
  if (formRef.value) {
    formRef.value.clearValidate()
  }
}

onMounted(() => {
  loadConfigs()
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

.empty-state {
  text-align: center;
  padding: 60px 20px;
}

.empty-icon {
  margin-bottom: 16px;
}

.empty-text {
  font-size: 16px;
  color: #606266;
  margin-bottom: 8px;
  font-weight: 500;
}

.empty-description {
  font-size: 14px;
  color: #909399;
  margin-bottom: 24px;
  line-height: 1.5;
}

.empty-action {
  margin-top: 8px;
}
</style>
