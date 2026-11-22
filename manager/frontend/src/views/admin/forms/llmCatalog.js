const defaultOption = { label: 'Mặc định', value: 'default' }
const enableOption = { label: 'bật', value: 'enabled' }
const disableOption = { label: 'Đóng', value: 'disabled' }
const clearHistoryOptions = [
  { label: 'Mặc định', value: 'default' },
  { label: 'Rõ ràng', value: true },
  { label: 'dành riêng', value: false }
]

function withDefault(options) {
  return [defaultOption, ...options]
}

function createModel(value, thinking, extra = {}) {
  return {
    value,
    label: value,
    thinking,
    ...extra
  }
}

const openAIReasoningStandard = withDefault([
  { label: 'cực thấp', value: 'minimal' },
  { label: 'thấp', value: 'low' },
  { label: 'vừa', value: 'medium' },
  { label: 'cao', value: 'high' }
])

const openAIReasoningCodex = withDefault([
  { label: 'Đóng', value: 'none' },
  { label: 'thấp', value: 'low' },
  { label: 'vừa', value: 'medium' },
  { label: 'cao', value: 'high' }
])

const openAIReasoningCodexMax = withDefault([
  { label: 'Đóng', value: 'none' },
  { label: 'thấp', value: 'low' },
  { label: 'vừa', value: 'medium' },
  { label: 'cao', value: 'high' },
  { label: 'cực cao', value: 'xhigh' }
])

const openAIReasoningLegacy = withDefault([
  { label: 'thấp', value: 'low' },
  { label: 'vừa', value: 'medium' },
  { label: 'cao', value: 'high' }
])

const openAIReasoningHighOnly = withDefault([
  { label: 'cao', value: 'high' }
])

const booleanThinkingOptions = withDefault([
  enableOption,
  disableOption
])

const doubaoReasoningOptions = withDefault([
  { label: 'Đóng', value: 'minimal' },
  { label: 'thấp', value: 'low' },
  { label: 'vừa', value: 'medium' },
  { label: 'cao', value: 'high' }
])

const anthropicAdaptiveOptions = [
  { label: 'thấp', value: 'low' },
  { label: 'vừa', value: 'medium' },
  { label: 'cao', value: 'high' },
  { label: 'cực cao', value: 'max' }
]

const openAIReasoningLatest = withDefault([
  { label: 'Đóng', value: 'none' },
  { label: 'thấp', value: 'low' },
  { label: 'vừa', value: 'medium' },
  { label: 'cao', value: 'high' },
  { label: 'cực cao', value: 'xhigh' }
])

const openAIReasoningLatestPro = withDefault([
  { label: 'vừa', value: 'medium' },
  { label: 'cao', value: 'high' },
  { label: 'cực cao', value: 'xhigh' }
])

const openAIReasoningRequest = {
  allowMaxTokens: false,
  allowTemperature: false,
  allowTopP: false
}

const anthropicManualThinking = {
  label: 'suy nghĩ sâu sắc',
  options: withDefault([{ label: 'Tư duy thủ công', value: 'enabled' }]),
  showBudgetFor: ['enabled'],
  budgetMin: 1024,
  budgetRequiredFor: ['enabled']
}

const anthropicAdaptiveThinking = {
  label: 'suy nghĩ sâu sắc',
  options: withDefault([
    { label: 'Tư duy thủ công', value: 'enabled' },
    { label: 'Tư duy thích ứng', value: 'adaptive' }
  ]),
  showBudgetFor: ['enabled'],
  budgetMin: 1024,
  budgetRequiredFor: ['enabled'],
  showEffortFor: ['adaptive'],
  effortOptions: anthropicAdaptiveOptions
}

const zhipuThinkingConfig = {
  label: 'suy nghĩ sâu sắc',
  options: booleanThinkingOptions,
  showClearThinkingFor: ['enabled'],
  clearThinkingOptions: clearHistoryOptions
}

const aliyunThinkingConfig = {
  label: 'suy nghĩ sâu sắc',
  options: booleanThinkingOptions,
  showBudgetFor: ['enabled'],
  budgetMin: 1,
  budgetStep: 256
}

const siliconflowThinkingConfig = {
  label: 'suy nghĩ sâu sắc',
  options: booleanThinkingOptions,
  showBudgetFor: ['enabled'],
  budgetMin: 128,
  budgetMax: 32768,
  budgetStep: 128
}

const providerTypeMap = {
  openai: 'openai',
  ollama: 'ollama',
  azure: 'openai',
  anthropic: 'openai',
  zhipu: 'openai',
  aliyun: 'openai',
  doubao: 'openai',
  siliconflow: 'openai',
  deepseek: 'openai',
  dify: 'dify',
  coze: 'coze'
}

const knownProviders = new Set(Object.keys(providerTypeMap))

const editableBaseURLProviders = new Set(['openai', 'ollama', 'azure', 'dify', 'coze'])

const catalog = {
  openai: {
    quickUrl: 'https://api.openai.com/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Theo mặc định,  ưu tiên các bí danh ổn định chính thức; nếu bạn cần khóa hành vi, bạn có thể nhập ID mô hình ảnh chụp nhanh chính xác theo cách thủ công.',
    models: [
      createModel('gpt-5.4', { label: 'Cường độ tư duy', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-pro', { label: 'Cường độ tư duy', options: openAIReasoningLatestPro }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-mini', { label: 'Cường độ tư duy', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-nano', { label: 'Cường độ tư duy', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2', { label: 'Cường độ tư duy', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-pro', { label: 'Cường độ tư duy', options: openAIReasoningLatestPro }, { request: openAIReasoningRequest }),
      createModel('gpt-5-chat-latest', false, { hint: 'Bí danh dành riêng cho  ChatGPT, phù hợp để tương thích với quy trình công việc cũ; quyền truy cập mới sẽ ưu tiên cho mẫu GPT-5.* dòng chính.' }),
      createModel('gpt-5-pro', { label: 'Cường độ tư duy', options: openAIReasoningHighOnly }, { request: openAIReasoningRequest }),
      createModel('gpt-5', { label: 'Cường độ tư duy', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-mini', { label: 'Cường độ tư duy', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-nano', { label: 'Cường độ tư duy', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5.3-codex', { label: 'Cường độ tư duy', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-codex', { label: 'Cường độ tư duy', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5-codex', { label: 'Cường độ tư duy', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1', { label: 'Cường độ tư duy', options: openAIReasoningCodex }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1-codex', { label: 'Cường độ tư duy', options: openAIReasoningCodex }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1-codex-mini', { label: 'Cường độ tư duy', options: openAIReasoningCodex }, { request: openAIReasoningRequest }),
      createModel('gpt-5.1-codex-max', { label: 'Cường độ tư duy', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('o3', { label: 'Cường độ tư duy', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o4-mini', { label: 'Cường độ tư duy', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o3-mini', { label: 'Cường độ tư duy', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o1', { label: 'Cường độ tư duy', options: openAIReasoningLegacy }, { request: openAIReasoningRequest })
    ],
    fallbackThinking: {
      label: 'Cường độ tư duy',
      options: openAIReasoningCodex,
      hint: 'Mô hình tùy chỉnh không khớp với danh sách trong tài liệu và đã quay trở lại cấu hình reasoning_effort chung; việc nó có hiệu lực hay không phụ thuộc vào mô hình thực tế.'
    }
  },
  ollama: {
    quickUrl: 'http://127.0.0.1:11434/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Ollama sử dụng các dịch vụ mô hình cục bộ hoặc riêng tư và cả danh sách mô hình và địa chỉ đều cho phép tùy chỉnh.',
    models: [],
    fallbackThinking: null
  },
  azure: {
    quickUrl: 'https://your-resource-name.openai.azure.com/openai/v1/',
    modelPlaceholder: 'Vui lòng chọn tên mẫu chính thức hoặc nhập tên triển khai tùy chỉnh',
    modelHint: 'Azure Tên triển khai được điền vào đây; các tên trong danh sách chủ yếu được sử dụng để chỉ các khả năng của mô hình cơ bản của nó.',
    models: [
      createModel('gpt-5.4', { label: 'Cường độ tư duy', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.4-pro', { label: 'Cường độ tư duy', options: openAIReasoningLatestPro }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2', { label: 'Cường độ tư duy', options: openAIReasoningLatest }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-chat', false, { hint: 'Mô hình Trò chuyện trong tài liệu Azure thường được truy cập thông qua tên triển khai; việc nó có mở hay không tùy thuộc vào khu vực và hạn ngạch.' }),
      createModel('gpt-5.3-codex', { label: 'Cường độ tư duy', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5.2-codex', { label: 'Cường độ tư duy', options: openAIReasoningCodexMax }, { request: openAIReasoningRequest }),
      createModel('gpt-5-mini', { label: 'Cường độ tư duy', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-nano', { label: 'Cường độ tư duy', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-chat', { label: 'Cường độ tư duy', options: openAIReasoningStandard }, { request: openAIReasoningRequest }),
      createModel('gpt-5-pro', { label: 'Cường độ tư duy', options: openAIReasoningHighOnly }, { request: openAIReasoningRequest }),
      createModel('o4-mini', { label: 'Cường độ tư duy', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o3', { label: 'Cường độ tư duy', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o3-mini', { label: 'Cường độ tư duy', options: openAIReasoningLegacy }, { request: openAIReasoningRequest }),
      createModel('o1', { label: 'Cường độ tư duy', options: openAIReasoningLegacy }, { request: openAIReasoningRequest })
    ],
    fallbackThinking: {
      label: 'Cường độ tư duy',
      options: openAIReasoningCodex,
      hint: 'Khi triển khai tùy chỉnh Azure bỏ sót mô hình tài liệu, nó sẽ quay trở lại cấu hình reasoning_effort chung; khả năng tương thích cụ thể tùy thuộc vào mô hình triển khai.'
    }
  },
  anthropic: {
    quickUrl: 'https://api.anthropic.com/v1/',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'theo mặc định ưu tiên cho bí danh ổn định chính thức; nếu cần có phiên bản cố định hoặc kiểm tra hồi quy, bạn có thể thay đổi nó thành ID mẫu chính xác kèm theo ngày.',
    models: [
      createModel('claude-opus-4-6', anthropicAdaptiveThinking),
      createModel('claude-sonnet-4-6', anthropicAdaptiveThinking),
      createModel('claude-haiku-4-5', anthropicManualThinking),
      createModel('claude-3-7-sonnet', anthropicManualThinking),
      createModel('claude-sonnet-4', anthropicManualThinking),
      createModel('claude-opus-4', anthropicManualThinking),
      createModel('claude-opus-4-1', anthropicManualThinking)
    ],
    fallbackThinking: {
      ...anthropicAdaptiveThinking,
      hint: 'Mẫu tùy chỉnh  thiếu danh sách trong tài liệu. Nếu bạn sử dụng tư duy thủ công, bạn cần điền rõ ràng budget_tokens; vui lòng chỉ sử dụng Thích ứng trên các kiểu máy được xác nhận là được tài liệu hỗ trợ.'
    }
  },
  zhipu: {
    quickUrl: 'https://open.bigmodel.cn/api/paas/v4',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Tài liệu  Zhipu hỗ trợ điều khiển chế độ tư duy thông qua thinking.type và clear_thinking.',
    models: [
      createModel('glm-5', zhipuThinkingConfig),
      createModel('glm-4.7', zhipuThinkingConfig),
      createModel('glm-4.7-flashx', zhipuThinkingConfig),
      createModel('glm-4.7-flash', zhipuThinkingConfig),
      createModel('glm-4.6', zhipuThinkingConfig),
      createModel('glm-4.6v', zhipuThinkingConfig),
      createModel('glm-4.5', zhipuThinkingConfig),
      createModel('glm-4.5-air', zhipuThinkingConfig),
      createModel('glm-4.5-airx', zhipuThinkingConfig),
      createModel('glm-4.5v', zhipuThinkingConfig)
    ],
    fallbackThinking: {
      ...zhipuThinkingConfig,
      hint: 'Mô hình tùy chỉnh  bỏ lỡ danh sách trong tài liệu và đã quay trở lại cấu hình thinking.type / clear_thinking chung.'
    }
  },
  aliyun: {
    quickUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Theo mặc định,  ưu tiên các bí danh ổn định chính thức; nếu bạn muốn khóa một phiên bản cụ thể, hãy điền thủ công ID mẫu máy với hậu tố ngày hoặc phiên bản phụ.',
    models: [
      createModel('qwen-plus-latest', aliyunThinkingConfig),
      createModel('qwen-turbo-latest', aliyunThinkingConfig),
      createModel('qwen3-max', aliyunThinkingConfig),
      createModel('qwen3-235b-a22b', aliyunThinkingConfig),
      createModel('qwen3-30b-a3b', aliyunThinkingConfig),
      createModel('qwen3-next-80b-a3b-thinking', aliyunThinkingConfig),
      createModel('glm-4.7', aliyunThinkingConfig),
      createModel('glm-4.6', aliyunThinkingConfig),
      createModel('glm-4.5', aliyunThinkingConfig),
      createModel('glm-4.5-air', aliyunThinkingConfig),
      createModel('kimi-k2-thinking', aliyunThinkingConfig),
      createModel('qwen3-235b-a22b-thinking-2507', aliyunThinkingConfig, { label: 'qwen3-235b-a22b-thinking-2507 (đã phiên bản)' }),
      createModel('qwen3-30b-a3b-thinking-2507', aliyunThinkingConfig, { label: 'qwen3-30b-a3b-thinking-2507 (đã phiên bản)' }),
      createModel('kimi/kimi-k2.5', aliyunThinkingConfig, { label: 'kimi/kimi-k2.5 (phiên bản)' })
    ],
    fallbackThinking: {
      ...aliyunThinkingConfig,
      hint: 'Mẫu tùy chỉnh  thiếu danh sách trong tài liệu. Nếu mô hình hỗ trợ thinking_budget thì có thể điền theo tài liệu; nếu để trống, trường này sẽ không được thông qua.'
    }
  },
  doubao: {
    quickUrl: 'https://ark.cn-beijing.volces.com/api/v3',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập ID model (thường có hậu tố phiên bản)',
    modelHint: 'Doubao ưu tiên điền ID Model thực chính thức. Hiện tại không có bí danh ổn định nào được xác nhận có thể được sử dụng như một sự thay thế phổ biến. Chúng tôi khuyên bạn nên sử dụng ID mẫu trong bảng điều khiển hoặc danh sách mẫu.',
    models: [
      createModel('doubao-seed-2-0-pro-260215', { label: 'Cường độ tư duy', options: doubaoReasoningOptions }, { label: 'Doubao Seed 2.0 Pro (doubao-seed-2-0-pro-260215)' }),
      createModel('doubao-seed-2-0-lite-260215', { label: 'Cường độ tư duy', options: doubaoReasoningOptions }, { label: 'Doubao Seed 2.0 Lite (doubao-seed-2-0-lite-260215)' }),
      createModel('doubao-seed-2-0-mini-260215', { label: 'Cường độ tư duy', options: doubaoReasoningOptions }, { label: 'Doubao Seed 2.0 Mini (doubao-seed-2-0-mini-260215)' }),
      createModel('doubao-seed-1-6-251015', { label: 'Cường độ tư duy', options: doubaoReasoningOptions }, { label: 'Doubao Seed 1.6 (doubao-seed-1-6-251015)' })
    ],
    fallbackThinking: {
      label: 'Cường độ tư duy',
      options: doubaoReasoningOptions,
      hint: 'Mô hình tùy chỉnh không khớp với danh sách trong tài liệu và đã quay trở lại cấu hình reasoning_effort chung; việc nó có hiệu lực hay không phụ thuộc vào mô hình thực tế.'
    }
  },
  siliconflow: {
    quickUrl: 'https://api.siliconflow.cn/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'Tài liệu  SiliconFlow liệt kê trực tiếp các mô hình được hỗ trợ enable_thinking; chỉ cấu hình ngân sách mới được hiển thị cho các mô hình được liệt kê trong tài liệu.',
    models: [
      createModel('Pro/zai-org/GLM-5', siliconflowThinkingConfig),
      createModel('Pro/zai-org/GLM-4.7', siliconflowThinkingConfig),
      createModel('deepseek-ai/DeepSeek-V3.2', siliconflowThinkingConfig),
      createModel('Pro/deepseek-ai/DeepSeek-V3.2', siliconflowThinkingConfig),
      createModel('zai-org/GLM-4.6', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-8B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-14B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-32B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3-30B-A3B', siliconflowThinkingConfig),
      createModel('tencent/Hunyuan-A13B-Instruct', siliconflowThinkingConfig),
      createModel('zai-org/GLM-4.5V', siliconflowThinkingConfig),
      createModel('deepseek-ai/DeepSeek-V3.1-Terminus', siliconflowThinkingConfig),
      createModel('Pro/deepseek-ai/DeepSeek-V3.1-Terminus', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-397B-A17B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-122B-A10B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-35B-A3B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-27B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-9B', siliconflowThinkingConfig),
      createModel('Qwen/Qwen3.5-4B', siliconflowThinkingConfig)
    ],
    fallbackThinking: {
      ...siliconflowThinkingConfig,
      hint: 'Mẫu tùy chỉnh  thiếu danh sách trong tài liệu. Nếu mô hình hỗ trợ enable_thinking/thinking_budget thì có thể điền theo tài liệu; thinking_budget sẽ không được thông qua khi để trống.'
    }
  },
  deepseek: {
    quickUrl: 'https://api.deepseek.com/v1',
    modelPlaceholder: 'Vui lòng chọn hoặc nhập tên model',
    modelHint: 'DeepSeek chính thức của  chuyển chế độ tư duy bằng cách chọn các mô hình khác nhau: deepseek-chat là không suy nghĩ, deepseek-reasoner là suy nghĩ.',
    models: [
      createModel('deepseek-chat', false, {
        hint: 'deepseek-chat là mô hình không cần tư duy và không yêu cầu thêm thông số tư duy.'
      }),
      createModel('deepseek-reasoner', false, {
        hint: 'deepseek-reasoner được tích hợp sẵn chế độ tư duy và không yêu cầu thêm thông số tư duy.'
      })
    ],
    fallbackThinking: {
      label: 'suy nghĩ sâu sắc',
      options: booleanThinkingOptions,
      hint: 'DeepSeek chính thức của  khuyên bạn nên chuyển chế độ suy nghĩ thông qua tên model. Nếu tác nhân tùy chỉnh hỗ trợ thêm thinking.type, thì bạn có thể bật công tắc tương thích tại đây.'
    }
  }
}

function cloneOptions(options = []) {
  return options.map(option => ({ ...option }))
}

function normalizeModelName(modelName) {
  return String(modelName || '').trim().toLowerCase()
}

export function resolveLLMProvider(provider, type) {
  const normalizedProvider = String(provider || '').trim().toLowerCase()
  const normalizedType = String(type || '').trim().toLowerCase()

  if (normalizedProvider === 'openai' && ['ollama', 'dify', 'coze'].includes(normalizedType)) {
    return normalizedType
  }
  if (knownProviders.has(normalizedProvider)) {
    return normalizedProvider
  }
  if (['ollama', 'dify', 'coze'].includes(normalizedType)) {
    return normalizedType
  }
  return 'openai'
}

export function getProviderFixedType(provider) {
  return providerTypeMap[provider] || 'openai'
}

export function isProviderBaseURLEditable(provider) {
  return editableBaseURLProviders.has(provider)
}

export function getProviderQuickUrl(provider) {
  return catalog[provider]?.quickUrl || ''
}

export function getProviderModelOptions(provider) {
  return (catalog[provider]?.models || []).map(model => ({
    label: model.label,
    value: model.value
  }))
}

export function getProviderModelHint(provider) {
  return catalog[provider]?.modelHint || ''
}

export function getProviderModelFieldLabel(provider) {
  if (provider === 'azure') {
    return 'Tên triển khai'
  }
  if (provider === 'doubao') {
    return 'ID mẫu'
  }
  return 'Tên mẫu'
}

export function getProviderModelPlaceholder(provider) {
  return catalog[provider]?.modelPlaceholder || 'Vui lòng chọn hoặc nhập tên model'
}

export function resolveProviderModel(provider, modelName) {
  const normalized = normalizeModelName(modelName)
  if (!normalized) {
    return null
  }

  const models = catalog[provider]?.models || []
  return models.find(model => normalizeModelName(model.value) === normalized) || null
}

export function getProviderRequestConfig(provider, modelName) {
  const model = resolveProviderModel(provider, modelName)
  return {
    allowMaxTokens: true,
    allowTemperature: true,
    allowTopP: true,
    temperatureMax: 2,
    ...(model?.request || {})
  }
}

export function getProviderThinkingConfig(provider, modelName) {
  const model = resolveProviderModel(provider, modelName)
  if (model?.thinking === false) {
    return {
      visible: false,
      hint: model.hint || ''
    }
  }

  const source = model?.thinking || catalog[provider]?.fallbackThinking
  if (!source) {
    return {
      visible: false,
      hint: model?.hint || ''
    }
  }

  return {
    visible: true,
    label: source.label || 'suy nghĩ sâu sắc',
    options: cloneOptions(source.options),
    showBudgetFor: [...(source.showBudgetFor || [])],
    budgetMin: source.budgetMin || 1,
    budgetMax: source.budgetMax || 100000,
    budgetStep: source.budgetStep || 1,
    budgetRequiredFor: [...(source.budgetRequiredFor || [])],
    showEffortFor: [...(source.showEffortFor || [])],
    effortOptions: cloneOptions(source.effortOptions || []),
    showClearThinkingFor: [...(source.showClearThinkingFor || [])],
    clearThinkingOptions: cloneOptions(source.clearThinkingOptions || clearHistoryOptions),
    hint: model?.hint || source.hint || ''
  }
}
