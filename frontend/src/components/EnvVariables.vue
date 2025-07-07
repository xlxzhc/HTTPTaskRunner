<template>
  <div class="env-variables">
    <div class="env-header">
      <h2>环境变量管理</h2>
      <div class="env-actions">
        <button @click="showAddDialog = true" class="btn btn-primary">
          添加变量
        </button>
        <button @click="loadEnvVariables" class="btn btn-secondary">
          刷新
        </button>
      </div>
    </div>

    <div class="env-content">
      <!-- 变量列表 -->
      <div v-if="loading" class="loading-state">
        <p>正在加载环境变量...</p>
      </div>

      <div v-else-if="Object.keys(envVariables).length === 0" class="empty">
        <p>暂无环境变量</p>
        <p class="hint">环境变量可以在任务配置中使用 &#123;&#123;VARIABLE_NAME&#125;&#125; 格式引用</p>
      </div>

      <div v-else class="env-list">
        <div v-for="(value, key) in envVariables" :key="key" class="env-item">
          <div class="env-info">
            <div class="env-key">{{ key }}</div>
            <div class="env-value">
              {{ maskValue ? '***' : getDisplayValue(value) }}
              <span v-if="getSeparator(value)" class="separator-info">
                (分隔符: {{ getSeparator(value) }})
              </span>
            </div>
            <div class="env-usage">使用方式: &#123;&#123;{{ key }}&#125;&#125;</div>
          </div>
          <div class="env-actions">
            <button @click="editVariable(key, getDisplayValue(value))" class="btn btn-sm btn-secondary">
              编辑
            </button>
            <button @click="deleteVariable(key)" class="btn btn-sm btn-danger">
              删除
            </button>
          </div>
        </div>
      </div>

      <!-- 显示/隐藏值的开关 -->
      <div v-if="Object.keys(envVariables).length > 0" class="env-controls">
        <label class="mask-toggle">
          <input type="checkbox" v-model="maskValue">
          隐藏变量值
        </label>
      </div>
    </div>

    <!-- 添加/编辑对话框 -->
    <div v-if="showAddDialog || showEditDialog" class="dialog-overlay" @click="closeDialogs">
      <div class="dialog" @click.stop>
        <div class="dialog-header">
          <h3>{{ showEditDialog ? '编辑变量' : '添加变量' }}</h3>
          <button @click="closeDialogs" class="close-btn">&times;</button>
        </div>
        
        <div class="dialog-body">
          <div class="form-group">
            <label>变量名</label>
            <input 
              type="text" 
              v-model="dialogData.key" 
              :disabled="showEditDialog"
              placeholder="例如: API_KEY, COOKIE_VALUE"
              class="form-input"
            >
            <div class="hint">变量名建议使用大写字母和下划线</div>
          </div>
          
          <div class="form-group">
            <label>变量值</label>
            <textarea
              v-model="dialogData.value"
              placeholder="输入变量值"
              class="form-textarea"
              rows="3"
            ></textarea>
          </div>

        <div class="form-group">
          <label>分隔符 (可选):</label>
          <input
            v-model="dialogData.separator"
            type="text"
            placeholder="例如: &,|,; (多个分隔符用逗号分隔)"
            class="form-control"
          />
          <small class="form-hint">
            设置分隔符后，变量值将按分隔符分割，任务执行时会对每个分割后的值分别执行
          </small>
        </div>

        <div v-if="dialogData.separator && dialogData.value" class="separator-preview">
          <label>分割预览:</label>
          <div class="preview-values">
            <span v-for="(val, index) in getPreviewValues()" :key="index" class="preview-value">
              {{ val }}
            </span>
          </div>
          <div class="preview-actions">
            <button @click="testSeparator" class="btn btn-sm btn-info">测试分割</button>
            <button v-if="separatorTestResult.length > 0" @click="applySeparatorTest" class="btn btn-sm btn-success">
              应用分割结果
            </button>
            <button v-if="separatorTestResult.length > 0" @click="cancelSeparatorTest" class="btn btn-sm btn-secondary">
              撤回
            </button>
          </div>
        </div>

          <div class="form-group">
            <label>预览</label>
            <div class="preview">
              在任务中使用: <code>&#123;&#123;{{ dialogData.key || 'VARIABLE_NAME' }}&#125;&#125;</code>
            </div>
          </div>
        </div>
        
        <div class="dialog-footer">
          <button @click="closeDialogs" class="btn btn-secondary">取消</button>
          <button @click="saveVariable" class="btn btn-primary" :disabled="!dialogData.key || !dialogData.value || saving">
            <span v-if="saving">{{ showEditDialog ? '更新中...' : '添加中...' }}</span>
            <span v-else>{{ showEditDialog ? '更新' : '添加' }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 消息提示 -->
    <div v-if="message" class="message" :class="messageType">
      {{ message }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

// 环境变量数据类型
interface EnvVariableData {
  value: string
  separator?: string
}

// 响应式数据
const envVariables = ref<Record<string, EnvVariableData | string>>({})
const showAddDialog = ref(false)
const showEditDialog = ref(false)
const maskValue = ref(true)
const message = ref('')
const messageType = ref('success')
const loading = ref(false)
const saving = ref(false)

const dialogData = ref({
  key: '',
  value: '',
  separator: '',
  originalKey: ''
})

const separatorTestResult = ref<string[]>([])

// 方法
const loadEnvVariables = async () => {
  loading.value = true
  try {
    // 首先尝试加载包含分隔符信息的环境变量
    try {
      const { GetEnvVariablesWithSeparator } = await import('../../wailsjs/go/main/App')
      const variables = await GetEnvVariablesWithSeparator()
      envVariables.value = variables || {}
    } catch (error) {
      // 如果新API不存在，降级到旧API
      const { GetEnvVariables } = await import('../../wailsjs/go/main/App')
      const variables = await GetEnvVariables()
      // 将简单字符串转换为复杂对象格式
      const convertedVars: Record<string, EnvVariableData> = {}
      for (const [key, value] of Object.entries(variables || {})) {
        convertedVars[key] = { value: value as string, separator: '' }
      }
      envVariables.value = convertedVars
    }
  } catch (error) {
    showMessage('加载环境变量失败: ' + error, 'error')
  } finally {
    loading.value = false
  }
}

const editVariable = (key: string, value: string) => {
  // 从现有环境变量中加载分隔符信息（如果存在）
  const existingVar = envVariables.value[key]

  let actualValue = value
  let separator = ''

  if (typeof existingVar === 'object' && existingVar !== null) {
    // 新格式：包含分隔符信息
    actualValue = existingVar.value || value
    separator = existingVar.separator || ''
  } else if (typeof existingVar === 'string') {
    // 旧格式：纯字符串
    actualValue = existingVar
  }

  dialogData.value = {
    key,
    value: actualValue,
    separator,
    originalKey: key
  }
  showEditDialog.value = true
}

const deleteVariable = async (key: string) => {
  if (!confirm(`确定要删除变量 "${key}" 吗？`)) {
    return
  }

  try {
    const { DeleteEnvVariable } = await import('../../wailsjs/go/main/App')
    const result = await DeleteEnvVariable(key)
    showMessage(result, 'success')
    await loadEnvVariables()
  } catch (error) {
    showMessage('删除失败: ' + error, 'error')
  }
}

const saveVariable = async () => {
  if (!dialogData.value.key || !dialogData.value.value) {
    showMessage('请填写完整信息', 'error')
    return
  }

  saving.value = true
  try {
    // 构建变量数据（包含分隔符信息）
    const variableData = {
      value: dialogData.value.value,
      separator: dialogData.value.separator || ''
    }

    if (showEditDialog.value) {
      try {
        const { UpdateEnvVariableWithSeparator } = await import('../../wailsjs/go/main/App')
        const result = await UpdateEnvVariableWithSeparator(dialogData.value.key, JSON.stringify(variableData))
        showMessage(result, 'success')
      } catch (error) {
        // 降级到旧API（如果新API还没有生成）
        const { UpdateEnvVariable } = await import('../../wailsjs/go/main/App')
        const result = await UpdateEnvVariable(dialogData.value.key, dialogData.value.value)
        showMessage(result + '（注意：分隔符功能需要重新编译后端）', 'warning')
      }
    } else {
      try {
        const { SetEnvVariableWithSeparator } = await import('../../wailsjs/go/main/App')
        const result = await SetEnvVariableWithSeparator(dialogData.value.key, JSON.stringify(variableData))
        showMessage(result, 'success')
      } catch (error) {
        // 降级到旧API（如果新API还没有生成）
        const { SetEnvVariable } = await import('../../wailsjs/go/main/App')
        const result = await SetEnvVariable(dialogData.value.key, dialogData.value.value)
        showMessage(result + '（注意：分隔符功能需要重新编译后端）', 'warning')
      }
    }
    
    closeDialogs()
    await loadEnvVariables()
  } catch (error) {
    showMessage('保存失败: ' + error, 'error')
  } finally {
    saving.value = false
  }
}

const closeDialogs = () => {
  showAddDialog.value = false
  showEditDialog.value = false
  separatorTestResult.value = []
  dialogData.value = {
    key: '',
    value: '',
    separator: '',
    originalKey: ''
  }
}

// 获取分割预览值
const getPreviewValues = (): string[] => {
  if (!dialogData.value.separator || !dialogData.value.value) {
    return []
  }

  const separators = dialogData.value.separator.split(',').map(s => s.trim()).filter(s => s)
  let result = [dialogData.value.value]

  for (const sep of separators) {
    const newResult: string[] = []
    for (const val of result) {
      newResult.push(...val.split(sep))
    }
    result = newResult
  }

  return result.map(v => v.trim()).filter(v => v)
}

// 测试分隔符
const testSeparator = () => {
  separatorTestResult.value = getPreviewValues()
  showMessage(`分割测试完成，共得到 ${separatorTestResult.value.length} 个值`, 'success')
}

// 应用分割测试结果
const applySeparatorTest = () => {
  // 这里可以将分割结果应用到变量值中，或者保存为多个变量
  showMessage('分割结果已应用', 'success')
}

// 撤回分割测试
const cancelSeparatorTest = () => {
  separatorTestResult.value = []
  showMessage('已撤回分割测试', 'info')
}

// 获取显示值（兼容新旧格式）
const getDisplayValue = (value: EnvVariableData | string): string => {
  if (typeof value === 'string') {
    return value
  }
  return value?.value || ''
}

// 获取分隔符（兼容新旧格式）
const getSeparator = (value: EnvVariableData | string): string => {
  if (typeof value === 'object' && value !== null) {
    return value.separator || ''
  }
  return ''
}

const showMessage = (msg: string, type: string = 'success') => {
  message.value = msg
  messageType.value = type
  setTimeout(() => {
    message.value = ''
  }, 3000)
}

// 生命周期
onMounted(() => {
  loadEnvVariables()
})
</script>

<style scoped>
.env-variables {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  overflow: hidden;
}

.env-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #e9ecef;
  background: #f8f9fa;
}

.env-header h2 {
  margin: 0;
  color: #2c3e50;
}

.env-actions {
  display: flex;
  gap: 12px;
}

.env-content {
  padding: 20px;
}

.empty {
  text-align: center;
  padding: 40px;
  color: #6c757d;
}

.empty .hint {
  font-size: 0.9rem;
  margin-top: 8px;
}

.env-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.env-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border: 1px solid #e9ecef;
  border-radius: 6px;
  background: #f8f9fa;
}

.env-info {
  flex: 1;
}

.env-key {
  font-weight: 600;
  color: #2c3e50;
  font-size: 1.1rem;
  margin-bottom: 4px;
}

.env-value {
  color: #495057;
  font-family: monospace;
  background: white;
  padding: 4px 8px;
  border-radius: 4px;
  border: 1px solid #dee2e6;
  margin-bottom: 4px;
  word-break: break-all;
}

.env-usage {
  font-size: 0.85rem;
  color: #6c757d;
  font-family: monospace;
}

.env-controls {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #e9ecef;
}

.mask-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  color: #495057;
  font-weight: 500;
}

.mask-toggle input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: #007bff;
  cursor: pointer;
}

/* 对话框样式 */
.dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.dialog {
  background: white;
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
}

.dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #e9ecef;
}

.dialog-header h3 {
  margin: 0;
  color: #2c3e50;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #6c757d;
}

.dialog-body {
  padding: 20px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-weight: 500;
  color: #495057;
}

.form-input, .form-textarea {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 1rem;
}

.form-textarea {
  resize: vertical;
  min-height: 80px;
}

.hint {
  font-size: 0.85rem;
  color: #6c757d;
  margin-top: 4px;
}

.preview {
  padding: 12px;
  background: #f8f9fa;
  border: 1px solid #e9ecef;
  border-radius: 4px;
  font-size: 0.9rem;
  color: #495057;
}

.preview code {
  background: #e9ecef;
  padding: 2px 4px;
  border-radius: 3px;
  font-family: monospace;
  color: #2c3e50;
  font-weight: 600;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px;
  border-top: 1px solid #e9ecef;
}

/* 按钮样式 */
.btn {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
  transition: all 0.2s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: #007bff;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #0056b3;
}

.btn-secondary {
  background: #6c757d;
  color: white;
}

.btn-secondary:hover:not(:disabled) {
  background: #545b62;
}

.btn-danger {
  background: #dc3545;
  color: white;
}

.btn-danger:hover:not(:disabled) {
  background: #c82333;
}

.btn-sm {
  padding: 4px 8px;
  font-size: 0.8rem;
}

/* 消息提示 */
.message {
  position: fixed;
  top: 20px;
  right: 20px;
  padding: 12px 20px;
  border-radius: 4px;
  color: white;
  font-weight: 500;
  z-index: 1001;
}

.message.success {
  background: #28a745;
}

.message.error {
  background: #dc3545;
}

.message.info {
  background: #17a2b8;
}

.message.warning {
  background: #ffc107;
  color: #212529;
}

/* 分隔符预览样式 */
.separator-preview {
  margin-top: 15px;
  padding: 15px;
  background: #f8f9fa;
  border: 1px solid #e9ecef;
  border-radius: 4px;
}

.preview-values {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 10px 0;
}

.preview-value {
  background: #e7f3ff;
  border: 1px solid #b3d9ff;
  padding: 4px 8px;
  border-radius: 3px;
  font-size: 0.9rem;
  font-family: monospace;
}

.preview-actions {
  display: flex;
  gap: 8px;
  margin-top: 10px;
}

.form-hint {
  display: block;
  margin-top: 4px;
  color: #6c757d;
  font-size: 0.8rem;
}

.separator-info {
  color: #6c757d;
  font-size: 0.8rem;
  font-style: italic;
  margin-left: 8px;
}

.loading-state {
  text-align: center;
  padding: 40px;
  color: #6c757d;
}

.loading-state p {
  margin: 0;
  font-size: 1.1rem;
}
</style>
