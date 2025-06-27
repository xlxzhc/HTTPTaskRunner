<template>
  <div class="task-form">
    <div class="form-header">
      <h2>{{ isEdit ? '编辑任务' : '创建任务' }}</h2>
    </div>

    <form @submit.prevent="handleSubmit" class="form-content">
      <div class="form-grid">
        <!-- 基本信息 -->
        <div class="form-section">
          <h3>基本信息</h3>
          
          <div class="form-group">
            <label for="name">任务名称 *</label>
            <input
              id="name"
              v-model="formData.name"
              type="text"
              required
              placeholder="请输入任务名称（支持格式：标签-任务名称）"
              @input="handleNameInput"
            />
            <small v-if="autoExtractedTag" class="auto-extract-hint">
              自动提取标签：<span class="extracted-tag">{{ autoExtractedTag }}</span>
            </small>
          </div>

          <div class="form-group">
            <label for="url">请求URL *</label>
            <input 
              id="url"
              v-model="formData.url" 
              type="url" 
              required 
              placeholder="https://example.com/api"
            />
          </div>

          <div class="form-group">
            <label for="method">请求方法</label>
            <select id="method" v-model="formData.method">
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="DELETE">DELETE</option>
            </select>
          </div>

          <div class="form-group">
            <label for="headers">请求头</label>
            <textarea
              id="headers"
              v-model="formData.headersText"
              placeholder="每行一个header，格式：Key: Value"
              rows="4"
            ></textarea>
          </div>

          <div class="form-group">
            <label for="data">请求数据</label>
            <textarea
              id="data"
              v-model="formData.data"
              placeholder="POST数据或JSON"
              rows="3"
            ></textarea>
          </div>
        </div>

        <!-- 执行设置 -->
        <div class="form-section">
          <h3>执行设置</h3>
          
          <div class="form-row">
            <div class="form-group">
              <label for="times">执行次数</label>
              <input 
                id="times"
                v-model.number="formData.times" 
                type="number" 
                min="1" 
                max="10000"
              />
            </div>

            <div class="form-group">
              <label for="threads">并发线程</label>
              <input 
                id="threads"
                v-model.number="formData.threads" 
                type="number" 
                min="1" 
                max="100"
              />
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="delayMin">最小延迟(ms)</label>
              <input 
                id="delayMin"
                v-model.number="formData.delayMin" 
                type="number" 
                min="0"
              />
            </div>

            <div class="form-group">
              <label for="delayMax">最大延迟(ms)</label>
              <input 
                id="delayMax"
                v-model.number="formData.delayMax" 
                type="number" 
                min="0"
              />
            </div>
          </div>

          <div class="form-group">
            <label for="tags">标签</label>
            <input 
              id="tags"
              v-model="tagsText" 
              type="text" 
              placeholder="多个标签用逗号分隔"
            />
          </div>

          <div class="form-group">
            <label for="cronExpr">定时表达式</label>
            <div class="cron-input-group">
              <input
                id="cronExpr"
                v-model="formData.cronExpr"
                type="text"
                placeholder="如: 0 0 9 * * 1-5 (工作日9点整)"
                @blur="validateCronInput"
                :class="{ 'error': cronError }"
              />
              <button type="button" @click="showCronBuilder = !showCronBuilder" class="btn btn-secondary">
                {{ showCronBuilder ? '隐藏' : '辅助' }}
              </button>
            </div>
            <div v-if="cronError" class="error-message">
              {{ cronError }}
            </div>

            <!-- Cron表达式辅助生成器 -->
            <div v-if="showCronBuilder" class="cron-builder">
              <div class="cron-presets">
                <button type="button" @click="setCronExpr('0 */1 * * * *')" class="preset-btn">每分钟</button>
                <button type="button" @click="setCronExpr('0 0 */1 * * *')" class="preset-btn">每小时</button>
                <button type="button" @click="setCronExpr('0 0 0 */1 * *')" class="preset-btn">每天</button>
                <button type="button" @click="setCronExpr('0 0 9 * * 1-5')" class="preset-btn">工作日9点</button>
                <button type="button" @click="setCronExpr('*/30 * * * * *')" class="preset-btn">每30秒</button>
                <button type="button" @click="setCronExpr('0 */5 * * * *')" class="preset-btn">每5分钟</button>
              </div>
              <div class="cron-custom">
                <label>自定义时间：</label>
                <div class="time-inputs">
                  <input v-model="cronHour" type="number" min="0" max="23" placeholder="时" />
                  <span>:</span>
                  <input v-model="cronMinute" type="number" min="0" max="59" placeholder="分" />
                  <span>:</span>
                  <input v-model="cronSecond" type="number" min="0" max="59" placeholder="秒" />
                  <button type="button" @click="buildCustomCron" class="btn btn-primary">生成</button>
                </div>
                <div class="cron-help">
                  <small>
                    <strong>Cron格式说明：</strong>秒 分 时 日 月 周（6字段）或 分 时 日 月 周（5字段）<br>
                    例如：0 0 9 * * 1-5 表示工作日上午9点整
                  </small>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Fiddler数据解析 -->
        <div class="form-section">
          <h3>Fiddler数据解析</h3>
          <div class="form-group">
            <label for="fiddlerData">粘贴Fiddler捕获的原始数据</label>
            <textarea
              id="fiddlerData"
              v-model="fiddlerData"
              placeholder="粘贴完整的HTTP请求数据..."
              rows="8"
            ></textarea>
            <button type="button" @click="parseFiddlerData" class="btn btn-primary">解析数据</button>
          </div>
        </div>
      </div>

      <!-- 变量预览区域 -->
      <div class="form-section">
        <h3>变量预览</h3>
        <div class="preview-area">
          <div class="preview-controls">
            <button type="button" @click="loadVariablePreview" class="btn btn-info" :disabled="!formData.url">
              预览变量替换结果
            </button>
            <div class="preview-hint">
              <small>使用 &#123;&#123;VARIABLE_NAME&#125;&#125; 格式引用环境变量</small>
            </div>
          </div>

          <div v-if="variablePreview" class="preview-result">
            <div class="preview-item">
              <label>URL (替换后):</label>
              <div class="preview-value">{{ variablePreview.url }}</div>
            </div>
            <div v-if="variablePreview.data" class="preview-item">
              <label>请求数据 (替换后):</label>
              <div class="preview-value">{{ variablePreview.data }}</div>
            </div>
            <div v-if="variablePreview.headersText" class="preview-item">
              <label>请求头文本 (替换后):</label>
              <div class="preview-value">{{ variablePreview.headersText }}</div>
            </div>
            <div v-if="Object.keys(variablePreview.headers || {}).length > 0" class="preview-item">
              <label>解析后的请求头:</label>
              <div class="preview-value">
                <div v-for="(value, key) in variablePreview.headers" :key="key" class="header-item">
                  <strong>{{ key }}:</strong> {{ value }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 成功条件配置 -->
      <div class="form-section">
        <h3>成功条件配置</h3>
        <div class="success-condition-area">
          <div class="condition-toggle">
            <label class="checkbox-label">
              <input
                type="checkbox"
                v-model="enableSuccessCondition"
              />
              启用自定义成功条件判断
            </label>
          </div>

          <div v-if="enableSuccessCondition" class="condition-config">
            <div class="form-row">
              <div class="form-group">
                <label>JSON路径:</label>
                <input
                  type="text"
                  v-model="successCondition.jsonPath"
                  placeholder="例如: data.code 或 result.status"
                  class="form-control"
                />
                <small class="form-hint">指定要检查的JSON字段路径</small>
              </div>

              <div class="form-group">
                <label>条件类型:</label>
                <select v-model="conditionType" class="form-control" @change="onConditionTypeChange">
                  <option value="json_path">JSON路径判断</option>
                  <option value="string_based">字符串内容判断</option>
                </select>
              </div>

              <div v-if="conditionType === 'json_path'" class="form-group">
                <label>JSON路径:</label>
                <input
                  v-model="successCondition.jsonPath"
                  type="text"
                  class="form-control"
                  placeholder="例如: data.status 或 message"
                />
              </div>

              <div class="form-group">
                <label>判断类型:</label>
                <select v-model="successCondition.operator" class="form-control">
                  <optgroup v-if="conditionType === 'json_path'" label="JSON路径判断">
                    <option value="equals">等于</option>
                    <option value="not_equals">不等于</option>
                    <option value="contains">包含</option>
                    <option value="not_contains">不包含</option>
                  </optgroup>
                  <optgroup v-if="conditionType === 'string_based'" label="字符串内容判断">
                    <option value="response_contains">响应包含</option>
                    <option value="response_not_contains">响应不包含</option>
                    <option value="response_equals">响应等于</option>
                    <option value="response_not_equals">响应不等于</option>
                  </optgroup>
                </select>
              </div>

              <div class="form-group">
                <label>期望值:</label>
                <input
                  type="text"
                  v-model="successCondition.expectedValue"
                  placeholder="例如: 0 或 success"
                  class="form-control"
                />
                <small class="form-hint">要比较的期望值（支持字符串和数字）</small>
              </div>
            </div>

            <div class="condition-example">
              <strong>示例配置：</strong>
              <ul>
                <li>JSON路径: <code>data.code</code>，判断类型: <code>等于</code>，期望值: <code>0</code></li>
                <li>JSON路径: <code>result.message</code>，判断类型: <code>包含</code>，期望值: <code>success</code></li>
              </ul>
            </div>
          </div>
        </div>
      </div>

      <!-- 后端测试功能 -->
      <div class="form-section">
        <h3>测试功能</h3>
        <div class="test-area">
          <div class="test-info">
            <p class="info-text">
              <strong>后端测试：</strong>使用服务器代理发送请求，绕过浏览器CORS限制，支持完整的请求头和认证信息。
            </p>
          </div>

          <div class="test-controls">
            <button
              type="button"
              @click="testCurrentTask"
              class="btn btn-info"
              :disabled="!formData.url || testing"
            >
              {{ testing ? '测试中...' : '后端测试' }}
            </button>
            <button
              type="button"
              @click="clearTestResult"
              class="btn btn-secondary"
              v-if="testResult"
            >
              清空结果
            </button>
          </div>

          <!-- 测试结果显示 -->
          <div v-if="testResult" class="test-result">
            <h4>测试结果</h4>

            <!-- 成功/失败状态 -->
            <div class="result-status">
              <span class="status-badge" :class="testResult.success ? 'success' : 'failed'">
                {{ testResult.success ? '✓ 测试成功' : '✗ 测试失败' }}
              </span>
              <span class="response-time">{{ formatDuration(testResult.responseTime / 1000) }}</span>
              <span class="status-code" :class="getStatusClass(testResult.statusCode)">
                {{ testResult.statusCode }}
              </span>
              <span class="status-text">{{ testResult.statusText }}</span>
            </div>

            <!-- 测试结果详细描述 -->
            <div class="test-result-description">
              <div class="description-header">
                <h5>{{ testResult.success ? '测试成功' : '测试失败' }}</h5>
              </div>
              <div class="description-content">
                <div v-if="testResult.successConditionDetails" class="condition-description">
                  <div class="description-title">成功条件详情：</div>
                  <ul class="condition-list">
                    <li><strong>条件类型：</strong>{{ getConditionTypeText(testResult.successConditionDetails.type) }}</li>
                    <li v-if="testResult.successConditionDetails.jsonPath">
                      <strong>JSON路径：</strong>{{ testResult.successConditionDetails.jsonPath }}
                    </li>
                    <li>
                      <strong>判断条件：</strong>
                      <span :class="{ 'negative-condition': isNegativeCondition(testResult.successConditionDetails.operator) }">
                        {{ getOperatorText(testResult.successConditionDetails.operator) }}
                      </span>
                    </li>
                    <li><strong>期望值：</strong>"{{ testResult.successConditionDetails.expectedValue }}"</li>
                    <li><strong>实际值：</strong>"{{ testResult.successConditionDetails.actualValue }}"</li>
                    <li class="failure-reason" :class="{ 'success-reason': testResult.success, 'failure-reason': !testResult.success }">
                      <strong>{{ testResult.success ? '成功原因：' : '失败原因：' }}</strong>
                      {{ getDetailedReason(testResult.successConditionDetails) }}
                    </li>
                  </ul>
                </div>
                <div v-else class="http-status-description">
                  <div class="description-title">HTTP状态判断：</div>
                  <p>{{ getHttpStatusDescription(testResult.statusCode, testResult.success) }}</p>
                </div>
              </div>
            </div>

            <!-- 成功条件评估详情 -->
            <div v-if="testResult.successConditionDetails" class="result-section">
              <h5>成功条件评估详情</h5>
              <div class="condition-details">
                <div class="condition-item">
                  <span class="condition-label">条件类型:</span>
                  <span class="condition-value">{{ testResult.successConditionDetails.type }}</span>
                </div>
                <div class="condition-item" v-if="testResult.successConditionDetails.jsonPath">
                  <span class="condition-label">JSON路径:</span>
                  <span class="condition-value">{{ testResult.successConditionDetails.jsonPath }}</span>
                </div>
                <div class="condition-item">
                  <span class="condition-label">操作符:</span>
                  <span class="condition-value">{{ testResult.successConditionDetails.operator }}</span>
                </div>
                <div class="condition-item">
                  <span class="condition-label">期望值:</span>
                  <span class="condition-value">{{ testResult.successConditionDetails.expectedValue }}</span>
                </div>
                <div class="condition-item">
                  <span class="condition-label">实际值:</span>
                  <span class="condition-value">{{ testResult.successConditionDetails.actualValue }}</span>
                </div>
                <div class="condition-item">
                  <span class="condition-label">判断结果:</span>
                  <span class="condition-value" :class="testResult.successConditionDetails.result ? 'success' : 'failed'">
                    {{ testResult.successConditionDetails.result ? '✓ 条件满足' : '✗ 条件不满足' }}
                  </span>
                </div>
                <div class="condition-item" v-if="testResult.successConditionDetails.reason">
                  <span class="condition-label">详细说明:</span>
                  <span class="condition-value">{{ testResult.successConditionDetails.reason }}</span>
                </div>
              </div>
            </div>

            <!-- 请求信息 -->
            <div class="result-section">
              <h5>请求信息</h5>
              <div class="request-info">
                <div class="info-item">
                  <span class="info-label">方法:</span>
                  <span class="info-value">{{ testResult.requestMethod }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">URL:</span>
                  <span class="info-value url-value">{{ testResult.requestUrl }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">请求体长度:</span>
                  <span class="info-value">{{ testResult.requestBodySize || 0 }} 字符</span>
                </div>
              </div>
            </div>

            <!-- 响应状态 -->
            <div class="result-section">
              <h5>响应状态</h5>
              <div class="status-info">
                <span class="status-code" :class="getStatusClass(testResult.statusCode)">
                  {{ testResult.statusCode }}
                </span>
                <span class="status-text">{{ testResult.statusText }}</span>
              </div>
            </div>

            <!-- 响应头 -->
            <div v-if="testResult.responseHeaders && Object.keys(testResult.responseHeaders).length > 0" class="result-section">
              <h5>响应头</h5>
              <div class="headers-display">
                <div v-for="(value, key) in testResult.responseHeaders" :key="key" class="header-item">
                  <span class="header-key">{{ key }}:</span>
                  <span class="header-value">{{ value }}</span>
                </div>
              </div>
            </div>

            <!-- 响应内容预览 -->
            <div class="result-section">
              <h5>响应内容预览</h5>
              <div class="response-preview">
                <div v-if="!testResult.responseBody || testResult.responseBody.trim() === ''" class="empty-response">
                  <em>响应内容为空</em>
                </div>
                <div v-else class="response-preview-content">
                  <div class="preview-info">
                    <span class="content-length">长度: {{ testResult.responseBody.length }} 字符</span>
                    <span class="content-type" v-if="getContentType(testResult.responseHeaders)">
                      类型: {{ getContentType(testResult.responseHeaders) }}
                    </span>
                  </div>
                  <div class="preview-text">
                    {{ getResponsePreview(testResult.responseBody) }}
                  </div>
                  <div v-if="testResult.responseBody.length > 200" class="preview-note">
                    <em>仅显示前200个字符，点击下方查看完整内容</em>
                  </div>
                </div>
              </div>
            </div>

            <!-- 完整响应内容 -->
            <div class="result-section">
              <h5>完整响应内容</h5>
              <div class="response-content">
                <div v-if="!testResult.responseBody || testResult.responseBody.trim() === ''" class="empty-response">
                  <em>响应内容为空</em>
                </div>
                <div v-else class="response-body">
                  <div class="response-actions">
                    <button
                      type="button"
                      @click="formatResponse"
                      class="btn-small"
                      v-if="isJsonResponse(testResult.responseBody)"
                    >
                      {{ formattedResponse ? '原始格式' : '格式化JSON' }}
                    </button>
                    <button
                      type="button"
                      @click="copyResponse"
                      class="btn-small"
                    >
                      复制内容
                    </button>
                    <button
                      type="button"
                      @click="toggleResponseExpansion"
                      class="btn-small"
                    >
                      {{ responseExpanded ? '收起内容' : '展开内容' }}
                    </button>
                  </div>
                  <pre
                    class="response-text"
                    :class="{ 'collapsed': !responseExpanded }"
                  >{{ displayResponseBody }}</pre>
                </div>
              </div>
            </div>

            <!-- 错误信息 -->
            <div v-if="testResult.error" class="result-section error">
              <h5>错误信息</h5>
              <div class="error-content">
                <pre>{{ testResult.error }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="form-actions">
        <button type="button" @click="$emit('cancel')" class="btn btn-secondary">
          取消
        </button>
        <button type="submit" class="btn btn-primary">
          {{ isEdit ? '更新' : '创建' }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'

// Props
const props = defineProps<{
  task?: any
}>()

// Emits
const emit = defineEmits<{
  save: [taskData: any]
  cancel: []
}>()

// 响应式数据
const formData = reactive({
  name: '',
  url: '',
  method: 'GET',
  headersText: '',
  data: '',
  times: 10,
  threads: 5,
  delayMin: 0,
  delayMax: 1000,
  cronExpr: ''
})

const tagsText = ref('')
const fiddlerData = ref('')
const showCronBuilder = ref(false)
const cronHour = ref(9)
const cronMinute = ref(0)
const cronSecond = ref(0)

const cronError = ref('')
const variablePreview = ref<any>(null)

// 成功条件配置
const enableSuccessCondition = ref(false)
const conditionType = ref('json_path') // 'json_path' 或 'string_based'
const successCondition = ref({
  jsonPath: '',
  operator: 'equals',
  expectedValue: ''
})

// 自动标签提取
const autoExtractedTag = ref('')

// 测试相关变量
const testing = ref(false)
const testResult = ref<any>(null)
const formattedResponse = ref(false)
const responseExpanded = ref(false)

// 计算属性
const isEdit = computed(() => !!props.task)

// 测试结果显示的计算属性
const displayResponseBody = computed(() => {
  if (!testResult.value?.responseBody) return ''

  if (formattedResponse.value && isJsonResponse(testResult.value.responseBody)) {
    try {
      const parsed = JSON.parse(testResult.value.responseBody)
      return JSON.stringify(parsed, null, 2)
    } catch {
      return testResult.value.responseBody
    }
  }

  return testResult.value.responseBody
})



// 监听task变化，填充表单
watch(() => props.task, (newTask) => {
  if (newTask) {
    formData.name = newTask.name || ''
    formData.url = newTask.url || ''
    formData.method = newTask.method || 'GET'
    formData.headersText = newTask.headersText || ''
    formData.data = newTask.data || ''
    formData.times = newTask.times || 10
    formData.threads = newTask.threads || 5
    formData.delayMin = newTask.delayMin || 0
    formData.delayMax = newTask.delayMax || 1000
    formData.cronExpr = newTask.cronExpr || ''
    tagsText.value = (newTask.tags || []).join(', ')

    // 加载成功条件配置
    if (newTask.successCondition) {
      enableSuccessCondition.value = newTask.successCondition.enabled || false
      successCondition.value = {
        jsonPath: newTask.successCondition.jsonPath || '',
        operator: newTask.successCondition.operator || 'equals',
        expectedValue: newTask.successCondition.expectedValue || ''
      }

      // 根据操作符判断条件类型
      const stringBasedOperators = ['response_contains', 'response_not_contains', 'response_equals', 'response_not_equals']
      conditionType.value = stringBasedOperators.includes(successCondition.value.operator) ? 'string_based' : 'json_path'
    } else {
      enableSuccessCondition.value = false
      conditionType.value = 'json_path'
      successCondition.value = {
        jsonPath: '',
        operator: 'equals',
        expectedValue: ''
      }
    }
  } else {
    // 重置表单
    formData.name = ''
    formData.url = ''
    formData.method = 'GET'
    formData.headersText = ''
    formData.data = ''
    formData.times = 10
    formData.threads = 5
    formData.delayMin = 0
    formData.delayMax = 1000
    formData.cronExpr = ''
    tagsText.value = ''

    // 重置成功条件
    enableSuccessCondition.value = false
    conditionType.value = 'json_path'
    successCondition.value = {
      jsonPath: '',
      operator: 'equals',
      expectedValue: ''
    }
  }
}, { immediate: true })

// 提交表单
const handleSubmit = () => {
  // 解析标签
  const tags = tagsText.value
    .split(',')
    .map(tag => tag.trim())
    .filter(tag => tag !== '')

  // 验证延迟设置
  if (formData.delayMax < formData.delayMin) {
    formData.delayMax = formData.delayMin
  }

  const taskData = {
    ...formData,
    tags,
    successCondition: enableSuccessCondition.value ? {
      enabled: true,
      jsonPath: successCondition.value.jsonPath,
      operator: successCondition.value.operator,
      expectedValue: successCondition.value.expectedValue
    } : {
      enabled: false,
      jsonPath: '',
      operator: 'equals',
      expectedValue: ''
    }
  }

  emit('save', taskData)
}

// 解析Fiddler数据
const parseFiddlerData = () => {
  if (!fiddlerData.value.trim()) {
    alert('请先粘贴Fiddler数据')
    return
  }

  try {
    const lines = fiddlerData.value.trim().split('\n').map(line => line.replace(/\r$/, ''))

    // 解析第一行获取方法和URL
    const firstLine = lines[0].trim()
    let methodMatch = firstLine.match(/^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\s+(.+?)\s+HTTP/i)

    if (!methodMatch) {
      throw new Error('无法解析HTTP请求行，请确保数据格式正确')
    }

    formData.method = methodMatch[1].toUpperCase()
    let url = methodMatch[2]

    // 如果URL没有协议头，需要从Host header中构建完整URL
    if (!url.startsWith('http://') && !url.startsWith('https://')) {
      // 查找Host header
      let host = ''
      let isHttps = false

      for (let i = 1; i < lines.length; i++) {
        const line = lines[i].trim()
        if (line === '') break

        const colonIndex = line.indexOf(':')
        if (colonIndex > 0) {
          const key = line.substring(0, colonIndex).trim().toLowerCase()
          const value = line.substring(colonIndex + 1).trim()

          if (key === 'host') {
            host = value
          }

          // 检查是否为HTTPS的各种指标
          if (key === 'x-forwarded-proto' && value.toLowerCase() === 'https') {
            isHttps = true
          }
          if (key === 'x-forwarded-ssl' && value.toLowerCase() === 'on') {
            isHttps = true
          }
          if (key === 'origin' && value.startsWith('https://')) {
            isHttps = true
          }
          if (key === 'referer' && value.startsWith('https://')) {
            isHttps = true
          }
        }
      }

      // 构建完整URL
      if (host) {
        // 判断是否使用HTTPS
        if (!isHttps) {
          isHttps = host.includes(':443') || host.endsWith('.com') || host.endsWith('.org')
        }
        const protocol = isHttps ? 'https://' : 'http://'
        formData.url = protocol + host + url
      } else {
        throw new Error('无法找到Host header，无法构建完整URL')
      }
    } else {
      formData.url = url
    }

    // 解析headers
    const headers: Record<string, string> = {}
    let bodyStartIndex = -1

    for (let i = 1; i < lines.length; i++) {
      const line = lines[i].trim()

      if (line === '') {
        bodyStartIndex = i + 1
        break
      }

      const colonIndex = line.indexOf(':')
      if (colonIndex > 0) {
        const key = line.substring(0, colonIndex).trim()
        const value = line.substring(colonIndex + 1).trim()

        // 处理多行header值（如果存在）
        if (headers[key]) {
          headers[key] += '; ' + value
        } else {
          headers[key] = value
        }
      }
    }

    // 设置headers（排除一些不需要的，但保留重要的）
    const excludeHeaders = ['host', 'content-length', 'connection', 'accept-encoding', 'transfer-encoding']
    const filteredHeaders: Record<string, string> = {}

    Object.entries(headers).forEach(([key, value]) => {
      if (!excludeHeaders.includes(key.toLowerCase())) {
        filteredHeaders[key] = value
      }
    })

    // 确保重要的headers被保留
    if (headers['Content-Type'] || headers['content-type']) {
      const contentType = headers['Content-Type'] || headers['content-type']
      filteredHeaders['Content-Type'] = contentType
    }

    if (headers['User-Agent'] || headers['user-agent']) {
      const userAgent = headers['User-Agent'] || headers['user-agent']
      filteredHeaders['User-Agent'] = userAgent
    }

    // 将headers转换为字符串格式存储（用于显示和编辑）
    const headersText = Object.entries(filteredHeaders)
      .map(([key, value]) => `${key}: ${value}`)
      .join('\n')

    // 存储到表单数据中
    formData.headersText = headersText

    // 解析body数据
    if (bodyStartIndex > 0 && bodyStartIndex < lines.length) {
      const bodyLines = lines.slice(bodyStartIndex)
      const bodyContent = bodyLines.join('\n').trim()

      // 处理URL编码的数据
      if (bodyContent) {
        formData.data = bodyContent

        // 如果没有设置Content-Type，根据数据格式自动设置
        if (!filteredHeaders['Content-Type'] && !filteredHeaders['content-type']) {
          if (bodyContent.includes('=') && bodyContent.includes('&')) {
            // 看起来像URL编码的表单数据
            filteredHeaders['Content-Type'] = 'application/x-www-form-urlencoded'
          } else if (bodyContent.startsWith('{') || bodyContent.startsWith('[')) {
            // 看起来像JSON数据
            filteredHeaders['Content-Type'] = 'application/json'
          }

          // 更新headersText
          formData.headersText = Object.entries(filteredHeaders)
            .map(([key, value]) => `${key}: ${value}`)
            .join('\n')
        }
      }
    }

    // 如果没有任务名称，从URL生成一个
    if (!formData.name) {
      try {
        const url = new URL(formData.url)
        const pathParts = url.pathname.split('/').filter(p => p)
        const lastPart = pathParts[pathParts.length - 1] || 'request'
        formData.name = `${formData.method} ${lastPart}`
      } catch {
        formData.name = `${formData.method} 请求`
      }
    }

    // 显示详细的解析结果
    const bodyLength = formData.data ? formData.data.length : 0
    const headerCount = Object.keys(filteredHeaders).length

    alert('Fiddler数据解析成功！\n\n' +
          `✓ 请求方法: ${formData.method}\n` +
          `✓ 请求URL: ${formData.url}\n` +
          `✓ 请求头数量: ${headerCount}个\n` +
          `✓ 请求体长度: ${bodyLength}字符\n` +
          `✓ 任务名称: ${formData.name}\n\n` +
          '数据已填入表单，您可以保存任务并执行验证配置是否正确。')

  } catch (error) {
    console.error('Fiddler解析错误:', error)
    alert(`解析失败: ${error instanceof Error ? error.message : String(error)}\n\n` +
          '请确保粘贴的是完整的HTTP请求数据，格式如下：\n' +
          'POST /path HTTP/1.1\n' +
          'Host: example.com\n' +
          'Content-Type: application/json\n' +
          '\n' +
          '{"key": "value"}')
  }
}

// 验证和格式化Cron表达式
const validateCronExpr = (expr: string): string => {
  if (!expr.trim()) return ''

  const fields = expr.trim().split(/\s+/)

  // 支持5字段和6字段格式
  if (fields.length === 5) {
    // 5字段格式：分 时 日 月 周，转换为6字段格式
    return '0 ' + expr.trim()
  }

  if (fields.length === 6) {
    // 6字段格式：秒 分 时 日 月 周，直接返回
    return expr.trim()
  }

  // 其他格式报错
  throw new Error(`Cron表达式格式错误：支持5字段（分 时 日 月 周）或6字段（秒 分 时 日 月 周），实际${fields.length}个字段`)
}

// 设置Cron表达式
const setCronExpr = (expr: string) => {
  try {
    formData.cronExpr = validateCronExpr(expr)
  } catch (error) {
    alert(`Cron表达式设置失败: ${error}`)
  }
}

// 构建自定义Cron表达式 (6字段格式: 秒 分 时 日 月 周)
const buildCustomCron = () => {
  const expr = `${cronSecond.value} ${cronMinute.value} ${cronHour.value} * * *`
  formData.cronExpr = validateCronExpr(expr)
}

// 验证Cron输入
const validateCronInput = () => {
  cronError.value = ''
  if (!formData.cronExpr.trim()) return

  try {
    formData.cronExpr = validateCronExpr(formData.cronExpr)
  } catch (error) {
    cronError.value = error instanceof Error ? error.message : String(error)
  }
}

// 加载变量预览
const loadVariablePreview = async () => {
  if (!formData.url) {
    alert('请先设置URL')
    return
  }

  try {
    // 直接获取环境变量并进行本地替换预览
    const { GetEnvVariables } = await import('../../wailsjs/go/main/App')
    const envVars = await GetEnvVariables()

    // 本地替换变量的函数
    const replaceVariables = (text: string): string => {
      if (!text) return text

      let result = text
      // 最多替换10次，防止无限递归
      for (let i = 0; i < 10; i++) {
        const oldResult = result
        for (const [key, value] of Object.entries(envVars)) {
          const placeholder = `{{${key}}}`
          result = result.replaceAll(placeholder, value)
        }
        // 如果没有变化，说明替换完成
        if (result === oldResult) {
          break
        }
      }
      return result
    }

    // 创建预览结果
    const preview = {
      id: 'preview',
      name: formData.name || 'preview_task',
      url: replaceVariables(formData.url),
      method: formData.method,
      data: replaceVariables(formData.data),
      headersText: replaceVariables(formData.headersText),
      headers: {}
    }

    // 处理headers
    const headersLines = formData.headersText.split('\n')
    const headers: Record<string, string> = {}
    for (const line of headersLines) {
      const trimmedLine = line.trim()
      if (trimmedLine && trimmedLine.includes(':')) {
        const [key, ...valueParts] = trimmedLine.split(':')
        const value = valueParts.join(':').trim()
        if (key && value) {
          headers[replaceVariables(key.trim())] = replaceVariables(value)
        }
      }
    }
    preview.headers = headers

    variablePreview.value = preview
  } catch (error) {
    alert('预览失败: ' + error)
    console.error('变量预览失败:', error)
  }
}

// 格式化耗时为简洁格式 (如: 5.000s, 1m2.123s, 1h23m45.678s)
const formatDuration = (seconds: number): string => {
  if (typeof seconds !== 'number' || isNaN(seconds)) {
    return '0.000s'
  }

  const totalMs = Math.round(seconds * 1000)
  const ms = totalMs % 1000
  const totalSeconds = Math.floor(totalMs / 1000)
  const secs = totalSeconds % 60
  const totalMinutes = Math.floor(totalSeconds / 60)
  const mins = totalMinutes % 60
  const hours = Math.floor(totalMinutes / 60)

  // 格式化毫秒部分
  const msStr = ms.toString().padStart(3, '0')

  // 构建时间字符串
  let result = ''

  if (hours > 0) {
    result += `${hours}h`
  }

  if (mins > 0) {
    result += `${mins}m`
  }

  // 秒数部分（包含毫秒）
  result += `${secs}.${msStr}s`

  return result
}

// 测试当前任务配置
const testCurrentTask = async () => {
  if (!formData.url) {
    alert('请先设置URL')
    return
  }

  testing.value = true
  testResult.value = null

  try {
    // 导入后端方法
    const { TestTaskDataWithBackend } = await import('../../wailsjs/go/main/App')

    // 构建成功条件数据
    const successConditionData = enableSuccessCondition.value ? {
      enabled: true,
      jsonPath: successCondition.value.jsonPath,
      operator: successCondition.value.operator,
      expectedValue: successCondition.value.expectedValue
    } : {
      enabled: false,
      jsonPath: '',
      operator: 'equals',
      expectedValue: ''
    }

    const result = await TestTaskDataWithBackend(
      formData.name || '测试任务',
      formData.url,
      formData.method,
      formData.headersText,
      formData.data,
      successConditionData
    )

    // 转换后端结果为前端格式
    testResult.value = {
      statusCode: result.statusCode,
      statusText: result.statusText,
      responseTime: result.responseTime,
      responseHeaders: result.responseHeaders,
      responseBody: result.responseBody,
      success: result.success,
      error: result.error,
      requestHeaders: result.requestHeaders,
      requestUrl: result.requestUrl,
      requestMethod: result.requestMethod,
      requestBodySize: result.requestBodySize,
      sensitiveHeaders: result.sensitiveHeaders,
      successConditionDetails: result.successConditionDetails // 添加成功条件详情
    }

  } catch (error) {
    console.error('后端测试失败:', error)
    testResult.value = {
      statusCode: 0,
      statusText: 'Backend Test Failed',
      responseTime: 0,
      responseHeaders: {},
      responseBody: '',
      error: `后端测试失败: ${error instanceof Error ? error.message : String(error)}`,
      success: false,
      requestHeaders: {},
      requestUrl: formData.url,
      requestMethod: formData.method,
      requestBodySize: formData.data ? formData.data.length : 0
    }
  } finally {
    testing.value = false
  }
}

// 清空测试结果
const clearTestResult = () => {
  testResult.value = null
  formattedResponse.value = false
}

// 获取状态码样式类
const getStatusClass = (statusCode: number) => {
  if (statusCode >= 200 && statusCode < 300) return 'success'
  if (statusCode >= 300 && statusCode < 400) return 'warning'
  if (statusCode >= 400) return 'error'
  return 'default'
}

// 检查是否为JSON响应
const isJsonResponse = (content: string) => {
  if (!content) return false
  const trimmed = content.trim()
  return (trimmed.startsWith('{') && trimmed.endsWith('}')) ||
         (trimmed.startsWith('[') && trimmed.endsWith(']'))
}

// 格式化响应内容
const formatResponse = () => {
  formattedResponse.value = !formattedResponse.value
}

// 复制响应内容
const copyResponse = async () => {
  if (!testResult.value?.responseBody) return

  try {
    await navigator.clipboard.writeText(testResult.value.responseBody)
    alert('响应内容已复制到剪贴板')
  } catch (error) {
    // 降级方案
    const textArea = document.createElement('textarea')
    textArea.value = testResult.value.responseBody
    document.body.appendChild(textArea)
    textArea.select()
    document.execCommand('copy')
    document.body.removeChild(textArea)
    alert('响应内容已复制到剪贴板')
  }
}

// 获取响应内容类型
const getContentType = (headers: Record<string, string>) => {
  if (!headers) return ''

  // 查找content-type头（不区分大小写）
  for (const [key, value] of Object.entries(headers)) {
    if (key.toLowerCase() === 'content-type') {
      return value.split(';')[0].trim() // 只返回主要类型，去掉charset等参数
    }
  }
  return ''
}

// 获取响应内容预览（前200个字符）
const getResponsePreview = (content: string) => {
  if (!content) return ''
  if (content.length <= 200) return content
  return content.substring(0, 200) + '...'
}

// 切换响应内容展开状态
const toggleResponseExpansion = () => {
  responseExpanded.value = !responseExpanded.value
}

// 条件类型变化处理
const onConditionTypeChange = () => {
  // 重置操作符为对应类型的默认值
  if (conditionType.value === 'json_path') {
    successCondition.value.operator = 'equals'
  } else if (conditionType.value === 'string_based') {
    successCondition.value.operator = 'response_contains'
    // 清空JSON路径，因为字符串基础条件不需要
    successCondition.value.jsonPath = ''
  }
}

// 获取条件类型文本
const getConditionTypeText = (type: string) => {
  switch (type) {
    case 'json_path': return 'JSON路径判断'
    case 'string_based': return '字符串内容判断'
    case 'http_status': return 'HTTP状态码判断'
    default: return type
  }
}

// 获取操作符文本
const getOperatorText = (operator: string) => {
  switch (operator) {
    case 'equals': return '等于'
    case 'not_equals': return '不等于'
    case 'contains': return '包含'
    case 'not_contains': return '不包含'
    case 'response_contains': return '响应包含'
    case 'response_not_contains': return '响应不包含'
    case 'response_equals': return '响应等于'
    case 'response_not_equals': return '响应不等于'
    default: return operator
  }
}

// 判断是否为否定条件（包含"不"的条件）
const isNegativeCondition = (operator: string) => {
  const operatorText = getOperatorText(operator)
  return operatorText.includes('不')
}

// 获取详细原因说明
const getDetailedReason = (details: any) => {
  if (!details) return ''

  const { operator, expectedValue, actualValue, result } = details

  if (result) {
    // 成功情况
    switch (operator) {
      case 'equals':
      case 'response_equals':
        return `实际值与期望值相等，条件满足`
      case 'not_equals':
      case 'response_not_equals':
        return `实际值与期望值不相等，条件满足`
      case 'contains':
      case 'response_contains':
        return `实际值包含期望值，条件满足`
      case 'not_contains':
      case 'response_not_contains':
        return `实际值不包含期望值，条件满足`
      default:
        return `条件判断通过`
    }
  } else {
    // 失败情况
    switch (operator) {
      case 'equals':
      case 'response_equals':
        return `实际值与期望值不相等，但条件要求相等`
      case 'not_equals':
      case 'response_not_equals':
        return `实际值与期望值相等，但条件要求不等于`
      case 'contains':
      case 'response_contains':
        return `实际值不包含期望值，但条件要求包含`
      case 'not_contains':
      case 'response_not_contains':
        return `实际值包含期望值，但条件要求不包含`
      default:
        return `条件判断失败`
    }
  }
}

// 获取HTTP状态描述
const getHttpStatusDescription = (statusCode: number, success: boolean) => {
  if (success) {
    return `HTTP状态码 ${statusCode} 在成功范围内（200-299），请求成功`
  } else {
    if (statusCode >= 400 && statusCode < 500) {
      return `HTTP状态码 ${statusCode} 表示客户端错误，请检查请求参数`
    } else if (statusCode >= 500) {
      return `HTTP状态码 ${statusCode} 表示服务器错误，请稍后重试`
    } else if (statusCode >= 300 && statusCode < 400) {
      return `HTTP状态码 ${statusCode} 表示重定向，可能需要处理跳转`
    } else {
      return `HTTP状态码 ${statusCode} 不在成功范围内`
    }
  }
}

// 处理任务名称输入，自动提取标签
const handleNameInput = () => {
  const name = formData.name
  if (!name) {
    autoExtractedTag.value = ''
    return
  }

  // 检查是否包含"-"分隔符
  const dashIndex = name.indexOf('-')
  if (dashIndex > 0 && dashIndex < name.length - 1) {
    // 提取标签（第一个"-"之前的内容）
    const tag = name.substring(0, dashIndex).trim()
    // 提取任务名称（第一个"-"之后的所有内容）
    const taskName = name.substring(dashIndex + 1).trim()

    if (tag && taskName) {
      autoExtractedTag.value = tag

      // 自动更新任务名称字段（只保留"-"后的内容）
      formData.name = taskName

      // 自动添加标签到tags字段
      const currentTags = tagsText.value ? tagsText.value.split(',').map(t => t.trim()).filter(t => t) : []
      if (!currentTags.includes(tag)) {
        currentTags.unshift(tag) // 添加到开头
        tagsText.value = currentTags.join(', ')
      }

      // 显示提示信息
      setTimeout(() => {
        autoExtractedTag.value = ''
      }, 3000)
    }
  } else {
    autoExtractedTag.value = ''
  }
}


</script>

<style scoped>
.task-form {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  overflow: hidden;
}

.form-header {
  padding: 20px;
  border-bottom: 1px solid #e9ecef;
  background: #f8f9fa;
}

.form-header h2 {
  margin: 0;
  color: #2c3e50;
}

.form-content {
  padding: 20px;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 30px;
  margin-bottom: 30px;
}

@media (max-width: 768px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
}

.form-section h3 {
  margin: 0 0 20px 0;
  color: #495057;
  font-size: 1.1rem;
  border-bottom: 2px solid #e9ecef;
  padding-bottom: 8px;
}

.form-group {
  margin-bottom: 16px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  font-weight: 500;
  color: #495057;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #ced4da;
  border-radius: 4px;
  font-size: 0.9rem;
  transition: border-color 0.2s;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
  outline: none;
  border-color: #007bff;
  box-shadow: 0 0 0 2px rgba(0,123,255,0.25);
}

.form-group textarea {
  resize: vertical;
  min-height: 80px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 20px;
  border-top: 1px solid #e9ecef;
}

.btn {
  padding: 10px 20px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
  font-weight: 500;
  transition: all 0.2s;
}

.btn-primary {
  background: #007bff;
  color: white;
}

.btn-primary:hover {
  background: #0056b3;
}

.btn-secondary {
  background: #6c757d;
  color: white;
}

.btn-secondary:hover {
  background: #545b62;
}

.cron-input-group {
  display: flex;
  gap: 8px;
  align-items: center;
}

.cron-input-group input {
  flex: 1;
}

.cron-builder {
  margin-top: 12px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 4px;
  border: 1px solid #e9ecef;
}

.cron-presets {
  margin-bottom: 12px;
}

.preset-btn {
  margin-right: 8px;
  margin-bottom: 8px;
  padding: 4px 12px;
  border: 1px solid #ddd;
  background: white;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.8rem;
}

.preset-btn:hover {
  background: #e9ecef;
}

.cron-custom label {
  display: block;
  margin-bottom: 8px;
  font-weight: 500;
}

.time-inputs {
  display: flex;
  align-items: center;
  gap: 8px;
}

.time-inputs input {
  width: 60px;
  padding: 4px 8px;
}

.cron-help {
  margin-top: 8px;
  padding: 8px;
  background: #e7f3ff;
  border-radius: 4px;
  border-left: 3px solid #007bff;
}

.cron-help small {
  color: #495057;
  line-height: 1.4;
}

.error-message {
  margin-top: 4px;
  padding: 8px;
  background: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
  border-radius: 4px;
  font-size: 0.85rem;
}

input.error {
  border-color: #dc3545;
  box-shadow: 0 0 0 0.2rem rgba(220, 53, 69, 0.25);
}



.info-box {
  padding: 12px;
  border-radius: 4px;
  border: 1px solid;
}

.info-box.warning {
  background: #fff8e1;
  border-color: #ffcc02;
  color: #663c00;
  font-weight: 500;
}

.info-box.success {
  background: #e8f5e8;
  border-color: #4caf50;
  color: #1b5e20;
  font-weight: 500;
}

.info-box strong {
  display: block;
  margin-bottom: 8px;
}

.info-box ul {
  margin: 0;
  padding-left: 20px;
}

.info-box li {
  margin-bottom: 4px;
}

.btn-info {
  background: #17a2b8;
  color: white;
}

.btn-info:hover:not(:disabled) {
  background: #138496;
}

.btn-info:disabled {
  background: #6c757d;
  cursor: not-allowed;
}



.result-section {
  margin-bottom: 16px;
}

.result-section h5 {
  margin: 0 0 8px 0;
  color: #6c757d;
  font-size: 0.9rem;
  font-weight: 600;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-code {
  padding: 4px 8px;
  border-radius: 4px;
  font-weight: bold;
  font-family: monospace;
}

.status-code.success {
  background: #d4edda;
  color: #155724;
}

.status-code.warning {
  background: #fff3cd;
  color: #856404;
}

.status-code.error {
  background: #f8d7da;
  color: #721c24;
}

.status-code.default {
  background: #e9ecef;
  color: #495057;
}

.status-text {
  font-weight: 500;
}

.response-time {
  color: #6c757d;
  font-size: 0.9rem;
}

.headers-display {
  max-height: 200px;
  overflow-y: auto;
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  padding: 8px;
}

.header-item {
  display: flex;
  margin-bottom: 4px;
  font-family: monospace;
  font-size: 0.85rem;
}

.header-key {
  font-weight: bold;
  color: #495057;
  min-width: 150px;
}

.header-value {
  color: #6c757d;
  word-break: break-all;
}

.response-content {
  max-height: 300px;
  overflow: auto;
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  padding: 12px;
}

.response-content pre {
  margin: 0;
  font-family: 'Courier New', monospace;
  font-size: 0.85rem;
  white-space: pre-wrap;
  word-break: break-word;
  color: #2c3e50;
  background: #ffffff;
  padding: 12px;
  border: 1px solid #e9ecef;
  border-radius: 4px;
  max-height: 400px;
  overflow-y: auto;
}

.content-length {
  font-size: 0.75rem;
  color: #6c757d;
  font-weight: normal;
  margin-left: 8px;
}

.empty-response {
  padding: 20px;
  text-align: center;
  color: #6c757d;
  background: #f8f9fa;
  border: 1px dashed #dee2e6;
  border-radius: 4px;
}

.response-body {
  position: relative;
}

.response-actions {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  gap: 4px;
  z-index: 10;
}

.btn-small {
  padding: 4px 8px;
  font-size: 0.75rem;
  border: 1px solid #dee2e6;
  background: white;
  border-radius: 3px;
  cursor: pointer;
  color: #495057;
  transition: all 0.2s;
}

.btn-small:hover {
  background: #e9ecef;
  border-color: #adb5bd;
}

.result-section.error .error-content {
  background: #f8d7da;
  color: #721c24;
  padding: 12px;
  border-radius: 4px;
  border: 1px solid #f5c6cb;
}

.result-section.warning .warning-content {
  background: #fff3cd;
  color: #856404;
  padding: 12px;
  border-radius: 4px;
  border: 1px solid #ffeaa7;
}

.request-info {
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  padding: 12px;
}

.info-item {
  display: flex;
  margin-bottom: 8px;
  align-items: flex-start;
}

.info-item:last-child {
  margin-bottom: 0;
}

.info-label {
  font-weight: bold;
  color: #495057;
  min-width: 100px;
  flex-shrink: 0;
}

.info-value {
  color: #6c757d;
  word-break: break-word;
}

.url-value {
  font-family: monospace;
  font-size: 0.85rem;
}

.warning-content ul,
.error-content ul {
  margin: 8px 0 0 0;
  padding-left: 20px;
}

.warning-content li,
.error-content li {
  margin-bottom: 4px;
  font-family: monospace;
  font-size: 0.85rem;
}

.error-content pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
}

/* 测试模式相关样式 */
.mode-badge {
  font-size: 0.7rem;
  padding: 2px 6px;
  border-radius: 3px;
  font-weight: bold;
  margin-left: 8px;
}



.header-value.sensitive {
  color: #dc3545;
  font-style: italic;
}

.sensitive-note {
  font-size: 0.7rem;
  color: #6c757d;
  margin-left: 4px;
}

.sensitive-info {
  margin-top: 8px;
  padding: 8px;
  background: #e7f3ff;
  border-radius: 4px;
  border-left: 3px solid #007bff;
}

.sensitive-info small {
  color: #495057;
}

/* 变量预览样式 */
.preview-area {
  border: 1px solid #e9ecef;
  border-radius: 6px;
  padding: 16px;
  background: #f8f9fa;
}

.preview-controls {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.preview-hint {
  color: #6c757d;
  font-style: italic;
}

.preview-result {
  border-top: 1px solid #dee2e6;
  padding-top: 16px;
}

.preview-item {
  margin-bottom: 12px;
}

.preview-item label {
  display: block;
  font-weight: 600;
  color: #495057;
  margin-bottom: 4px;
}

.preview-value {
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  padding: 8px;
  font-family: monospace;
  font-size: 0.9rem;
  word-break: break-all;
  max-height: 120px;
  overflow-y: auto;
  color: #2c3e50;
}

.header-item {
  margin-bottom: 4px;
  padding: 2px 0;
}

.header-item strong {
  color: #495057;
}

.btn-info {
  background: #17a2b8;
  color: white;
}

.btn-info:hover:not(:disabled) {
  background: #138496;
}

/* 成功条件配置样式 */
.success-condition-area {
  background: #f8f9fa;
  border: 1px solid #e9ecef;
  border-radius: 6px;
  padding: 20px;
  color: #495057; /* 确保文字颜色为深色 */
}

.condition-toggle {
  margin-bottom: 20px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  cursor: pointer;
  color: #495057; /* 确保标签文字颜色为深色 */
}

.checkbox-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
}

.condition-config {
  border-top: 1px solid #dee2e6;
  padding-top: 20px;
}

.form-control {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #ced4da;
  border-radius: 4px;
  font-size: 0.9rem;
  transition: border-color 0.2s;
}

.form-control:focus {
  outline: none;
  border-color: #007bff;
  box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
}

.form-hint {
  display: block;
  margin-top: 4px;
  color: #6c757d;
  font-size: 0.8rem;
}

.condition-example {
  margin-top: 15px;
  padding: 15px;
  background: #e7f3ff;
  border: 1px solid #b3d9ff;
  border-radius: 4px;
  font-size: 0.9rem;
  color: #495057; /* 确保示例文字颜色为深色 */
}

.condition-example ul {
  margin: 10px 0 0 0;
  padding-left: 20px;
}

.condition-example code {
  background: #f1f3f4;
  padding: 2px 4px;
  border-radius: 3px;
  font-family: monospace;
}

/* 自动标签提取样式 */
.auto-extract-hint {
  display: block;
  margin-top: 4px;
  color: #28a745;
  font-size: 0.8rem;
  animation: fadeInOut 3s ease-in-out;
}

.extracted-tag {
  background: #d4edda;
  color: #155724;
  padding: 2px 6px;
  border-radius: 3px;
  font-weight: 500;
}

@keyframes fadeInOut {
  0% { opacity: 0; }
  20% { opacity: 1; }
  80% { opacity: 1; }
  100% { opacity: 0; }
}

/* 测试功能样式 */
.test-area {
  margin-top: 16px;
}

.test-info {
  margin-bottom: 16px;
  padding: 12px;
  background: #e7f3ff;
  border: 1px solid #b3d9ff;
  border-radius: 6px;
}

.info-text {
  margin: 0;
  color: #0056b3;
  font-size: 0.9rem;
}

.test-controls {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.test-result {
  border: 1px solid #e9ecef;
  border-radius: 6px;
  padding: 16px;
  background: #f8f9fa;
  margin-top: 16px;
}

.test-result h4 {
  margin: 0 0 16px 0;
  color: #495057;
}

.result-status {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding: 12px;
  background: white;
  border-radius: 6px;
  border: 1px solid #dee2e6;
}

.status-badge {
  padding: 6px 12px;
  border-radius: 4px;
  font-weight: bold;
  font-size: 0.9rem;
}

.status-badge.success {
  background: #d4edda;
  color: #155724;
}

.status-badge.failed {
  background: #f8d7da;
  color: #721c24;
}

.response-time {
  color: #28a745;
  font-weight: 500;
  font-family: monospace;
}

.result-section {
  margin-bottom: 16px;
}

.result-section h5 {
  margin: 0 0 8px 0;
  color: #6c757d;
  font-size: 0.9rem;
  font-weight: 600;
}

.request-info, .status-info {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding: 12px;
  background: white;
  border-radius: 4px;
  border: 1px solid #dee2e6;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-label {
  font-weight: 600;
  color: #495057;
  min-width: 60px;
}

.info-value {
  color: #2c3e50;
  font-family: monospace;
  font-size: 0.9rem;
}

.url-value {
  word-break: break-all;
  max-width: 300px;
}

.status-code {
  padding: 4px 8px;
  border-radius: 4px;
  font-weight: bold;
  font-family: monospace;
}

.status-code.success {
  background: #d4edda;
  color: #155724;
}

.status-code.warning {
  background: #fff3cd;
  color: #856404;
}

.status-code.error {
  background: #f8d7da;
  color: #721c24;
}

.status-code.default {
  background: #e9ecef;
  color: #495057;
}

.status-text {
  color: #6c757d;
  font-style: italic;
}

.headers-display {
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  padding: 12px;
  max-height: 200px;
  overflow-y: auto;
}

.header-item {
  margin-bottom: 4px;
  padding: 2px 0;
  font-family: monospace;
  font-size: 0.85rem;
}

.header-key {
  color: #007bff;
  font-weight: 600;
}

.header-value {
  color: #2c3e50;
  margin-left: 8px;
}

.response-content {
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  padding: 12px;
}

.empty-response {
  color: #6c757d;
  font-style: italic;
  text-align: center;
  padding: 20px;
}

.response-body {
  position: relative;
}

.response-actions {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.btn-small {
  padding: 4px 8px;
  font-size: 0.8rem;
  border: 1px solid #dee2e6;
  background: #f8f9fa;
  color: #495057;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-small:hover {
  background: #e9ecef;
  border-color: #adb5bd;
}

.response-text {
  background: #f8f9fa;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  padding: 12px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: monospace;
  font-size: 0.85rem;
  max-height: 300px;
  overflow-y: auto;
  color: #2c3e50;
}

.result-section.error {
  border-left: 4px solid #dc3545;
  padding-left: 16px;
}

.error-content {
  background: #f8d7da;
  border: 1px solid #f5c6cb;
  border-radius: 4px;
  padding: 12px;
}

.error-content pre {
  margin: 0;
  color: #721c24;
  font-family: monospace;
  font-size: 0.85rem;
  white-space: pre-wrap;
  word-break: break-word;
}

/* 成功条件详情样式 */
.condition-details {
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  padding: 12px;
}

.condition-item {
  display: flex;
  margin-bottom: 8px;
  align-items: flex-start;
}

.condition-item:last-child {
  margin-bottom: 0;
}

.condition-label {
  font-weight: 600;
  color: #495057;
  min-width: 80px;
  margin-right: 12px;
}

.condition-value {
  color: #2c3e50;
  font-family: monospace;
  font-size: 0.9rem;
  word-break: break-word;
  flex: 1;
}

.condition-value.success {
  color: #155724;
  font-weight: 600;
}

.condition-value.failed {
  color: #721c24;
  font-weight: 600;
}

/* 响应预览样式 */
.response-preview {
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  padding: 12px;
}

.response-preview-content {
  position: relative;
}

.preview-info {
  display: flex;
  gap: 16px;
  margin-bottom: 8px;
  font-size: 0.85rem;
  color: #6c757d;
}

.content-length, .content-type {
  font-family: monospace;
}

.preview-text {
  background: #f8f9fa;
  border: 1px solid #dee2e6;
  border-radius: 4px;
  padding: 8px;
  font-family: monospace;
  font-size: 0.85rem;
  color: #2c3e50;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 120px;
  overflow-y: auto;
}

.preview-note {
  margin-top: 8px;
  font-size: 0.8rem;
  color: #6c757d;
  font-style: italic;
}

/* 响应内容展开/收起 */
.response-text.collapsed {
  max-height: 200px;
  overflow: hidden;
  position: relative;
}

.response-text.collapsed::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 40px;
  background: linear-gradient(transparent, #f8f9fa);
  pointer-events: none;
}

/* 状态码样式增强 */
.result-status .status-code {
  margin-left: 8px;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 0.8rem;
}

.result-status .status-text {
  margin-left: 4px;
  color: #6c757d;
  font-size: 0.85rem;
}

/* 测试结果详细描述样式 */
.test-result-description {
  margin-top: 16px;
  background: white;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  padding: 16px;
}

.description-header h5 {
  margin: 0 0 12px 0;
  color: #495057;
  font-size: 1rem;
  font-weight: 600;
}

.description-content {
  font-size: 0.9rem;
}

.description-title {
  font-weight: 600;
  color: #495057;
  margin-bottom: 8px;
}

.condition-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.condition-list li {
  margin-bottom: 6px;
  padding: 4px 0;
  color: #2c3e50;
}

.condition-list li strong {
  color: #495057;
  min-width: 80px;
  display: inline-block;
}

.success-reason {
  color: #155724 !important;
  background: #d4edda;
  padding: 8px;
  border-radius: 4px;
  border-left: 4px solid #28a745;
}

.failure-reason {
  color: #721c24 !important;
  background: #f8d7da;
  padding: 8px;
  border-radius: 4px;
  border-left: 4px solid #dc3545;
}

.http-status-description p {
  margin: 0;
  color: #2c3e50;
  line-height: 1.5;
}

/* 否定条件样式 */
.negative-condition {
  color: #dc3545 !important;
  font-weight: 600;
  background: rgba(220, 53, 69, 0.1);
  padding: 2px 4px;
  border-radius: 3px;
  border: 1px solid rgba(220, 53, 69, 0.2);
}
</style>
