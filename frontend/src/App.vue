<template>
  <div class="container">
    <div class="nav-bar">
      <div class="nav-tabs">
        <div 
          class="nav-tab" 
          :class="{ 'active': activeTab === 'requestConfig' }"
          @click="activeTab = 'requestConfig'"
        >
          请求配置
        </div>
        <div 
          class="nav-tab" 
          :class="{ 'active': activeTab === 'taskList' }"
          @click="activeTab = 'taskList'"
        >
          任务列表
          <span v-if="scheduledTasks.length > 0" class="badge">{{ scheduledTasks.length }}</span>
        </div>
      </div>
      <div class="current-time">{{ currentTimeDisplay }}</div>
    </div>
    
    <div class="content" v-if="activeTab === 'requestConfig'">
      <div class="left-panel">
        <div class="card">
          <div class="card-header">
            <h2>请求配置</h2>
            <button @click="showSaveTaskModal = true" class="btn">保存任务</button>
          </div>
          <div class="input-group">
            <label>网址</label>
            <input type="text" v-model="url" placeholder="请输入请求URL" />
          </div>
          <div class="input-group">
            <label>访问方式</label>
            <select v-model="visitMethod">
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="DELETE">DELETE</option>
            </select>
          </div>
          <div class="input-group">
            <label>CK</label>
            <input type="text" v-model="ck" placeholder="Cookie信息" />
          </div>
          <div class="input-group">
            <label>协议头</label>
            <textarea v-model="headers" placeholder="每行一个请求头，格式为 Key: Value" class="headers-input"></textarea>
          </div>
          <div class="input-group">
            <label>提交数据</label>
            <input type="text" v-model="submitData" placeholder="POST数据" />
          </div>
          <div class="input-group">
            <label>请求延迟</label>
            <div class="delay-inputs">
              <input 
                type="text" 
                v-model="delayMin" 
                placeholder="最小值" 
                class="small-input"
                @input="validateDelayInput"
              />
              <span>-</span>
              <input 
                type="text" 
                v-model="delayMax" 
                placeholder="最大值" 
                class="small-input"
                @input="validateDelayInput"
              />
              <span>毫秒</span>
            </div>
          </div>
          <div class="checkbox-group">
            <input type="checkbox" id="virtualIP" v-model="useVirtualIP" />
            <label for="virtualIP">使用虚拟IP (中国IP)</label>
          </div>
          <div class="notification-box" v-if="hasRemovedGzipHeader">
            <div class="notification-content">
              <i class="notification-icon">ℹ️</i>
              <span>已自动移除包含 gzip 的 Accept-Encoding 协议头</span>
            </div>
          </div>
        </div>

        <div class="card raw-data-card">
          <div class="card-header">
            <h2>Fiddler原始数据</h2>
            <button @click="parseRawData" class="btn primary">解析</button>
          </div>
          <div class="raw-data-container">
            <textarea v-model="rawData" placeholder="粘贴Fiddler捕获的原始数据" class="raw-data"></textarea>
          </div>
        </div>
      </div>

      <div class="right-panel">
        <div class="card">
          <div class="card-header">
            <h2>卡包设置</h2>
          </div>
          <div class="card-settings">
            <div class="settings-row">
              <div class="setting-item">
                <label>卡包值/线程数</label>
                <input type="text" v-model="cardPackage" class="small-input" />
              </div>
              <div class="setting-item">
                <label>请求次数</label>
                <input type="text" v-model="times" class="small-input" />
              </div>
            </div>
            
            <div class="settings-row">
              <div class="setting-item timer-setting">
                <label>定时执行 (设定执行时间)</label>
                <div class="time-input-group">
                  <input 
                    type="text" 
                    v-model="targetHours" 
                    class="time-input" 
                    placeholder="时" 
                    :disabled="timerActive"
                    @input="validateHours"
                    @focus="initTimeInput('hours')" 
                  />
                  <span>:</span>
                  <input 
                    type="text" 
                    v-model="targetMinutes" 
                    class="time-input" 
                    placeholder="分" 
                    :disabled="timerActive"
                    @input="validateMinutes"
                    @focus="initTimeInput('minutes')" 
                  />
                  <span>:</span>
                  <input 
                    type="text" 
                    v-model="targetSeconds" 
                    class="time-input" 
                    placeholder="秒" 
                    :disabled="timerActive"
                    @input="validateSeconds"
                    @focus="initTimeInput('seconds')" 
                  />
                </div>
              </div>
            </div>
          </div>
          
          <div class="button-group">
            <button @click="startCardPackage" class="btn primary" :disabled="executionInProgress">卡包</button>
            <button @click="toggleScheduledTimer" class="btn" :class="{ 'warning': timerActive }">
              {{ timerActive ? '取消定时' : '定时' }}
            </button>
          </div>

          <div class="timer-display" :class="{ 'active': timerRunning || timerActive }">
            <div v-if="timerActive" class="countdown">
              定时执行: {{ targetTimeDisplay }} 
              <span class="time-remaining">(剩余: {{ timeRemainingDisplay }})</span>
            </div>
            <div v-if="taskDuration > 0" class="time">任务耗时: {{ formattedTaskDuration }}</div>
          </div>
        </div>

        <div class="card output-card">
          <div class="card-header">
            <h2>执行日志</h2>
            <button @click="clearLog" class="btn">清空</button>
          </div>
          <div class="output-area">
            <textarea v-model="outputLog" readonly ref="outputLogRef"></textarea>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Task List View -->
    <div class="content" v-if="activeTab === 'taskList'">
      <div class="tasks-container">
        <div class="card">
          <div class="card-header">
            <h2>任务列表</h2>
            <div class="task-actions">
              <button @click="createNewTask" class="btn primary">创建任务</button>
              <button @click="showImportExportModal = true; importExportMode = 'export'" class="btn">导出全部</button>
              <button @click="showImportExportModal = true; importExportMode = 'exportSelected'" class="btn">导出选中</button>
              <button @click="showImportExportModal = true; importExportMode = 'exportTags'" class="btn">按标签导出</button>
              <button @click="showImportExportModal = true; importExportMode = 'import'" class="btn">导入</button>
            </div>
          </div>
          
          <div class="task-filters">
            <div class="search-box">
              <input type="text" v-model="taskFilter" placeholder="搜索任务名称或URL" />
            </div>
            <div class="tag-filters">
              <div 
                v-for="tag in availableTags" 
                :key="tag" 
                @click="toggleTagFilter(tag)"
                class="tag-filter"
                :class="{ 'active': selectedTags.includes(tag) }"
              >
                {{ tag }}
              </div>
            </div>
          </div>
          
          <div class="task-list">
            <div 
              v-for="task in filteredTasks" 
              :key="task.id" 
              class="task-item"
              :class="{ 
                'scheduled': scheduledTasks.includes(task.id),
                'running': task.isRunning,
                'cron-scheduled': hasCronSchedule(task)
              }"
            >
              <div class="task-selection" v-if="importExportMode === 'exportSelected'">
                <input 
                  type="checkbox" 
                  :id="'select-' + task.id" 
                  :checked="selectedTasksForExport.includes(task.id)"
                  @change="toggleTaskSelection(task.id)"
                />
              </div>
              <div class="task-info" @click="loadTaskToForm(task.id)">
                <div class="task-name">
                  {{ task.name }}
                  <span v-if="task.isRunning" class="task-status running">运行中</span>
                  <span v-else-if="hasCronSchedule(task)" class="task-status cron">
                    定时 ({{ formatCronExpression(task.cronExpression) }})
                  </span>
                </div>
                <div class="task-url">{{ task.url }}</div>
                <div class="task-meta">
                  <span>{{ task.method }}</span>
                  <span>线程: {{ task.threads }}</span>
                  <span>次数: {{ task.times }}</span>
                  <span>更新: {{ new Date(task.updatedAt).toLocaleString() }}</span>
                </div>
                <div v-if="task.isRunning" class="task-progress">
                  <div class="progress-bar">
                    <div 
                      class="progress-fill" 
                      :style="{ width: formatProgress(task.id) }"
                    ></div>
                  </div>
                  <div class="progress-text">
                    {{ runningTaskProgress[task.id]?.currentRequest || 0 }}/{{ task.times }}
                    <span class="progress-time">{{ formatElapsedTime(task.id) }}</span>
                  </div>
                </div>
                <div v-else-if="hasCronSchedule(task)" class="task-next-execution">
                  下次执行: {{ getNextCronExecutionForTask(task) }}
                </div>
                <div class="task-tags">
                  <span v-for="tag in task.tags" :key="tag" class="tag">{{ tag }}</span>
                </div>
              </div>
              <div class="task-actions">
                <button 
                  v-if="!task.isRunning && !scheduledTasks.includes(task.id)" 
                  @click="scheduleTask(task.id)" 
                  class="btn"
                  title="添加到定时队列"
                >
                  定时
                </button>
                <button 
                  v-else-if="scheduledTasks.includes(task.id)" 
                  @click="unscheduleTask(task.id)" 
                  class="btn warning"
                  title="从定时队列移除"
                >
                  取消定时
                </button>
                <button 
                  v-if="task.isRunning" 
                  @click="stopTask(task.id)" 
                  class="btn danger"
                  title="停止任务"
                >
                  停止
                </button>
                <button 
                  v-else
                  @click="executeTask(task.id)" 
                  class="btn primary"
                  title="立即执行"
                  :disabled="task.isRunning"
                >
                  执行
                </button>
                <button 
                  @click.stop="testTask(task.id)" 
                  class="btn"
                  title="测试单次请求"
                >
                  测试
                </button>
                <button 
                  @click.stop="showTaskLogs(task.id)" 
                  class="btn"
                  title="查看执行日志"
                >
                  日志
                </button>
              </div>
            </div>
            
            <div v-if="filteredTasks.length === 0" class="no-tasks">
              没有找到符合条件的任务
            </div>
          </div>
        </div>
        
        <div class="card output-card">
          <div class="card-header">
            <h2>执行日志</h2>
            <div class="output-actions">
              <div class="auto-scroll-toggle">
                <input type="checkbox" id="autoScroll" v-model="autoScrollLogs" />
                <label for="autoScroll">自动滚动</label>
              </div>
              <button @click="clearLog" class="btn">清空</button>
            </div>
          </div>
          <div class="output-area">
            <textarea v-model="outputLog" readonly ref="outputLogRef"></textarea>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Task Detail View -->
    <div class="content" v-if="activeTab === 'taskDetail'">
      <div class="left-panel">
        <div class="card">
          <div class="card-header">
            <h2>{{ selectedTaskId ? '编辑任务' : '创建任务' }}</h2>
            <div class="task-actions">
              <button 
                v-if="selectedTaskId" 
                @click="testTask(selectedTaskId)"
                class="btn"
                title="测试任务"
              >
                测试
              </button>
              <button 
                v-if="selectedTaskId" 
                @click="showConfirmDeleteModal = true" 
                class="btn danger"
              >
                删除
              </button>
            </div>
          </div>
          <div class="input-group">
            <label>任务名称</label>
            <input type="text" v-model="taskName" placeholder="请输入任务名称" />
          </div>
          <div class="input-group">
            <label>标签</label>
            <input type="text" v-model="taskTags" placeholder="多个标签用英文逗号分隔" />
          </div>
          <div class="input-group">
            <label>网址</label>
            <input type="text" v-model="url" placeholder="请输入请求URL" />
          </div>
          <div class="input-group">
            <label>访问方式</label>
            <select v-model="visitMethod">
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="DELETE">DELETE</option>
            </select>
          </div>
          <div class="input-group">
            <label>CK</label>
            <input type="text" v-model="ck" placeholder="Cookie信息" />
          </div>
          <div class="input-group">
            <label>协议头</label>
            <textarea v-model="headers" placeholder="每行一个请求头，格式为 Key: Value" class="headers-input"></textarea>
          </div>
          <div class="input-group">
            <label>提交数据</label>
            <input type="text" v-model="submitData" placeholder="POST数据" />
          </div>
          <div class="input-group">
            <label>请求延迟</label>
            <div class="delay-inputs">
              <input 
                type="text" 
                v-model="delayMin" 
                placeholder="最小值" 
                class="small-input"
                @input="validateDelayInput"
              />
              <span>-</span>
              <input 
                type="text" 
                v-model="delayMax" 
                placeholder="最大值" 
                class="small-input"
                @input="validateDelayInput"
              />
              <span>毫秒</span>
            </div>
          </div>
          <div class="checkbox-group">
            <input type="checkbox" id="taskVirtualIP" v-model="useVirtualIP" />
            <label for="taskVirtualIP">使用虚拟IP (中国IP)</label>
          </div>
        </div>
      </div>

      <div class="right-panel">
        <div class="card">
          <div class="card-header">
            <h2>任务设置</h2>
          </div>
          <div class="card-settings">
            <div class="settings-row">
              <div class="setting-item">
                <label>卡包值/线程数</label>
                <input type="text" v-model="cardPackage" class="small-input" />
              </div>
              <div class="setting-item">
                <label>请求次数</label>
                <input type="text" v-model="times" class="small-input" />
              </div>
            </div>
            
            <div class="settings-row">
              <div class="setting-item timer-setting">
                <label>快捷定时设置</label>
                <div class="cron-builder">
                  <div class="cron-builder-row">
                    <select v-model="cronBuilderType" @change="updateCronBuilder">
                      <option value="specific">指定时间</option>
                      <option value="every_second">每秒</option>
                      <option value="every_minute">每分钟</option>
                      <option value="every_hour">每小时</option>
                      <option value="every_day">每天</option>
                      <option value="workday">工作日</option>
                      <option value="weekend">周末</option>
                      <option value="range">时间范围</option>
                    </select>
                    
                    <!-- 每秒选项 -->
                    <div v-if="cronBuilderType === 'every_second'" class="cron-builder-inputs">
                      <div class="cron-builder-item">
                        <label>每隔</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderSecond" 
                          class="small-input"
                        />
                        <label>秒</label>
                      </div>
                    </div>
                    
                    <!-- 每分钟选项 -->
                    <div v-if="cronBuilderType === 'every_minute'" class="cron-builder-inputs">
                      <div class="cron-builder-item">
                        <label>每隔</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderMinute" 
                          class="small-input"
                        />
                        <label>分钟</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>第</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderSecond" 
                          class="small-input"
                        />
                        <label>秒</label>
                      </div>
                    </div>
                    
                    <!-- 每小时选项 -->
                    <div v-if="cronBuilderType === 'every_hour'" class="cron-builder-inputs">
                      <div class="cron-builder-item">
                        <label>每隔</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderHour" 
                          class="small-input"
                        />
                        <label>小时</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>第</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderMinute" 
                          class="small-input"
                        />
                        <label>分</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>第</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderSecond" 
                          class="small-input"
                        />
                        <label>秒</label>
                      </div>
                    </div>
                    
                    <!-- 每天选项 -->
                    <div v-if="cronBuilderType === 'every_day'" class="cron-builder-inputs">
                      <div class="cron-builder-item">
                        <label>每天</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderHour" 
                          class="small-input"
                        />
                        <label>点</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>第</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderMinute" 
                          class="small-input"
                        />
                        <label>分</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>第</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderSecond" 
                          class="small-input"
                        />
                        <label>秒</label>
                      </div>
                    </div>
                    
                    <!-- 工作日选项 -->
                    <div v-if="cronBuilderType === 'workday'" class="cron-builder-inputs">
                      <div class="cron-builder-item">
                        <label>每个工作日</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderHour" 
                          class="small-input"
                        />
                        <label>点</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>第</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderMinute" 
                          class="small-input"
                        />
                        <label>分</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>第</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderSecond" 
                          class="small-input"
                        />
                        <label>秒</label>
                      </div>
                    </div>
                    
                    <!-- 周末选项 -->
                    <div v-if="cronBuilderType === 'weekend'" class="cron-builder-inputs">
                      <div class="cron-builder-item">
                        <label>每个周末</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderHour" 
                          class="small-input"
                        />
                        <label>点</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>第</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderMinute" 
                          class="small-input"
                        />
                        <label>分</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>第</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderSecond" 
                          class="small-input"
                        />
                        <label>秒</label>
                      </div>
                    </div>
                    
                    <!-- 时间范围选项 -->
                    <div v-if="cronBuilderType === 'range'" class="cron-builder-inputs">
                      <div class="cron-builder-item">
                        <label>从</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderStartHour" 
                          class="small-input"
                        />
                        <label>点 到</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderEndHour" 
                          class="small-input"
                        />
                        <label>点</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>每隔</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderRangeInterval" 
                          class="small-input"
                        />
                        <label>小时</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>第</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderMinute" 
                          class="small-input"
                        />
                        <label>分</label>
                      </div>
                      <div class="cron-builder-item">
                        <label>第</label>
                        <input 
                          type="text" 
                          v-model="cronBuilderSecond" 
                          class="small-input"
                        />
                        <label>秒</label>
                      </div>
                    </div>
                    
                    <!-- 指定时间选项 -->
                    <div v-if="cronBuilderType === 'specific'" class="cron-builder-inputs">
                      <div class="time-input-group">
                        <input 
                          type="text" 
                          v-model="targetHours" 
                          class="time-input" 
                          placeholder="时" 
                          @input="validateHours()"
                          @focus="initTimeInput('hours')" 
                        />
                        <span>:</span>
                        <input 
                          type="text" 
                          v-model="targetMinutes" 
                          class="time-input" 
                          placeholder="分" 
                          @input="validateMinutes()"
                          @focus="initTimeInput('minutes')" 
                        />
                        <span>:</span>
                        <input 
                          type="text" 
                          v-model="targetSeconds" 
                          class="time-input" 
                          placeholder="秒" 
                          @input="validateSeconds()"
                          @focus="initTimeInput('seconds')" 
                        />
                      </div>
                    </div>
                    
                    <!-- 设置按钮 -->
                    <div class="cron-builder-actions">
                      <button @click="buildCronExpression" class="btn primary">设置</button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            
            <div class="settings-row">
              <div class="setting-item timer-setting">
                <label>Cron表达式 (秒级定时规则)</label>
                <div class="cron-input-group">
                  <input 
                    type="text" 
                    v-model="cronExpression" 
                    class="cron-input" 
                    placeholder="例如: */5 * * * * *" 
                    :class="{ 'invalid': !cronIsValid && cronExpression }"
                  />
                  <div class="cron-help">
                    <span class="tooltip" @click.stop="showCronHelpModal = true">?
                      <span class="tooltip-text">
                        点击查看Cron表达式完整帮助
                      </span>
                    </span>
                  </div>
                </div>
                <div v-if="!cronIsValid && cronExpression" class="cron-validation-error">
                  {{ cronValidationMessage }}
                </div>
                <div v-else-if="cronExpression && nextCronExecution" class="cron-next-execution">
                  下次执行时间: {{ formatNextExecutionDate(nextCronExecution) }}
                </div>
              </div>
            </div>
          </div>
          
          <div class="button-group">
            <button 
              @click="selectedTaskId ? updateTask() : saveCurrentAsTask()" 
              class="btn primary"
            >
              {{ selectedTaskId ? '更新任务' : '保存任务' }}
            </button>
            <button @click="activeTab = 'taskList'" class="btn">取消</button>
          </div>
        </div>
        
        <!-- Add task detail output log -->
        <div class="card output-card">
          <div class="card-header">
            <h2>执行日志</h2>
            <div class="output-actions">
              <div class="auto-scroll-toggle">
                <input type="checkbox" id="detailAutoScroll" v-model="autoScrollLogs" />
                <label for="detailAutoScroll">自动滚动</label>
              </div>
              <button @click="clearLog" class="btn">清空</button>
            </div>
          </div>
          <div class="output-area">
            <textarea v-model="outputLog" readonly ref="detailOutputLogRef"></textarea>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Save Task Modal -->
    <div class="modal" v-if="showSaveTaskModal">
      <div class="modal-content">
        <div class="modal-header">
          <h2>保存任务</h2>
          <button @click="showSaveTaskModal = false" class="close-btn">&times;</button>
        </div>
        <div class="modal-body">
          <div class="input-group">
            <label>任务名称</label>
            <input type="text" v-model="taskName" placeholder="请输入任务名称" />
          </div>
          <div class="input-group">
            <label>标签</label>
            <input type="text" v-model="taskTags" placeholder="多个标签用英文逗号分隔" />
          </div>
        </div>
        <div class="modal-footer">
          <button @click="saveCurrentAsTask" class="btn primary">保存</button>
          <button @click="showSaveTaskModal = false" class="btn">取消</button>
        </div>
      </div>
    </div>
    
    <!-- Delete Confirmation Modal -->
    <div class="modal" v-if="showConfirmDeleteModal">
      <div class="modal-content">
        <div class="modal-header">
          <h2>确认删除</h2>
          <button @click="showConfirmDeleteModal = false" class="close-btn">&times;</button>
        </div>
        <div class="modal-body">
          <p>确定要删除任务 "{{ currentTask?.name }}" 吗？此操作不可撤销。</p>
        </div>
        <div class="modal-footer">
          <button @click="deleteTask" class="btn danger">删除</button>
          <button @click="showConfirmDeleteModal = false" class="btn">取消</button>
        </div>
      </div>
    </div>
    
    <!-- Import/Export Modal -->
    <div class="modal" v-if="showImportExportModal">
      <div class="modal-content">
        <div class="modal-header">
          <h2>
            {{ 
              importExportMode === 'export' ? '导出全部任务' : 
              importExportMode === 'exportSelected' ? '导出选中任务' :
              importExportMode === 'exportTags' ? '按标签导出任务' : '导入任务' 
            }}
          </h2>
          <button @click="showImportExportModal = false" class="close-btn">&times;</button>
        </div>
        <div class="modal-body">
          <!-- Tag selection for export by tags -->
          <div v-if="importExportMode === 'exportTags'" class="tag-selection">
            <h3>选择要导出的标签:</h3>
            <div class="export-tags">
              <div 
                v-for="tag in availableTags" 
                :key="tag" 
                @click="toggleTagSelectionForExport(tag)"
                class="tag-filter"
                :class="{ 'active': selectedTagsForExport.includes(tag) }"
              >
                {{ tag }}
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button 
            @click="importExportMode.includes('export') ? exportFilteredTasks() : importTasksFromFile()" 
            class="btn primary"
          >
            {{ importExportMode.includes('export') ? '导出' : '导入' }}
          </button>
          <button @click="showImportExportModal = false" class="btn">取消</button>
        </div>
      </div>
    </div>
    
    <!-- Cron Help Modal -->
    <div class="modal" v-if="showCronHelpModal">
      <div class="modal-content modal-lg">
        <div class="modal-header">
          <h2>Cron 表达式帮助</h2>
          <button @click="showCronHelpModal = false" class="close-btn">&times;</button>
        </div>
        <div class="modal-body">
          <div class="cron-help-content">
            <h3>基本语法</h3>
            <p>Cron 表达式格式: <code>秒 分 时 日 月 星期</code></p>
            
            <div class="cron-example">
              <h4>常见例子:</h4>
              <ul>
                <li><code>*/5 * * * * *</code> - 每5秒执行一次</li>
                <li><code>0 */1 * * * *</code> - 每分钟执行一次</li>
                <li><code>0 0 9 * * *</code> - 每天早上9点执行</li>
                <li><code>0 0 9-17 * * *</code> - 每天早上9点到下午5点，每小时执行一次</li>
                <li><code>0 0 9-17 * * 1-5</code> - 每周一到周五，早上9点到下午5点，每小时执行一次</li>
                <li><code>0 0 0 1 * *</code> - 每月1号午夜执行</li>
              </ul>
            </div>
            
            <h3>特殊字符</h3>
            <table class="cron-table">
              <tr>
                <th>字符</th>
                <th>描述</th>
                <th>示例</th>
              </tr>
              <tr>
                <td>*</td>
                <td>匹配该字段的所有值</td>
                <td><code>* * * * * *</code> - 每秒执行</td>
              </tr>
              <tr>
                <td>,</td>
                <td>指定多个值</td>
                <td><code>0 0 9,12,15 * * *</code> - 每天9点、12点和15点执行</td>
              </tr>
              <tr>
                <td>-</td>
                <td>指定一个范围</td>
                <td><code>0 0 9-17 * * *</code> - 每天9点到17点每小时执行</td>
              </tr>
              <tr>
                <td>/</td>
                <td>指定步长</td>
                <td><code>0 */30 * * * *</code> - 每30分钟执行</td>
              </tr>
            </table>
            
            <h3>预定义表达式</h3>
            <table class="cron-table">
              <tr>
                <th>表达式</th>
                <th>描述</th>
                <th>等效于</th>
              </tr>
              <tr>
                <td>@yearly</td>
                <td>每年执行一次</td>
                <td><code>0 0 0 1 1 *</code></td>
              </tr>
              <tr>
                <td>@monthly</td>
                <td>每月执行一次</td>
                <td><code>0 0 0 1 * *</code></td>
              </tr>
              <tr>
                <td>@weekly</td>
                <td>每周执行一次</td>
                <td><code>0 0 0 * * 0</code></td>
              </tr>
              <tr>
                <td>@daily</td>
                <td>每天执行一次</td>
                <td><code>0 0 0 * * *</code></td>
              </tr>
              <tr>
                <td>@hourly</td>
                <td>每小时执行一次</td>
                <td><code>0 0 * * * *</code></td>
              </tr>
              <tr>
                <td>@every 1h30m</td>
                <td>每隔1小时30分钟执行</td>
                <td>-</td>
              </tr>
            </table>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="showCronHelpModal = false" class="btn">关闭</button>
        </div>
      </div>
    </div>
    
    <!-- Task Logs Modal -->
    <div class="modal" v-if="showTaskLogsModal">
      <div class="modal-content modal-lg">
        <div class="modal-header">
          <h2>任务执行日志 - {{ currentTaskLogId ? tasks[currentTaskLogId]?.name || currentTaskLogId : '' }}</h2>
          <button @click="showTaskLogsModal = false" class="close-btn">&times;</button>
        </div>
        <div class="modal-body">
          <div class="task-logs-actions">
            <div class="task-logs-retention">
              <label>保留日志天数:</label>
              <input type="text" v-model="logRetentionDays" class="small-input" />
              <button @click="deleteOldLogs" class="btn">删除旧日志</button>
            </div>
            <div class="task-logs-path" v-if="taskLogsPath">
              <span>日志保存路径: {{ taskLogsPath }}</span>
            </div>
            <div class="task-logs-clear">
              <button @click="currentTaskLogId && clearTaskLogs(currentTaskLogId)" class="btn warning">清空当前任务日志</button>
              <button @click="clearAllTaskLogs" class="btn danger">清空所有日志</button>
            </div>
          </div>
          
          <div class="task-logs-content">
            <div v-if="currentTaskLogId && taskLogs[currentTaskLogId] && taskLogs[currentTaskLogId].length > 0">
              <div v-for="(log, index) in taskLogs[currentTaskLogId]" :key="index" class="task-log-entry">
                {{ log }}
              </div>
            </div>
            <div v-else class="no-logs">
              没有可用的日志记录
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="showTaskLogsModal = false" class="btn">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, reactive, watch, nextTick } from 'vue';
import { 
  SendRequest, ExecuteCardPackage, SaveTask, GetAllTasks, GetTaskByID, UpdateTask, 
  DeleteTask, ExportTasks, ImportTasks, StopTask, TestTask, ExecuteTask, GetTaskProgress,
  ExportTasksByIDs, ExportTasksByTags, OpenSaveFileDialog, OpenFileDialog, 
  SaveTaskLogs, LoadTaskLogs, ClearTaskLogs, DeleteOldTaskLogs, EnsureLogDirectory
} from '../wailsjs/go/main/App';
import { debounce } from 'lodash-es';

// Types
interface Task {
  id: string;
  name: string;
  url: string;
  method: string;
  cookie: string;
  headers: string;
  data: string;
  useVirtualIP: boolean;
  times: number;
  threads: number;
  scheduledTime: string;
  cronExpression: string;
  delayMin: number;
  delayMax: number;
  tags: string[];
  createdAt: string;
  updatedAt: string;
  isRunning: boolean;
}

interface TaskProgress {
  currentRequest: number;
  totalRequests: number;
  startTime: string;
  elapsedTime: number;
  delayInfo: number[];
}

// Application state
const activeTab = ref('requestConfig'); // 'requestConfig', 'taskList', 'taskDetail'
const tasks = ref<Record<string, Task>>({});
const selectedTaskId = ref<string | null>(null);
const showSaveTaskModal = ref(false);
const showConfirmDeleteModal = ref(false);
const showImportExportModal = ref(false);
const showCronHelpModal = ref(false);
const importExportMode = ref<'import' | 'export' | 'exportSelected' | 'exportTags'>('export');
const importExportPath = ref('');
const isFileBrowserOpen = ref(false);
const autoScrollLogs = ref(true);
const outputLogRef = ref<HTMLTextAreaElement | null>(null);
const selectedTasksForExport = ref<string[]>([]);
const selectedTagsForExport = ref<string[]>([]);
const runningTaskProgress = ref<Record<string, TaskProgress>>({});
const progressUpdateInterval = ref<number | null>(null);

// State variables for request configuration
const url = ref('');
const visitMethod = ref('GET');
const ck = ref('');
const headers = ref('');
const useVirtualIP = ref(true);
const submitData = ref('');
const rawData = ref('');
const cardPackage = ref('5');
const times = ref('10');
const outputLog = ref('');
const executionInProgress = ref(false);
const currentTime = ref(new Date());
const timerTick = ref(0);
const taskName = ref('');
const taskTags = ref('');
const delayMin = ref('0');
const delayMax = ref('0');
const hasRemovedGzipHeader = ref(false);
const cronExpression = ref('');

// Add new state variables for cron validation
const cronIsValid = ref(true);
const cronValidationMessage = ref('');
const nextCronExecution = ref<Date | null>(null);
const cronExpressionChanged = ref(false);

// Add new state variables for cron builder
const cronBuilderType = ref('specific');
const cronBuilderSecond = ref('0');
const cronBuilderMinute = ref('0');
const cronBuilderHour = ref('0');
const cronBuilderStartHour = ref('9');
const cronBuilderEndHour = ref('17');
const cronBuilderRangeInterval = ref('1');
const detailOutputLogRef = ref<HTMLTextAreaElement | null>(null);

// Add new state variables for task logs
const taskLogs = ref<Record<string, string[]>>({});
const showTaskLogsModal = ref(false);
const currentTaskLogId = ref<string | null>(null);
const logRetentionDays = ref('7');
const taskLogsPath = ref('');

// Auto-scroll logs when enabled
watch(outputLog, () => {
  if (autoScrollLogs.value) {
    nextTick(() => {
      if (outputLogRef.value) {
        // Ensure scroll happens after content is rendered
        outputLogRef.value.scrollTop = outputLogRef.value.scrollHeight;
      }
      if (detailOutputLogRef.value && activeTab.value === 'taskDetail') {
        detailOutputLogRef.value.scrollTop = detailOutputLogRef.value.scrollHeight;
      }
    });
  }
}, { flush: 'post' }); // Use post flush timing to ensure DOM is updated

// Add a watcher for activeTab changes to maintain log scroll position
watch(activeTab, (newTab, oldTab) => {
  // When switching between tabs, preserve log scroll position
  nextTick(() => {
    if (newTab === 'taskList' && outputLogRef.value) {
      // When switching to task list, restore scroll position
      outputLogRef.value.scrollTop = outputLogRef.value.scrollHeight;
    } else if (newTab === 'taskDetail' && detailOutputLogRef.value && outputLogRef.value) {
      // When switching to task detail, copy scroll position from main log
      detailOutputLogRef.value.scrollTop = detailOutputLogRef.value.scrollHeight;
    } else if (newTab === 'requestConfig' && outputLogRef.value) {
      // When switching to request config, restore scroll position
      outputLogRef.value.scrollTop = outputLogRef.value.scrollHeight;
    }
  });
});

// Task filter
const taskFilter = ref('');
const selectedTags = ref<string[]>([]);
const availableTags = computed(() => {
  const tagsSet = new Set<string>();
  Object.values(tasks.value).forEach(task => {
    task.tags.forEach(tag => tagsSet.add(tag));
  });
  return Array.from(tagsSet).sort();
});

const filteredTasks = computed(() => {
  const filter = taskFilter.value.toLowerCase();
  const tasksArray = Object.values(tasks.value);
  
  return tasksArray.filter(task => {
    // Filter by search text
    const matchesFilter = 
      task.name.toLowerCase().includes(filter) ||
      task.url.toLowerCase().includes(filter);
    
    // Filter by selected tags
    const matchesTags = selectedTags.value.length === 0 || 
      selectedTags.value.some(tag => task.tags.includes(tag));
    
    return matchesFilter && matchesTags;
  }).sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime());
});

// Current task in edit mode
const currentTask = computed(() => {
  if (!selectedTaskId.value || !tasks.value[selectedTaskId.value]) {
    return null;
  }
  return tasks.value[selectedTaskId.value];
});

// Multiple scheduled tasks
const scheduledTasks = ref<string[]>([]);

// Timer functionality
const timerRunning = ref(false);
const timerActive = ref(false);
const taskStartTime = ref<Date | null>(null);
const taskEndTime = ref<Date | null>(null);
const taskDuration = ref(0);
const taskDurationMs = ref(0);

// Time input for scheduled timer (target time)
const targetHours = ref('');
const targetMinutes = ref('');
const targetSeconds = ref('');

// Current time display with milliseconds
const currentTimeDisplay = computed(() => {
  const h = currentTime.value.getHours().toString().padStart(2, '0');
  const m = currentTime.value.getMinutes().toString().padStart(2, '0');
  const s = currentTime.value.getSeconds().toString().padStart(2, '0');
  return `${h}:${m}:${s}`;
});

// Format task duration with milliseconds
const formattedTaskDuration = computed(() => {
  // For short durations, just show milliseconds
  if (taskDurationMs.value < 1000) {
    return `${taskDurationMs.value}ms`;
  }
  
  // For longer durations, format as needed
  const totalSeconds = Math.floor(taskDurationMs.value / 1000);
  const ms = taskDurationMs.value % 1000;
  
  if (totalSeconds < 60) {
    // Less than a minute: show as seconds.milliseconds
    return `${totalSeconds}.${ms.toString().padStart(3, '0')}s`;
  } else if (totalSeconds < 3600) {
    // Less than an hour: show as minutes:seconds.milliseconds
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}:${seconds.toString().padStart(2, '0')}.${ms.toString().padStart(3, '0')}`;
  } else {
    // More than an hour: show full format
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;
    return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}.${ms.toString().padStart(3, '0')}`;
  }
});

// Target time for scheduled execution
const targetTimeDisplay = computed(() => {
  const h = targetHours.value.padStart(2, '0');
  const m = targetMinutes.value.padStart(2, '0');
  const s = targetSeconds.value.padStart(2, '0');
  return `${h}:${m}:${s}`;
});

// Calculate time remaining until target time
const timeRemainingDisplay = computed(() => {
  if (!timerActive.value) return "00:00:00";
  
  // Include timerTick in the dependency to force updates
  timerTick.value;
  
  const now = new Date();
  const targetTime = new Date();
  
  const h = parseInt(targetHours.value) || 0;
  const m = parseInt(targetMinutes.value) || 0;
  const s = parseInt(targetSeconds.value) || 0;
  
  targetTime.setHours(h, m, s, 0);
  
  // If target time is earlier than current time, set it to tomorrow
  if (targetTime < now) {
    targetTime.setDate(targetTime.getDate() + 1);
  }
  
  const diffMs = targetTime.getTime() - now.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  
  const hours = Math.floor(diffSec / 3600).toString().padStart(2, '0');
  const minutes = Math.floor((diffSec % 3600) / 60).toString().padStart(2, '0');
  const seconds = (diffSec % 60).toString().padStart(2, '0');
  
  return `${hours}:${minutes}:${seconds}`;
});

let clockInterval: number | null = null;

// Update current time every 100ms for more accurate timing
onMounted(() => {
  // Initialize time inputs with current time
  const now = new Date();
  targetHours.value = now.getHours().toString().padStart(2, '0');
  targetMinutes.value = now.getMinutes().toString().padStart(2, '0');
  targetSeconds.value = now.getSeconds().toString().padStart(2, '0');
  
  clockInterval = setInterval(() => {
    currentTime.value = new Date();
    timerTick.value++; // Increment the timer tick to trigger recomputation
    checkScheduledTasks();
  }, 100) as unknown as number;
  
  // Load all saved tasks
  loadTasks();
  
  // Ensure log directory exists and load task logs
  EnsureLogDirectory().then(() => {
    loadTaskLogsFromDisk();
  });
  
  // Start polling for task progress updates
  progressUpdateInterval.value = setInterval(updateTaskProgress, 500) as unknown as number;
});

// Validate and initialize time input fields
function initTimeInput(field: 'hours' | 'minutes' | 'seconds') {
  if (timerActive.value) return;
  
  // Set default value to current time if empty
  const now = new Date();
  if (field === 'hours' && targetHours.value === '') {
    targetHours.value = now.getHours().toString().padStart(2, '0');
  } else if (field === 'minutes' && targetMinutes.value === '') {
    targetMinutes.value = now.getMinutes().toString().padStart(2, '0');
  } else if (field === 'seconds' && targetSeconds.value === '') {
    targetSeconds.value = now.getSeconds().toString().padStart(2, '0');
  }
}

function validateHours() {
  if (timerActive.value) return;
  
  // Remove non-digit characters
  targetHours.value = targetHours.value.replace(/\D/g, '');
  
  // Ensure value is a valid hour (0-23)
  const hours = parseInt(targetHours.value);
  if (isNaN(hours)) {
    targetHours.value = '00';
  } else if (hours > 23) {
    targetHours.value = '23';
  } else {
    targetHours.value = hours.toString().padStart(2, '0');
  }
}

function validateMinutes() {
  if (timerActive.value) return;
  
  // Remove non-digit characters
  targetMinutes.value = targetMinutes.value.replace(/\D/g, '');
  
  // Ensure value is a valid minute (0-59)
  const minutes = parseInt(targetMinutes.value);
  if (isNaN(minutes)) {
    targetMinutes.value = '00';
  } else if (minutes > 59) {
    targetMinutes.value = '59';
  } else {
    targetMinutes.value = minutes.toString().padStart(2, '0');
  }
}

function validateSeconds() {
  if (timerActive.value) return;
  
  // Remove non-digit characters
  targetSeconds.value = targetSeconds.value.replace(/\D/g, '');
  
  // Ensure value is a valid second (0-59)
  const seconds = parseInt(targetSeconds.value);
  if (isNaN(seconds)) {
    targetSeconds.value = '00';
  } else if (seconds > 59) {
    targetSeconds.value = '59';
  } else {
    targetSeconds.value = seconds.toString().padStart(2, '0');
  }
}

// Validate delay input fields
function validateDelayInput() {
  // Remove non-digit characters
  delayMin.value = delayMin.value.replace(/\D/g, '');
  delayMax.value = delayMax.value.replace(/\D/g, '');
  
  // Ensure values are valid
  const min = parseInt(delayMin.value);
  const max = parseInt(delayMax.value);
  
  if (isNaN(min)) {
    delayMin.value = '0';
  }
  
  if (isNaN(max)) {
    delayMax.value = '0';
  } else if (max < min) {
    delayMax.value = delayMin.value;
  }
}

// Load all tasks from the backend
async function loadTasks() {
  try {
    const allTasks = await GetAllTasks();
    tasks.value = allTasks;
  } catch (error) {
    outputLog.value += getCurrentTime() + ` 加载任务失败: ${error}\n`;
  }
}

// Save current configuration as a task
async function saveCurrentAsTask() {
  if (!url.value) {
    outputLog.value += getCurrentTime() + " 错误: 保存任务需要 URL\n";
    return;
  }
  
  if (!taskName.value) {
    outputLog.value += getCurrentTime() + " 错误: 请输入任务名称\n";
    return;
  }
  
  // Parse tags
  const tags = taskTags.value.split(',')
    .map(tag => tag.trim())
    .filter(tag => tag !== '');
  
  // Create scheduled time string
  const scheduledTime = `${targetHours.value}:${targetMinutes.value}:${targetSeconds.value}`;
  
  try {
    const result = await SaveTask(
      taskName.value,
      url.value,
      visitMethod.value,
      ck.value,
      submitData.value,
      headers.value,
      useVirtualIP.value,
      parseInt(times.value),
      parseInt(cardPackage.value),
      scheduledTime,
      cronExpression.value,
      parseInt(delayMin.value),
      parseInt(delayMax.value),
      tags
    );
    
    outputLog.value += getCurrentTime() + ` ${result}\n`;
    showSaveTaskModal.value = false;
    
    // Reload tasks
    await loadTasks();
    
    // Switch to task list view
    activeTab.value = 'taskList';
  } catch (error) {
    outputLog.value += getCurrentTime() + ` 保存任务失败: ${error}\n`;
  }
}

// Update an existing task
async function updateTask() {
  if (!selectedTaskId.value) return;
  
  // Parse tags
  const tags = taskTags.value.split(',')
    .map(tag => tag.trim())
    .filter(tag => tag !== '');
  
  // Create scheduled time string
  const scheduledTime = `${targetHours.value}:${targetMinutes.value}:${targetSeconds.value}`;
  
  try {
    const result = await UpdateTask(
      selectedTaskId.value,
      taskName.value,
      url.value,
      visitMethod.value,
      ck.value,
      submitData.value,
      headers.value,
      useVirtualIP.value,
      parseInt(times.value),
      parseInt(cardPackage.value),
      scheduledTime,
      cronExpression.value,
      parseInt(delayMin.value),
      parseInt(delayMax.value),
      tags
    );
    
    outputLog.value += getCurrentTime() + ` ${result}\n`;
    
    // Reload tasks
    await loadTasks();
    
    // Switch to task list view
    activeTab.value = 'taskList';
  } catch (error) {
    outputLog.value += getCurrentTime() + ` 更新任务失败: ${error}\n`;
  }
}

// Delete a task
async function deleteTask() {
  if (!selectedTaskId.value) return;
  
  try {
    const result = await DeleteTask(selectedTaskId.value);
    outputLog.value += getCurrentTime() + ` ${result}\n`;
    
    // Remove from scheduled tasks if it's there
    const index = scheduledTasks.value.indexOf(selectedTaskId.value);
    if (index !== -1) {
      scheduledTasks.value.splice(index, 1);
    }
    
    // Reload tasks
    await loadTasks();
    
    // Close confirm modal and switch to task list
    showConfirmDeleteModal.value = false;
    activeTab.value = 'taskList';
    selectedTaskId.value = null;
  } catch (error) {
    outputLog.value += getCurrentTime() + ` 删除任务失败: ${error}\n`;
  }
}

// Load task details into form
function loadTaskToForm(taskId: string) {
  const task = tasks.value[taskId];
  if (!task) return;
  
  taskName.value = task.name;
  url.value = task.url;
  visitMethod.value = task.method;
  ck.value = task.cookie;
  headers.value = task.headers;
  submitData.value = task.data;
  useVirtualIP.value = task.useVirtualIP;
  times.value = task.times.toString();
  cardPackage.value = task.threads.toString();
  taskTags.value = task.tags.join(', ');
  delayMin.value = task.delayMin.toString();
  delayMax.value = task.delayMax.toString();
  cronExpression.value = task.cronExpression || '';
  hasRemovedGzipHeader.value = false;
  
  // Parse scheduled time
  const timeParts = task.scheduledTime.split(':');
  if (timeParts.length === 3) {
    targetHours.value = timeParts[0];
    targetMinutes.value = timeParts[1];
    targetSeconds.value = timeParts[2];
  }
  
  // Initialize cron builder from existing expression
  initCronBuilderFromExpression(task.cronExpression);
  
  selectedTaskId.value = taskId;
  activeTab.value = 'taskDetail';
  
  // Preserve log scroll position by adding a reference to the detail log area
  nextTick(() => {
    if (detailOutputLogRef.value && outputLogRef.value) {
      detailOutputLogRef.value.scrollTop = outputLogRef.value.scrollTop;
    }
  });
}

// Create a new task
function createNewTask() {
  // Reset form
  taskName.value = '';
  url.value = '';
  visitMethod.value = 'GET';
  ck.value = '';
  headers.value = '';
  submitData.value = '';
  useVirtualIP.value = true;
  times.value = '10';
  cardPackage.value = '5';
  taskTags.value = '';
  delayMin.value = '0';
  delayMax.value = '0';
  hasRemovedGzipHeader.value = false;
  
  // Initialize time inputs with current time
  const now = new Date();
  targetHours.value = now.getHours().toString().padStart(2, '0');
  targetMinutes.value = now.getMinutes().toString().padStart(2, '0');
  targetSeconds.value = now.getSeconds().toString().padStart(2, '0');
  
  selectedTaskId.value = null;
  activeTab.value = 'taskDetail';
}

// Add a task to scheduled tasks
function scheduleTask(taskId: string) {
  if (!scheduledTasks.value.includes(taskId)) {
    scheduledTasks.value.push(taskId);
    outputLog.value += getCurrentTime() + ` 已添加任务到定时队列: ${tasks.value[taskId].name}\n`;
  }
}

// Remove a task from scheduled tasks
function unscheduleTask(taskId: string) {
  const index = scheduledTasks.value.indexOf(taskId);
  if (index !== -1) {
    scheduledTasks.value.splice(index, 1);
    outputLog.value += getCurrentTime() + ` 已从定时队列移除任务: ${tasks.value[taskId].name}\n`;
  }
}

// Export tasks with filters
async function exportFilteredTasks() {
  try {
    // Open file dialog to select save location
    const defaultPath = importExportPath.value || 'tasks.json';
    const selectedPath = await OpenSaveFileDialog('导出任务', defaultPath, '*.json');
    
    if (!selectedPath) {
      outputLog.value += getCurrentTime() + " 已取消导出\n";
      return;
    }
    
    // Ensure path has .json extension
    let filePath = selectedPath;
    if (!filePath.endsWith('.json')) {
      filePath += '.json';
    }
    
    importExportPath.value = filePath;
    
    let result = '';
    
    if (importExportMode.value === 'export') {
      // Export all tasks
      result = await ExportTasks(filePath);
    } else if (importExportMode.value === 'exportSelected') {
      // Export selected tasks
      if (selectedTasksForExport.value.length === 0) {
        outputLog.value += getCurrentTime() + " 错误: 请选择要导出的任务\n";
        return;
      }
      result = await ExportTasksByIDs(filePath, selectedTasksForExport.value);
    } else if (importExportMode.value === 'exportTags') {
      // Export tasks with selected tags
      if (selectedTagsForExport.value.length === 0) {
        outputLog.value += getCurrentTime() + " 错误: 请选择要导出的标签\n";
        return;
      }
      result = await ExportTasksByTags(filePath, selectedTagsForExport.value);
    }
    
    outputLog.value += getCurrentTime() + ` ${result}\n`;
    showImportExportModal.value = false;
    
    // Reset selections
    selectedTasksForExport.value = [];
    selectedTagsForExport.value = [];
  } catch (error) {
    outputLog.value += getCurrentTime() + ` 导出任务失败: ${error}\n`;
  }
}

// Toggle task selection for export
function toggleTaskSelection(taskId: string) {
  const index = selectedTasksForExport.value.indexOf(taskId);
  if (index === -1) {
    selectedTasksForExport.value.push(taskId);
  } else {
    selectedTasksForExport.value.splice(index, 1);
  }
}

// Toggle tag selection for export
function toggleTagSelectionForExport(tag: string) {
  const index = selectedTagsForExport.value.indexOf(tag);
  if (index === -1) {
    selectedTagsForExport.value.push(tag);
  } else {
    selectedTagsForExport.value.splice(index, 1);
  }
}

// Check if a task has a cron schedule
function hasCronSchedule(task: Task) {
  return task.cronExpression && task.cronExpression.trim() !== '';
}

// Format cron expression for display
function formatCronExpression(expr: string) {
  if (!expr || expr.trim() === '') return '无';
  return expr;
}

// Execute a task by ID with option to stop (update to show logs)
async function executeTask(taskId: string) {
  const task = tasks.value[taskId];
  if (!task) {
    outputLog.value += getCurrentTime() + ` 错误: 找不到任务 ${taskId}\n`;
    return;
  }
  
  if (task.isRunning) {
    outputLog.value += getCurrentTime() + ` 任务 ${task.name} 已在运行中\n`;
    return;
  }
  
  try {
    const logMessage = `开始执行任务: ${task.name}`;
    outputLog.value += getCurrentTime() + ` ${logMessage}\n`;
    addTaskLog(taskId, logMessage);
    
    // For cron tasks, add additional log info
    if (task.cronExpression && task.cronExpression.trim() !== '') {
      const cronLogMessage = `定时规则: ${task.cronExpression}`;
      outputLog.value += getCurrentTime() + ` ${cronLogMessage}\n`;
      addTaskLog(taskId, cronLogMessage);
    }
    
    const result = await ExecuteTask(taskId);
    outputLog.value += getCurrentTime() + ` ${result}\n`;
    addTaskLog(taskId, `执行状态: ${result}`);
    
    // Refresh task list to update running status
    await loadTasks();
  } catch (error) {
    const errorMessage = `执行任务失败: ${error}`;
    outputLog.value += getCurrentTime() + ` ${errorMessage}\n`;
    addTaskLog(taskId, errorMessage);
  }
}

// Stop a running task (update to add logs)
async function stopTask(taskId: string) {
  try {
    const result = await StopTask(taskId);
    outputLog.value += getCurrentTime() + ` ${result}\n`;
    addTaskLog(taskId, `停止任务: ${result}`);
    
    // Refresh task list to update running status
    await loadTasks();
    
    // Clean up progress data
    delete runningTaskProgress.value[taskId];
  } catch (error) {
    const errorMessage = `停止任务失败: ${error}`;
    outputLog.value += getCurrentTime() + ` ${errorMessage}\n`;
    addTaskLog(taskId, errorMessage);
  }
}

// Test a task with a single request (update to add logs)
async function testTask(taskId: string) {
  try {
    const taskName = tasks.value[taskId]?.name || taskId;
    const logMessage = `测试任务: ${taskName}`;
    outputLog.value += getCurrentTime() + ` ${logMessage}\n`;
    addTaskLog(taskId, logMessage);
    
    // If we're in task detail view, make sure to scroll to the bottom
    if (activeTab.value === 'taskDetail' && detailOutputLogRef.value) {
      nextTick(() => {
        if (detailOutputLogRef.value && autoScrollLogs.value) {
          detailOutputLogRef.value.scrollTop = detailOutputLogRef.value.scrollHeight;
        }
      });
    }
    
    const result = await TestTask(taskId);
    outputLog.value += getCurrentTime() + ` ${result}\n`;
    addTaskLog(taskId, `测试结果: ${result}`);
    
    // Make sure to scroll to the bottom again after the result comes in
    if (autoScrollLogs.value) {
      nextTick(() => {
        if (activeTab.value === 'taskDetail' && detailOutputLogRef.value) {
          detailOutputLogRef.value.scrollTop = detailOutputLogRef.value.scrollHeight;
        } else if (outputLogRef.value) {
          outputLogRef.value.scrollTop = outputLogRef.value.scrollHeight;
        }
      });
    }
  } catch (error) {
    const errorMessage = `测试任务失败: ${error}`;
    outputLog.value += getCurrentTime() + ` ${errorMessage}\n`;
    addTaskLog(taskId, errorMessage);
  }
}

// Format progress percentage
function formatProgress(taskId: string) {
  const progress = runningTaskProgress.value[taskId];
  if (!progress) return '0%';
  
  const percentage = Math.floor((progress.currentRequest / progress.totalRequests) * 100);
  return `${percentage}%`;
}

// Format elapsed time in milliseconds
function formatElapsedTime(taskId: string) {
  const progress = runningTaskProgress.value[taskId];
  if (!progress) return '0ms';
  
  const ms = progress.elapsedTime;
  
  // For short durations, just show milliseconds
  if (ms < 1000) {
    return `${ms}ms`;
  }
  
  // For longer durations, format as needed
  const totalSeconds = Math.floor(ms / 1000);
  
  if (totalSeconds < 60) {
    // Less than a minute: show as seconds.milliseconds
    return `${totalSeconds}.${(ms % 1000).toString().padStart(3, '0')}s`;
  } else if (totalSeconds < 3600) {
    // Less than an hour: show as minutes:seconds.milliseconds
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}:${seconds.toString().padStart(2, '0')}.${(ms % 1000).toString().padStart(3, '0')}`;
  } else {
    // More than an hour: show full format
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;
    return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}.${(ms % 1000).toString().padStart(3, '0')}`;
  }
}

function toggleScheduledTimer() {
  if (timerActive.value) {
    // Cancel scheduled timer
    timerActive.value = false;
    executionInProgress.value = false;
    outputLog.value += getCurrentTime() + " 已取消定时执行\n";
  } else {
    // Start scheduled timer with validation
    validateHours();
    validateMinutes();
    validateSeconds();
    
    const h = parseInt(targetHours.value) || 0;
    const m = parseInt(targetMinutes.value) || 0;
    const s = parseInt(targetSeconds.value) || 0;
    
    // Ensure we have valid time values
    targetHours.value = h.toString().padStart(2, '0');
    targetMinutes.value = m.toString().padStart(2, '0');
    targetSeconds.value = s.toString().padStart(2, '0');
    
    timerActive.value = true;
    executionInProgress.value = true;
    
    outputLog.value += getCurrentTime() + ` 已设置定时执行，执行时间: ${targetTimeDisplay.value}\n`;
  }
}

function getCurrentTime() {
  const now = new Date();
  const h = now.getHours().toString().padStart(2, '0');
  const m = now.getMinutes().toString().padStart(2, '0');
  const s = now.getSeconds().toString().padStart(2, '0');
  const ms = now.getMilliseconds().toString().padStart(3, '0');
  return `[${h}:${m}:${s}.${ms}]`;
}

// Check scheduled time for the single timer case
function checkScheduledTime() {
  if (!timerActive.value) return;
  
  const now = new Date();
  const h = now.getHours();
  const m = now.getMinutes();
  const s = now.getSeconds();
  
  const targetH = parseInt(targetHours.value) || 0;
  const targetM = parseInt(targetMinutes.value) || 0;
  const targetS = parseInt(targetSeconds.value) || 0;
  
  if (h === targetH && m === targetM && s === targetS) {
    // Time matches, execute the task
    timerActive.value = false;
    startCardPackage();
  }
}

function startTaskTimer() {
  taskStartTime.value = new Date();
  taskDuration.value = 0;
  taskDurationMs.value = 0;
  timerRunning.value = true;
  
  // Start updating the task duration every 10ms for millisecond precision
  const intervalId = setInterval(() => {
    if (taskStartTime.value) {
      const now = new Date();
      const elapsedMs = now.getTime() - taskStartTime.value.getTime();
      taskDurationMs.value = elapsedMs;
      taskDuration.value = Math.floor(elapsedMs / 1000);
    }
  }, 10);
  
  return intervalId;
}

function stopTaskTimer(intervalId: number) {
  clearInterval(intervalId);
  taskEndTime.value = new Date();
  
  // Calculate final duration
  if (taskStartTime.value && taskEndTime.value) {
    taskDurationMs.value = taskEndTime.value.getTime() - taskStartTime.value.getTime();
    taskDuration.value = Math.floor(taskDurationMs.value / 1000);
  }
  
  timerRunning.value = false;
}

async function startCardPackage() {
  if (!url.value) {
    outputLog.value += getCurrentTime() + " 错误: URL不能为空\n";
    executionInProgress.value = false;
    return;
  }
  
  const timesNum = parseInt(times.value);
  const threadsNum = parseInt(cardPackage.value);
  const delayMinNum = parseInt(delayMin.value);
  const delayMaxNum = parseInt(delayMax.value);
  
  if (isNaN(timesNum) || timesNum <= 0) {
    outputLog.value += getCurrentTime() + " 错误: 次数必须是正整数\n";
    executionInProgress.value = false;
    return;
  }
  
  if (isNaN(threadsNum) || threadsNum <= 0) {
    outputLog.value += getCurrentTime() + " 错误: 卡包值/线程数必须是正整数\n";
    executionInProgress.value = false;
    return;
  }
  
  if (isNaN(delayMinNum) || delayMinNum < 0) {
    outputLog.value += getCurrentTime() + " 错误: 最小延迟必须是非负整数\n";
    executionInProgress.value = false;
    return;
  }
  
  if (isNaN(delayMaxNum) || delayMaxNum < 0) {
    outputLog.value += getCurrentTime() + " 错误: 最大延迟必须是非负整数\n";
    executionInProgress.value = false;
    return;
  }
  
  // Start the task timer
  const timerIntervalId = startTaskTimer();
  
  outputLog.value += getCurrentTime() + ` 开始卡包操作，次数: ${timesNum}, 卡包值/线程数: ${threadsNum}, 延迟: ${delayMinNum}-${delayMaxNum}ms\n`;
  executionInProgress.value = true;
  
  try {
    // Execute the card package operation with delay parameters
    const result = await ExecuteCardPackage(
      url.value, 
      visitMethod.value, 
      ck.value, 
      submitData.value,
      headers.value,
      useVirtualIP.value, 
      timesNum,
      threadsNum,
      delayMinNum,
      delayMaxNum
    );
    
    // Update the output log
    outputLog.value += result + "\n";
    
    // Stop the timer and log completion with millisecond precision
    stopTaskTimer(timerIntervalId);
    outputLog.value += getCurrentTime() + ` 卡包操作完成，总耗时: ${formattedTaskDuration.value}\n`;
  } catch (error) {
    outputLog.value += getCurrentTime() + ` 错误: ${error}\n`;
    stopTaskTimer(timerIntervalId);
  } finally {
    executionInProgress.value = false;
  }
}

function parseRawData() {
  try {
    const raw = rawData.value.trim();
    if (!raw) {
      outputLog.value += getCurrentTime() + " 错误: 原始数据为空\n";
      return;
    }
    
    // Reset gzip header notification
    hasRemovedGzipHeader.value = false;
    
    // Parse the raw data
    const lines = raw.split('\n');
    const firstLine = lines[0].trim();
    
    // Extract method and URL
    const methodUrlMatch = firstLine.match(/^(\w+)\s+(\S+)/);
    let extractedUrl = '';
    let extractedMethod = '';
    
    if (methodUrlMatch) {
      extractedMethod = methodUrlMatch[1];
      visitMethod.value = extractedMethod;
      
      // Extract URL - handle both full URLs and relative paths
      extractedUrl = methodUrlMatch[2];
      if (!extractedUrl.startsWith('http')) {
        // Try to find Host header
        const hostLine = lines.find(line => line.toLowerCase().startsWith('host:'));
        if (hostLine) {
          const host = hostLine.split(':', 2)[1].trim();
          extractedUrl = `http://${host}${extractedUrl}`;
        }
      }
      url.value = extractedUrl;
    }
    
    // Extract headers
    let extractedHeaders = '';
    let cookieValue = '';
    let headerStarted = false;
    let hasRemovedGzip = false;
    
    // Find blank line that separates headers from body
    let bodyStartIndex = -1;
    
    for (let i = 1; i < lines.length; i++) {
      const line = lines[i].trim();
      
      if (line === '') {
        bodyStartIndex = i;
        break;
      }
      
      if (line.toLowerCase().startsWith('host:')) {
        headerStarted = true;
      }
      
      if (headerStarted) {
        if (line.toLowerCase().startsWith('cookie:')) {
          cookieValue = line.substring(line.indexOf(':') + 1).trim();
          ck.value = cookieValue;
          // Skip adding cookie to headers
        } else if (line.toLowerCase().startsWith('accept-encoding:') && 
                  line.toLowerCase().includes('gzip')) {
          // Skip this header and mark it as removed
          hasRemovedGzip = true;
          hasRemovedGzipHeader.value = true;
        } else {
          extractedHeaders += line + '\n';
        }
      }
    }
    
    headers.value = extractedHeaders;
    
    // Extract body data
    if (bodyStartIndex !== -1 && bodyStartIndex < lines.length - 1) {
      submitData.value = lines.slice(bodyStartIndex + 1).join('\n');
    }
    
    // Log the parsed data
    outputLog.value += getCurrentTime() + " 原始数据解析成功\n";
    if (hasRemovedGzip) {
      outputLog.value += getCurrentTime() + " 自动移除了包含 gzip 的 Accept-Encoding 协议头\n";
    }
    outputLog.value += getCurrentTime() + " 解析结果:\n";
    outputLog.value += `  网址: ${extractedUrl}\n`;
    outputLog.value += `  方式: ${extractedMethod}\n`;
    if (cookieValue) {
      outputLog.value += `  Cookie: ${cookieValue.length > 50 ? cookieValue.substring(0, 50) + '...' : cookieValue}\n`;
    }
    if (extractedHeaders) {
      outputLog.value += `  协议头: ${extractedHeaders.split('\n').length} 行\n`;
    }
    if (submitData.value) {
      outputLog.value += `  提交数据: ${submitData.value.length > 50 ? submitData.value.substring(0, 50) + '...' : submitData.value}\n`;
    }
  } catch (error) {
    outputLog.value += getCurrentTime() + ` 解析错误: ${error}\n`;
  }
}

function clearLog() {
  outputLog.value = '';
}

// Filter tasks by tag
function toggleTagFilter(tag: string) {
  const index = selectedTags.value.indexOf(tag);
  if (index === -1) {
    selectedTags.value.push(tag);
  } else {
    selectedTags.value.splice(index, 1);
  }
}

// Update progress for running tasks
async function updateTaskProgress() {
  // Get tasks that are marked as running
  const runningTaskIds = Object.keys(tasks.value).filter(id => tasks.value[id].isRunning);
  
  // If no tasks are running, no need to update
  if (runningTaskIds.length === 0) {
    return;
  }
  
  // Update progress for each running task
  for (const taskId of runningTaskIds) {
    try {
      const progress = await GetTaskProgress(taskId);
      if (progress) {
        runningTaskProgress.value[taskId] = progress;
      }
    } catch (error) {
      console.error(`Error fetching progress for task ${taskId}:`, error);
    }
  }
}

// Import tasks from a file
async function importTasksFromFile() {
  try {
    // Open file dialog to select file to import
    const selectedPath = await OpenFileDialog('导入任务', '*.json');
    
    if (!selectedPath) {
      outputLog.value += getCurrentTime() + " 已取消导入\n";
      return;
    }
    
    importExportPath.value = selectedPath;
    
    const result = await ImportTasks(selectedPath);
    outputLog.value += getCurrentTime() + ` ${result}\n`;
    showImportExportModal.value = false;
    
    // Reload tasks
    await loadTasks();
  } catch (error) {
    outputLog.value += getCurrentTime() + ` 导入任务失败: ${error}\n`;
  }
}

// Check if any scheduled tasks need to be executed
function checkScheduledTasks() {
  if (scheduledTasks.value.length === 0) return;
  
  const now = new Date();
  const h = now.getHours();
  const m = now.getMinutes();
  const s = now.getSeconds();
  
  // Check each scheduled task
  for (let i = scheduledTasks.value.length - 1; i >= 0; i--) {
    const taskId = scheduledTasks.value[i];
    const task = tasks.value[taskId];
    
    if (!task) {
      // Task doesn't exist anymore, remove it
      scheduledTasks.value.splice(i, 1);
      continue;
    }
    
    // Parse scheduled time
    const timeParts = task.scheduledTime.split(':');
    if (timeParts.length !== 3) continue;
    
    const targetH = parseInt(timeParts[0]) || 0;
    const targetM = parseInt(timeParts[1]) || 0;
    const targetS = parseInt(timeParts[2]) || 0;
    
    if (h === targetH && m === targetM && s === targetS) {
      // Time matches, execute the task
      outputLog.value += getCurrentTime() + ` 执行定时任务: ${task.name}\n`;
      
      // Remove from scheduled tasks to prevent duplicate execution
      scheduledTasks.value.splice(i, 1);
      
      // Execute the task
      executeTask(taskId);
    }
  }
}

// Clean up intervals when component is unmounted
onUnmounted(() => {
  if (clockInterval) {
    clearInterval(clockInterval);
  }
  
  if (progressUpdateInterval.value) {
    clearInterval(progressUpdateInterval.value);
  }
});

// Add a watcher for cron expression changes
watch(cronExpression, (newValue) => {
  if (newValue) {
    validateCronExpression(newValue);
    cronExpressionChanged.value = true;
  } else {
    cronIsValid.value = true;
    cronValidationMessage.value = '';
    nextCronExecution.value = null;
    cronExpressionChanged.value = false;
  }
});

// Add function to validate cron expression
function validateCronExpression(expr: string) {
  // Basic validation for cron expression
  if (!expr || expr.trim() === '') {
    cronIsValid.value = true;
    cronValidationMessage.value = '';
    nextCronExecution.value = null;
    return;
  }

  try {
    // Check if it's a predefined expression
    if (expr.startsWith('@')) {
      cronIsValid.value = ['@yearly', '@monthly', '@weekly', '@daily', '@hourly'].includes(expr);
      if (cronIsValid.value) {
        cronValidationMessage.value = '';
        calculateNextCronExecution(expr);
      } else {
        cronValidationMessage.value = '无效的预定义表达式';
        nextCronExecution.value = null;
      }
      return;
    }

    // Regular cron expression validation
    const parts = expr.split(' ');
    
    // Cron with seconds should have 6 parts
    if (parts.length !== 6) {
      cronIsValid.value = false;
      cronValidationMessage.value = 'Cron表达式应包含6个部分: 秒 分 时 日 月 星期';
      nextCronExecution.value = null;
      return;
    }

    // Validate each part
    const ranges = [
      { min: 0, max: 59 },  // seconds
      { min: 0, max: 59 },  // minutes
      { min: 0, max: 23 },  // hours
      { min: 1, max: 31 },  // day of month
      { min: 1, max: 12 },  // month
      { min: 0, max: 7 }    // day of week (0 or 7 is Sunday)
    ];

    for (let i = 0; i < 6; i++) {
      const part = parts[i];
      
      // Skip validation for wildcards
      if (part === '*') continue;
      
      // Validate step values (*/n)
      if (part.includes('/')) {
        const step = parseInt(part.split('/')[1]);
        if (isNaN(step) || step <= 0) {
          cronIsValid.value = false;
          cronValidationMessage.value = `步长值必须是正整数`;
          nextCronExecution.value = null;
          return;
        }
        continue;
      }
      
      // Validate comma-separated values
      if (part.includes(',')) {
        const values = part.split(',');
        for (const val of values) {
          const num = parseInt(val);
          if (isNaN(num) || num < ranges[i].min || num > ranges[i].max) {
            cronIsValid.value = false;
            cronValidationMessage.value = `第${i+1}部分包含无效值: ${val}`;
            nextCronExecution.value = null;
            return;
          }
        }
        continue;
      }
      
      // Validate ranges
      if (part.includes('-')) {
        const [start, end] = part.split('-').map(v => parseInt(v));
        if (isNaN(start) || isNaN(end) || 
            start < ranges[i].min || start > ranges[i].max ||
            end < ranges[i].min || end > ranges[i].max ||
            start > end) {
          cronIsValid.value = false;
          cronValidationMessage.value = `第${i+1}部分的范围无效: ${part}`;
          nextCronExecution.value = null;
          return;
        }
        continue;
      }
      
      // Validate single values
      const num = parseInt(part);
      if (isNaN(num) || num < ranges[i].min || num > ranges[i].max) {
        cronIsValid.value = false;
        cronValidationMessage.value = `第${i+1}部分的值无效: ${part}`;
        nextCronExecution.value = null;
        return;
      }
    }
    
    // If we get here, the expression is valid
    cronIsValid.value = true;
    cronValidationMessage.value = '';
    
    // Calculate next execution time
    calculateNextCronExecution(expr);
    
  } catch (error) {
    cronIsValid.value = false;
    cronValidationMessage.value = '无效的Cron表达式格式';
    nextCronExecution.value = null;
  }
}

// Calculate next execution time for a cron expression
function calculateNextCronExecution(expr: string) {
  try {
    // Handle predefined expressions
    if (expr === '@yearly' || expr === '@annually') {
      const now = new Date();
      const next = new Date(now.getFullYear() + 1, 0, 1, 0, 0, 0);
      nextCronExecution.value = next;
      return;
    }
    
    if (expr === '@monthly') {
      const now = new Date();
      const next = new Date(now.getFullYear(), now.getMonth() + 1, 1, 0, 0, 0);
      nextCronExecution.value = next;
      return;
    }
    
    if (expr === '@weekly') {
      const now = new Date();
      const daysUntilSunday = 7 - now.getDay();
      const next = new Date(now);
      next.setDate(now.getDate() + (daysUntilSunday === 7 ? 0 : daysUntilSunday));
      next.setHours(0, 0, 0, 0);
      nextCronExecution.value = next;
      return;
    }
    
    if (expr === '@daily' || expr === '@midnight') {
      const now = new Date();
      const next = new Date(now);
      next.setDate(now.getDate() + 1);
      next.setHours(0, 0, 0, 0);
      nextCronExecution.value = next;
      return;
    }
    
    if (expr === '@hourly') {
      const now = new Date();
      const next = new Date(now);
      next.setHours(now.getHours() + 1, 0, 0, 0);
      nextCronExecution.value = next;
      return;
    }
    
    // Parse regular cron expression
    const parts = expr.split(' ');
    if (parts.length !== 6) {
      nextCronExecution.value = null;
      return;
    }
    
    const now = new Date();
    let next = new Date(now);
    
    // Handle time range expression (e.g., "0 0 9-17 * * *")
    const hourRangeMatch = parts[2].match(/^(\d+)-(\d+)$/);
    if (hourRangeMatch && parts[0] === '0' && parts[1] === '0') {
      const startHour = parseInt(hourRangeMatch[1]);
      const endHour = parseInt(hourRangeMatch[2]);
      const currentHour = now.getHours();
      
      if (currentHour < startHour) {
        // Before the range starts today
        next.setHours(startHour, 0, 0, 0);
      } else if (currentHour >= endHour) {
        // After the range ends today, move to tomorrow
        next.setDate(now.getDate() + 1);
        next.setHours(startHour, 0, 0, 0);
      } else {
        // Within the range, move to the next hour in the range
        next.setHours(currentHour + 1, 0, 0, 0);
      }
      
      nextCronExecution.value = next;
      return;
    }
    
    // Handle simple cases
    if (expr === '* * * * * *') {
      // Every second
      next.setSeconds(now.getSeconds() + 1);
      nextCronExecution.value = next;
      return;
    }
    
    // Handle specific time of day (e.g., "0 30 9 * * *" - 9:30:00 every day)
    if (/^\d+ \d+ \d+ \* \* \*$/.test(expr)) {
      const second = parseInt(parts[0]);
      const minute = parseInt(parts[1]);
      const hour = parseInt(parts[2]);
      
      next.setHours(hour, minute, second, 0);
      
      // If the time is already past for today, move to tomorrow
      if (next <= now) {
        next.setDate(next.getDate() + 1);
      }
      
      nextCronExecution.value = next;
      return;
    }
    
    // Handle day of week restrictions
    if (parts[5] !== '*') {
      const dayOfWeekSpec = parts[5];
      let allowedDays: number[] = [];
      
      // Parse day of week specification (0-6, where 0 is Sunday)
      if (dayOfWeekSpec.includes(',')) {
        // Comma-separated list of days
        allowedDays = dayOfWeekSpec.split(',').map(d => parseInt(d) % 7);
      } else if (dayOfWeekSpec.includes('-')) {
        // Range of days
        const [start, end] = dayOfWeekSpec.split('-').map(d => parseInt(d) % 7);
        if (start <= end) {
          for (let i = start; i <= end; i++) {
            allowedDays.push(i);
          }
        } else {
          // Handle wrap-around (e.g., 5-1 means Fri,Sat,Sun,Mon)
          for (let i = start; i <= 6; i++) {
            allowedDays.push(i);
          }
          for (let i = 0; i <= end; i++) {
            allowedDays.push(i);
          }
        }
      } else {
        // Single day
        allowedDays = [parseInt(dayOfWeekSpec) % 7];
      }
      
      // Get the hour, minute, second from the cron expression
      const second = parts[0] === '*' ? 0 : parseInt(parts[0]);
      const minute = parts[1] === '*' ? 0 : parseInt(parts[1]);
      const hour = parts[2] === '*' ? 0 : parseInt(parts[2]);
      
      // Set the time components
      next.setHours(hour, minute, second, 0);
      
      // Find the next allowed day
      let daysToAdd = 0;
      let currentDayOfWeek = next.getDay();
      
      // If the specified time is earlier today and today is an allowed day
      if (next <= now && allowedDays.includes(currentDayOfWeek)) {
        daysToAdd = 7; // Move to next week
      } else {
        // Find the next allowed day
        while (!allowedDays.includes((currentDayOfWeek + daysToAdd) % 7) || 
               (daysToAdd === 0 && next <= now)) {
          daysToAdd++;
          if (daysToAdd > 7) break; // Safety check
        }
      }
      
      next.setDate(next.getDate() + daysToAdd);
      nextCronExecution.value = next;
      return;
    }
    
    // For step values in seconds (e.g., */5 * * * * *)
    if (parts[0].startsWith('*/')) {
      const step = parseInt(parts[0].substring(2));
      const currentSecond = now.getSeconds();
      const nextSecond = currentSecond + step - (currentSecond % step);
      
      if (nextSecond >= 60) {
        // Move to next minute
        next.setMinutes(now.getMinutes() + 1, nextSecond % 60, 0);
      } else {
        next.setSeconds(nextSecond, 0);
      }
      
      nextCronExecution.value = next;
      return;
    }
    
    // For step values in minutes (e.g., 0 */5 * * * *)
    if (parts[1].startsWith('*/')) {
      const step = parseInt(parts[1].substring(2));
      const second = parseInt(parts[0]);
      const currentMinute = now.getMinutes();
      const nextMinute = currentMinute + step - (currentMinute % step);
      
      next.setSeconds(second, 0);
      
      if (nextMinute >= 60) {
        // Move to next hour
        next.setHours(now.getHours() + 1, nextMinute % 60);
      } else {
        next.setMinutes(nextMinute);
        // If the time is already past, move to the next occurrence
        if (next <= now) {
          next.setMinutes(next.getMinutes() + step);
        }
      }
      
      nextCronExecution.value = next;
      return;
    }
    
    // For step values in hours (e.g., 0 0 */2 * * *)
    if (parts[2].startsWith('*/')) {
      const step = parseInt(parts[2].substring(2));
      const second = parseInt(parts[0]);
      const minute = parseInt(parts[1]);
      const currentHour = now.getHours();
      const nextHour = currentHour + step - (currentHour % step);
      
      next.setMinutes(minute, second, 0);
      
      if (nextHour >= 24) {
        // Move to next day
        next.setDate(now.getDate() + 1);
        next.setHours(nextHour % 24);
      } else {
        next.setHours(nextHour);
        // If the time is already past, move to the next occurrence
        if (next <= now) {
          next.setHours(next.getHours() + step);
        }
      }
      
      nextCronExecution.value = next;
      return;
    }
    
    // For other complex expressions, provide a reasonable estimate
    // Default fallback - add 1 minute
    next.setMinutes(now.getMinutes() + 1, 0, 0);
    nextCronExecution.value = next;
    
  } catch (error) {
    nextCronExecution.value = null;
  }
}

// Format next execution time
function formatNextExecution(date: Date | null): string {
  if (!date) return '无法计算';
  
  const now = new Date();
  const diffMs = date.getTime() - now.getTime();
  
  if (diffMs < 0) return '已过期';
  
  // If it's within the next minute
  if (diffMs < 60000) {
    return `${Math.ceil(diffMs / 1000)}秒后`;
  }
  
  // If it's within the next hour
  if (diffMs < 3600000) {
    return `${Math.floor(diffMs / 60000)}分钟后`;
  }
  
  // If it's within the next day
  if (diffMs < 86400000) {
    return `${Math.floor(diffMs / 3600000)}小时后`;
  }
  
  // If it's within the next week
  if (diffMs < 604800000) {
    return `${Math.floor(diffMs / 86400000)}天后`;
  }
  
  // Otherwise, return the date and time
  return `${date.toLocaleDateString()} ${date.toLocaleTimeString()}`;
}

// Calculate next execution time for a task with cron expression
function getNextCronExecutionForTask(task: Task): string {
  if (!task.cronExpression || task.cronExpression.trim() === '') {
    return '未设置';
  }
  
  try {
    // Reuse the existing calculation function
    const savedExpr = cronExpression.value;
    cronExpression.value = task.cronExpression;
    validateCronExpression(task.cronExpression);
    const result = nextCronExecution.value ? formatNextExecution(nextCronExecution.value) : '无法计算';
    cronExpression.value = savedExpr;
    return result;
  } catch (error) {
    return '无法计算';
  }
}

// Add function to build cron expression from builder inputs
function buildCronExpression() {
  switch (cronBuilderType.value) {
    case 'every_second':
      const seconds = parseInt(cronBuilderSecond.value) || 1;
      cronExpression.value = seconds === 1 ? '* * * * * *' : `*/${seconds} * * * * *`;
      break;
    case 'every_minute':
      const minutes = parseInt(cronBuilderMinute.value) || 1;
      const secondsInMinute = parseInt(cronBuilderSecond.value) || 0;
      cronExpression.value = minutes === 1 
        ? `${secondsInMinute} * * * * *` 
        : `${secondsInMinute} */${minutes} * * * *`;
      break;
    case 'every_hour':
      const hours = parseInt(cronBuilderHour.value) || 1;
      const minutesInHour = parseInt(cronBuilderMinute.value) || 0;
      const secondsInHour = parseInt(cronBuilderSecond.value) || 0;
      cronExpression.value = hours === 1 
        ? `${secondsInHour} ${minutesInHour} * * * *` 
        : `${secondsInHour} ${minutesInHour} */${hours} * * *`;
      break;
    case 'every_day':
      const hoursInDay = parseInt(cronBuilderHour.value) || 0;
      const minutesInDay = parseInt(cronBuilderMinute.value) || 0;
      const secondsInDay = parseInt(cronBuilderSecond.value) || 0;
      cronExpression.value = `${secondsInDay} ${minutesInDay} ${hoursInDay} * * *`;
      break;
    case 'workday':
      const hoursWorkday = parseInt(cronBuilderHour.value) || 0;
      const minutesWorkday = parseInt(cronBuilderMinute.value) || 0;
      const secondsWorkday = parseInt(cronBuilderSecond.value) || 0;
      cronExpression.value = `${secondsWorkday} ${minutesWorkday} ${hoursWorkday} * * 1-5`;
      break;
    case 'weekend':
      const hoursWeekend = parseInt(cronBuilderHour.value) || 0;
      const minutesWeekend = parseInt(cronBuilderMinute.value) || 0;
      const secondsWeekend = parseInt(cronBuilderSecond.value) || 0;
      cronExpression.value = `${secondsWeekend} ${minutesWeekend} ${hoursWeekend} * * 0,6`;
      break;
    case 'range':
      const startHour = parseInt(cronBuilderStartHour.value) || 9;
      const endHour = parseInt(cronBuilderEndHour.value) || 17;
      const interval = parseInt(cronBuilderRangeInterval.value) || 1;
      const minutesRange = parseInt(cronBuilderMinute.value) || 0;
      const secondsRange = parseInt(cronBuilderSecond.value) || 0;
      
      if (interval === 1) {
        cronExpression.value = `${secondsRange} ${minutesRange} ${startHour}-${endHour} * * *`;
      } else {
        cronExpression.value = `${secondsRange} ${minutesRange} ${startHour}-${endHour}/${interval} * * *`;
      }
      break;
    case 'specific':
      buildCronFromTime();
      break;
  }
  
  // Validate the expression after building it
  validateCronExpression(cronExpression.value);
}

// Build cron expression from time inputs
function buildCronFromTime() {
  const h = parseInt(targetHours.value) || 0;
  const m = parseInt(targetMinutes.value) || 0;
  const s = parseInt(targetSeconds.value) || 0;
  
  // Create a cron expression for the specific time
  cronExpression.value = `${s} ${m} ${h} * * *`;
}

// Initialize cron builder from existing cron expression
function initCronBuilderFromExpression(expr: string) {
  if (!expr || expr.trim() === '') {
    cronBuilderType.value = 'specific';
    cronBuilderSecond.value = '0';
    cronBuilderMinute.value = '0';
    cronBuilderHour.value = '0';
    return;
  }
  
  // Try to match common patterns
  const parts = expr.split(' ');
  
  // Every second: * * * * * *
  if (expr === '* * * * * *') {
    cronBuilderType.value = 'every_second';
    cronBuilderSecond.value = '1';
    return;
  }
  
  // Every N seconds: */N * * * * *
  const everyNSecondsMatch = expr.match(/^\*\/(\d+) \* \* \* \* \*$/);
  if (everyNSecondsMatch) {
    cronBuilderType.value = 'every_second';
    cronBuilderSecond.value = everyNSecondsMatch[1];
    return;
  }
  
  // Every minute at specific second: S * * * * *
  const everyMinuteMatch = expr.match(/^(\d+) \* \* \* \* \*$/);
  if (everyMinuteMatch) {
    cronBuilderType.value = 'every_minute';
    cronBuilderSecond.value = everyMinuteMatch[1];
    cronBuilderMinute.value = '1';
    return;
  }
  
  // Every N minutes at specific second: S */N * * * *
  const everyNMinutesMatch = expr.match(/^(\d+) \*\/(\d+) \* \* \* \*$/);
  if (everyNMinutesMatch) {
    cronBuilderType.value = 'every_minute';
    cronBuilderSecond.value = everyNMinutesMatch[1];
    cronBuilderMinute.value = everyNMinutesMatch[2];
    return;
  }
  
  // Every hour at specific minute and second: S M * * * *
  const everyHourMatch = expr.match(/^(\d+) (\d+) \* \* \* \*$/);
  if (everyHourMatch) {
    cronBuilderType.value = 'every_hour';
    cronBuilderSecond.value = everyHourMatch[1];
    cronBuilderMinute.value = everyHourMatch[2];
    cronBuilderHour.value = '1';
    return;
  }
  
  // Every N hours at specific minute and second: S M */N * * *
  const everyNHoursMatch = expr.match(/^(\d+) (\d+) \*\/(\d+) \* \* \*$/);
  if (everyNHoursMatch) {
    cronBuilderType.value = 'every_hour';
    cronBuilderSecond.value = everyNHoursMatch[1];
    cronBuilderMinute.value = everyNHoursMatch[2];
    cronBuilderHour.value = everyNHoursMatch[3];
    return;
  }
  
  // Every day at specific hour, minute, and second: S M H * * *
  const everyDayMatch = expr.match(/^(\d+) (\d+) (\d+) \* \* \*$/);
  if (everyDayMatch) {
    cronBuilderType.value = 'every_day';
    cronBuilderSecond.value = everyDayMatch[1];
    cronBuilderMinute.value = everyDayMatch[2];
    cronBuilderHour.value = everyDayMatch[3];
    return;
  }
  
  // For other expressions, set to specific time if possible
  if (parts.length === 6) {
    if (/^\d+$/.test(parts[0]) && /^\d+$/.test(parts[1]) && /^\d+$/.test(parts[2])) {
      cronBuilderType.value = 'specific';
      targetSeconds.value = parts[0].padStart(2, '0');
      targetMinutes.value = parts[1].padStart(2, '0');
      targetHours.value = parts[2].padStart(2, '0');
      return;
    }
  }
  
  // Default to specific time
  cronBuilderType.value = 'specific';
}

// Format next execution time as a date string
function formatNextExecutionDate(date: Date | null): string {
  if (!date) return '无法计算';
  
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  });
}

// Update cron builder inputs when type changes
function updateCronBuilder() {
  // Reset values to defaults when changing type
  switch (cronBuilderType.value) {
    case 'every_second':
      cronBuilderSecond.value = '1';
      break;
    case 'every_minute':
      cronBuilderMinute.value = '1';
      cronBuilderSecond.value = '0';
      break;
    case 'every_hour':
      cronBuilderHour.value = '1';
      cronBuilderMinute.value = '0';
      cronBuilderSecond.value = '0';
      break;
    case 'every_day':
      cronBuilderHour.value = '9';
      cronBuilderMinute.value = '0';
      cronBuilderSecond.value = '0';
      break;
    case 'workday':
      cronBuilderHour.value = '9';
      cronBuilderMinute.value = '0';
      cronBuilderSecond.value = '0';
      break;
    case 'weekend':
      cronBuilderHour.value = '9';
      cronBuilderMinute.value = '0';
      cronBuilderSecond.value = '0';
      break;
    case 'range':
      cronBuilderStartHour.value = '9';
      cronBuilderEndHour.value = '17';
      cronBuilderRangeInterval.value = '1';
      cronBuilderMinute.value = '0';
      cronBuilderSecond.value = '0';
      break;
    case 'specific':
      // Use current time
      const now = new Date();
      targetHours.value = now.getHours().toString().padStart(2, '0');
      targetMinutes.value = now.getMinutes().toString().padStart(2, '0');
      targetSeconds.value = now.getSeconds().toString().padStart(2, '0');
      break;
  }
}

// Add function to save task logs to disk
async function saveTaskLogsToDisk() {
  try {
    if (Object.keys(taskLogs.value).length === 0) {
      return true;
    }
    
    // Call backend to save logs
    const result = await SaveTaskLogs(taskLogs.value);
    taskLogsPath.value = result.path;
    return true;
  } catch (error) {
    console.error('Error saving task logs:', error);
    return false;
  }
}

// Add function to load task logs from disk
async function loadTaskLogsFromDisk() {
  try {
    // Call backend to load logs
    const result = await LoadTaskLogs();
    taskLogs.value = result.logs || {};
    taskLogsPath.value = result.path;
  } catch (error) {
    console.error('Error loading task logs:', error);
  }
}

// Add function to add log entry for a task
function addTaskLog(taskId: string, message: string) {
  if (!taskLogs.value[taskId]) {
    taskLogs.value[taskId] = [];
  }
  
  // Add date to the log entry
  const now = new Date();
  const dateStr = now.toISOString().split('T')[0]; // YYYY-MM-DD
  const logEntry = `[${dateStr}] ${getCurrentTime()} ${message}`;
  
  taskLogs.value[taskId].push(logEntry);
  
  // Save logs to disk (debounced)
  debouncedSaveTaskLogs();
}

// Create a debounced version of saveTaskLogsToDisk
const debouncedSaveTaskLogs = debounce(saveTaskLogsToDisk, 2000);

// Add function to clear logs for a task
async function clearTaskLogs(taskId: string) {
  if (!taskId) return;
  
  try {
    await ClearTaskLogs(taskId);
    if (taskLogs.value[taskId]) {
      taskLogs.value[taskId] = [];
    }
    outputLog.value += getCurrentTime() + ` 已清空任务 ${tasks.value[taskId]?.name || taskId} 的日志\n`;
  } catch (error) {
    outputLog.value += getCurrentTime() + ` 清空日志失败: ${error}\n`;
  }
}

// Add function to clear all logs
async function clearAllTaskLogs() {
  try {
    await ClearTaskLogs('all');
    taskLogs.value = {};
    outputLog.value += getCurrentTime() + " 已清空所有任务日志\n";
  } catch (error) {
    outputLog.value += getCurrentTime() + ` 清空所有日志失败: ${error}\n`;
  }
}

// Add function to delete old logs
async function deleteOldLogs() {
  try {
    const days = parseInt(logRetentionDays.value) || 7;
    const result = await DeleteOldTaskLogs(days);
    outputLog.value += getCurrentTime() + ` 已删除 ${result.count} 条过期日志\n`;
    
    // Reload logs
    await loadTaskLogsFromDisk();
  } catch (error) {
    outputLog.value += getCurrentTime() + ` 删除过期日志失败: ${error}\n`;
  }
}

// Add function to show task logs
function showTaskLogs(taskId: string) {
  currentTaskLogId.value = taskId;
  showTaskLogsModal.value = true;
}

// Utility function for debouncing
function debounce(fn: Function, delay: number) {
  let timeout: number | null = null;
  return function(...args: any[]) {
    if (timeout !== null) {
      clearTimeout(timeout);
    }
    timeout = setTimeout(() => {
      fn.apply(null, args);
      timeout = null;
    }, delay) as unknown as number;
  };
}

// Mock file system functions for now (these would be replaced with actual Wails bindings)
async function getHomeDir() {
  return os.homedir ? os.homedir() : '.';
}

async function ensureDir(dir: string) {
  // In a real implementation, this would create the directory if it doesn't exist
  return true;
}

async function writeFile(path: string, content: string) {
  // In a real implementation, this would write content to a file
  return true;
}

async function readFile(path: string) {
  // In a real implementation, this would read content from a file
  return '';
}

async function readDir(path: string) {
  // In a real implementation, this would list files in a directory
  return [];
}

async function pathExists(path: string) {
  // In a real implementation, this would check if a path exists
  return false;
}
</script>

<style>
:root {
  --primary-color: #1976d2;
  --primary-light: #e3f2fd;
  --primary-dark: #0d47a1;
  --secondary-color: #e0f7fa;
  --text-color: #333;
  --light-gray: #f5f5f5;
  --border-color: #ddd;
  --danger-color: #f44336;
  --success-color: #4caf50;
  --warning-color: #ff9800;
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

.container {
  font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
  width: 100%;
  height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: var(--light-gray);
  color: var(--text-color);
  overflow: hidden;
}

.nav-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 15px;
  background-color: white;
  border-bottom: 1px solid var(--border-color);
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}

.nav-tabs {
  display: flex;
  gap: 10px;
}

.nav-tab {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s, transform 0.1s;
  background-color: #e0e0e0;
  color: var(--text-color);
  position: relative;
}

.nav-tab.active {
  background-color: var(--primary-color);
  color: white;
}

.nav-tab:hover:not(.active) {
  background-color: #d0d0d0;
}

.badge {
  position: absolute;
  top: -5px;
  right: -5px;
  background-color: var(--danger-color);
  color: white;
  border-radius: 50%;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.7rem;
  font-weight: bold;
}

.content {
  display: flex;
  flex: 1;
  padding: 15px;
  gap: 15px;
  overflow: auto;
  height: calc(100vh - 60px);
  min-height: 0;
}

.left-panel, .right-panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  gap: 15px;
  overflow-y: auto;
  height: 100%;
  min-width: 0;
}

.card {
  background-color: white;
  border-radius: 8px;
  box-shadow: 0 2px 5px rgba(0,0,0,0.1);
  padding: 15px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border-color);
}

.card-header h2 {
  font-size: 1rem;
  font-weight: 500;
  color: var(--primary-color);
  margin: 0;
}

.current-time {
  font-family: monospace;
  font-size: 1rem;
  font-weight: bold;
  color: var(--primary-dark);
  background-color: var(--primary-light);
  padding: 4px 8px;
  border-radius: 4px;
}

.input-group {
  display: flex;
  align-items: center;
  gap: 10px;
}

.input-group label {
  width: 80px;
  text-align: right;
  color: var(--primary-color);
  font-weight: 500;
}

.input-group input[type="text"], 
.input-group select,
.input-group textarea {
  flex: 1;
  border: 1px solid var(--border-color);
  padding: 8px 10px;
  border-radius: 4px;
  font-size: 0.9rem;
  transition: border-color 0.2s;
}

.input-group input[type="text"]:focus, 
.input-group select:focus,
.input-group textarea:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(25, 118, 210, 0.2);
}

.headers-input {
  height: 80px;
  font-family: monospace;
  resize: none;
}

.checkbox-group {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: 90px;
}

.checkbox-group input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: var(--primary-color);
}

.small-input {
  width: 80px !important;
  flex: none !important;
  text-align: center;
}

.delay-inputs {
  display: flex;
  align-items: center;
  gap: 5px;
}

.time-input-group {
  display: flex;
  align-items: center;
  gap: 5px;
}

.time-input {
  width: 40px !important;
  text-align: center;
}

.raw-data-card {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 200px;
}

.raw-data-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.raw-data {
  width: 100%;
  height: 100%;
  min-height: 150px;
  border: 1px solid var(--border-color);
  padding: 10px;
  border-radius: 4px;
  font-family: monospace;
  resize: none;
}

.card-settings {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.settings-row {
  display: flex;
  gap: 15px;
  justify-content: space-between;
}

.setting-item {
  display: flex;
  flex-direction: column;
  gap: 5px;
  flex: 1;
}

.timer-setting {
  flex: 2;
}

.setting-item label {
  font-size: 0.85rem;
  color: var(--text-color);
}

.button-group {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}

.btn {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s, transform 0.1s;
  background-color: #e0e0e0;
  color: var(--text-color);
}

.btn:hover:not(:disabled) {
  background-color: #d0d0d0;
  transform: translateY(-1px);
}

.btn:active:not(:disabled) {
  transform: translateY(1px);
}

.btn.primary {
  background-color: var(--primary-color);
  color: white;
}

.btn.primary:hover:not(:disabled) {
  background-color: var(--primary-dark);
}

.btn.warning {
  background-color: var(--warning-color);
  color: white;
}

.btn.danger {
  background-color: var(--danger-color);
  color: white;
}

.btn.danger:hover:not(:disabled) {
  background-color: #d32f2f;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.timer-display {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  margin-top: 10px;
  padding: 10px;
  border-radius: 4px;
  background-color: var(--light-gray);
  font-family: monospace;
}

.timer-display.active {
  background-color: var(--primary-light);
  border: 1px solid var(--primary-color);
}

.countdown {
  font-weight: bold;
  color: var(--primary-dark);
}

.time-remaining {
  font-size: 0.9em;
  color: var(--text-color);
  opacity: 0.8;
}

.output-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 300px;
}

.output-area {
  flex: 1;
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.output-area textarea {
  width: 100%;
  height: 100%;
  min-height: 200px;
  border: 1px solid var(--border-color);
  padding: 10px;
  border-radius: 4px;
  font-family: monospace;
  resize: none;
  background-color: var(--light-gray);
  white-space: pre-wrap;
}

/* Task list styles */
.tasks-container {
  display: flex;
  flex-direction: column;
  gap: 15px;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: auto;
}

.task-filters {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 10px;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-color);
}

.search-box input {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 0.9rem;
}

.tag-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 5px;
}

.tag-filter {
  background-color: var(--light-gray);
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.8rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.tag-filter:hover {
  background-color: var(--secondary-color);
}

.tag-filter.active {
  background-color: var(--primary-light);
  color: var(--primary-dark);
  font-weight: bold;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
  padding-right: 5px;
  flex: 1;
}

.task-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px;
  border-radius: 4px;
  background-color: white;
  border: 1px solid var(--border-color);
  transition: box-shadow 0.2s;
}

.task-item:hover {
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
}

.task-item.scheduled {
  border: 1px solid var(--warning-color);
  background-color: #fff8e1;
}

.task-info {
  flex: 1;
  cursor: pointer;
}

.task-name {
  font-weight: 500;
  font-size: 1rem;
  margin-bottom: 5px;
  color: var(--primary-dark);
}

.task-url {
  font-size: 0.85rem;
  color: #666;
  margin-bottom: 5px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 500px;
}

.task-meta {
  display: flex;
  gap: 10px;
  font-size: 0.8rem;
  color: #666;
  margin-bottom: 5px;
}

.task-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.tag {
  background-color: var(--primary-light);
  color: var(--primary-dark);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 500;
}

.task-actions {
  display: flex;
  gap: 5px;
}

.no-tasks {
  text-align: center;
  padding: 20px;
  color: #666;
  font-style: italic;
}

/* Modal styles */
.modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 100;
}

.modal-content {
  background-color: white;
  border-radius: 8px;
  width: 400px;
  max-width: 90%;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h2 {
  font-size: 1.1rem;
  font-weight: 500;
  color: var(--primary-color);
  margin: 0;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #666;
}

.close-btn:hover {
  color: var(--danger-color);
}

.modal-body {
  padding: 15px;
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.modal-footer {
  padding: 15px;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  border-top: 1px solid var(--border-color);
}

.notification-box {
  background-color: var(--primary-light);
  padding: 8px 10px;
  border-radius: 4px;
  margin-top: 10px;
}

.notification-content {
  display: flex;
  align-items: center;
  gap: 5px;
}

.notification-icon {
  font-size: 1rem;
  color: var(--primary-dark);
}

.output-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.auto-scroll-toggle {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 0.8rem;
}

.task-progress {
  margin-top: 5px;
  margin-bottom: 5px;
}

.progress-bar {
  height: 8px;
  background-color: #f0f0f0;
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background-color: var(--primary-color);
  transition: width 0.3s ease;
}

.progress-text {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
  color: #666;
  margin-top: 2px;
}

.progress-time {
  font-family: monospace;
}

.task-item.running {
  border: 1px solid var(--primary-color);
  background-color: var(--primary-light);
}

.task-item.cron-scheduled {
  border: 1px solid #9c27b0;
  background-color: #f3e5f5;
}

.task-status {
  display: inline-block;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 500;
  margin-left: 10px;
}

.task-status.running {
  background-color: var(--primary-color);
  color: white;
}

.task-status.cron {
  background-color: #9c27b0;
  color: white;
}

.task-selection {
  display: flex;
  align-items: center;
  padding: 0 10px;
}

.cron-input-group {
  display: flex;
  align-items: center;
  gap: 5px;
}

.cron-input {
  flex: 1;
  border: 1px solid var(--border-color);
  padding: 8px 10px;
  border-radius: 4px;
  font-size: 0.9rem;
  font-family: monospace;
}

.cron-help {
  cursor: help;
}

.tooltip {
  position: relative;
  display: inline-block;
  width: 20px;
  height: 20px;
  background-color: var(--primary-light);
  color: var(--primary-dark);
  border-radius: 50%;
  text-align: center;
  line-height: 20px;
  font-weight: bold;
  cursor: help;
}

.tooltip-text {
  visibility: hidden;
  width: 300px;
  background-color: #333;
  color: #fff;
  text-align: left;
  border-radius: 6px;
  padding: 8px;
  position: absolute;
  z-index: 1;
  bottom: 125%;
  left: 50%;
  transform: translateX(-50%);
  opacity: 0;
  transition: opacity 0.3s;
  font-weight: normal;
  font-size: 0.8rem;
  line-height: 1.4;
}

.tooltip:hover .tooltip-text {
  visibility: visible;
  opacity: 1;
}

.tag-selection {
  margin-top: 15px;
}

.export-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.modal-lg {
  width: 700px;
  max-width: 90%;
}

.cron-help-content {
  max-height: 70vh;
  overflow-y: auto;
  padding-right: 10px;
}

.cron-help-content h3 {
  margin-top: 20px;
  margin-bottom: 10px;
  color: var(--primary-dark);
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 5px;
}

.cron-help-content code {
  background-color: #f0f0f0;
  padding: 2px 4px;
  border-radius: 4px;
  font-family: monospace;
  color: #e91e63;
}

.cron-table {
  width: 100%;
  border-collapse: collapse;
  margin: 15px 0;
}

.cron-table th, .cron-table td {
  border: 1px solid var(--border-color);
  padding: 8px;
  text-align: left;
}

.cron-table th {
  background-color: var(--primary-light);
  color: var(--primary-dark);
}

.cron-example {
  background-color: #f5f5f5;
  padding: 10px 15px;
  border-radius: 4px;
  margin: 15px 0;
}

.cron-example h4 {
  margin-top: 0;
  margin-bottom: 10px;
  color: var(--primary-dark);
}

.cron-example ul {
  margin: 0;
  padding-left: 20px;
}

.cron-example li {
  margin-bottom: 5px;
}

.cron-input.invalid {
  border-color: var(--danger-color);
  background-color: rgba(244, 67, 54, 0.05);
}

.cron-validation-error {
  color: var(--danger-color);
  font-size: 0.8rem;
  margin-top: 5px;
}

.cron-next-execution {
  color: var(--success-color);
  font-size: 0.8rem;
  margin-top: 5px;
}

.task-next-execution {
  font-size: 0.8rem;
  color: var(--primary-dark);
  margin-bottom: 5px;
}

.cron-builder {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.cron-builder-row {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.cron-builder-row select {
  padding: 8px;
  border-radius: 4px;
  border: 1px solid var(--border-color);
  width: 100%;
}

.cron-builder-inputs {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 5px;
}

.cron-builder-item {
  display: flex;
  align-items: center;
  gap: 5px;
}

.cron-builder-item label {
  font-size: 0.8rem;
}

.cron-builder-item input {
  width: 50px;
  text-align: center;
  padding: 5px;
  border-radius: 4px;
  border: 1px solid var(--border-color);
}

.cron-builder-actions {
  margin-top: 10px;
  display: flex;
  justify-content: flex-end;
}

.task-logs-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
  flex-wrap: wrap;
  gap: 10px;
}

.task-logs-retention {
  display: flex;
  align-items: center;
  gap: 10px;
}

.task-logs-clear {
  display: flex;
  gap: 10px;
}

.task-logs-content {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid var(--border-color);
  padding: 10px;
  border-radius: 4px;
  background-color: var(--light-gray);
  font-family: monospace;
  white-space: pre-wrap;
}

.task-log-entry {
  margin-bottom: 5px;
  border-bottom: 1px solid rgba(0,0,0,0.05);
  padding-bottom: 5px;
}

.no-logs {
  text-align: center;
  padding: 20px;
  color: #666;
  font-style: italic;
}
</style>


