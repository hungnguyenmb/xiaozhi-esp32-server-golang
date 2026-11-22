<template>
  <div class="config-page">
    <div class="page-actions">
      <el-button type="primary" @click="openDialog()">bổ sung thêm kiến thức cơ bản</el-button>
    </div>

    <el-table :data="items" v-loading="loading" stripe table-layout="fixed" style="width: 100%">
      <el-table-column prop="id" label="ID" width="56" />
      <el-table-column prop="name" label="Tên" width="124" show-overflow-tooltip />
      <el-table-column label="Mô tả" min-width="180" show-overflow-tooltip>
        <template #default="scope">
          <span class="kb-desc-text" :class="{ 'is-empty': !(scope.row.description || '').trim() }">
            {{ (scope.row.description || '').trim() || '-' }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="Nhà cung cấp" width="88" show-overflow-tooltip>
        <template #default="scope">
          <el-tag size="small" effect="plain">{{ formatProviderText(scope.row.sync_provider) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="số lượng tài liệu" width="72" align="center">
        <template #default="scope">
          <el-tag size="small" type="info">{{ formatDocCount(scope.row.doc_count) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Trạng thái đồng bộ hóa" width="132">
        <template #default="scope">
          <div class="kb-sync-status-cell">
            <el-tag :type="getSyncStatusTagType(scope.row.sync_status)" size="small">{{ getSyncStatusText(scope.row.sync_status) }}</el-tag>
            <el-tooltip v-if="shouldShowSyncErrorTip(scope.row)" placement="top">
              <template #content>
                <div class="kb-sync-error-tooltip">{{ scope.row.sync_error }}</div>
              </template>
              <el-icon class="kb-sync-error-icon"><WarningFilled /></el-icon>
            </el-tooltip>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="Đồng bộ hóa mới nhất" width="168" show-overflow-tooltip>
        <template #default="scope">
          <span>{{ formatDateTimeCell(scope.row.last_synced_at) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="Trạng thái" width="92" align="center">
        <template #default="scope">
          <el-switch
            :model-value="String(scope.row.status || '').trim() === 'active'"
            inline-prompt
            active-text="mở"
            inactive-text="Đóng"
            :loading="isStatusSwitchLoading(scope.row.id)"
            @change="(checked) => toggleKnowledgeBaseStatus(scope.row, checked)"
          />
        </template>
      </el-table-column>
      <el-table-column label="Thao tác" width="176">
        <template #default="scope">
          <div class="action-buttons">
            <el-button size="small" type="primary" plain @click="openDocuments(scope.row)">Tài liệu</el-button>
            <el-button size="small" type="success" plain @click="openSearchTestDialog(scope.row)">Kiểm tra</el-button>
            <el-dropdown trigger="click" @command="(cmd) => handleKnowledgeBaseAction(cmd, scope.row)">
              <el-button size="small">Thêm</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">Chỉnh sửa</el-dropdown-item>
                  <el-dropdown-item command="sync">thử lại đồng bộ hóa</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>Xóa</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editing ? 'Cơ sở kiến thức soạn thảo' : 'bổ sung thêm kiến thức cơ bản'" width="680px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="Tên">
          <el-input v-model="form.name" maxlength="100" show-word-limit />
        </el-form-item>
        <el-form-item label="Mô tả">
          <el-input v-model="form.description" />
        </el-form-item>
        <el-form-item label="Hướng dẫn đồng bộ hóa">
          <div class="kb-helper-text">Sau khi  được lưu, nó sẽ tự động đồng bộ hóa không đồng bộ với nhà cung cấp cơ sở kiến thức do quản trị viên định cấu hình (chẳng hạn như Dify / RAGFlow / WeKnora). Vui lòng thêm tài liệu vào "Quản lý tài liệu".</div>
        </el-form-item>
        <el-form-item label="Ngưỡng tìm kiếm">
          <el-input
            v-model="form.retrieval_threshold_text"
            placeholder="Vui lòng nhập số thập phân trong khoảng từ 0~1, chẳng hạn như 0,2"
            clearable
          />
          <div class="kb-helper-text is-spaced">
            điền ngưỡng dùng chung của nhà cung cấp theo mặc định. Nhà cung cấp hiện tại:{{ form.threshold_provider || '-' }}, ngưỡng dùng chung:{{ formatKnowledgeThreshold(form.global_threshold) }}。
          </div>
        </el-form-item>
        <el-form-item label="Trạng thái">
          <el-select v-model="form.status" style="width: 100%">
            <el-option value="active" label="Đang hoạt động" />
            <el-option value="inactive" label="Ngừng hoạt động" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">Hủy</el-button>
        <el-button type="primary" @click="submit">Lưu</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="documentsVisible" title="Quản lý tài liệu" width="900px">
      <div class="dialog-toolbar">
        <div>
          Cơ sở kiến thức hiện tại của : <strong>{{ currentKb?.name || '-' }}</strong>
        </div>
        <div class="dialog-toolbar-actions">
          <el-upload
            :show-file-list="false"
            :http-request="uploadDocumentFile"
            :accept="uploadAcceptByProvider"
            :disabled="!isUploadProviderSupported"
          >
            <el-button type="success" plain>Tệp tải lên</el-button>
          </el-upload>
          <el-button type="primary" @click="openDocumentDialog()">tài liệu mới</el-button>
        </div>
      </div>
      <div class="kb-helper-text is-bottom">
        {{ uploadTipText }}
      </div>
      <el-table :data="documentItems" v-loading="documentsLoading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="Tên tài liệu" width="180" />
        <el-table-column prop="external_doc_id" label="ID tài liệu" width="220" />
        <el-table-column label="Xem trước nội dung">
          <template #default="scope">
            {{ getDocumentPreview(scope.row) }}
          </template>
        </el-table-column>
        <el-table-column label="Trạng thái đồng bộ hóa" width="110">
          <template #default="scope">
            <el-tag :type="getSyncStatusTagType(scope.row.sync_status)">{{ getSyncStatusText(scope.row.sync_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_synced_at" label="Thời gian đồng bộ hóa mới nhất của" width="170" />
        <el-table-column label="Thao tác" width="250">
          <template #default="scope">
            <div class="action-buttons">
              <el-button size="small" :disabled="isUploadedFileDocument(scope.row)" @click="openDocumentDialog(scope.row)">Chỉnh sửa</el-button>
              <el-button size="small" type="primary" plain @click="syncDocument(scope.row.id)">thử lại đồng bộ hóa</el-button>
              <el-button size="small" type="danger" @click="removeDocument(scope.row.id)">Xóa</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog v-model="documentDialogVisible" :title="documentEditing ? 'Chỉnh sửa tài liệu' : 'tài liệu mới'" width="700px">
      <el-form :model="documentForm" label-width="90px">
        <el-form-item label="Tên tài liệu">
          <el-input v-model="documentForm.name" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="Nội dung">
          <el-input v-model="documentForm.content" type="textarea" :rows="12" placeholder="Vui lòng nhập nội dung tài liệu" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="documentDialogVisible = false">Hủy</el-button>
        <el-button type="primary" @click="submitDocument">Lưu</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="searchTestVisible" title="Kiểm tra truy xuất" width="960px">
      <div style="display: flex; justify-content: space-between; gap: 12px; margin-bottom: 12px; flex-wrap: wrap;">
        <div>
          Cơ sở kiến thức hiện tại của : <strong>{{ searchTestKb?.name || '-' }}</strong>
          <el-tag size="small" style="margin-left: 8px;">{{ searchTestKb?.sync_provider || '-' }}</el-tag>
        </div>
        <div style="display: flex; gap: 8px; flex: 1; min-width: 420px; justify-content: flex-end;">
          <el-input
            v-model="searchTestForm.query"
            placeholder="Nhập từ khóa hoặc câu hỏi kiểm tra, chẳng hạn như: quy trình hoàn tiền/xác thực giao diện"
            clearable
            @keyup.enter="runSearchTest"
          />
          <el-tooltip content="TopK: Trả về kết quả triệu hồi K top đầu" placement="top">
            <span style="display:inline-flex;align-items:center;color:#909399;font-size:12px;white-space:nowrap;">TopK</span>
          </el-tooltip>
          <el-select v-model="searchTestForm.top_k" style="width: 110px;">
            <el-option v-for="k in topKOptions" :key="k" :value="k" :label="String(k)" />
          </el-select>
          <el-tooltip content="Chỉ thử nghiệm truy xuất này mới có hiệu lực; nếu nó trống thì ngưỡng hiện tại của cơ sở tri thức (hoặc ngưỡng dùng chung) sẽ được sử dụng." placement="top">
            <span style="display:inline-flex;align-items:center;color:#909399;font-size:12px;white-space:nowrap;">Ngưỡng</span>
          </el-tooltip>
          <el-input
            v-model="searchTestForm.threshold_text"
            placeholder="là 0,2"
            clearable
            style="width: 120px;"
          />
          <el-button type="primary" :loading="searchTestLoading" @click="runSearchTest">Bắt đầu thử nghiệm</el-button>
        </div>
      </div>
      <div class="kb-helper-text is-bottom">
        Bài kiểm tra truy xuất  sẽ gọi trực tiếp giao diện truy xuất (Dify/RAGFlow/WeKnora) của nhà cung cấp tương ứng với cơ sở kiến thức hiện tại để xác minh hiệu quả truy xuất từ khóa.
      </div>
      <div v-if="searchTestElapsedMs !== null" class="kb-helper-text is-bottom is-regular">
        Thời gian phản hồi của :{{ searchTestElapsedMs }} ms
      </div>
      <el-table :data="searchTestResult.hits" v-loading="searchTestLoading" style="width: 100%" max-height="420">
        <el-table-column type="index" label="#" width="60" />
        <el-table-column prop="title" label="Nguồn" width="200" />
        <el-table-column label="Điểm" width="110">
          <template #default="scope">
            {{ formatHitScore(scope.row.score) }}
          </template>
        </el-table-column>
        <el-table-column prop="content" label="Nội dung đình đám" min-width="480">
          <template #default="scope">
            <div class="search-hit-content">
              {{ scope.row.content }}
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!searchTestLoading && hasRunSearchTest && searchTestResult.hits.length === 0" class="kb-helper-text is-empty">
        Thiếu nội dung, bạn có thể thử thay đổi từ khóa hoặc kiểm tra xem cơ sở kiến thức đã được đồng bộ chưa.
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { WarningFilled } from '@element-plus/icons-vue'
import api from '@/utils/api'

const loading = ref(false)
const items = ref([])
const statusSwitchLoadingMap = ref({})
const dialogVisible = ref(false)
const editing = ref(false)
const currentId = ref(null)

const documentsVisible = ref(false)
const documentsLoading = ref(false)
const documentItems = ref([])
const currentKb = ref(null)

const documentDialogVisible = ref(false)
const documentEditing = ref(false)
const currentDocumentId = ref(null)
const searchTestVisible = ref(false)
const searchTestLoading = ref(false)
const searchTestKb = ref(null)
const hasRunSearchTest = ref(false)
const searchTestElapsedMs = ref(null)
const searchTestResult = reactive({
  query: '',
  count: 0,
  hits: []
})

const form = reactive({
  name: '',
  description: '',
  status: 'active',
  inherit_global_threshold: true,
  retrieval_threshold_text: '0.2',
  threshold_provider: 'dify',
  global_threshold: 0.2
})

const documentForm = reactive({
  name: '',
  content: ''
})
const searchTestForm = reactive({
  query: '',
  top_k: 5,
  threshold_text: ''
})
const topKOptions = Array.from({ length: 20 }, (_, i) => i + 1)

const FILE_UPLOAD_CONTENT_PREFIX = '__KB_FILE_UPLOAD_V1__:'
const DIFY_UPLOAD_ACCEPT = '.txt,.md,.markdown,.pdf,.html,.htm,.xlsx,.xls,.docx,.csv,.eml,.msg,.pptx,.ppt,.xml,.epub'
const RAGFLOW_UPLOAD_ACCEPT = '.txt,.text,.md,.markdown,.pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx,.wps,.json,.csv,.log,.xml,.html,.htm,.yml,.yaml,.rtf,.sql,.ini,.jpg,.jpeg,.png,.gif,.bmp,.webp,.tif,.tiff,.eml,.msg'
const WEKNORA_UPLOAD_ACCEPT = '.txt,.text,.md,.markdown,.pdf,.doc,.docx,.ppt,.pptx,.xls,.xlsx,.wps,.json,.csv,.log,.xml,.html,.htm,.yml,.yaml,.rtf,.sql,.ini,.jpg,.jpeg,.png,.gif,.bmp,.webp,.tif,.tiff,.eml,.msg'
const DEFAULT_DIFY_THRESHOLD = 0.2
const DEFAULT_RAGFLOW_THRESHOLD = 0.2
const DEFAULT_WEKNORA_THRESHOLD = 0.2

const knowledgeGlobalConfig = reactive({
  default_provider: 'dify',
  providers: {}
})

const currentKBProvider = computed(() => (currentKb.value?.sync_provider || 'dify').toLowerCase())
const uploadAcceptByProvider = computed(() => {
  if (currentKBProvider.value === 'dify') return DIFY_UPLOAD_ACCEPT
  if (currentKBProvider.value === 'ragflow') return RAGFLOW_UPLOAD_ACCEPT
  if (currentKBProvider.value === 'weknora') return WEKNORA_UPLOAD_ACCEPT
  return ''
})
const isUploadProviderSupported = computed(() => currentKBProvider.value === 'dify' || currentKBProvider.value === 'ragflow' || currentKBProvider.value === 'weknora')
const uploadTipText = computed(() => {
  if (currentKBProvider.value === 'dify') {
    return 'hỗ trợ tải lên có giới hạn định dạng bởi Dify (txt/md/pdf/html/xlsx/docx/csv/eml/msg/pptx/xml/epub). Sau khi tải lên, tài liệu sẽ tự động được tạo và đồng bộ hóa không đồng bộ.'
  }
  if (currentKBProvider.value === 'ragflow') {
    return 'giới hạn tải lên theo các định dạng được RAGFlow hỗ trợ (chẳng hạn như txt/md/pdf/docx/xlsx/pptx/jpg/png/eml, v.v.). Sau khi tải lên, tài liệu sẽ tự động được tạo và đồng bộ hóa không đồng bộ.'
  }
  if (currentKBProvider.value === 'weknora') {
    return 'giới hạn tải lên theo các định dạng được WeKnora hỗ trợ (chẳng hạn như txt/md/pdf/docx/xlsx/pptx/jpg/png/eml, v.v.). Sau khi tải lên, tài liệu sẽ tự động được tạo và đồng bộ hóa không đồng bộ.'
  }
  return `Nhà cung cấp hiện tại ${currentKBProvider.value} hiện tại chưa hỗ trợ upload và tạo tài liệu.`
})

const applyKnowledgeGlobalConfig = (knowledge) => {
  const payload = knowledge && typeof knowledge === 'object' ? knowledge : {}
  knowledgeGlobalConfig.default_provider = normalizeProvider(payload.default_provider || 'dify')
  knowledgeGlobalConfig.providers = payload.providers && typeof payload.providers === 'object' ? payload.providers : {}
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await api.get('/user/knowledge-bases')
    items.value = res.data.data || []
    applyKnowledgeGlobalConfig(res.data.knowledge)
  } finally {
    loading.value = false
  }
}

const normalizeProvider = (provider) => {
  const p = String(provider || '').trim().toLowerCase()
  if (p === 'dify' || p === 'ragflow' || p === 'weknora') return p
  return 'dify'
}

const getGlobalThresholdByProvider = (provider) => {
  const p = normalizeProvider(provider)
  const cfg = knowledgeGlobalConfig.providers?.[p] || {}
  if (p === 'dify') {
    const v = Number(cfg.score_threshold)
    if (!Number.isNaN(v) && v >= 0 && v <= 1) return v
    return DEFAULT_DIFY_THRESHOLD
  }
  if (p === 'ragflow') {
    const v = Number(cfg.similarity_threshold)
    if (!Number.isNaN(v) && v >= 0 && v <= 1) return v
    return DEFAULT_RAGFLOW_THRESHOLD
  }
  if (p === 'weknora') {
    const v = Number(cfg.score_threshold)
    if (!Number.isNaN(v) && v >= 0 && v <= 1) return v
    return DEFAULT_WEKNORA_THRESHOLD
  }
  return DEFAULT_DIFY_THRESHOLD
}

const openDialog = (row = null) => {
  editing.value = !!row
  currentId.value = row?.id || null
  form.name = row?.name || ''
  form.description = row?.description || ''
  form.status = row?.status || 'active'
  const provider = normalizeProvider(row?.sync_provider || knowledgeGlobalConfig.default_provider || 'dify')
  const globalThreshold = getGlobalThresholdByProvider(provider)
  form.threshold_provider = provider
  form.global_threshold = globalThreshold
  if (row && row.retrieval_threshold !== null && row.retrieval_threshold !== undefined) {
    form.inherit_global_threshold = false
    form.retrieval_threshold_text = String(row.retrieval_threshold)
  } else {
    form.inherit_global_threshold = true
    form.retrieval_threshold_text = String(globalThreshold)
  }
  dialogVisible.value = true
}

const submit = async () => {
  if (!form.name.trim()) {
    ElMessage.error('Tên  không được để trống')
    return
  }
  const rawThreshold = String(form.retrieval_threshold_text || '').trim()
  const threshold = Number(rawThreshold)
  if (!rawThreshold || Number.isNaN(threshold) || threshold < 0 || threshold > 1) {
    ElMessage.error('Ngưỡng tìm kiếm phải nằm trong khoảng 0~1')
    return
  }
  const globalThreshold = Number(form.global_threshold)
  const sameAsGlobal = !Number.isNaN(globalThreshold) && Math.abs(threshold - globalThreshold) < 0.000001
  if (form.inherit_global_threshold && !sameAsGlobal) {
    form.inherit_global_threshold = false
  }
  try {
    const useInheritGlobal = form.inherit_global_threshold && sameAsGlobal
    const payload = {
      name: form.name,
      description: form.description,
      status: form.status,
      inherit_global_threshold: useInheritGlobal,
      retrieval_threshold: useInheritGlobal ? null : threshold
    }
    let res = null
    if (editing.value) {
      res = await api.put(`/user/knowledge-bases/${currentId.value}`, payload)
    } else {
      res = await api.post('/user/knowledge-bases', payload)
    }
    ElMessage.success('Đã lưu')
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    }
    dialogVisible.value = false
    await loadData()
  } catch (e) {
    ElMessage.error('Không thể lưu')
  }
}

const removeItem = async (id) => {
  try {
    await ElMessageBox.confirm('Bạn có chắc chắn xóa cơ sở kiến thức này và tất cả tài liệu của nó không?', 'Thông báo', { type: 'warning' })
    const res = await api.delete(`/user/knowledge-bases/${id}`)
    ElMessage.success('Đã xóa')
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    }
    await loadData()
  } catch {}
}

const isStatusSwitchLoading = (id) => !!statusSwitchLoadingMap.value?.[id]

const toggleKnowledgeBaseStatus = async (row, checked) => {
  if (!row?.id) return
  const id = row.id
  const prevStatus = String(row.status || 'inactive').trim() === 'active' ? 'active' : 'inactive'
  const nextStatus = checked ? 'active' : 'inactive'
  if (prevStatus === nextStatus) return
  if (isStatusSwitchLoading(id)) return

  statusSwitchLoadingMap.value = {
    ...statusSwitchLoadingMap.value,
    [id]: true
  }
  row.status = nextStatus

  try {
    const res = await api.put(`/user/knowledge-bases/${id}`, {
      name: row.name || '',
      description: row.description || '',
      content: row.content || '',
      status: nextStatus
    })
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    } else {
      ElMessage.success(`có${nextStatus === 'active' ? 'Bật' : 'bị vô hiệu hóa'}`)
    }
    await loadData()
  } catch (e) {
    row.status = prevStatus
    const msg = e?.response?.data?.error || 'Cập nhật trạng thái  không thành công'
    ElMessage.error(msg)
  } finally {
    statusSwitchLoadingMap.value = {
      ...statusSwitchLoadingMap.value,
      [id]: false
    }
  }
}

const handleKnowledgeBaseAction = async (command, row) => {
  if (!row?.id) return
  if (command === 'edit') {
    openDialog(row)
    return
  }
  if (command === 'sync') {
    await syncItem(row.id)
    return
  }
  if (command === 'delete') {
    await removeItem(row.id)
  }
}

const syncItem = async (id) => {
  try {
    const res = await api.post(`/user/knowledge-bases/${id}/sync`)
    ElMessage.success(res?.data?.message || 'Nhiệm vụ đồng bộ hóa  đã được gửi')
    await loadData()
  } catch (e) {
    const msg = e?.response?.data?.error || 'Đồng bộ hóa  không thành công'
    ElMessage.error(msg)
    await loadData()
  }
}

const openSearchTestDialog = (row) => {
  searchTestKb.value = row || null
  searchTestForm.query = ''
  searchTestForm.top_k = 5
  const provider = normalizeProvider(row?.sync_provider || knowledgeGlobalConfig.default_provider || 'dify')
  const globalThreshold = getGlobalThresholdByProvider(provider)
  const kbThreshold = row?.retrieval_threshold
  const effectiveThreshold = (kbThreshold !== null && kbThreshold !== undefined) ? Number(kbThreshold) : Number(globalThreshold)
  searchTestForm.threshold_text = Number.isNaN(effectiveThreshold) ? '' : String(effectiveThreshold)
  searchTestResult.query = ''
  searchTestResult.count = 0
  searchTestResult.hits = []
  searchTestElapsedMs.value = null
  hasRunSearchTest.value = false
  searchTestVisible.value = true
}

const runSearchTest = async () => {
  if (!searchTestKb.value?.id) {
    ElMessage.error('Vui lòng chọn cơ sở kiến thức trước')
    return
  }
  const query = (searchTestForm.query || '').trim()
  if (!query) {
    ElMessage.error('Vui lòng nhập từ khóa kiểm tra')
    return
  }
  searchTestLoading.value = true
  const startedAt = Date.now()
  try {
    const rawThreshold = String(searchTestForm.threshold_text || '').trim()
    let threshold = null
    if (rawThreshold !== '') {
      const parsed = Number(rawThreshold)
      if (Number.isNaN(parsed) || parsed < 0 || parsed > 1) {
        ElMessage.error('Ngưỡng  phải nằm trong khoảng 0~1')
        return
      }
      threshold = parsed
    }
    const payload = {
      query,
      top_k: Number(searchTestForm.top_k) || 5,
      threshold
    }
    const res = await api.post(`/user/knowledge-bases/${searchTestKb.value.id}/test-search`, payload)
    const data = res?.data?.data || {}
    searchTestResult.query = data.query || query
    searchTestResult.count = Number(data.count || 0)
    searchTestResult.hits = Array.isArray(data.hits) ? data.hits : []
    const elapsed = Number(data.elapsed_ms)
    searchTestElapsedMs.value = Number.isNaN(elapsed) ? Date.now() - startedAt : elapsed
    hasRunSearchTest.value = true
    ElMessage.success(`Việc truy xuất  đã hoàn tất, tổng số tiền truy xuất ${searchTestResult.count} kết quả`)
  } catch (e) {
    const msg = e?.response?.data?.error || 'Thử nghiệm  không thành công'
    ElMessage.error(msg)
  } finally {
    searchTestLoading.value = false
  }
}

const openDocuments = async (row) => {
  currentKb.value = row
  documentsVisible.value = true
  await loadDocuments()
}

const loadDocuments = async () => {
  if (!currentKb.value?.id) return
  documentsLoading.value = true
  try {
    const res = await api.get(`/user/knowledge-bases/${currentKb.value.id}/documents`)
    documentItems.value = res.data.data || []
  } finally {
    documentsLoading.value = false
  }
}

const openDocumentDialog = (row = null) => {
  if (row && isUploadedFileDocument(row)) {
    ElMessage.warning('Tài liệu dạng file  không hỗ trợ chỉnh sửa trực tuyến, vui lòng xóa và tải lên lại.')
    return
  }
  documentEditing.value = !!row
  currentDocumentId.value = row?.id || null
  documentForm.name = row?.name || ''
  documentForm.content = row?.content || ''
  documentDialogVisible.value = true
}

const submitDocument = async () => {
  if (!currentKb.value?.id) return
  if (!documentForm.name.trim()) {
    ElMessage.error('Tên tài liệu không được để trống')
    return
  }
  if (!documentForm.content.trim()) {
    ElMessage.error('Nội dung tài liệu  không được để trống')
    return
  }
  try {
    let res = null
    if (documentEditing.value) {
      res = await api.put(`/user/knowledge-bases/${currentKb.value.id}/documents/${currentDocumentId.value}`, documentForm)
    } else {
      res = await api.post(`/user/knowledge-bases/${currentKb.value.id}/documents`, documentForm)
    }
    ElMessage.success('Tài liệu  đã được lưu thành công')
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    }
    documentDialogVisible.value = false
    await loadDocuments()
    await loadData()
  } catch (e) {
    const msg = e?.response?.data?.error || 'Lưu tài liệu không thành công'
    ElMessage.error(msg)
  }
}

const removeDocument = async (docId) => {
  if (!currentKb.value?.id) return
  try {
    await ElMessageBox.confirm('Bạn có chắc chắn xóa tài liệu này không?', 'Thông báo', { type: 'warning' })
    const res = await api.delete(`/user/knowledge-bases/${currentKb.value.id}/documents/${docId}`)
    ElMessage.success('Đã xóa')
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    }
    await loadDocuments()
    await loadData()
  } catch {}
}

const syncDocument = async (docId) => {
  if (!currentKb.value?.id) return
  try {
    const res = await api.post(`/user/knowledge-bases/${currentKb.value.id}/documents/${docId}/sync`)
    ElMessage.success(res?.data?.message || 'Nhiệm vụ đồng bộ hóa  đã được gửi')
    await loadDocuments()
    await loadData()
  } catch (e) {
    const msg = e?.response?.data?.error || 'Đồng bộ hóa  không thành công'
    ElMessage.error(msg)
  }
}

const uploadDocumentFile = async (options) => {
  if (!currentKb.value?.id) {
    ElMessage.error('Vui lòng chọn cơ sở kiến thức trước')
    options?.onError?.(new Error('missing knowledge base'))
    return
  }
  if (!isUploadProviderSupported.value) {
    ElMessage.error(`Nhà cung cấp cơ sở tri thức hiện tại là ${currentKBProvider.value}, hiện tại chưa hỗ trợ upload file để tạo tài liệu`)
    options?.onError?.(new Error('provider not supported'))
    return
  }
  const file = options?.file
  if (!file) {
    ElMessage.error('Vui lòng chọn file để upload')
    options?.onError?.(new Error('missing file'))
    return
  }

  const formData = new FormData()
  formData.append('file', file)
  const fileName = (file.name || '').replace(/\.[^/.]+$/, '')
  if (fileName) {
    formData.append('name', fileName)
  }

  try {
    const res = await api.post(`/user/knowledge-bases/${currentKb.value.id}/documents/upload`, formData)
    ElMessage.success(res?.data?.message || 'Tệp  được tải lên thành công')
    if (res?.data?.warning) {
      ElMessage.warning(res.data.warning)
    }
    await loadDocuments()
    await loadData()
    options?.onSuccess?.(res?.data)
  } catch (e) {
    const msg = e?.response?.data?.error || 'Tải lên tệp  không thành công'
    ElMessage.error(msg)
    options?.onError?.(e)
  }
}

const isUploadedFileDocument = (doc) => {
  const content = doc?.content
  return typeof content === 'string' && content.startsWith(FILE_UPLOAD_CONTENT_PREFIX)
}

const getDocumentPreview = (doc) => {
  const content = doc?.content || ''
  if (isUploadedFileDocument(doc)) {
    try {
      const payload = JSON.parse(content.slice(FILE_UPLOAD_CONTENT_PREFIX.length))
      const fileName = payload?.file_name || doc?.name || 'Tệp tải lên'
      return `[Tệp] ${fileName}`
    } catch {
      return `[Tệp] ${doc?.name || 'Tệp tải lên'}`
    }
  }
  const text = String(content)
  return `${text.slice(0, 120)}${text.length > 120 ? '...' : ''}`
}

const getSyncStatusText = (status) => {
  if (status === 'uploading') return 'Đang tải lên'
  if (status === 'uploaded') return 'đã tải lên'
  if (status === 'parsing') return 'đang phân tích'
  if (status === 'upload_failed') return 'Tải lên  không thành công'
  if (status === 'parse_failed') return 'Phân tích cú pháp  không thành công'
  if (status === 'synced') return 'được đồng bộ hóa'
  if (status === 'failed') return 'Không thành công'
  return 'được đồng bộ hóa'
}

const getSyncStatusTagType = (status) => {
  if (status === 'upload_failed' || status === 'parse_failed') return 'danger'
  if (status === 'uploading' || status === 'parsing') return 'warning'
  if (status === 'uploaded') return 'info'
  if (status === 'synced') return 'success'
  if (status === 'failed') return 'danger'
  return 'warning'
}

const getKnowledgeStatusText = (status) => {
  return String(status || '').trim() === 'active' ? 'Bật' : 'bị vô hiệu hóa'
}

const formatProviderText = (provider) => {
  const p = String(provider || '').trim().toLowerCase()
  if (p === 'ragflow') return 'RAGFlow'
  if (p === 'weknora') return 'WeKnora'
  if (p === 'dify') return 'Dify'
  return provider || '-'
}

const shouldShowSyncErrorTip = (row) => {
  const status = String(row?.sync_status || '').trim()
  const syncError = String(row?.sync_error || '').trim()
  if (!syncError) return false
  return status === 'failed' || status === 'upload_failed' || status === 'parse_failed'
}

const formatHitScore = (score) => {
  const n = Number(score)
  if (Number.isNaN(n)) return '-'
  return n.toFixed(4)
}

const formatDocCount = (value) => {
  const n = Number(value)
  if (Number.isNaN(n) || n < 0) return 0
  return n
}

const formatDateTimeCell = (value) => {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  return d.toLocaleString('vi-VN')
}

const formatKnowledgeThreshold = (value) => {
  if (value === null || value === undefined || value === '') return 'Dùng chung'
  const n = Number(value)
  if (Number.isNaN(n)) return 'Dùng chung'
  return n.toFixed(2)
}

onMounted(async () => {
  await loadData()
})
</script>

<style scoped>
.page-actions {
  display: flex;
  justify-content: flex-end;
  margin: 10px 0 14px;
}

.kb-sync-status-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.kb-sync-error-tooltip {
  max-width: 320px;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.5;
}

.kb-sync-error-icon {
  color: var(--el-color-danger);
  cursor: pointer;
  font-size: 14px;
}

.kb-desc-text {
  color: var(--el-text-color-regular);
}

.kb-desc-text.is-empty {
  color: var(--el-text-color-placeholder);
}

.action-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}

.action-buttons :deep(.el-button) {
  margin: 0;
  white-space: nowrap;
}

.action-buttons :deep(.el-dropdown) {
  display: inline-flex;
}

.dialog-toolbar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.dialog-toolbar-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.kb-helper-text {
  color: var(--apple-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.kb-helper-text.is-spaced {
  margin-top: 6px;
}

.kb-helper-text.is-bottom {
  margin-bottom: 8px;
}

.kb-helper-text.is-empty {
  margin-top: 10px;
}

.kb-helper-text.is-regular {
  color: var(--el-text-color-regular);
}

.search-hit-content {
  white-space: pre-wrap;
  line-height: 1.4;
}
</style>
