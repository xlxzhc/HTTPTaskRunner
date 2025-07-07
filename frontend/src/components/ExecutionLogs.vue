<template>
  <div class="execution-logs">
    <div class="logs-header">
      <h2>执行日志</h2>
      <div class="logs-search">
        <input
          v-model="searchText"
          type="text"
          placeholder="搜索日志内容（支持搜索请求URL、参数、响应状态码、响应内容）"
          class="search-input"
        />
        <button v-if="searchText" @click="clearSearch" class="btn btn-clear">
          ✕
        </button>
      </div>
      <div class="logs-actions">
        <button @click="clearLogs" class="btn btn-secondary">
          清空日志
        </button>
        <button @click="toggleAutoScroll" class="btn" :class="{ active: autoScroll }">
          {{ autoScroll ? '停止滚动' : '自动滚动' }}
        </button>
      </div>
    </div>

    <div class="logs-content">
      <div 
        ref="logsContainer"
        class="logs-container"
        @scroll="handleScroll"
      >
        <div v-if="filteredLogs.length === 0" class="empty-logs">
          {{ searchText ? '未找到匹配的日志' : '暂无执行日志' }}
        </div>
        <div
          v-for="(log, index) in filteredLogs"
          :key="index"
          class="log-entry"
          :class="getLogClass(log)"
        >
          <span class="log-time">{{ formatTime(log.time) }}</span>
          <span class="log-message" v-html="highlightText(log.message)"></span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, watch, computed } from 'vue'

// Props
const props = defineProps<{
  logs: Array<{
    time: string
    message: string
    level?: 'info' | 'success' | 'warning' | 'error'
  }>
}>()

// 搜索相关
const searchText = ref('')

// Emits
const emit = defineEmits<{
  clear: []
}>()

// 响应式数据
const logsContainer = ref<HTMLElement>()
const autoScroll = ref(true)

// 过滤后的日志
const filteredLogs = computed(() => {
  if (!searchText.value) {
    return props.logs
  }

  const search = searchText.value.toLowerCase()
  return props.logs.filter(log =>
    log.message.toLowerCase().includes(search) ||
    log.time.toLowerCase().includes(search)
  )
})

// 方法
const clearLogs = () => {
  emit('clear')
}

const toggleAutoScroll = () => {
  autoScroll.value = !autoScroll.value
  if (autoScroll.value) {
    scrollToBottom()
  }
}

const clearSearch = () => {
  searchText.value = ''
}

// 高亮搜索关键词
const highlightText = (text: string): string => {
  if (!searchText.value) {
    return text
  }

  const search = searchText.value
  const regex = new RegExp(`(${search.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
  return text.replace(regex, '<mark class="highlight">$1</mark>')
}

const handleScroll = () => {
  if (!logsContainer.value) return
  
  const { scrollTop, scrollHeight, clientHeight } = logsContainer.value
  const isAtBottom = scrollTop + clientHeight >= scrollHeight - 10
  
  if (!isAtBottom) {
    autoScroll.value = false
  }
}

const scrollToBottom = () => {
  if (!logsContainer.value) return
  
  nextTick(() => {
    if (logsContainer.value) {
      logsContainer.value.scrollTop = logsContainer.value.scrollHeight
    }
  })
}

const formatTime = (time: string) => {
  try {
    return new Date(time).toLocaleTimeString()
  } catch {
    return time
  }
}

const getLogClass = (log: any) => {
  if (log.level) {
    return `log-${log.level}`
  }
  
  // 根据消息内容自动判断级别
  const message = log.message.toLowerCase()
  if (message.includes('错误') || message.includes('失败')) {
    return 'log-error'
  }
  if (message.includes('成功') || message.includes('完成')) {
    return 'log-success'
  }
  if (message.includes('警告')) {
    return 'log-warning'
  }
  
  return 'log-info'
}

// 监听日志变化，自动滚动
watch(() => props.logs, () => {
  if (autoScroll.value) {
    scrollToBottom()
  }
}, { deep: true })
</script>

<style scoped>
.execution-logs {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  overflow: hidden;
  height: 600px;
  display: flex;
  flex-direction: column;
}

.logs-header {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 20px;
  border-bottom: 1px solid #e9ecef;
  background: #f8f9fa;
  flex-shrink: 0;
}

.logs-search {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.search-input {
  flex: 1;
  padding: 0.5rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.9rem;
}

.search-input:focus {
  outline: none;
  border-color: #007bff;
  box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
}

.btn-clear {
  padding: 0.25rem 0.5rem;
  background: #6c757d;
  color: white;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.8rem;
}

.btn-clear:hover {
  background: #5a6268;
}

.logs-header h2 {
  margin: 0;
  color: #2c3e50;
}

.logs-actions {
  display: flex;
  gap: 12px;
}

.logs-content {
  flex: 1;
  overflow: hidden;
}

.logs-container {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
  font-family: 'Courier New', monospace;
  font-size: 0.9rem;
  line-height: 1.4;
  background: #1e1e1e;
  color: #d4d4d4;
}

.empty-logs {
  text-align: center;
  color: #6c757d;
  padding: 40px;
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
}

.log-entry {
  display: flex;
  margin-bottom: 4px;
  padding: 4px 8px;
  border-radius: 3px;
  transition: background-color 0.2s;
}

.log-entry:hover {
  background: rgba(255,255,255,0.05);
}

.log-time {
  color: #569cd6;
  margin-right: 12px;
  flex-shrink: 0;
  min-width: 80px;
}

.log-message {
  flex: 1;
  word-break: break-word;
}

.log-info .log-message {
  color: #d4d4d4;
}

.log-success .log-message {
  color: #4ec9b0;
}

.log-warning .log-message {
  color: #dcdcaa;
}

.log-error .log-message {
  color: #f44747;
}

.btn {
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
  transition: all 0.2s;
}

.btn-secondary {
  background: #6c757d;
  color: white;
}

.btn-secondary:hover {
  background: #545b62;
}

.btn.active {
  background: #28a745;
  color: white;
}

.btn:not(.active):not(.btn-secondary) {
  background: #e9ecef;
  color: #495057;
}

.btn:not(.active):not(.btn-secondary):hover {
  background: #dee2e6;
}

/* 滚动条样式 */
.logs-container::-webkit-scrollbar {
  width: 8px;
}

.logs-container::-webkit-scrollbar-track {
  background: #2d2d30;
}

.logs-container::-webkit-scrollbar-thumb {
  background: #464647;
  border-radius: 4px;
}

.logs-container::-webkit-scrollbar-thumb:hover {
  background: #5a5a5c;
}

.highlight {
  background-color: #ffeb3b;
  color: #333;
  padding: 0 2px;
  border-radius: 2px;
}
</style>
