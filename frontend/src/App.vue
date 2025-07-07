<template>
  <div id="app">
    <!-- 导航栏 -->
    <nav class="navbar">
      <div class="nav-brand">
        <h1>HTTPTaskRunner</h1>
      </div>
      <div class="nav-tabs">
        <button 
          v-for="tab in tabs" 
          :key="tab.id"
          @click="activeTab = tab.id"
          :class="{ active: activeTab === tab.id }"
          class="nav-tab"
        >
          {{ tab.name }}
        </button>
      </div>
      <div class="nav-info">
        <span class="version">{{ versionInfo.name }} v{{ versionInfo.version }}</span>
        <span class="time">{{ currentTime }}</span>
      </div>
    </nav>

    <!-- 主内容区 -->
    <main class="main-content">
      <!-- 任务列表 -->
      <div v-if="activeTab === 'list'" class="tab-content">
        <TaskList
          ref="taskListRef"
          :tasks="tasks"
          :loading="loading"
          :scheduledTasks="scheduledTasks"
          @edit="editTask"
          @test="testTask"
          @execute="executeTask"
          @stop="stopTask"
          @logs="showTaskLogs"
          @schedule="scheduleTask"
          @unschedule="unscheduleTask"
          @delete="deleteTask"
          @create="createTask"
        />
      </div>

      <!-- 任务表单 -->
      <div v-if="activeTab === 'form'" class="tab-content">
        <TaskForm 
          :task="currentTask"
          @save="saveTask"
          @cancel="cancelEdit"
        />
      </div>

      <!-- 执行日志 -->
      <div v-if="activeTab === 'logs'" class="tab-content">
        <ExecutionLogs :logs="logs" @clear="clearLogs" />
      </div>

      <!-- 环境变量 -->
      <div v-if="activeTab === 'env'" class="tab-content">
        <EnvVariables />
      </div>
    </main>

    <!-- 状态栏 -->
    <footer class="status-bar">
      <span>任务总数: {{ taskCount }}</span>
      <span>运行中: {{ runningCount }}</span>
      <span v-if="message" class="message">{{ message }}</span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import TaskList from './components/TaskList.vue'
import TaskForm from './components/TaskForm.vue'
import ExecutionLogs from './components/ExecutionLogs.vue'
import EnvVariables from './components/EnvVariables.vue'
import { GetTasks, GetTaskCount, SaveTask, UpdateTask, DeleteTask, ExecuteTask, TestTaskWithBackend, StopTask, GetTaskLogs, ScheduleTask, UnscheduleTask, GetScheduledTasks, GetVersionInfo } from '../wailsjs/go/main/App'

// 响应式数据
const activeTab = ref('list')
const loading = ref(false)
const message = ref('')
const currentTime = ref('')
const tasks = ref<Record<string, any>>({})
const taskCount = ref(0)
const currentTask = ref(null)
const logs = ref<Array<{time: string, message: string, level?: 'info' | 'success' | 'warning' | 'error'}>>([])
const scheduledTasks = ref<string[]>([])
const taskListRef = ref<any>(null)
const versionInfo = ref({ name: 'HTTPTaskRunner', version: '1.0.0', buildDate: '2025-01-07' })

// 标签页配置
const tabs = [
  { id: 'list', name: '任务列表' },
  { id: 'form', name: '创建任务' },
  { id: 'logs', name: '执行日志' },
  { id: 'env', name: '环境变量' }
]

// 计算属性
const runningCount = computed(() => {
  return Object.values(tasks.value).filter((task: any) => task.isRunning).length
})

// 时间更新
let timeInterval: number | null = null

const updateTime = () => {
  const now = new Date()
  currentTime.value = now.toLocaleTimeString()
}

// 加载任务列表
const loadTasks = async () => {
  try {
    loading.value = true
    const result = await GetTasks(1, 50)
    tasks.value = result.tasks || {}
    taskCount.value = result.total || 0
    addLog('success', `成功加载 ${taskCount.value} 个任务`)
  } catch (error) {
    addLog('error', `加载任务失败: ${error}`)
    showMessage(`加载任务失败: ${error}`)
  } finally {
    loading.value = false
  }
}

// 创建新任务
const createTask = () => {
  currentTask.value = null
  activeTab.value = 'form'
}

// 保存任务
const saveTask = async (taskData: any) => {
  try {
    loading.value = true
    let result: string
    
    if (currentTask.value) {
      result = await UpdateTask(
        (currentTask.value as any).id,
        taskData.name,
        taskData.url,
        taskData.method,
        taskData.headersText,
        taskData.data,
        taskData.times,
        taskData.threads,
        taskData.delayMin,
        taskData.delayMax,
        taskData.tags,
        taskData.cronExpr,
        taskData.successCondition
      )
    } else {
      result = await SaveTask(
        taskData.name,
        taskData.url,
        taskData.method,
        taskData.headersText,
        taskData.data,
        taskData.times,
        taskData.threads,
        taskData.delayMin,
        taskData.delayMax,
        taskData.tags,
        taskData.cronExpr,
        taskData.successCondition
      )
    }
    
    addLog('success', result)
    showMessage(result)
    await loadTasks()
    activeTab.value = 'list'
    currentTask.value = null
  } catch (error) {
    const errorMsg = `保存失败: ${error}`
    addLog('error', errorMsg)
    showMessage(errorMsg)
  } finally {
    loading.value = false
  }
}

// 编辑任务
const editTask = (task: any) => {
  currentTask.value = task
  activeTab.value = 'form'
}

// 取消编辑
const cancelEdit = () => {
  currentTask.value = null
  activeTab.value = 'list'
}

// 测试任务
const testTask = async (taskId: string) => {
  try {
    const result = await TestTaskWithBackend(taskId)
    if (result.success) {
      const successMsg = `任务测试成功\n状态码: ${result.statusCode}\n响应时间: ${formatDuration(result.responseTime / 1000)}\n响应内容: ${result.responseBody ? result.responseBody.substring(0, 500) + (result.responseBody.length > 500 ? '...' : '') : '无'}`
      addLog('info', successMsg)
      showMessage(successMsg)
    } else {
      const errorMsg = `任务测试失败\n状态码: ${result.statusCode || '无'}\n错误: ${result.error || '未知错误'}\n响应内容: ${result.responseBody ? result.responseBody.substring(0, 500) + (result.responseBody.length > 500 ? '...' : '') : '无'}`
      addLog('error', errorMsg)
      showMessage(errorMsg, 'error')
    }
  } catch (error) {
    const errorMsg = `测试失败: ${error}`
    addLog('error', errorMsg)
    showMessage(errorMsg, 'error')
  }
}

// 执行任务
const executeTask = async (taskId: string) => {
  try {
    const result = await ExecuteTask(taskId)
    addLog('info', result)
    showMessage(result)

    // 立即刷新一次任务状态
    await loadTasks()

    // 启动定期刷新，直到任务完成
    const refreshInterval = setInterval(async () => {
      await loadTasks()

      // 检查任务是否还在运行
      const currentTasks = await GetTasks(1, 1000)
      const task = currentTasks.tasks[taskId]

      if (!task || !task.isRunning) {
        // 任务已完成，停止刷新
        clearInterval(refreshInterval)

        // 最后刷新一次日志
        setTimeout(async () => {
          if (taskListRef.value) {
            await taskListRef.value.refreshTaskLogs(taskId)
          }
        }, 500)
      }
    }, 1000) // 每秒刷新一次

    // 设置最大刷新时间（5分钟），防止无限刷新
    setTimeout(() => {
      clearInterval(refreshInterval)
    }, 5 * 60 * 1000)

  } catch (error) {
    const errorMsg = `执行失败: ${error}`
    addLog('error', errorMsg)
    showMessage(errorMsg)
  }
}

// 停止任务
const stopTask = async (taskId: string) => {
  try {
    const result = await StopTask(taskId)
    addLog('warning', result)
    showMessage(result)
    await loadTasks()
  } catch (error) {
    const errorMsg = `停止失败: ${error}`
    addLog('error', errorMsg)
    showMessage(errorMsg)
  }
}

// 查看任务日志
const showTaskLogs = async (taskId: string) => {
  try {
    const taskLogs = await GetTaskLogs(taskId)
    // 将任务日志添加到主日志中
    taskLogs.forEach(log => {
      logs.value.push({
        time: new Date().toISOString(),
        message: log,
        level: 'info'
      })
    })
    activeTab.value = 'logs'
    showMessage('任务日志已加载')
  } catch (error) {
    const errorMsg = `获取日志失败: ${error}`
    addLog('error', errorMsg)
    showMessage(errorMsg)
  }
}

// 添加定时任务
const scheduleTask = async (taskId: string) => {
  try {
    const result = await ScheduleTask(taskId)
    addLog('success', result)
    showMessage(result)
    await loadScheduledTasks()
  } catch (error) {
    const errorMsg = `添加定时失败: ${error}`
    addLog('error', errorMsg)
    showMessage(errorMsg)
  }
}

// 移除定时任务
const unscheduleTask = async (taskId: string) => {
  try {
    const result = await UnscheduleTask(taskId)
    addLog('warning', result)
    showMessage(result)
    await loadScheduledTasks()
  } catch (error) {
    const errorMsg = `移除定时失败: ${error}`
    addLog('error', errorMsg)
    showMessage(errorMsg)
  }
}

// 加载定时任务列表
const loadScheduledTasks = async () => {
  try {
    const scheduled = await GetScheduledTasks()
    scheduledTasks.value = scheduled
  } catch (error) {
    console.error('加载定时任务失败:', error)
  }
}

// 删除任务
const deleteTask = async (taskId: string) => {
  if (!confirm('确定要删除这个任务吗？')) {
    return
  }
  
  try {
    const result = await DeleteTask(taskId)
    addLog('success', result)
    showMessage(result)
    await loadTasks()
  } catch (error) {
    const errorMsg = `删除失败: ${error}`
    addLog('error', errorMsg)
    showMessage(errorMsg)
  }
}

// 添加日志
const addLog = (level: 'info' | 'success' | 'warning' | 'error', message: string) => {
  logs.value.push({
    time: new Date().toISOString(),
    message,
    level
  })

  if (logs.value.length > 1000) {
    logs.value = logs.value.slice(-500)
  }
}

// 清空日志
const clearLogs = () => {
  logs.value = []
}

// 显示消息
const showMessage = (msg: string, type: string = 'success') => {
  message.value = msg
  setTimeout(() => {
    message.value = ''
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

// 生命周期
// 加载版本信息
const loadVersionInfo = async () => {
  try {
    const version = await GetVersionInfo()
    versionInfo.value = version
  } catch (error) {
    console.error('获取版本信息失败:', error)
  }
}

onMounted(() => {
  updateTime()
  timeInterval = setInterval(updateTime, 1000)
  loadTasks()
  loadScheduledTasks()
  loadVersionInfo()
})

onUnmounted(() => {
  if (timeInterval) {
    clearInterval(timeInterval)
  }
})
</script>

<style scoped>
#app {
  height: 100vh;
  display: flex;
  flex-direction: column;
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
}

.navbar {
  display: flex;
  align-items: center;
  padding: 0 20px;
  background: #2c3e50;
  color: white;
  height: 60px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.nav-brand h1 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
}

.nav-tabs {
  display: flex;
  margin-left: 40px;
  gap: 10px;
}

.nav-tab {
  padding: 8px 16px;
  border: none;
  background: transparent;
  color: #bdc3c7;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.2s;
}

.nav-tab:hover {
  background: rgba(255,255,255,0.1);
  color: white;
}

.nav-tab.active {
  background: #3498db;
  color: white;
}

.nav-info {
  margin-left: auto;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.version {
  font-size: 0.85rem;
  color: #666;
  font-weight: 500;
}

.time {
  font-family: 'Courier New', monospace;
  font-size: 1.1rem;
}

.main-content {
  flex: 1;
  padding: 20px;
  overflow: auto;
  background: #f8f9fa;
}

.tab-content {
  max-width: 1200px;
  margin: 0 auto;
}

.status-bar {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 10px 20px;
  background: #34495e;
  color: white;
  font-size: 0.9rem;
}

.message {
  margin-left: auto;
  padding: 4px 12px;
  background: #27ae60;
  border-radius: 4px;
  font-weight: 500;
}
</style>
