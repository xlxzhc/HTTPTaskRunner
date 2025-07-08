<template>
  <div class="task-list">
    <div class="list-header">
      <h2>任务列表</h2>
      <div class="list-actions">
        <input
          v-model="searchText"
          type="text"
          placeholder="搜索任务..."
          class="search-input"
        />
        <div class="tag-filters">
          <span class="filter-label">标签过滤:</span>
          <button
            @click="selectedTags = []"
            class="tag-filter-btn"
            :class="{ active: selectedTags.length === 0 }"
          >
            全部 ({{ Object.keys(props.tasks).length }})
          </button>
          <button
            v-for="tag in availableTags"
            :key="tag.name"
            @click="toggleTagFilter(tag.name)"
            class="tag-filter-btn"
            :class="{ active: selectedTags.includes(tag.name) }"
          >
            {{ tag.name }} ({{ tag.count }})
          </button>
        </div>
        <button @click="$emit('create')" class="btn btn-primary">
          新建任务
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading">
      加载中...
    </div>

    <div v-else-if="filteredTasks.length === 0" class="empty">
      <p>暂无任务</p>
    </div>

    <div v-else class="task-table-container">
      <table class="task-table">
        <thead>
          <tr>
            <th>任务名</th>
            <th>URL</th>
            <th>方法</th>
            <th>次数</th>
            <th>线程</th>
            <th>定时规则</th>
            <th>下次执行</th>
            <th>最后执行</th>
            <th>成功次数</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="task in filteredTasks" :key="task.id">
            <!-- 第一行：任务信息 -->
            <tr
              class="task-row task-info-row"
              :class="{ running: task.isRunning }"
            >
            <!-- 任务名 -->
            <td class="task-name-cell">
              <div class="task-name-container">
                <span class="task-name">{{ task.name }}</span>
                <div class="task-tags" v-if="task.tags && task.tags.length > 0">
                  <span v-for="tag in task.tags" :key="tag" class="tag">{{ tag }}</span>
                </div>
              </div>
            </td>

            <!-- URL -->
            <td class="url-cell">
              <span
                class="url-display"
                :title="task.url"
                @click="copyToClipboard(task.url)"
              >
                {{ formatUrl(task.url) }}
              </span>
            </td>

            <!-- 方法 -->
            <td class="method-cell">{{ task.method }}</td>

            <!-- 次数 -->
            <td class="times-cell">
              <div class="editable-field">
                <input
                  v-if="editingFields[task.id]?.times"
                  v-model.number="tempValues[task.id].times"
                  type="number"
                  min="1"
                  max="10000"
                  class="quick-edit-input"
                  @blur="saveQuickEdit(task.id, 'times')"
                  @keyup.enter="saveQuickEdit(task.id, 'times')"
                  @keyup.escape="cancelQuickEdit(task.id, 'times')"
                />
                <span
                  v-else
                  class="value editable"
                  @click="startQuickEdit(task.id, 'times', task.times)"
                  :title="'点击编辑执行次数'"
                >
                  {{ task.times }}
                  <span class="edit-icon">✏️</span>
                </span>
              </div>
            </td>

            <!-- 线程 -->
            <td class="threads-cell">
              <div class="editable-field">
                <input
                  v-if="editingFields[task.id]?.threads"
                  v-model.number="tempValues[task.id].threads"
                  type="number"
                  min="1"
                  max="100"
                  class="quick-edit-input"
                  @blur="saveQuickEdit(task.id, 'threads')"
                  @keyup.enter="saveQuickEdit(task.id, 'threads')"
                  @keyup.escape="cancelQuickEdit(task.id, 'threads')"
                />
                <span
                  v-else
                  class="value editable"
                  @click="startQuickEdit(task.id, 'threads', task.threads)"
                  :title="'点击编辑线程数量'"
                >
                  {{ task.threads }}
                  <span class="edit-icon">✏️</span>
                </span>
              </div>
            </td>

            <!-- 定时规则 -->
            <td class="cron-cell">
              <span v-if="task.cronExpr" class="cron-expr" :title="task.cronExpr">
                {{ getScheduleInfo(task.id).cronDescription || task.cronExpr }}
              </span>
              <span v-else class="no-cron">未设置</span>
            </td>

            <!-- 下次执行 -->
            <td class="next-run-cell">
              <span v-if="getScheduleInfo(task.id).nextRunTime" class="next-run-time">
                {{ formatDateTime(getScheduleInfo(task.id).nextRunTime) }}
              </span>
              <span v-else class="no-schedule">-</span>
            </td>

            <!-- 最后执行 -->
            <td class="last-run-cell">
              <div v-if="task.lastRunTime" class="last-run-info">
                <div class="last-run-time">{{ formatDateTime(task.lastRunTime) }}</div>
                <div class="last-run-status" :class="task.lastRunStatus">
                  {{ getLastRunStatusText(task.lastRunStatus) }}
                </div>
              </div>
              <span v-else class="no-run">未执行</span>
            </td>

            <!-- 成功次数 -->
            <td class="success-count-cell">
              <span v-if="task.lastRunResult" class="success-count">
                {{ extractSuccessCount(task.lastRunResult) }}
              </span>
              <span v-else>-</span>
            </td>

            <!-- 状态 -->
            <td class="status-cell">
              <span v-if="task.isRunning" class="status-badge running">运行中</span>
              <span v-else-if="getScheduleInfo(task.id).isScheduled" class="status-badge scheduled">已定时</span>
              <span v-else-if="task.cronExpr" class="status-badge idle-scheduled">待定时</span>
              <span v-else class="status-badge idle">空闲</span>
            </td>





            </tr>

            <!-- 第二行：操作按钮 -->
            <tr class="task-row task-actions-row" :class="{ running: task.isRunning }">
              <td colspan="10" class="actions-cell-full">
                <div class="action-buttons-container">
                  <button
                    @click="$emit('edit', task)"
                    class="btn btn-sm btn-primary"
                    title="编辑任务"
                  >
                    编辑
                  </button>
                  <button
                    @click="$emit('test', task.id)"
                    class="btn btn-sm btn-info"
                    title="测试任务"
                  >
                    测试
                  </button>
                  <button
                    v-if="!task.isRunning"
                    @click="$emit('execute', task.id)"
                    class="btn btn-sm btn-success"
                    title="执行任务"
                  >
                    执行
                  </button>
                  <button
                    v-if="task.isRunning"
                    @click="$emit('stop', task.id)"
                    class="btn btn-sm btn-warning"
                    title="停止任务"
                  >
                    停止
                  </button>
                  <button
                    @click="toggleLogs(task.id)"
                    class="btn btn-sm btn-secondary"
                    title="查看日志"
                  >
                    日志
                  </button>
                  <button
                    v-if="task.cronExpr && !isScheduled(task.id)"
                    @click="$emit('schedule', task.id)"
                    class="btn btn-sm btn-info"
                    :disabled="task.isRunning"
                    title="启用定时"
                  >
                    定时
                  </button>
                  <button
                    v-if="task.cronExpr && isScheduled(task.id)"
                    @click="$emit('unschedule', task.id)"
                    class="btn btn-sm btn-warning"
                    :disabled="task.isRunning"
                    title="取消定时"
                  >
                    取消定时
                  </button>
                  <button
                    @click="$emit('delete', task.id)"
                    class="btn btn-sm btn-danger"
                    :disabled="task.isRunning"
                    title="删除任务"
                  >
                    删除
                  </button>
                </div>
              </td>
            </tr>

            <!-- 第三行：日志显示区域（仅在展开时显示） -->
            <tr v-if="showLogs[task.id]" class="task-row task-logs-row">
              <td colspan="10" class="logs-cell-full">
                <div class="task-logs-container">
                  <div class="logs-header">
                    <div class="logs-title">{{ task.name }} - 执行日志</div>
                    <div class="logs-controls">
                      <input
                        v-model="logSearchQuery"
                        type="text"
                        placeholder="搜索日志..."
                        class="log-search"
                      />
                      <label class="json-format-checkbox">
                        <input
                          v-model="formatJsonResponse"
                          type="checkbox"
                        />
                        <span class="checkbox-label">格式化JSON</span>
                      </label>
                      <button @click="refreshLogs(task.id)" class="btn btn-sm btn-info">刷新</button>
                      <button @click="clearLogs(task.id)" class="btn btn-sm btn-danger btn-clear">清空</button>
                      <button @click="toggleLogs(task.id)" class="btn btn-sm btn-secondary">关闭</button>
                    </div>
                  </div>

                  <div class="logs-content">
                    <div v-if="!taskLogEntries[task.id] || taskLogEntries[task.id].length === 0" class="no-logs">
                      暂无执行日志
                    </div>
                    <div v-else class="logs-list">
                      <div
                        v-for="logEntry in getFilteredLogEntries(task.id)"
                        :key="logEntry.id"
                        class="log-entry-container"
                      >
                        <div
                          class="task-log-entry"
                          :class="getLogEntryClass(logEntry)"
                        >
                          <div class="log-main">
                            <div class="log-timestamp">{{ formatDateTime(logEntry.timestamp) }}</div>
                            <div class="log-message">{{ logEntry.message }}</div>
                            <div class="log-status" :class="logEntry.status">
                              {{ getLogStatusText(logEntry.status) }}
                            </div>
                          </div>
                          <button
                            v-if="logEntry.executionLogId"
                            @click="toggleDetailedLog(logEntry.executionLogId)"
                            class="expand-btn"
                            :class="{ expanded: expandedLogs[logEntry.executionLogId] }"
                          >
                            {{ expandedLogs[logEntry.executionLogId] ? '收起' : '详情' }}
                          </button>
                        </div>

                        <!-- 详细执行日志 -->
                        <div
                          v-if="expandedLogs[logEntry.executionLogId]"
                          class="detailed-logs"
                        >
                          <div v-if="!executionLogs[logEntry.executionLogId]" class="loading-detailed">
                            加载详细日志中...
                          </div>
                          <div v-else class="execution-summary">
                            <div class="summary-header">
                              <div class="summary-title">执行详情</div>
                              <div class="summary-stats">
                                总计: {{ executionLogs[logEntry.executionLogId].totalRequests }} 次 |
                                成功: {{ executionLogs[logEntry.executionLogId].successCount }} 次 |
                                失败: {{ executionLogs[logEntry.executionLogId].failedCount }} 次 |
                                耗时: {{ formatDuration(executionLogs[logEntry.executionLogId].duration / 1000) }}
                              </div>
                            </div>

                            <!-- 失败日志聚合显示 -->
                            <div v-if="executionLogs[logEntry.executionLogId].failedCount > 0" class="failure-groups">
                              <div class="failure-groups-header">
                                <span class="failure-groups-title">失败原因汇总</span>
                                <span class="failure-groups-count">({{ executionLogs[logEntry.executionLogId].failedCount }} 次失败)</span>
                              </div>
                              <div
                                v-for="(requests, errorSummary) in groupFailedRequests(executionLogs[logEntry.executionLogId].detailedLogs)"
                                :key="errorSummary"
                                class="failure-group"
                              >
                                <div class="failure-group-header" @click="toggleFailureGroup(logEntry.executionLogId, errorSummary)">
                                  <span class="failure-icon">⚠️</span>
                                  <span class="failure-summary">{{ errorSummary }}</span>
                                  <span class="failure-count">({{ requests.length }} 次)</span>
                                  <span class="expand-icon">{{ expandedFailureGroups[`${logEntry.executionLogId}_${errorSummary}`] ? '▼' : '▶' }}</span>
                                </div>
                                <div v-if="expandedFailureGroups[`${logEntry.executionLogId}_${errorSummary}`]" class="failure-group-details">
                                  <div
                                    v-for="(request, index) in requests"
                                    :key="index"
                                    class="failure-detail-item"
                                  >
                                    <div class="detail-item-header">
                                      <span class="failure-detail-status">{{ request.statusCode || 'ERROR' }}</span>
                                      <div class="failure-detail-reason">{{ getDetailedReasonText(request) }}</div>
                                      <span class="failure-detail-time">{{ formatDuration(request.responseTime / 1000) }}</span>
                                      <button
                                        v-if="request.response"
                                        @click="toggleResponseDetail(request.requestId)"
                                        class="response-toggle-inline"
                                      >
                                        {{ showResponseDetails[request.requestId] ? '隐藏' : '详情' }}
                                      </button>
                                    </div>
                                    <div v-if="request.response && showResponseDetails[request.requestId]" class="response-content">
                                      <pre>{{ formatJsonContent(request.response) }}</pre>
                                    </div>
                                  </div>
                                </div>
                              </div>
                            </div>

                            <!-- 成功日志聚合显示 -->
                            <div v-if="executionLogs[logEntry.executionLogId].successCount > 0" class="success-groups">
                              <div class="success-groups-header">
                                <span class="success-groups-title">成功原因汇总</span>
                                <span class="success-groups-count">({{ executionLogs[logEntry.executionLogId].successCount }} 次成功)</span>
                              </div>
                              <div
                                v-for="(requests, successSummary) in groupSuccessRequests(executionLogs[logEntry.executionLogId].detailedLogs)"
                                :key="successSummary"
                                class="success-group"
                              >
                                <div class="success-group-header" @click="toggleFailureGroup(logEntry.executionLogId, successSummary)">
                                  <span class="success-icon">✅</span>
                                  <span class="success-summary">{{ successSummary }}</span>
                                  <span class="success-count">({{ requests.length }} 次)</span>
                                  <span class="expand-icon">{{ expandedFailureGroups[`${logEntry.executionLogId}_${successSummary}`] ? '▼' : '▶' }}</span>
                                </div>
                                <div v-if="expandedFailureGroups[`${logEntry.executionLogId}_${successSummary}`]" class="success-group-details">
                                  <div
                                    v-for="(request, index) in requests"
                                    :key="index"
                                    class="success-detail-item"
                                  >
                                    <div class="detail-item-header">
                                      <span class="success-detail-status">{{ request.statusCode || 'OK' }}</span>
                                      <div class="success-detail-reason">{{ getDetailedReasonText(request) }}</div>
                                      <span class="success-detail-time">{{ formatDuration(request.responseTime / 1000) }}</span>
                                      <button
                                        v-if="request.response"
                                        @click="toggleResponseDetail(request.requestId)"
                                        class="response-toggle-inline"
                                      >
                                        {{ showResponseDetails[request.requestId] ? '隐藏' : '详情' }}
                                      </button>
                                    </div>
                                    <div v-if="request.response && showResponseDetails[request.requestId]" class="response-content">
                                      <pre>{{ formatJsonContent(request.response) }}</pre>
                                    </div>
                                  </div>
                                </div>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="logs-footer">
                    <div class="logs-count">
                      共 {{ taskLogEntries[task.id]?.length || 0 }} 条日志
                    </div>
                  </div>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>


    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

// Props
const props = defineProps<{
  tasks: Record<string, any>
  loading: boolean
  scheduledTasks?: string[]
}>()

// Emits
defineEmits<{
  edit: [task: any]
  test: [taskId: string]
  execute: [taskId: string]
  stop: [taskId: string]
  logs: [taskId: string]
  schedule: [taskId: string]
  unschedule: [taskId: string]
  delete: [taskId: string]
  create: []
}>()

// 响应式数据
const searchText = ref('')
const scheduleInfoCache = ref<Record<string, any>>({})
const showLogs = ref<Record<string, boolean>>({})
const taskLogEntries = ref<Record<string, any[]>>({})
const executionLogs = ref<Record<string, any>>({})
const expandedLogs = ref<Record<string, boolean>>({})
const showResponseDetails = ref<Record<string, boolean>>({})
const logSearchQuery = ref('')
const selectedTags = ref<string[]>([])
// JSON格式化选项
const formatJsonResponse = ref(false)
// 失败日志聚合相关
const expandedFailureGroups = ref<Record<string, boolean>>({})
// 快捷编辑相关
const editingFields = ref<Record<string, Record<string, boolean>>>({})
const tempValues = ref<Record<string, Record<string, number>>>({})

// 方法
const isScheduled = (taskId: string) => {
  return props.scheduledTasks?.includes(taskId) || false
}

// 获取调度信息
const getScheduleInfo = (taskId: string) => {
  return scheduleInfoCache.value[taskId] || {
    taskId,
    isScheduled: false,
    cronExpr: '',
    nextRunTime: '',
    cronDescription: '',
    status: 'idle'
  }
}

// 获取状态文字
const getStatusText = (taskId: string) => {
  const info = getScheduleInfo(taskId)
  switch (info.status) {
    case 'scheduled': return '已启用'
    case 'running': return '运行中'
    case 'error': return '错误'
    default: return '未启用'
  }
}



// 获取下次执行时间的样式类
const getNextRunClass = (taskId: string) => {
  const info = getScheduleInfo(taskId)
  if (info.status === 'error') return 'error'
  if (!info.nextRunTime) return 'loading'

  // 检查是否即将执行（30分钟内）
  try {
    const nextTime = new Date(info.nextRunTime.replace(' ', 'T'))
    const now = new Date()
    const diffMinutes = (nextTime.getTime() - now.getTime()) / (1000 * 60)

    if (diffMinutes < 0) return 'overdue'
    if (diffMinutes < 30) return 'soon'
    if (diffMinutes < 60) return 'upcoming'
    return 'normal'
  } catch {
    return 'normal'
  }
}

// 加载调度信息
const loadScheduleInfo = async () => {
  try {
    // 动态导入后端方法
    const { GetTaskScheduleInfo } = await import('../../wailsjs/go/main/App')

    const taskIds = Object.keys(props.tasks)
    const promises = taskIds.map(async (taskId) => {
      try {
        const info = await GetTaskScheduleInfo(taskId)
        scheduleInfoCache.value[taskId] = info
      } catch (error) {
        console.error(`获取任务 ${taskId} 调度信息失败:`, error)
      }
    })

    await Promise.all(promises)
  } catch (error) {
    console.error('加载调度信息失败:', error)
  }
}

// 定时更新调度信息
let updateInterval: number | null = null

onMounted(() => {
  loadScheduleInfo()
  // 每30秒更新一次调度信息
  updateInterval = setInterval(loadScheduleInfo, 30000)
})

onUnmounted(() => {
  if (updateInterval) {
    clearInterval(updateInterval)
  }
})

// 日志相关方法
const toggleLogs = async (taskId: string) => {
  showLogs.value[taskId] = !showLogs.value[taskId]

  if (showLogs.value[taskId] && !taskLogEntries.value[taskId]) {
    await loadTaskLogEntries(taskId)
  }
}

const loadTaskLogEntries = async (taskId: string) => {
  try {
    const { GetTaskLogEntries } = await import('../../wailsjs/go/main/App')
    const logEntries = await GetTaskLogEntries(taskId)
    taskLogEntries.value[taskId] = logEntries || []
  } catch (error) {
    console.error('加载任务日志失败:', error)
    taskLogEntries.value[taskId] = []
  }
}

const toggleDetailedLog = async (logEntryId: string) => {
  expandedLogs.value[logEntryId] = !expandedLogs.value[logEntryId]

  if (expandedLogs.value[logEntryId] && !executionLogs.value[logEntryId]) {
    await loadExecutionLog(logEntryId)
  }
}

const loadExecutionLog = async (logEntryId: string) => {
  try {
    const { GetExecutionLog } = await import('../../wailsjs/go/main/App')
    const executionLog = await GetExecutionLog(logEntryId)
    if (executionLog) {
      executionLogs.value[logEntryId] = executionLog

      // 默认折叠所有响应详情
      if (executionLog.detailedLogs && executionLog.detailedLogs.length > 0) {
        executionLog.detailedLogs.forEach((log: any) => {
          showResponseDetails.value[log.requestId] = false // 默认折叠，用户可以点击查看详情
        })
      }
    }
  } catch (error) {
    console.error('加载详细日志失败:', error)
  }
}

const toggleResponseDetail = (requestId: string) => {
  showResponseDetails.value[requestId] = !showResponseDetails.value[requestId]
}

// 获取错误类型文本
const getErrorTypeText = (errorType: string) => {
  switch (errorType) {
    case 'network': return '网络错误'
    case 'parsing': return '解析错误'
    case 'condition': return '成功条件失败'
    case 'http': return 'HTTP状态错误'
    default: return '未知错误'
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



// 获取简化的错误摘要
const getSimplifiedErrorSummary = (request: any) => {
  if (!request.error && !request.detailedError) return '未知错误'

  // 根据错误类型返回简洁的摘要
  switch (request.errorType) {
    case 'network':
      return '网络连接失败'
    case 'parsing':
      return '响应解析失败'
    case 'condition':
      if (request.successConditionDetails) {
        const conditionType = getConditionTypeText(request.successConditionDetails.type)
        if (request.successConditionDetails.jsonPath) {
          return `${conditionType}失败 (${request.successConditionDetails.jsonPath})`
        } else {
          return `${conditionType}失败`
        }
      }
      return '成功条件不满足'
    case 'http':
      return `HTTP ${request.statusCode} 错误`
    default:
      // 如果有基础错误信息，使用它
      if (request.error) {
        return request.error
      }
      return '请求失败'
  }
}

// 对失败日志进行分组聚合
const groupFailedRequests = (detailedLogs: any[]) => {
  const failedRequests = detailedLogs.filter(log => !log.success)
  const groups: Record<string, any[]> = {}

  failedRequests.forEach(request => {
    const errorSummary = getSimplifiedErrorSummary(request)
    if (!groups[errorSummary]) {
      groups[errorSummary] = []
    }
    groups[errorSummary].push(request)
  })

  return groups
}

// 对成功日志进行分组聚合
const groupSuccessRequests = (detailedLogs: any[]) => {
  const successRequests = detailedLogs.filter(log => log.success)
  const groups: Record<string, any[]> = {}

  successRequests.forEach(request => {
    let successSummary = '请求成功'
    if (request.successConditionDetails) {
      const conditionType = getConditionTypeText(request.successConditionDetails.type)
      if (request.successConditionDetails.jsonPath) {
        successSummary = `${conditionType}成功 (${request.successConditionDetails.jsonPath})`
      } else {
        successSummary = `${conditionType}成功`
      }
    }

    if (!groups[successSummary]) {
      groups[successSummary] = []
    }
    groups[successSummary].push(request)
  })

  return groups
}

// 切换失败分组的展开状态
const toggleFailureGroup = (logEntryId: string, errorSummary: string) => {
  const key = `${logEntryId}_${errorSummary}`
  expandedFailureGroups.value[key] = !expandedFailureGroups.value[key]
}

// 格式化JSON响应内容
const formatJsonContent = (content: string) => {
  if (!formatJsonResponse.value) {
    return content
  }

  try {
    // 尝试解析JSON
    const parsed = JSON.parse(content)
    // 格式化输出，缩进2个空格
    return JSON.stringify(parsed, null, 2)
  } catch (error) {
    // 如果不是有效的JSON，返回原内容
    return content
  }
}

// 获取详细的成功/失败原因说明
const getDetailedReasonText = (request: any) => {
  if (request.success) {
    // 成功的情况
    if (request.successConditionDetails) {
      const conditionType = getConditionTypeText(request.successConditionDetails.type)
      const operator = getOperatorText(request.successConditionDetails.operator)
      const expectedValue = request.successConditionDetails.expectedValue
      const actualValue = request.successConditionDetails.actualValue

      if (request.successConditionDetails.jsonPath) {
        return `成功原因：检查 '${actualValue}' ${operator} '${expectedValue}' (${request.successConditionDetails.jsonPath})`
      } else {
        return `成功原因：检查 '${actualValue}' ${operator} '${expectedValue}'`
      }
    } else {
      return `成功原因：HTTP ${request.statusCode} 响应正常`
    }
  } else {
    // 失败的情况
    if (request.successConditionDetails && request.successConditionDetails.reason) {
      const conditionType = getConditionTypeText(request.successConditionDetails.type)
      const operator = getOperatorText(request.successConditionDetails.operator)
      const expectedValue = request.successConditionDetails.expectedValue
      const actualValue = request.successConditionDetails.actualValue

      if (request.successConditionDetails.jsonPath) {
        return `失败原因：检查 '${actualValue}' ${operator} '${expectedValue}' (${request.successConditionDetails.jsonPath})`
      } else {
        return `失败原因：检查 '${actualValue}' ${operator} '${expectedValue}'`
      }
    } else if (request.error) {
      return `失败原因：${request.error}`
    } else {
      return `失败原因：HTTP ${request.statusCode || 'ERROR'} 错误`
    }
  }
}

const refreshLogs = async (taskId: string) => {
  await loadTaskLogEntries(taskId)
}

// 刷新特定任务的日志（供父组件调用）
const refreshTaskLogs = async (taskId: string) => {
  // 如果该任务的日志已经加载过，则刷新
  if (taskLogEntries.value[taskId]) {
    await loadTaskLogEntries(taskId)
  }
}

// 暴露方法给父组件
defineExpose({
  refreshTaskLogs
})

const clearLogs = async (taskId: string) => {
  if (confirm('确定要清空该任务的所有日志吗？')) {
    try {
      // 调用后端API清空日志
      const { ClearTaskLogs } = await import('../../wailsjs/go/main/App')
      const result = await ClearTaskLogs(taskId)

      // 清空前端显示的日志数据
      taskLogEntries.value[taskId] = []

      // 清空相关的详细日志
      Object.keys(executionLogs.value).forEach(key => {
        if (key.startsWith(taskId)) {
          delete executionLogs.value[key]
        }
      })

      // 显示成功消息
      showQuickEditMessage(result, 'success')
    } catch (error) {
      console.error('清空日志失败:', error)
      showQuickEditMessage(`清空日志失败: ${error}`, 'error')
    }
  }
}

const getFilteredLogEntries = (taskId: string) => {
  const logs = taskLogEntries.value[taskId] || []
  const search = logSearchQuery.value

  if (!search) return logs

  return logs.filter((log: any) =>
    log.message.toLowerCase().includes(search.toLowerCase()) ||
    log.status.toLowerCase().includes(search.toLowerCase())
  )
}

const getLogEntryClass = (logEntry: any) => {
  return `log-${logEntry.status}`
}

const getLogStatusText = (status: string) => {
  switch (status) {
    case 'success': return '成功'
    case 'failed': return '失败'
    case 'partial': return '部分成功'
    case 'running': return '执行中'
    default: return status
  }
}

// 标签过滤方法
const toggleTagFilter = (tagName: string) => {
  const index = selectedTags.value.indexOf(tagName)
  if (index > -1) {
    selectedTags.value.splice(index, 1)
  } else {
    selectedTags.value.push(tagName)
  }
}

// 计算属性
const availableTags = computed(() => {
  const tagCounts: Record<string, number> = {}

  Object.values(props.tasks).forEach((task: any) => {
    if (task.tags && Array.isArray(task.tags)) {
      task.tags.forEach((tag: string) => {
        tagCounts[tag] = (tagCounts[tag] || 0) + 1
      })
    }
  })

  return Object.entries(tagCounts)
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count)
})

const filteredTasks = computed(() => {
  let tasksArray = Object.values(props.tasks)

  // 标签过滤
  if (selectedTags.value.length > 0) {
    tasksArray = tasksArray.filter((task: any) =>
      task.tags && task.tags.some((tag: string) => selectedTags.value.includes(tag))
    )
  }

  // 文本搜索
  if (searchText.value) {
    const search = searchText.value.toLowerCase()
    tasksArray = tasksArray.filter((task: any) =>
      task.name.toLowerCase().includes(search) ||
      task.url.toLowerCase().includes(search) ||
      (task.tags && task.tags.some((tag: string) => tag.toLowerCase().includes(search)))
    )
  }

  return tasksArray
})

// 快捷编辑方法
const startQuickEdit = (taskId: string, field: string, currentValue: number) => {
  if (!editingFields.value[taskId]) {
    editingFields.value[taskId] = {}
  }
  if (!tempValues.value[taskId]) {
    tempValues.value[taskId] = {}
  }

  editingFields.value[taskId][field] = true
  tempValues.value[taskId][field] = currentValue

  // 下一帧聚焦输入框
  setTimeout(() => {
    const inputRef = field === 'times' ? 'timesInput' : 'threadsInput'
    const inputs = document.querySelectorAll('.quick-edit-input')
    const targetInput = Array.from(inputs).find(input =>
      input.closest('.task-card')?.querySelector('.task-name')?.textContent?.includes(props.tasks[taskId]?.name)
    ) as HTMLInputElement
    if (targetInput) {
      targetInput.focus()
      targetInput.select()
    }
  }, 50)
}

const saveQuickEdit = async (taskId: string, field: string) => {
  const newValue = tempValues.value[taskId]?.[field]

  if (!newValue || newValue < 1) {
    cancelQuickEdit(taskId, field)
    return
  }

  // 验证范围
  if (field === 'times' && (newValue < 1 || newValue > 10000)) {
    alert('执行次数必须在1-10000之间')
    cancelQuickEdit(taskId, field)
    return
  }

  if (field === 'threads' && (newValue < 1 || newValue > 100)) {
    alert('线程数量必须在1-100之间')
    cancelQuickEdit(taskId, field)
    return
  }

  try {
    const { UpdateTask } = await import('../../wailsjs/go/main/App')
    const task = props.tasks[taskId]

    // 更新任务
    const result = await UpdateTask(
      taskId,
      task.name,
      task.url,
      task.method,
      task.headersText,
      task.data,
      field === 'times' ? newValue : task.times,
      field === 'threads' ? newValue : task.threads,
      task.delayMin,
      task.delayMax,
      task.tags,
      task.cronExpr,
      task.successCondition || {
        enabled: false,
        jsonPath: '',
        operator: 'equals',
        expectedValue: ''
      }
    )

    if (result.includes('成功')) {
      // 更新本地数据
      if (field === 'times') {
        task.times = newValue
      } else if (field === 'threads') {
        task.threads = newValue
      }

      // 显示成功消息
      showQuickEditMessage(`${field === 'times' ? '执行次数' : '线程数量'}已更新为 ${newValue}`, 'success')
    } else {
      showQuickEditMessage(result, 'error')
    }
  } catch (error) {
    showQuickEditMessage(`更新失败: ${error}`, 'error')
  }

  cancelQuickEdit(taskId, field)
}

const cancelQuickEdit = (taskId: string, field: string) => {
  if (editingFields.value[taskId]) {
    editingFields.value[taskId][field] = false
  }
  if (tempValues.value[taskId]) {
    delete tempValues.value[taskId][field]
  }
}

const showQuickEditMessage = (message: string, type: 'success' | 'error') => {
  // 创建临时消息元素
  const messageEl = document.createElement('div')
  messageEl.className = `quick-edit-message ${type}`
  messageEl.textContent = message
  messageEl.style.cssText = `
    position: fixed;
    top: 20px;
    right: 20px;
    padding: 12px 20px;
    border-radius: 4px;
    color: white;
    font-weight: 500;
    z-index: 1000;
    background: ${type === 'success' ? '#28a745' : '#dc3545'};
  `

  document.body.appendChild(messageEl)

  setTimeout(() => {
    document.body.removeChild(messageEl)
  }, 3000)
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

// 格式化URL显示，超过50字符时省略
const formatUrl = (url: string): string => {
  if (!url) return ''
  if (url.length <= 50) return url
  return url.substring(0, 30) + '...'
}

// 复制到剪贴板
const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    showQuickEditMessage('URL已复制到剪贴板', 'success')
  } catch (error) {
    // 降级方案：使用传统方法
    const textArea = document.createElement('textarea')
    textArea.value = text
    document.body.appendChild(textArea)
    textArea.select()
    document.execCommand('copy')
    document.body.removeChild(textArea)
    showQuickEditMessage('URL已复制到剪贴板', 'success')
  }
}

// 格式化日期时间
const formatDateTime = (dateTimeStr: string): string => {
  if (!dateTimeStr) return '-'
  try {
    const date = new Date(dateTimeStr)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  } catch (error) {
    return dateTimeStr
  }
}

// 获取最后执行状态文本
const getLastRunStatusText = (status: string): string => {
  switch (status) {
    case 'success': return '成功'
    case 'failed': return '失败'
    case 'error': return '错误'
    case 'stopped': return '已停止'
    default: return status || '未知'
  }
}

// 提取成功次数
const extractSuccessCount = (result: string): string => {
  if (!result) return '-'
  const match = result.match(/成功(\d+)次/)
  return match ? match[1] : '-'
}
</script>

<style scoped>
.task-list {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  overflow: hidden;
}

.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #e9ecef;
  background: #f8f9fa;
}

.list-header h2 {
  margin: 0;
  color: #2c3e50;
}

.list-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.search-input {
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
  width: 200px;
}

.tag-filters {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.filter-label {
  font-weight: 500;
  color: #495057;
  font-size: 0.9rem;
}

.tag-filter-btn {
  padding: 4px 8px;
  border: 1px solid #dee2e6;
  background: white;
  border-radius: 12px;
  cursor: pointer;
  font-size: 0.8rem;
  color: #495057;
  transition: all 0.2s;
}

.tag-filter-btn:hover {
  background: #e9ecef;
  border-color: #adb5bd;
}

.tag-filter-btn.active {
  background: #007bff;
  color: white;
  border-color: #007bff;
}

.loading, .empty {
  padding: 40px;
  text-align: center;
  color: #6c757d;
}

.task-table-container {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  overflow: hidden;
  margin: 20px;
}

.task-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

.task-table th {
  background: #f8f9fa;
  padding: 12px 8px;
  text-align: left;
  font-weight: 600;
  color: #495057;
  border-bottom: 2px solid #dee2e6;
  white-space: nowrap;
  font-size: 0.85rem;
}

.task-table td {
  padding: 12px 8px;
  border-bottom: 1px solid #e9ecef;
  vertical-align: middle;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  color: #495057; /* 确保文字颜色为深色 */
}

.task-row:hover {
  background-color: #f8f9fa;
}

.task-row.running {
  background-color: #fff3cd;
}

.task-row.running:hover {
  background-color: #ffeaa7;
}

/* 各列特定样式 */
.task-name-cell {
  min-width: 150px;
  max-width: 200px;
}

.task-name-container {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.task-name {
  font-weight: 600;
  color: #2c3e50;
}

.task-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.url-cell {
  min-width: 200px;
  max-width: 250px;
  color: #495057; /* 确保URL文字颜色可见 */
}

.method-cell {
  width: 80px;
  text-align: center;
  font-weight: 500;
  color: #495057; /* 确保方法文字颜色可见 */
}

.times-cell, .threads-cell {
  width: 80px;
  text-align: center;
}

.cron-cell {
  min-width: 120px;
  max-width: 150px;
}

.cron-expr {
  font-family: monospace;
  font-size: 0.8rem;
  color: #6c757d;
}

.no-cron {
  color: #adb5bd;
  font-style: italic;
}

.next-run-cell, .last-run-cell {
  min-width: 140px;
  font-size: 0.8rem;
}

.next-run-time, .last-run-time {
  display: block;
  color: #495057;
}

.last-run-status {
  display: block;
  font-size: 0.75rem;
  margin-top: 2px;
}

.last-run-status.success {
  color: #28a745;
}

.last-run-status.failed, .last-run-status.error {
  color: #dc3545;
}

.no-schedule, .no-run {
  color: #adb5bd;
  font-style: italic;
}

.success-count-cell {
  width: 80px;
  text-align: center;
  font-weight: 500;
  color: #28a745;
}

.status-cell {
  min-width: 120px;
  width: 120px;
  text-align: center;
  color: #495057; /* 确保状态文字颜色可见 */
}

/* 三行布局样式 */
.task-info-row {
  border-bottom: none; /* 移除信息行的底部边框 */
}

.task-actions-row {
  border-bottom: none; /* 移除操作行的底部边框，与日志行连接 */
}

.task-actions-row td {
  padding: 8px;
  background-color: #f8f9fa;
}

.task-actions-row.running td {
  background-color: #fff3cd;
}

.task-logs-row {
  border-bottom: 2px solid #dee2e6; /* 日志行有明显的分隔 */
}

.task-logs-row td {
  padding: 0; /* 日志区域内部自己控制padding */
  background-color: #ffffff;
}

.logs-cell-full {
  padding: 0;
}

.task-logs-container {
  border: 1px solid #e9ecef;
  border-radius: 6px;
  margin: 8px;
  background: white;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.actions-cell-full {
  padding: 8px 16px;
}

.action-buttons-container {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: center;
  align-items: center;
}

.btn-sm {
  padding: 3px 6px;
  font-size: 0.7rem;
  border-radius: 3px;
  white-space: nowrap;
  min-width: 50px;
  text-align: center;
}

.task-card {
  border: 1px solid #e9ecef;
  border-radius: 8px;
  padding: 16px;
  background: white;
  transition: all 0.2s;
}

.task-card:hover {
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  transform: translateY(-2px);
}

.task-card.running {
  border-color: #28a745;
  background: #f8fff9;
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.task-name {
  margin: 0;
  font-size: 1.1rem;
  color: #2c3e50;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 0.8rem;
  font-weight: 500;
}

.status-badge.running {
  background: #d4edda;
  color: #155724;
}

.status-badge.idle {
  background: #f8f9fa;
  color: #6c757d;
}

.status-badge.scheduled {
  background: #d1ecf1;
  color: #0c5460;
}

.status-badge.idle-scheduled {
  background: #fff3cd;
  color: #856404;
}

.task-info {
  margin-bottom: 12px;
}

.info-row {
  display: flex;
  margin-bottom: 4px;
}

.label {
  font-weight: 500;
  color: #6c757d;
  width: 60px;
  flex-shrink: 0;
}

.value {
  color: #2c3e50;
  word-break: break-all;
}

.value.editable {
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 3px;
  transition: background-color 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.value.editable:hover {
  background-color: #f8f9fa;
}

.edit-icon {
  font-size: 0.8rem;
  opacity: 0.6;
}

.editable-field {
  display: flex;
  align-items: center;
}

.quick-edit-input {
  width: 80px;
  padding: 2px 6px;
  border: 1px solid #007bff;
  border-radius: 3px;
  font-size: 0.9rem;
  background: white;
  outline: none;
}

.quick-edit-input:focus {
  box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
}

.url-display {
  cursor: pointer;
  transition: color 0.2s;
  word-break: break-all;
}

.url-display:hover {
  color: #007bff;
  text-decoration: underline;
}

/* 响应式设计 */
@media (max-width: 1400px) {
  .task-table th:nth-child(6),
  .task-table td:nth-child(6) {
    display: none; /* 隐藏定时规则列 */
  }


}

@media (max-width: 1200px) {
  .task-table th:nth-child(5),
  .task-table td:nth-child(5) {
    display: none; /* 隐藏线程列 */
  }
}

@media (max-width: 1000px) {
  .task-table th:nth-child(7),
  .task-table td:nth-child(7),
  .task-table th:nth-child(8),
  .task-table td:nth-child(8) {
    display: none; /* 隐藏下次执行和最后执行列 */
  }


}

@media (max-width: 800px) {
  .task-table th:nth-child(9),
  .task-table td:nth-child(9) {
    display: none; /* 隐藏成功次数列 */
  }

  .action-buttons-container {
    flex-direction: column;
    gap: 4px;
    justify-content: center;
    align-items: center;
  }

  .action-buttons-container .btn-sm {
    width: 100%;
    max-width: 200px;
  }

  .btn-sm {
    font-size: 0.65rem;
    padding: 2px 4px;
  }
}

@media (max-width: 600px) {
  .task-table-container {
    margin: 10px;
  }

  .task-table th,
  .task-table td {
    padding: 8px 4px;
    font-size: 0.8rem;
  }

  .task-table th:nth-child(3),
  .task-table td:nth-child(3) {
    display: none; /* 隐藏方法列 */
  }
}

.task-tags {
  margin-bottom: 12px;
}

.tag {
  display: inline-block;
  padding: 2px 8px;
  margin-right: 6px;
  background: #e9ecef;
  color: #495057;
  border-radius: 12px;
  font-size: 0.8rem;
}

.task-actions {
  display: flex;
  gap: 8px;
}

.btn {
  padding: 6px 12px;
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

.btn-info {
  background: #17a2b8;
  color: white;
}

.btn-info:hover:not(:disabled) {
  background: #138496;
}

.btn-warning {
  background: #ffc107;
  color: #212529;
}

.btn-warning:hover:not(:disabled) {
  background: #e0a800;
}

.btn-success {
  background: #28a745;
  color: white;
}

.btn-success:hover:not(:disabled) {
  background: #218838;
}

/* 定时信息样式 */
.schedule-info {
  margin-bottom: 12px;
  padding: 12px;
  background: #f8f9fa;
  border: 1px solid #e9ecef;
  border-radius: 6px;
}

.schedule-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.schedule-icon {
  font-size: 1.1rem;
  margin-right: 6px;
}

.schedule-title {
  font-weight: 600;
  color: #495057;
  font-size: 0.9rem;
  flex: 1;
}

.schedule-status {
  font-size: 0.75rem;
  padding: 2px 6px;
  border-radius: 3px;
  font-weight: 500;
}

.schedule-status.scheduled {
  background: #d4edda;
  color: #155724;
}

.schedule-status.running {
  background: #d1ecf1;
  color: #0c5460;
}

.schedule-status.error {
  background: #f8d7da;
  color: #721c24;
}

.schedule-status.idle {
  background: #e2e3e5;
  color: #383d41;
}

.schedule-details {
  font-size: 0.85rem;
}

.schedule-description {
  color: #2c3e50;
  font-weight: 500;
  margin-bottom: 6px;
}

.next-run-info {
  display: flex;
  align-items: center;
  gap: 6px;
}

.next-run-label {
  color: #6c757d;
  font-size: 0.8rem;
}

.next-run-time {
  font-weight: 500;
  font-size: 0.8rem;
}

.next-run-time.normal {
  color: #28a745;
}

.next-run-time.upcoming {
  color: #ffc107;
}

.next-run-time.soon {
  color: #fd7e14;
  animation: pulse 2s infinite;
}

.next-run-time.overdue {
  color: #dc3545;
  font-weight: 600;
}

.next-run-time.loading {
  color: #6c757d;
  font-style: italic;
}

.error-info {
  color: #dc3545;
  font-size: 0.8rem;
  font-weight: 500;
}

.last-run-info {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
  font-size: 0.75rem;
}

.last-run-label {
  color: #6c757d;
}

.last-run-time {
  color: #495057;
}

.last-run-status {
  padding: 1px 4px;
  border-radius: 2px;
  font-weight: 500;
}

.last-run-status.success {
  background: #d4edda;
  color: #155724;
}

.last-run-status.failed {
  background: #f8d7da;
  color: #721c24;
}

.last-run-status.running {
  background: #d1ecf1;
  color: #0c5460;
}

.last-run-result {
  margin-top: 2px;
  font-size: 0.75rem;
  color: #6c757d;
  font-style: italic;
}

/* 日志显示样式 */
.task-logs {
  margin-top: 12px;
  border: 1px solid #e9ecef;
  border-radius: 6px;
  background: #ffffff;
}

.logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #f8f9fa;
  border-bottom: 1px solid #e9ecef;
  border-radius: 6px 6px 0 0;
}

.logs-title {
  font-weight: 600;
  color: #495057;
  font-size: 0.9rem;
}

.logs-controls {
  display: flex;
  align-items: center;
  gap: 6px;
}

.log-search {
  padding: 4px 8px;
  border: 1px solid #dee2e6;
  border-radius: 3px;
  font-size: 0.8rem;
  width: 120px;
}

.json-format-checkbox {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.8rem;
  color: #495057;
  cursor: pointer;
  user-select: none;
}

.json-format-checkbox input[type="checkbox"] {
  margin: 0;
  cursor: pointer;
}

.checkbox-label {
  cursor: pointer;
  white-space: nowrap;
}

.btn-refresh, .btn-clear {
  padding: 4px 8px;
  border: 1px solid #dee2e6;
  background: white;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.75rem;
  color: #495057;
  transition: all 0.2s;
}

.btn-refresh:hover, .btn-clear:hover {
  background: #e9ecef;
}

.btn-clear {
  color: #dc3545;
  border-color: #dc3545;
}

.btn-clear:hover {
  background: #f8d7da;
}

.logs-content {
  max-height: 300px;
  overflow-y: auto;
  padding: 8px;
  color: #495057; /* 确保日志内容文字颜色为深色 */
}

.logs-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.log-entry-container {
  border: 1px solid #e9ecef;
  border-radius: 6px;
  overflow: hidden;
}

.task-log-entry {
  padding: 8px 12px;
  background: #f8f9fa;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.task-log-entry.log-success {
  border-left: 4px solid #28a745;
}

.task-log-entry.log-failed {
  border-left: 4px solid #dc3545;
}

.task-log-entry.log-partial {
  border-left: 4px solid #ffc107;
}

.task-log-entry.log-running {
  border-left: 4px solid #007bff;
}

.log-main {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.log-timestamp {
  font-size: 0.75rem;
  color: #6c757d;
  font-family: 'Courier New', monospace;
  min-width: 130px;
}

.log-message {
  font-size: 0.85rem;
  color: #495057;
  flex: 1;
}

.log-status {
  padding: 2px 6px;
  border-radius: 12px;
  font-size: 0.7rem;
  font-weight: 500;
  min-width: 60px;
  text-align: center;
}

.log-status.success {
  background: #d4edda;
  color: #155724;
}

.log-status.failed {
  background: #f8d7da;
  color: #721c24;
}

.log-status.partial {
  background: #fff3cd;
  color: #856404;
}

.log-status.running {
  background: #d1ecf1;
  color: #0c5460;
}

.expand-btn {
  padding: 4px 8px;
  border: 1px solid #dee2e6;
  background: white;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.75rem;
  color: #007bff;
  transition: all 0.2s;
}

.expand-btn:hover {
  background: #e9ecef;
}

.expand-btn.expanded {
  background: #007bff;
  color: white;
  border-color: #007bff;
}

.no-logs {
  padding: 20px;
  text-align: center;
  color: #6c757d;
  font-style: italic;
}

.logs-footer {
  padding: 6px 12px;
  background: #f8f9fa;
  border-top: 1px solid #e9ecef;
  border-radius: 0 0 6px 6px;
}

.logs-count {
  font-size: 0.75rem;
  color: #6c757d;
}

/* 详细日志样式 */
.detailed-logs {
  background: white;
  border-top: 1px solid #e9ecef;
  color: #495057; /* 确保详细日志文字颜色为深色 */
}

.execution-summary {
  padding: 12px;
}

.summary-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e9ecef;
}

.summary-title {
  font-weight: 600;
  color: #495057;
  font-size: 0.9rem;
}

.summary-stats {
  font-size: 0.8rem;
  color: #6c757d;
}

.detailed-requests {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.request-detail {
  border: 1px solid #e9ecef;
  border-radius: 4px;
  padding: 8px;
  background: #f8f9fa;
  color: #495057; /* 确保请求详情文字颜色为深色 */
}

.request-detail.success {
  border-left: 3px solid #28a745;
}

.request-detail.failed {
  border-left: 3px solid #dc3545;
}

.request-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.request-status {
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 0.7rem;
  font-weight: 500;
  background: #dc3545;
  color: white;
  min-width: 35px;
  text-align: center;
}

.request-status.success {
  background: #28a745;
}

.request-time {
  font-size: 0.7rem;
  color: #6c757d;
  min-width: 50px;
  text-align: right;
  margin-left: 8px;
}

.request-result-indicator {
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 0.7rem;
  font-weight: 500;
  margin-left: 8px;
  min-width: 35px;
  text-align: center;
}

.request-result-indicator.success {
  background: #d4edda;
  color: #155724;
}

.request-result-indicator.failed {
  background: #f8d7da;
  color: #721c24;
}

/* 失败日志聚合样式 */
.failure-groups {
  margin-bottom: 12px;
  border: 1px solid #f8d7da;
  border-radius: 6px;
  background: #fff5f5;
}

.failure-groups-header {
  padding: 8px 12px;
  background: #f8d7da;
  border-bottom: 1px solid #f5c6cb;
  border-radius: 6px 6px 0 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.failure-groups-title {
  font-weight: 600;
  color: #721c24;
  font-size: 0.85rem;
}

.failure-groups-count {
  font-size: 0.75rem;
  color: #856404;
  background: #fff3cd;
  padding: 2px 6px;
  border-radius: 3px;
}

.failure-group {
  border-bottom: 1px solid #f5c6cb;
}

.failure-group:last-child {
  border-bottom: none;
}

.failure-group-header {
  padding: 8px 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: background-color 0.2s;
}

.failure-group-header:hover {
  background: #f5c6cb;
}

.failure-icon {
  font-size: 0.9rem;
}

.failure-summary {
  flex: 1;
  font-size: 0.8rem;
  color: #721c24;
  font-weight: 500;
}

.failure-count {
  font-size: 0.75rem;
  color: #856404;
  background: #fff3cd;
  padding: 2px 6px;
  border-radius: 3px;
}

.expand-icon {
  font-size: 0.7rem;
  color: #6c757d;
}

.failure-group-details {
  padding: 8px 12px;
  background: #ffffff;
  border-top: 1px solid #f5c6cb;
}

.failure-detail-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid #f8f9fa;
}

.detail-item-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.failure-detail-item:last-child {
  border-bottom: none;
}

.failure-detail-status {
  padding: 2px 6px;
  background: #dc3545;
  color: white;
  border-radius: 3px;
  font-size: 0.7rem;
  font-weight: 500;
  min-width: 35px;
  text-align: center;
  flex-shrink: 0;
  margin-top: 2px;
}

.failure-detail-reason {
  flex: 1;
  font-size: 0.75rem;
  color: #495057;
  line-height: 1.4;
  padding: 2px 4px;
  background: #f8f9fa;
  border-radius: 3px;
  border-left: 3px solid #dc3545;
  font-family: 'Courier New', monospace;
}

.failure-detail-time {
  font-size: 0.7rem;
  color: #6c757d;
  min-width: 50px;
  flex-shrink: 0;
  margin-top: 2px;
}

/* 成功日志聚合样式 */
.success-groups {
  margin-bottom: 12px;
  border: 1px solid #d4edda;
  border-radius: 6px;
  background: #f8fff9;
}

.success-groups-header {
  padding: 8px 12px;
  background: #d4edda;
  border-bottom: 1px solid #c3e6cb;
  border-radius: 6px 6px 0 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.success-groups-title {
  font-weight: 600;
  color: #155724;
  font-size: 0.85rem;
}

.success-groups-count {
  font-size: 0.75rem;
  color: #155724;
  background: #d1ecf1;
  padding: 2px 6px;
  border-radius: 3px;
}

.success-group {
  border-bottom: 1px solid #c3e6cb;
}

.success-group:last-child {
  border-bottom: none;
}

.success-group-header {
  padding: 8px 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: background-color 0.2s;
}

.success-group-header:hover {
  background: #c3e6cb;
}

.success-icon {
  font-size: 0.9rem;
}

.success-summary {
  flex: 1;
  font-size: 0.8rem;
  color: #155724;
  font-weight: 500;
}

.success-count {
  font-size: 0.75rem;
  color: #155724;
  background: #d1ecf1;
  padding: 2px 6px;
  border-radius: 3px;
}

.success-group-details {
  padding: 8px 12px;
  background: #ffffff;
  border-top: 1px solid #c3e6cb;
}

.success-detail-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid #f8f9fa;
}

.success-detail-item:last-child {
  border-bottom: none;
}

.success-detail-status {
  padding: 2px 6px;
  background: #28a745;
  color: white;
  border-radius: 3px;
  font-size: 0.7rem;
  font-weight: 500;
  min-width: 35px;
  text-align: center;
  flex-shrink: 0;
  margin-top: 2px;
}

.success-detail-reason {
  flex: 1;
  font-size: 0.75rem;
  color: #495057;
  line-height: 1.4;
  padding: 2px 4px;
  background: #f8f9fa;
  border-radius: 3px;
  border-left: 3px solid #28a745;
  font-family: 'Courier New', monospace;
}

.success-detail-time {
  font-size: 0.7rem;
  color: #6c757d;
  min-width: 50px;
  flex-shrink: 0;
  margin-top: 2px;
}

.request-error {
  margin-top: 4px;
  padding: 4px 8px;
  background: #f8d7da;
  color: #721c24;
  border-radius: 3px;
  font-size: 0.75rem;
}

.request-response {
  margin-top: 8px;
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid #dee2e6;
  border-radius: 3px;
}

.request-response pre {
  margin: 0;
  padding: 8px;
  font-size: 0.7rem;
  background: #f8f9fa;
  color: #495057;
  white-space: pre-wrap;
  word-break: break-all;
}

.response-toggle {
  margin-top: 6px;
  padding: 2px 6px;
  border: 1px solid #dee2e6;
  background: white;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.7rem;
  color: #007bff;
}

.response-toggle:hover {
  background: #e9ecef;
}

.response-toggle-inline {
  padding: 1px 4px;
  border: 1px solid #dee2e6;
  background: white;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.65rem;
  color: #007bff;
  margin-left: 8px;
  flex-shrink: 0;
  height: fit-content;
  align-self: center;
}

.response-toggle-inline:hover {
  background: #e9ecef;
}

.loading-detailed {
  padding: 20px;
  text-align: center;
  color: #6c757d;
  font-style: italic;
  font-size: 0.85rem;
}

@keyframes pulse {
  0% { opacity: 1; }
  50% { opacity: 0.6; }
  100% { opacity: 1; }
}

/* 日志详情增强样式 */
.request-time {
  color: #28a745;
  font-weight: 500;
  font-family: monospace;
  font-size: 0.85rem;
}

.request-error {
  margin-top: 8px;
  padding: 8px;
  background: #f8d7da;
  border: 1px solid #f5c6cb;
  border-radius: 4px;
}

.error-label {
  font-weight: 600;
  color: #721c24;
  margin-bottom: 4px;
  font-size: 0.85rem;
}

.error-content {
  color: #721c24;
  font-family: monospace;
  font-size: 0.8rem;
  white-space: pre-wrap;
  word-break: break-word;
}

.request-header {
  flex-wrap: wrap;
}

/* 增强的错误信息样式 */
.request-error {
  margin-top: 8px;
  border: 1px solid #f5c6cb;
  border-radius: 6px;
  background: #f8d7da;
  overflow: hidden;
}

.error-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #f5c6cb;
  border-bottom: 1px solid #f1b0b7;
}

.error-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  color: #721c24;
  font-size: 0.85rem;
}

.error-icon {
  font-size: 1rem;
}

.error-type-badge {
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 0.7rem;
  font-weight: 500;
  text-transform: uppercase;
}

.error-badge-network {
  background: #dc3545;
  color: white;
}

.error-badge-parsing {
  background: #fd7e14;
  color: white;
}

.error-badge-condition {
  background: #6f42c1;
  color: white;
}

.error-badge-http {
  background: #e83e8c;
  color: white;
}

.error-badge-unknown {
  background: #6c757d;
  color: white;
}

.error-summary {
  padding: 8px 12px;
  color: #721c24;
  font-weight: 500;
  border-bottom: 1px solid #f1b0b7;
}

.detailed-error {
  border-top: 1px solid #f1b0b7;
}

.detailed-error-toggle {
  padding: 6px 12px;
  background: #f1b0b7;
}

.error-toggle-btn {
  background: none;
  border: none;
  color: #721c24;
  font-size: 0.8rem;
  cursor: pointer;
  text-decoration: underline;
  padding: 0;
}

.error-toggle-btn:hover {
  color: #491217;
}

.detailed-error-content {
  padding: 8px 12px;
  background: #fff5f5;
}

.detailed-error-content pre {
  margin: 0;
  color: #721c24;
  font-family: monospace;
  font-size: 0.8rem;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.4;
}

.condition-error-details {
  border-top: 1px solid #f1b0b7;
  background: #fff5f5;
}

.condition-error-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  font-weight: 600;
  color: #721c24;
  font-size: 0.85rem;
  background: #f1b0b7;
}

.condition-icon {
  font-size: 1rem;
}

.condition-error-content {
  padding: 8px 12px;
}

.condition-item {
  margin-bottom: 4px;
  color: #721c24;
  font-size: 0.8rem;
}

.condition-item strong {
  color: #491217;
  margin-right: 4px;
}

.condition-item.failure-reason {
  margin-top: 6px;
  padding: 6px;
  background: #f8d7da;
  border-radius: 4px;
  border-left: 3px solid #dc3545;
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

/* 简化的错误信息样式 */
.request-error-simplified {
  margin-top: 8px;
  border: 1px solid #f5c6cb;
  border-radius: 6px;
  background: #f8d7da;
  overflow: hidden;
}

.error-summary-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: #f5c6cb;
  border-bottom: 1px solid #f1b0b7;
}

.error-summary-text {
  flex: 1;
  color: #721c24;
  font-weight: 500;
  font-size: 0.85rem;
  margin: 0 8px;
}

.error-expand-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: 1px solid #721c24;
  color: #721c24;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.error-expand-btn:hover {
  background: #721c24;
  color: white;
}

.error-expand-btn.expanded {
  background: #721c24;
  color: white;
}

.expand-icon {
  font-size: 0.7rem;
  transition: transform 0.2s ease;
}

.error-expand-btn.expanded .expand-icon {
  transform: rotate(0deg);
}

.error-details-expanded {
  border-top: 1px solid #f1b0b7;
  background: #fff5f5;
}

.error-details-expanded .error-header {
  padding: 8px 12px;
  background: #f1b0b7;
  border-bottom: 1px solid #e9a6ab;
}

.error-details-expanded .error-summary {
  padding: 8px 12px;
  border-bottom: 1px solid #f1b0b7;
}

.error-details-expanded .detailed-error {
  border-top: none;
}

.error-details-expanded .detailed-error-content {
  padding: 8px 12px;
  background: #fff5f5;
}

.error-details-expanded .condition-error-details {
  border-top: 1px solid #f1b0b7;
}
</style>
