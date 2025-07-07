package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// 版本信息
const (
	AppVersion = "1.0.0"
	AppName    = "HTTPTaskRunner"
	BuildDate  = "2025-01-07"
)

// VersionInfo - 版本信息结构
type VersionInfo struct {
	Version   string `json:"version"`
	Name      string `json:"name"`
	BuildDate string `json:"buildDate"`
}

// App - 重新设计的应用结构，优化性能
type App struct {
	ctx           context.Context
	runningTasks  map[string]*TaskProgress
	taskMutex     sync.RWMutex
	cronScheduler *cron.Cron
	cronJobs      map[string]cron.EntryID
	cronMutex     sync.RWMutex
	// 添加缓存机制
	tasksCache    map[string]*Task
	cacheMutex    sync.RWMutex
	lastCacheTime time.Time
	// 日志管理
	taskLogs      map[string][]TaskLogEntry // 任务级别日志
	executionLogs map[string]ExecutionLog   // 执行详细日志
	logMutex      sync.RWMutex              // 日志锁
	// 环境变量管理
	envVariables map[string]EnvVariableData // 环境变量存储（支持分隔符）
	envMutex     sync.RWMutex               // 环境变量锁
}

// SuccessCondition - 成功条件配置
type SuccessCondition struct {
	Enabled       bool   `json:"enabled"`
	JsonPath      string `json:"jsonPath"` // JSON路径（用于JSON路径判断）
	Operator      string `json:"operator"` // equals, not_equals, contains, not_contains, response_contains, response_not_contains, response_equals, response_not_equals
	ExpectedValue string `json:"expectedValue"`
}

// EnvVariableData - 环境变量数据结构（支持分隔符）
type EnvVariableData struct {
	Value     string `json:"value"`
	Separator string `json:"separator"`
}

// Task - 简化的任务结构，优化内存使用
type Task struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	URL              string            `json:"url"`
	Method           string            `json:"method"`
	Headers          map[string]string `json:"headers"`     // 使用map，更高效
	HeadersText      string            `json:"headersText"` // 用于前端显示和编辑
	Data             string            `json:"data"`
	Times            int               `json:"times"`
	Threads          int               `json:"threads"`
	DelayMin         int               `json:"delayMin"`
	DelayMax         int               `json:"delayMax"`
	Tags             []string          `json:"tags"`
	CronExpr         string            `json:"cronExpr"`
	SuccessCondition SuccessCondition  `json:"successCondition"` // 成功条件配置
	CreatedAt        int64             `json:"createdAt"`        // 时间戳，更高效
	UpdatedAt        int64             `json:"updatedAt"`
	IsRunning        bool              `json:"isRunning"`
	LastRunTime      string            `json:"lastRunTime"`   // 最后执行时间
	LastRunStatus    string            `json:"lastRunStatus"` // 最后执行状态: success, failed, running
	LastRunResult    string            `json:"lastRunResult"` // 最后执行结果描述
}

// TaskProgress - 简化的进度结构
type TaskProgress struct {
	Current   int   `json:"current"`
	Total     int   `json:"total"`
	StartTime int64 `json:"startTime"`
	IsRunning bool  `json:"isRunning"`
}

// TaskList - 任务列表响应
type TaskList struct {
	Tasks map[string]*Task `json:"tasks"`
	Total int              `json:"total"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		runningTasks:  make(map[string]*TaskProgress),
		cronJobs:      make(map[string]cron.EntryID),
		tasksCache:    make(map[string]*Task),
		taskLogs:      make(map[string][]TaskLogEntry),
		executionLogs: make(map[string]ExecutionLog),
		envVariables:  make(map[string]EnvVariableData),
		// 支持秒字段的cron调度器
		cronScheduler: cron.New(cron.WithSeconds()),
	}
}

// OnStartup is called when the app starts up
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	a.cronScheduler.Start()
}

// OnDomReady is called after front-end resources have been loaded
func (a *App) OnDomReady(ctx context.Context) {
	// 预加载任务数据
	go a.preloadTasks()
	// 恢复定时任务状态
	go a.restoreScheduledTasks()
	// 加载历史日志数据
	go a.loadHistoryLogs()
	// 加载环境变量
	go a.loadEnvVariables()
}

// OnShutdown is called when the app is shutting down
func (a *App) OnShutdown(ctx context.Context) {
	a.cronScheduler.Stop()
}

// preloadTasks 预加载任务数据到缓存
func (a *App) preloadTasks() {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	tasks := a.loadTasksFromDisk()
	a.tasksCache = tasks
	a.lastCacheTime = time.Now()
}

// getTasksPath 获取任务文件路径
func (a *App) getTasksPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "tasks.json"
	}

	exeDir := filepath.Dir(exePath)
	dataDir := filepath.Join(exeDir, "data")
	os.MkdirAll(dataDir, 0755)

	return filepath.Join(dataDir, "tasks.json")
}

// loadTasksFromDisk 从磁盘加载任务
func (a *App) loadTasksFromDisk() map[string]*Task {
	tasks := make(map[string]*Task)

	data, err := os.ReadFile(a.getTasksPath())
	if err != nil {
		return tasks
	}

	json.Unmarshal(data, &tasks)
	return tasks
}

// saveTasksToDisk 保存任务到磁盘
func (a *App) saveTasksToDisk(tasks map[string]*Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(a.getTasksPath(), data, 0644)
}

// GetTasks 获取任务列表（带分页）
func (a *App) GetTasks(page, pageSize int) *TaskList {
	a.cacheMutex.RLock()

	// 检查缓存是否需要更新
	if time.Since(a.lastCacheTime) > 5*time.Second {
		a.cacheMutex.RUnlock()
		a.preloadTasks()
		a.cacheMutex.RLock()
	}

	tasks := make(map[string]*Task)
	total := len(a.tasksCache)

	// 简单分页逻辑
	start := (page - 1) * pageSize
	end := start + pageSize

	i := 0
	for id, task := range a.tasksCache {
		if i >= start && i < end {
			tasks[id] = task
		}
		i++
		if i >= end {
			break
		}
	}

	a.cacheMutex.RUnlock()

	return &TaskList{
		Tasks: tasks,
		Total: total,
	}
}

// GetTaskCount 获取任务总数
func (a *App) GetTaskCount() int {
	a.cacheMutex.RLock()
	defer a.cacheMutex.RUnlock()
	return len(a.tasksCache)
}

// SaveTask 保存任务
func (a *App) SaveTask(name, url, method, headersText, data string, times, threads, delayMin, delayMax int, tags []string, cronExpr string, successCondition SuccessCondition) string {
	if name == "" || url == "" {
		return "错误：任务名称和URL不能为空"
	}

	// 生成任务ID
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())

	// 解析headers文本
	headers := a.parseHeadersText(headersText)

	task := &Task{
		ID:               taskID,
		Name:             name,
		URL:              url,
		Method:           method,
		Headers:          headers,
		HeadersText:      headersText,
		Data:             data,
		Times:            times,
		Threads:          threads,
		DelayMin:         delayMin,
		DelayMax:         delayMax,
		Tags:             tags,
		CronExpr:         cronExpr,
		SuccessCondition: successCondition,
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
		IsRunning:        false,
	}

	// 更新缓存和磁盘
	a.cacheMutex.Lock()
	a.tasksCache[taskID] = task
	tasks := make(map[string]*Task)
	for k, v := range a.tasksCache {
		tasks[k] = v
	}
	a.cacheMutex.Unlock()

	if err := a.saveTasksToDisk(tasks); err != nil {
		return fmt.Sprintf("保存失败：%v", err)
	}

	return fmt.Sprintf("任务 '%s' 保存成功", name)
}

// UpdateTask 更新任务
func (a *App) UpdateTask(taskID, name, url, method, headersText, data string, times, threads, delayMin, delayMax int, tags []string, cronExpr string, successCondition SuccessCondition) string {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	task, exists := a.tasksCache[taskID]
	if !exists {
		return "错误：任务不存在"
	}

	// 更新任务信息
	task.Name = name
	task.URL = url
	task.Method = method
	task.Headers = a.parseHeadersText(headersText)
	task.HeadersText = headersText
	task.Data = data
	task.Times = times
	task.Threads = threads
	task.DelayMin = delayMin
	task.DelayMax = delayMax
	task.Tags = tags
	task.CronExpr = cronExpr
	task.SuccessCondition = successCondition
	task.UpdatedAt = time.Now().Unix()

	// 保存到磁盘
	tasks := make(map[string]*Task)
	for k, v := range a.tasksCache {
		tasks[k] = v
	}

	if err := a.saveTasksToDisk(tasks); err != nil {
		return fmt.Sprintf("更新失败：%v", err)
	}

	return fmt.Sprintf("任务 '%s' 更新成功", name)
}

// DeleteTask 删除任务
func (a *App) DeleteTask(taskID string) string {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	task, exists := a.tasksCache[taskID]
	if !exists {
		return "错误：任务不存在"
	}

	name := task.Name
	delete(a.tasksCache, taskID)

	// 保存到磁盘
	tasks := make(map[string]*Task)
	for k, v := range a.tasksCache {
		tasks[k] = v
	}

	if err := a.saveTasksToDisk(tasks); err != nil {
		return fmt.Sprintf("删除失败：%v", err)
	}

	return fmt.Sprintf("任务 '%s' 删除成功", name)
}

// ExecuteTask 执行任务
func (a *App) ExecuteTask(taskID string) string {
	a.cacheMutex.RLock()
	task, exists := a.tasksCache[taskID]
	a.cacheMutex.RUnlock()

	if !exists {
		return "错误：任务不存在"
	}

	// 检查是否已在运行
	a.taskMutex.RLock()
	if _, running := a.runningTasks[taskID]; running {
		a.taskMutex.RUnlock()
		return "错误：任务正在运行中"
	}
	a.taskMutex.RUnlock()

	// 启动任务
	go a.runTask(task)

	return fmt.Sprintf("任务 '%s' 开始执行", task.Name)
}

// runTask 运行任务的核心逻辑
func (a *App) runTask(task *Task) {
	// 创建支持分隔符的任务副本列表
	tasksWithVars := a.createTasksWithSeparatedVariables(task)
	totalTasks := len(tasksWithVars)
	totalTimes := task.Times * totalTasks
	// 设置运行状态
	progress := &TaskProgress{
		Current:   0,
		Total:     totalTimes,
		StartTime: time.Now().Unix(),
		IsRunning: true,
	}

	a.taskMutex.Lock()
	a.runningTasks[task.ID] = progress
	a.taskMutex.Unlock()

	// 更新任务状态
	a.cacheMutex.Lock()
	a.tasksCache[task.ID].IsRunning = true
	a.cacheMutex.Unlock()

	// 创建详细日志收集器
	var detailedLogs []DetailedLogEntry
	detailLogsChan := make(chan DetailedLogEntry, totalTimes)

	// 创建工作通道
	jobs := make(chan *Task, totalTimes)
	results := make(chan bool, totalTimes)

	// 启动工作协程
	for w := 0; w < task.Threads; w++ {
		go a.workerWithDetailedLogForTask(jobs, results, detailLogsChan)
	}

	// 发送任务（每个任务副本执行指定次数）
	for _, taskVar := range tasksWithVars {
		for i := 0; i < task.Times; i++ {
			jobs <- taskVar
		}
	}
	close(jobs)

	// 等待完成并收集详细日志
	completed := 0
	successCount := 0
	for completed < totalTimes {
		success := <-results
		if success {
			successCount++
		}
		completed++

		// 收集详细日志
		select {
		case detailLog := <-detailLogsChan:
			detailedLogs = append(detailedLogs, detailLog)
		default:
		}

		// 更新进度
		a.taskMutex.Lock()
		if prog, exists := a.runningTasks[task.ID]; exists {
			prog.Current = completed
		}
		a.taskMutex.Unlock()
	}

	// 收集剩余的详细日志
	close(detailLogsChan)
	for detailLog := range detailLogsChan {
		detailedLogs = append(detailedLogs, detailLog)
	}

	// 记录任务完成（只记录关键结果）
	duration := time.Now().Unix() - progress.StartTime
	status := "success"
	if successCount == 0 {
		status = "failed"
	} else if successCount < totalTimes {
		status = "partial"
	}

	var message string
	if totalTasks > 1 {
		message = fmt.Sprintf("任务 '%s' 执行完成，耗时: %d秒，成功: %d/%d（分隔符产生%d个变体，每个执行%d次）",
			task.Name, duration, successCount, totalTimes, totalTasks, task.Times)
	} else {
		message = fmt.Sprintf("任务 '%s' 执行完成，耗时: %d秒，成功: %d/%d", task.Name, duration, successCount, totalTimes)
	}
	logID := a.writeTaskLog(task.ID, message, "execution", status)

	// 保存详细日志
	summary := fmt.Sprintf("执行完成，成功率: %.1f%%", float64(successCount)/float64(totalTimes)*100)
	a.writeExecutionLog(logID, detailedLogs, summary, totalTimes, successCount, totalTimes-successCount, duration)

	// 清理
	a.taskMutex.Lock()
	delete(a.runningTasks, task.ID)
	a.taskMutex.Unlock()

	a.cacheMutex.Lock()
	a.tasksCache[task.ID].IsRunning = false
	a.cacheMutex.Unlock()
}

// runTaskWithResult 运行任务并返回结果（用于定时任务）
func (a *App) runTaskWithResult(task *Task) (bool, int, string) {
	// 创建替换了环境变量的任务副本
	taskWithVars := a.createTaskWithVariables(task)
	// 设置运行状态
	progress := &TaskProgress{
		Current:   0,
		Total:     task.Times,
		StartTime: time.Now().Unix(),
		IsRunning: true,
	}

	a.taskMutex.Lock()
	a.runningTasks[task.ID] = progress
	a.taskMutex.Unlock()

	// 更新任务状态
	a.cacheMutex.Lock()
	a.tasksCache[task.ID].IsRunning = true
	a.cacheMutex.Unlock()

	// 创建详细日志收集器
	var detailedLogs []DetailedLogEntry
	detailLogsChan := make(chan DetailedLogEntry, task.Times)

	// 创建工作通道
	jobs := make(chan int, task.Times)
	results := make(chan bool, task.Times)

	// 启动工作协程
	for w := 0; w < task.Threads; w++ {
		go a.workerWithDetailedLog(taskWithVars, jobs, results, detailLogsChan)
	}

	// 发送任务
	for i := 0; i < task.Times; i++ {
		jobs <- i + 1
	}
	close(jobs)

	// 等待完成并收集详细日志
	completed := 0
	successCount := 0
	for completed < task.Times {
		success := <-results
		if success {
			successCount++
		}
		completed++

		// 收集详细日志
		select {
		case detailLog := <-detailLogsChan:
			detailedLogs = append(detailedLogs, detailLog)
		default:
		}

		// 更新进度
		a.taskMutex.Lock()
		if prog, exists := a.runningTasks[task.ID]; exists {
			prog.Current = completed
		}
		a.taskMutex.Unlock()
	}

	// 收集剩余的详细日志
	close(detailLogsChan)
	for detailLog := range detailLogsChan {
		detailedLogs = append(detailedLogs, detailLog)
	}

	// 记录任务完成（只记录关键结果）
	duration := time.Now().Unix() - progress.StartTime
	status := "success"
	if successCount == 0 {
		status = "failed"
	} else if successCount < task.Times {
		status = "partial"
	}

	message := fmt.Sprintf("定时任务 '%s' 执行完成，耗时: %d秒，成功: %d/%d", task.Name, duration, successCount, task.Times)
	logID := a.writeTaskLog(task.ID, message, "execution", status)

	// 保存详细日志
	summary := fmt.Sprintf("定时执行完成，成功率: %.1f%%", float64(successCount)/float64(task.Times)*100)
	a.writeExecutionLog(logID, detailedLogs, summary, task.Times, successCount, task.Times-successCount, duration)

	// 清理
	a.taskMutex.Lock()
	delete(a.runningTasks, task.ID)
	a.taskMutex.Unlock()

	a.cacheMutex.Lock()
	a.tasksCache[task.ID].IsRunning = false
	a.cacheMutex.Unlock()

	// 返回结果
	if successCount > 0 {
		return true, successCount, fmt.Sprintf("成功%d次，失败%d次", successCount, task.Times-successCount)
	} else {
		return false, 0, fmt.Sprintf("全部失败，共%d次请求", task.Times)
	}
}

// worker 工作协程
func (a *App) worker(task *Task, jobs <-chan int, results chan<- bool) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for range jobs {
		success := a.makeRequest(client, task)
		results <- success

		// 随机延迟
		if task.DelayMax > task.DelayMin {
			delay := task.DelayMin + rand.Intn(task.DelayMax-task.DelayMin)
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}
}

// workerWithDetailedLog 带详细日志的工作协程
func (a *App) workerWithDetailedLog(task *Task, jobs <-chan int, results chan<- bool, detailLogs chan<- DetailedLogEntry) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for range jobs {
		success, detailLog := a.makeRequestWithDetailedLog(client, task)
		results <- success
		detailLogs <- detailLog

		// 随机延迟
		if task.DelayMax > task.DelayMin {
			delay := task.DelayMin + rand.Intn(task.DelayMax-task.DelayMin)
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}
}

// workerWithDetailedLogForTask 支持分隔符的带详细日志工作协程
func (a *App) workerWithDetailedLogForTask(jobs <-chan *Task, results chan<- bool, detailLogs chan<- DetailedLogEntry) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for task := range jobs {
		success, detailLog := a.makeRequestWithDetailedLog(client, task)
		results <- success
		detailLogs <- detailLog

		// 随机延迟
		if task.DelayMax > task.DelayMin {
			delay := task.DelayMin + rand.Intn(task.DelayMax-task.DelayMin)
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}
}

// makeRequest 发送HTTP请求
func (a *App) makeRequest(client *http.Client, task *Task) bool {
	var body io.Reader
	if task.Data != "" {
		body = strings.NewReader(task.Data)
	}

	req, err := http.NewRequest(task.Method, task.URL, body)
	if err != nil {
		return false
	}

	// 设置headers
	for key, value := range task.Headers {
		req.Header.Set(key, value)
	}

	// 智能设置Content-Type（只在用户未设置时才自动设置）
	if task.Method != "GET" && task.Data != "" {
		if req.Header.Get("Content-Type") == "" {
			if strings.Contains(task.Data, "=") && strings.Contains(task.Data, "&") {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else if strings.HasPrefix(strings.TrimSpace(task.Data), "{") {
				req.Header.Set("Content-Type", "application/json")
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 读取响应体用于成功条件判断
	responseBody := ""
	if task.SuccessCondition.Enabled && task.SuccessCondition.JsonPath != "" {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			responseBody = string(bodyBytes)
		}
	}

	// 使用自定义成功条件判断
	return a.evaluateSuccessCondition(task, resp, responseBody)
}

// makeRequestWithDetailedLog 发送HTTP请求并记录详细日志
func (a *App) makeRequestWithDetailedLog(client *http.Client, task *Task) (bool, DetailedLogEntry) {
	startTime := time.Now()
	var body io.Reader
	if task.Data != "" {
		body = strings.NewReader(task.Data)
	}

	req, err := http.NewRequest(task.Method, task.URL, body)
	if err != nil {
		return false, a.addDetailedLogEntryWithError(task.ID, task.URL, task.Method, 0, 0, "", err.Error(), false, "network", fmt.Sprintf("创建HTTP请求失败: %v", err), nil)
	}

	// 设置headers
	for key, value := range task.Headers {
		req.Header.Set(key, value)
	}

	// 智能设置Content-Type（只在用户未设置时才自动设置）
	if task.Method != "GET" && task.Data != "" {
		if req.Header.Get("Content-Type") == "" {
			if strings.Contains(task.Data, "=") && strings.Contains(task.Data, "&") {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else if strings.HasPrefix(strings.TrimSpace(task.Data), "{") {
				req.Header.Set("Content-Type", "application/json")
			}
		}
	}

	resp, err := client.Do(req)
	responseTime := time.Since(startTime).Milliseconds()

	if err != nil {
		detailedError := fmt.Sprintf("网络请求失败: %v", err)
		return false, a.addDetailedLogEntryWithError(task.ID, task.URL, task.Method, 0, responseTime, "", err.Error(), false, "network", detailedError, nil)
	}
	defer resp.Body.Close()

	// 读取响应内容（限制大小）
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*10)) // 限制10KB
	responseStr := ""
	if err != nil {
		detailedError := fmt.Sprintf("读取响应内容失败: %v", err)
		return false, a.addDetailedLogEntryWithError(task.ID, task.URL, task.Method, resp.StatusCode, responseTime, "", err.Error(), false, "parsing", detailedError, nil)
	}
	responseStr = string(responseBody)

	// 使用增强的成功条件判断
	success, successConditionDetails := a.evaluateSuccessConditionWithDetails(task, resp, responseStr)

	var errorMsg, errorType, detailedError string
	if !success {
		if successConditionDetails != nil {
			// 成功条件失败
			errorType = "condition"
			errorMsg = "成功条件不满足"
			detailedError = a.generateConditionFailureDescription(successConditionDetails)
		} else {
			// HTTP状态码失败
			errorType = "http"
			errorMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
			detailedError = a.generateHttpErrorDescription(resp.StatusCode)
		}
	}

	detailLog := a.addDetailedLogEntryWithError(task.ID, task.URL, task.Method, resp.StatusCode, responseTime, responseStr, errorMsg, success, errorType, detailedError, successConditionDetails)
	return success, detailLog
}

// GetTaskProgress 获取任务进度
func (a *App) GetTaskProgress(taskID string) *TaskProgress {
	a.taskMutex.RLock()
	defer a.taskMutex.RUnlock()

	if progress, exists := a.runningTasks[taskID]; exists {
		return progress
	}

	return &TaskProgress{
		Current:   0,
		Total:     0,
		StartTime: 0,
		IsRunning: false,
	}
}

// StopTask 停止任务
func (a *App) StopTask(taskID string) string {
	a.taskMutex.Lock()
	defer a.taskMutex.Unlock()

	if _, exists := a.runningTasks[taskID]; !exists {
		return "错误：任务未在运行"
	}

	delete(a.runningTasks, taskID)

	a.cacheMutex.Lock()
	if task, exists := a.tasksCache[taskID]; exists {
		task.IsRunning = false
	}
	a.cacheMutex.Unlock()

	return "任务已停止"
}

// GetTaskLogs 获取任务执行日志（兼容性方法）
func (a *App) GetTaskLogs(taskID string) []string {
	logsPath := a.getTaskLogPath(taskID)

	data, err := os.ReadFile(logsPath)
	if err != nil {
		return []string{}
	}

	lines := strings.Split(string(data), "\n")
	// 过滤空行
	var logs []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			logs = append(logs, line)
		}
	}

	return logs
}

// GetTaskLogEntries 获取任务级别日志条目（按时间倒序）
func (a *App) GetTaskLogEntries(taskID string) []TaskLogEntry {
	a.logMutex.RLock()
	defer a.logMutex.RUnlock()

	if logs, exists := a.taskLogs[taskID]; exists {
		// 返回副本，避免并发问题
		result := make([]TaskLogEntry, len(logs))
		copy(result, logs)

		// 按时间倒序排列（最新的在前面）
		sort.Slice(result, func(i, j int) bool {
			timeI, errI := time.Parse("2006-01-02 15:04:05", result[i].Timestamp)
			timeJ, errJ := time.Parse("2006-01-02 15:04:05", result[j].Timestamp)

			// 如果时间解析失败，按ID倒序（ID包含时间戳）
			if errI != nil || errJ != nil {
				return result[i].ID > result[j].ID
			}

			return timeI.After(timeJ)
		})

		return result
	}

	return []TaskLogEntry{}
}

// GetExecutionLog 获取执行详细日志（按时间倒序）
func (a *App) GetExecutionLog(taskLogID string) *ExecutionLog {
	a.logMutex.RLock()
	defer a.logMutex.RUnlock()

	if log, exists := a.executionLogs[taskLogID]; exists {
		// 返回副本，避免并发问题
		result := log
		result.DetailedLogs = make([]DetailedLogEntry, len(log.DetailedLogs))
		copy(result.DetailedLogs, log.DetailedLogs)

		// 按时间倒序排列详细日志（最新的在前面）
		sort.Slice(result.DetailedLogs, func(i, j int) bool {
			timeI, errI := time.Parse("2006-01-02 15:04:05", result.DetailedLogs[i].Timestamp)
			timeJ, errJ := time.Parse("2006-01-02 15:04:05", result.DetailedLogs[j].Timestamp)

			// 如果时间解析失败，按RequestID倒序（RequestID包含时间戳）
			if errI != nil || errJ != nil {
				return result.DetailedLogs[i].RequestID > result.DetailedLogs[j].RequestID
			}

			return timeI.After(timeJ)
		})

		return &result
	}

	return nil
}

// ClearTaskLogs 清空任务日志
func (a *App) ClearTaskLogs(taskID string) string {
	a.logMutex.Lock()
	defer a.logMutex.Unlock()

	if taskID == "all" {
		// 清空所有任务的日志
		a.taskLogs = make(map[string][]TaskLogEntry)
		a.executionLogs = make(map[string]ExecutionLog)

		// 保存清空后的日志到磁盘
		go a.saveTaskLogs()
		go a.saveExecutionLogs()

		return "所有任务日志已清空"
	} else {
		// 清空指定任务的日志
		if _, exists := a.taskLogs[taskID]; !exists {
			return "任务日志不存在"
		}

		// 获取要删除的执行日志ID列表
		var executionLogIDs []string
		if logs, exists := a.taskLogs[taskID]; exists {
			for _, log := range logs {
				if log.ExecutionLogId != "" {
					executionLogIDs = append(executionLogIDs, log.ExecutionLogId)
				}
			}
		}

		// 删除任务级别日志
		delete(a.taskLogs, taskID)

		// 删除对应的详细执行日志
		for _, logID := range executionLogIDs {
			delete(a.executionLogs, logID)
		}

		// 保存更新后的日志到磁盘
		go a.saveTaskLogs()
		go a.saveExecutionLogs()

		return fmt.Sprintf("任务 '%s' 的日志已清空", taskID)
	}
}

// getTaskLogPath 获取任务日志文件路径
func (a *App) getTaskLogPath(taskID string) string {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Sprintf("logs/task_%s.log", taskID)
	}

	exeDir := filepath.Dir(exePath)
	logsDir := filepath.Join(exeDir, "logs")
	os.MkdirAll(logsDir, 0755)

	return filepath.Join(logsDir, fmt.Sprintf("task_%s.log", taskID))
}

// writeTaskLog 写入任务级别日志（简洁版本）
func (a *App) writeTaskLog(taskID, message, logType, status string) string {
	a.logMutex.Lock()
	defer a.logMutex.Unlock()

	// 生成日志条目ID
	logID := fmt.Sprintf("%s_%d", taskID, time.Now().UnixNano())
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	logEntry := TaskLogEntry{
		ID:        logID,
		Timestamp: timestamp,
		Message:   message,
		Type:      logType,
		Status:    status,
	}

	// 如果是执行类型的日志，设置ExecutionLogId
	if logType == "execution" {
		logEntry.ExecutionLogId = logID
	}

	// 添加到内存中的任务日志
	if a.taskLogs[taskID] == nil {
		a.taskLogs[taskID] = make([]TaskLogEntry, 0)
	}
	a.taskLogs[taskID] = append(a.taskLogs[taskID], logEntry)

	// 保持最近100条日志
	if len(a.taskLogs[taskID]) > 100 {
		a.taskLogs[taskID] = a.taskLogs[taskID][len(a.taskLogs[taskID])-100:]
	}

	// 同时写入文件（兼容性）
	logPath := a.getTaskLogPath(taskID)
	logFileEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)

	// 追加写入日志文件
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return logID
	}
	defer file.Close()

	file.WriteString(logFileEntry)

	// 异步保存任务级别日志到JSON文件
	go a.saveTaskLogs()

	return logID
}

// writeExecutionLog 写入执行详细日志
func (a *App) writeExecutionLog(taskLogID string, detailedLogs []DetailedLogEntry, summary string, totalRequests, successCount, failedCount int, duration int64) {
	a.logMutex.Lock()
	defer a.logMutex.Unlock()

	executionLog := ExecutionLog{
		TaskLogID:     taskLogID,
		DetailedLogs:  detailedLogs,
		Summary:       summary,
		TotalRequests: totalRequests,
		SuccessCount:  successCount,
		FailedCount:   failedCount,
		Duration:      duration,
	}

	a.executionLogs[taskLogID] = executionLog

	// 异步保存详细执行日志到JSON文件
	go a.saveExecutionLogs()
}

// addDetailedLogEntry 添加详细日志条目（保持向后兼容）
func (a *App) addDetailedLogEntry(taskID, url, method string, statusCode int, responseTime int64, response, errorMsg string, success bool) DetailedLogEntry {
	return a.addDetailedLogEntryWithError(taskID, url, method, statusCode, responseTime, response, errorMsg, success, "", "", nil)
}

// addDetailedLogEntryWithError 添加带详细错误信息的日志条目
func (a *App) addDetailedLogEntryWithError(taskID, url, method string, statusCode int, responseTime int64, response, errorMsg string, success bool, errorType, detailedError string, successConditionDetails *SuccessConditionDetails) DetailedLogEntry {
	requestID := fmt.Sprintf("%s_%d", taskID, time.Now().UnixNano())
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	return DetailedLogEntry{
		RequestID:               requestID,
		Timestamp:               timestamp,
		URL:                     url,
		Method:                  method,
		StatusCode:              statusCode,
		ResponseTime:            responseTime,
		Response:                response,
		Error:                   errorMsg,
		Success:                 success,
		SuccessConditionDetails: successConditionDetails,
		ErrorType:               errorType,
		DetailedError:           detailedError,
	}
}

// parseHeadersText 解析headers文本为map
func (a *App) parseHeadersText(headersText string) map[string]string {
	headers := make(map[string]string)

	if headersText == "" {
		return headers
	}

	lines := strings.Split(headersText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		colonIndex := strings.Index(line, ":")
		if colonIndex > 0 {
			key := strings.TrimSpace(line[:colonIndex])
			value := strings.TrimSpace(line[colonIndex+1:])
			if key != "" && value != "" {
				headers[key] = value
			}
		}
	}

	return headers
}

// ScheduleTask 添加定时任务
func (a *App) ScheduleTask(taskID string) string {
	a.cacheMutex.RLock()
	task, exists := a.tasksCache[taskID]
	a.cacheMutex.RUnlock()

	if !exists {
		return "错误：任务不存在"
	}

	if task.CronExpr == "" {
		return "错误：任务没有设置定时表达式"
	}

	a.cronMutex.Lock()
	defer a.cronMutex.Unlock()

	// 如果已经有定时任务，先移除
	if entryID, exists := a.cronJobs[taskID]; exists {
		a.cronScheduler.Remove(entryID)
	}

	// 添加新的定时任务
	entryID, err := a.cronScheduler.AddFunc(task.CronExpr, func() {
		// 不记录定时任务触发日志，这属于系统级别日志

		// 更新最后执行时间和状态
		a.updateLastRunInfo(taskID, "running", "定时任务执行中...")

		// 执行任务
		go func() {
			// 执行任务并获取结果
			success, completed, errorMsg := a.runTaskWithResult(task)

			// 更新执行结果
			if success {
				a.updateLastRunInfo(taskID, "success", fmt.Sprintf("执行成功，完成%d次请求", completed))
			} else {
				a.updateLastRunInfo(taskID, "failed", errorMsg)
			}
		}()
	})

	if err != nil {
		return fmt.Sprintf("添加定时任务失败: %v", err)
	}

	a.cronJobs[taskID] = entryID
	// 不记录调度添加日志，这属于系统级别日志

	// 保存定时任务状态到文件
	go a.saveScheduledTasks()

	return fmt.Sprintf("任务 '%s' 已添加到定时调度", task.Name)
}

// UnscheduleTask 移除定时任务
func (a *App) UnscheduleTask(taskID string) string {
	a.cronMutex.Lock()
	defer a.cronMutex.Unlock()

	entryID, exists := a.cronJobs[taskID]
	if !exists {
		return "任务没有定时调度"
	}

	a.cronScheduler.Remove(entryID)
	delete(a.cronJobs, taskID)

	// 保存定时任务状态到文件
	go a.saveScheduledTasks()

	a.cacheMutex.RLock()
	task, exists := a.tasksCache[taskID]
	a.cacheMutex.RUnlock()

	if exists {
		// 不记录调度移除日志，这属于系统级别日志
		return fmt.Sprintf("任务 '%s' 已从定时调度中移除", task.Name)
	}

	return "定时任务已移除"
}

// GetScheduledTasks 获取所有定时任务
func (a *App) GetScheduledTasks() []string {
	a.cronMutex.RLock()
	defer a.cronMutex.RUnlock()

	var scheduledTasks []string
	for taskID := range a.cronJobs {
		scheduledTasks = append(scheduledTasks, taskID)
	}

	return scheduledTasks
}

// TaskLogEntry 任务级别日志条目
type TaskLogEntry struct {
	ID             string `json:"id"`             // 日志条目唯一ID
	Timestamp      string `json:"timestamp"`      // 时间戳
	Message        string `json:"message"`        // 日志消息
	Type           string `json:"type"`           // 日志类型: execution, schedule, system
	Status         string `json:"status"`         // 执行状态: success, failed, running
	ExecutionLogId string `json:"executionLogId"` // 关联的详细执行日志ID
}

// DetailedLogEntry 详细日志条目
type DetailedLogEntry struct {
	RequestID               string                   `json:"requestId"`               // 请求ID
	Timestamp               string                   `json:"timestamp"`               // 时间戳
	URL                     string                   `json:"url"`                     // 请求URL
	Method                  string                   `json:"method"`                  // 请求方法
	StatusCode              int                      `json:"statusCode"`              // 响应状态码
	ResponseTime            int64                    `json:"responseTime"`            // 响应时间(毫秒)
	Response                string                   `json:"response"`                // 响应内容
	Error                   string                   `json:"error"`                   // 错误信息
	Success                 bool                     `json:"success"`                 // 基于自定义成功条件的判断结果
	SuccessConditionDetails *SuccessConditionDetails `json:"successConditionDetails"` // 成功条件评估详情
	ErrorType               string                   `json:"errorType"`               // 错误类型: network, parsing, condition, http
	DetailedError           string                   `json:"detailedError"`           // 详细错误描述
}

// ExecutionLog 执行日志（包含任务级别和详细日志）
type ExecutionLog struct {
	TaskLogID     string             `json:"taskLogId"`     // 对应的任务日志ID
	DetailedLogs  []DetailedLogEntry `json:"detailedLogs"`  // 详细日志列表
	Summary       string             `json:"summary"`       // 执行摘要
	TotalRequests int                `json:"totalRequests"` // 总请求数
	SuccessCount  int                `json:"successCount"`  // 成功数
	FailedCount   int                `json:"failedCount"`   // 失败数
	Duration      int64              `json:"duration"`      // 执行时长(秒)
}

// TaskScheduleInfo 任务调度信息
type TaskScheduleInfo struct {
	TaskID          string `json:"taskId"`
	IsScheduled     bool   `json:"isScheduled"`
	CronExpr        string `json:"cronExpr"`
	NextRunTime     string `json:"nextRunTime"`
	CronDescription string `json:"cronDescription"`
	Status          string `json:"status"` // idle, scheduled, running, error
	LastRunTime     string `json:"lastRunTime"`
	LastRunStatus   string `json:"lastRunStatus"` // success, failed, running
	LastRunResult   string `json:"lastRunResult"`
}

// GetTaskScheduleInfo 获取任务调度信息
func (a *App) GetTaskScheduleInfo(taskID string) TaskScheduleInfo {
	a.cacheMutex.RLock()
	task, exists := a.tasksCache[taskID]
	a.cacheMutex.RUnlock()

	info := TaskScheduleInfo{
		TaskID:      taskID,
		IsScheduled: false,
		Status:      "idle",
	}

	if !exists {
		info.Status = "error"
		return info
	}

	info.CronExpr = task.CronExpr
	info.LastRunTime = task.LastRunTime
	info.LastRunStatus = task.LastRunStatus
	info.LastRunResult = task.LastRunResult

	// 检查是否正在运行
	if task.IsRunning {
		info.Status = "running"
		return info
	}

	// 检查是否有定时调度
	a.cronMutex.RLock()
	entryID, isScheduled := a.cronJobs[taskID]
	a.cronMutex.RUnlock()

	if isScheduled && task.CronExpr != "" {
		info.IsScheduled = true
		info.Status = "scheduled"

		// 计算下次执行时间
		nextTime, err := a.getNextRunTime(task.CronExpr)
		if err != nil {
			info.Status = "error"
			info.NextRunTime = "计算失败: " + err.Error()
		} else {
			info.NextRunTime = nextTime
		}

		// 生成人性化描述
		info.CronDescription = a.describeCronExpr(task.CronExpr)

		// 验证entry是否还存在
		entry := a.cronScheduler.Entry(entryID)
		if entry.ID == 0 {
			info.Status = "error"
			info.NextRunTime = "调度已失效"
		}
	} else if task.CronExpr != "" {
		info.Status = "idle"
		info.CronDescription = a.describeCronExpr(task.CronExpr)
	}

	return info
}

// updateLastRunInfo 更新任务的最后执行信息
func (a *App) updateLastRunInfo(taskID, status, result string) {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	task, exists := a.tasksCache[taskID]
	if !exists {
		return
	}

	task.LastRunTime = time.Now().Format("2006-01-02 15:04:05")
	task.LastRunStatus = status
	task.LastRunResult = result

	// 不记录状态更新日志，这属于系统级别日志
}

// getNextRunTime 计算下次执行时间
func (a *App) getNextRunTime(cronExpr string) (string, error) {
	fields := strings.Fields(cronExpr)
	var schedule cron.Schedule
	var err error

	if len(fields) == 6 {
		// 6字段格式（秒 分 时 日 月 周）- 使用支持秒的解析器
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err = parser.Parse(cronExpr)
	} else if len(fields) == 5 {
		// 5字段格式（分 时 日 月 周）- 使用标准解析器
		schedule, err = cron.ParseStandard(cronExpr)
	} else {
		return "", fmt.Errorf("Cron表达式格式错误：期望5字段或6字段，实际%d字段", len(fields))
	}

	if err != nil {
		return "", fmt.Errorf("解析Cron表达式失败: %v", err)
	}

	nextTime := schedule.Next(time.Now())
	return nextTime.Format("2006-01-02 15:04:05"), nil
}

// describeCronExpr 生成Cron表达式的人性化描述
func (a *App) describeCronExpr(cronExpr string) string {
	if cronExpr == "" {
		return ""
	}

	fields := strings.Fields(cronExpr)

	var second, minute, hour, day, month, weekday string

	if len(fields) == 6 {
		// 6字段格式：秒 分 时 日 月 周
		second, minute, hour, day, month, weekday = fields[0], fields[1], fields[2], fields[3], fields[4], fields[5]
	} else if len(fields) == 5 {
		// 5字段格式：分 时 日 月 周
		second = "0"
		minute, hour, day, month, weekday = fields[0], fields[1], fields[2], fields[3], fields[4]
	} else {
		return cronExpr
	}

	// 简单的描述生成逻辑
	if minute == "*" && hour == "*" && day == "*" && month == "*" && weekday == "*" {
		if second == "0" || second == "*" {
			return "每分钟"
		}
		return fmt.Sprintf("每分钟第%s秒", second)
	}

	if hour == "*" && day == "*" && month == "*" && weekday == "*" {
		if minute == "0" {
			return "每小时整点"
		}
		if strings.HasPrefix(minute, "*/") {
			interval := strings.TrimPrefix(minute, "*/")
			return fmt.Sprintf("每%s分钟", interval)
		}
		return fmt.Sprintf("每小时第%s分钟", minute)
	}

	if day == "*" && month == "*" && weekday == "*" {
		if minute == "0" && (second == "0" || second == "*") {
			if strings.HasPrefix(hour, "*/") {
				interval := strings.TrimPrefix(hour, "*/")
				return fmt.Sprintf("每%s小时", interval)
			}
			return fmt.Sprintf("每天%s点", hour)
		}
		if second == "0" || second == "*" {
			return fmt.Sprintf("每天%s:%s", hour, minute)
		}
		return fmt.Sprintf("每天%s:%s:%s", hour, minute, second)
	}

	if month == "*" && weekday != "*" {
		weekdayDesc := ""
		switch weekday {
		case "1-5":
			weekdayDesc = "工作日"
		case "6,0", "0,6":
			weekdayDesc = "周末"
		default:
			weekdayDesc = "每周" + weekday
		}

		if minute == "0" {
			return fmt.Sprintf("%s %s点", weekdayDesc, hour)
		}
		return fmt.Sprintf("%s %s:%s", weekdayDesc, hour, minute)
	}

	// 默认返回原表达式
	return cronExpr
}

// getScheduledTasksPath 获取定时任务状态文件路径
func (a *App) getScheduledTasksPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "scheduled_tasks.json"
	}

	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "scheduled_tasks.json")
}

// saveScheduledTasks 保存定时任务状态到文件
func (a *App) saveScheduledTasks() error {
	a.cronMutex.RLock()
	defer a.cronMutex.RUnlock()

	// 只保存任务ID列表，因为cron表达式已经在任务配置中
	scheduledTaskIDs := make([]string, 0, len(a.cronJobs))
	for taskID := range a.cronJobs {
		scheduledTaskIDs = append(scheduledTaskIDs, taskID)
	}

	data, err := json.MarshalIndent(scheduledTaskIDs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(a.getScheduledTasksPath(), data, 0644)
}

// restoreScheduledTasks 从文件恢复定时任务状态
func (a *App) restoreScheduledTasks() {
	scheduledPath := a.getScheduledTasksPath()

	// 检查文件是否存在
	if _, err := os.Stat(scheduledPath); os.IsNotExist(err) {
		return // 文件不存在，跳过恢复
	}

	data, err := os.ReadFile(scheduledPath)
	if err != nil {
		return // 读取失败，跳过恢复
	}

	var scheduledTaskIDs []string
	if err := json.Unmarshal(data, &scheduledTaskIDs); err != nil {
		return // 解析失败，跳过恢复
	}

	// 等待任务缓存加载完成
	time.Sleep(1 * time.Second)

	// 恢复每个定时任务
	for _, taskID := range scheduledTaskIDs {
		a.cacheMutex.RLock()
		task, exists := a.tasksCache[taskID]
		a.cacheMutex.RUnlock()

		if !exists || task.CronExpr == "" {
			continue // 任务不存在或没有cron表达式，跳过
		}

		// 重新添加定时任务
		a.cronMutex.Lock()
		entryID, err := a.cronScheduler.AddFunc(task.CronExpr, func() {
			// 更新最后执行时间和状态
			a.updateLastRunInfo(taskID, "running", "定时任务执行中...")

			// 执行任务
			success, _, result := a.runTaskWithResult(task)

			// 更新执行结果
			status := "success"
			if !success {
				status = "failed"
			}
			a.updateLastRunInfo(taskID, status, result)
		})

		if err == nil {
			a.cronJobs[taskID] = entryID
		}
		a.cronMutex.Unlock()
	}
}

// TestTask 测试任务（单次请求）
func (a *App) TestTask(taskID string) string {
	a.cacheMutex.RLock()
	task, exists := a.tasksCache[taskID]
	a.cacheMutex.RUnlock()

	if !exists {
		return "错误：任务不存在"
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 发送详细的测试请求
	result := a.makeDetailedRequest(client, task)
	return result
}

// SuccessConditionDetails 成功条件评估详情
type SuccessConditionDetails struct {
	Type          string `json:"type"`          // "json_path" 或 "string_based"
	JsonPath      string `json:"jsonPath"`      // JSON路径（仅当type为json_path时）
	Operator      string `json:"operator"`      // 操作符
	ExpectedValue string `json:"expectedValue"` // 期望值
	ActualValue   string `json:"actualValue"`   // 实际值
	Result        bool   `json:"result"`        // 判断结果
	Reason        string `json:"reason"`        // 详细说明
}

// TestTaskResult 测试结果结构体
type TestTaskResult struct {
	Success                 bool                     `json:"success"`
	StatusCode              int                      `json:"statusCode"`
	StatusText              string                   `json:"statusText"`
	ResponseTime            int64                    `json:"responseTime"`
	RequestHeaders          map[string]string        `json:"requestHeaders"`
	ResponseHeaders         map[string]string        `json:"responseHeaders"`
	ResponseBody            string                   `json:"responseBody"`
	Error                   string                   `json:"error"`
	RequestURL              string                   `json:"requestUrl"`
	RequestMethod           string                   `json:"requestMethod"`
	RequestBodySize         int                      `json:"requestBodySize"`
	SensitiveHeaders        []string                 `json:"sensitiveHeaders"`
	SuccessConditionDetails *SuccessConditionDetails `json:"successConditionDetails"`
}

// TestTaskWithBackend 使用后端发送HTTP请求（绕过浏览器限制）
func (a *App) TestTaskWithBackend(taskID string) TestTaskResult {
	a.cacheMutex.RLock()
	task, exists := a.tasksCache[taskID]
	a.cacheMutex.RUnlock()

	if !exists {
		return TestTaskResult{
			Success: false,
			Error:   "错误：任务不存在",
		}
	}

	// 创建替换了环境变量的任务副本
	taskWithVars := a.createTaskWithVariables(task)
	return a.makeDetailedRequestWithResult(taskWithVars)
}

// TestTaskDataWithBackend 直接使用任务数据测试（不需要保存任务）
func (a *App) TestTaskDataWithBackend(name, url, method, headersText, data string, successCondition SuccessCondition) TestTaskResult {
	// 解析headers
	headers := a.parseHeadersText(headersText)

	task := &Task{
		Name:             name,
		URL:              url,
		Method:           method,
		Headers:          headers,
		HeadersText:      headersText,
		Data:             data,
		SuccessCondition: successCondition,
	}

	// 创建替换了环境变量的任务副本
	taskWithVars := a.createTaskWithVariables(task)
	return a.makeDetailedRequestWithResult(taskWithVars)
}

// makeDetailedRequestWithResult 发送详细的HTTP请求并返回结构化结果
func (a *App) makeDetailedRequestWithResult(task *Task) TestTaskResult {
	startTime := time.Now()

	result := TestTaskResult{
		RequestURL:      task.URL,
		RequestMethod:   task.Method,
		RequestBodySize: len(task.Data),
		RequestHeaders:  make(map[string]string),
		ResponseHeaders: make(map[string]string),
	}

	// 检查敏感headers
	sensitiveHeaders := []string{}
	for key, value := range task.Headers {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "cookie") ||
			strings.Contains(lowerKey, "authorization") ||
			strings.Contains(lowerKey, "token") {
			sensitiveHeaders = append(sensitiveHeaders, key)
			// 脱敏处理
			if len(value) > 20 {
				result.RequestHeaders[key] = value[:10] + "***" + value[len(value)-7:]
			} else {
				result.RequestHeaders[key] = "***"
			}
		} else {
			result.RequestHeaders[key] = value
		}
	}
	result.SensitiveHeaders = sensitiveHeaders

	var body io.Reader
	if task.Data != "" {
		body = strings.NewReader(task.Data)
	}

	req, err := http.NewRequest(task.Method, task.URL, body)
	if err != nil {
		result.Error = fmt.Sprintf("创建请求失败: %v", err)
		return result
	}

	// 设置所有headers（后端没有浏览器限制）
	for key, value := range task.Headers {
		req.Header.Set(key, value)
	}

	// 智能设置Content-Type
	if task.Method != "GET" && task.Data != "" {
		if req.Header.Get("Content-Type") == "" {
			if strings.Contains(task.Data, "=") && strings.Contains(task.Data, "&") {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else if strings.HasPrefix(strings.TrimSpace(task.Data), "{") {
				req.Header.Set("Content-Type", "application/json")
			}
		}
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
		// 不跟随重定向，让用户看到原始响应
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("请求失败: %v", err)
		result.ResponseTime = time.Since(startTime).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.StatusText = resp.Status
	result.ResponseTime = time.Since(startTime).Milliseconds()

	// 获取响应headers
	for key, values := range resp.Header {
		if len(values) > 0 {
			result.ResponseHeaders[key] = values[0]
		}
	}

	// 读取响应内容
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("读取响应失败: %v", err)
		return result
	}

	// 限制响应内容长度
	respContent := string(respBody)
	if len(respContent) > 5000 {
		result.ResponseBody = respContent[:5000] + "\n\n... (内容过长，已截断，总长度: " +
			fmt.Sprintf("%d", len(respContent)) + " 字符)"
	} else {
		result.ResponseBody = respContent
	}

	// 使用自定义成功条件判断（与正式执行保持一致）
	success, details := a.evaluateSuccessConditionWithDetails(task, resp, respContent)
	result.Success = success
	result.SuccessConditionDetails = details

	return result
}

// makeDetailedRequest 保持向后兼容的简单版本
func (a *App) makeDetailedRequest(client *http.Client, task *Task) string {
	result := a.makeDetailedRequestWithResult(task)

	if result.Error != "" {
		return fmt.Sprintf("任务 '%s' 测试失败: %s", task.Name, result.Error)
	}

	if result.Success {
		return fmt.Sprintf("任务 '%s' 测试成功\n状态码: %d\n响应时间: %dms\n响应内容: %s",
			task.Name, result.StatusCode, result.ResponseTime, result.ResponseBody)
	} else {
		return fmt.Sprintf("任务 '%s' 测试失败\n状态码: %d\n响应时间: %dms\n响应内容: %s",
			task.Name, result.StatusCode, result.ResponseTime, result.ResponseBody)
	}
}

// getTaskLogsPath 获取任务级别日志文件路径
func (a *App) getTaskLogsPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "task_logs.json"
	}

	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "task_logs.json")
}

// getExecutionLogsPath 获取详细执行日志文件路径
func (a *App) getExecutionLogsPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "execution_logs.json"
	}

	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "execution_logs.json")
}

// saveTaskLogs 保存任务级别日志到文件
func (a *App) saveTaskLogs() error {
	a.logMutex.RLock()
	defer a.logMutex.RUnlock()

	data, err := json.MarshalIndent(a.taskLogs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(a.getTaskLogsPath(), data, 0644)
}

// saveExecutionLogs 保存详细执行日志到文件
func (a *App) saveExecutionLogs() error {
	a.logMutex.RLock()
	defer a.logMutex.RUnlock()

	data, err := json.MarshalIndent(a.executionLogs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(a.getExecutionLogsPath(), data, 0644)
}

// loadHistoryLogs 加载历史日志数据
func (a *App) loadHistoryLogs() {
	// 加载任务级别日志
	taskLogsPath := a.getTaskLogsPath()
	if data, err := os.ReadFile(taskLogsPath); err == nil {
		var taskLogs map[string][]TaskLogEntry
		if err := json.Unmarshal(data, &taskLogs); err == nil {
			a.logMutex.Lock()
			a.taskLogs = taskLogs
			a.logMutex.Unlock()
		}
	}

	// 加载详细执行日志
	executionLogsPath := a.getExecutionLogsPath()
	if data, err := os.ReadFile(executionLogsPath); err == nil {
		var executionLogs map[string]ExecutionLog
		if err := json.Unmarshal(data, &executionLogs); err == nil {
			a.logMutex.Lock()
			a.executionLogs = executionLogs
			a.logMutex.Unlock()
		}
	}

	// 清理过期日志（保留最近30天）
	a.cleanupOldLogs()
}

// cleanupOldLogs 清理过期日志
func (a *App) cleanupOldLogs() {
	a.logMutex.Lock()
	defer a.logMutex.Unlock()

	cutoffTime := time.Now().AddDate(0, 0, -30) // 30天前

	// 清理任务级别日志
	for taskID, logs := range a.taskLogs {
		var filteredLogs []TaskLogEntry
		for _, log := range logs {
			if logTime, err := time.Parse("2006-01-02 15:04:05", log.Timestamp); err == nil {
				if logTime.After(cutoffTime) {
					filteredLogs = append(filteredLogs, log)
				}
			} else {
				// 如果解析时间失败，保留日志
				filteredLogs = append(filteredLogs, log)
			}
		}

		if len(filteredLogs) == 0 {
			delete(a.taskLogs, taskID)
		} else {
			a.taskLogs[taskID] = filteredLogs
		}
	}

	// 清理详细执行日志（基于任务级别日志的存在性）
	for logID := range a.executionLogs {
		// 检查对应的任务级别日志是否还存在
		found := false
		for _, logs := range a.taskLogs {
			for _, log := range logs {
				if log.ID == logID {
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			delete(a.executionLogs, logID)
		}
	}
}

// ==================== 环境变量管理 ====================

// getEnvVariablesPath 获取环境变量文件路径
func (a *App) getEnvVariablesPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "env_variables.json"
	}

	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "env_variables.json")
}

// loadEnvVariables 加载环境变量
func (a *App) loadEnvVariables() {
	envPath := a.getEnvVariablesPath()
	if data, err := os.ReadFile(envPath); err == nil {
		var envVars map[string]EnvVariableData
		if err := json.Unmarshal(data, &envVars); err == nil {
			a.envMutex.Lock()
			a.envVariables = envVars
			a.envMutex.Unlock()
		} else {
			// 尝试加载旧格式（字符串格式）
			var oldEnvVars map[string]string
			if err := json.Unmarshal(data, &oldEnvVars); err == nil {
				a.envMutex.Lock()
				a.envVariables = make(map[string]EnvVariableData)
				for k, v := range oldEnvVars {
					a.envVariables[k] = EnvVariableData{Value: v, Separator: ""}
				}
				a.envMutex.Unlock()
			}
		}
	}
}

// saveEnvVariables 保存环境变量到文件
func (a *App) saveEnvVariables() error {
	a.envMutex.RLock()
	defer a.envMutex.RUnlock()

	data, err := json.MarshalIndent(a.envVariables, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(a.getEnvVariablesPath(), data, 0644)
}

// GetEnvVariables 获取所有环境变量（向后兼容，返回字符串格式）
func (a *App) GetEnvVariables() map[string]string {
	a.envMutex.RLock()
	defer a.envMutex.RUnlock()

	result := make(map[string]string)
	for k, v := range a.envVariables {
		result[k] = v.Value
	}
	return result
}

// GetEnvVariablesWithSeparator 获取所有环境变量（包含分隔符信息）
func (a *App) GetEnvVariablesWithSeparator() map[string]EnvVariableData {
	a.envMutex.RLock()
	defer a.envMutex.RUnlock()

	result := make(map[string]EnvVariableData)
	for k, v := range a.envVariables {
		result[k] = v
	}
	return result
}

// SetEnvVariable 设置环境变量（向后兼容）
func (a *App) SetEnvVariable(key, value string) string {
	if key == "" {
		return "错误：变量名不能为空"
	}

	a.envMutex.Lock()
	a.envVariables[key] = EnvVariableData{Value: value, Separator: ""}
	a.envMutex.Unlock()

	if err := a.saveEnvVariables(); err != nil {
		return fmt.Sprintf("保存失败：%v", err)
	}

	return fmt.Sprintf("环境变量 '%s' 设置成功", key)
}

// SetEnvVariableWithSeparator 设置环境变量（支持分隔符）
func (a *App) SetEnvVariableWithSeparator(key, dataJson string) string {
	if key == "" {
		return "错误：变量名不能为空"
	}

	var data EnvVariableData
	if err := json.Unmarshal([]byte(dataJson), &data); err != nil {
		return fmt.Sprintf("数据格式错误：%v", err)
	}

	a.envMutex.Lock()
	a.envVariables[key] = data
	a.envMutex.Unlock()

	if err := a.saveEnvVariables(); err != nil {
		return fmt.Sprintf("保存失败：%v", err)
	}

	return fmt.Sprintf("环境变量 '%s' 设置成功", key)
}

// DeleteEnvVariable 删除环境变量
func (a *App) DeleteEnvVariable(key string) string {
	a.envMutex.Lock()
	defer a.envMutex.Unlock()

	if _, exists := a.envVariables[key]; !exists {
		return "错误：变量不存在"
	}

	delete(a.envVariables, key)

	if err := a.saveEnvVariables(); err != nil {
		return fmt.Sprintf("删除失败：%v", err)
	}

	return fmt.Sprintf("环境变量 '%s' 删除成功", key)
}

// UpdateEnvVariable 更新环境变量（向后兼容）
func (a *App) UpdateEnvVariable(key, value string) string {
	if key == "" {
		return "错误：变量名不能为空"
	}

	a.envMutex.Lock()
	defer a.envMutex.Unlock()

	if _, exists := a.envVariables[key]; !exists {
		return "错误：变量不存在"
	}

	// 保留原有的分隔符设置
	existing := a.envVariables[key]
	a.envVariables[key] = EnvVariableData{Value: value, Separator: existing.Separator}

	if err := a.saveEnvVariables(); err != nil {
		return fmt.Sprintf("更新失败：%v", err)
	}

	return fmt.Sprintf("环境变量 '%s' 更新成功", key)
}

// UpdateEnvVariableWithSeparator 更新环境变量（支持分隔符）
func (a *App) UpdateEnvVariableWithSeparator(key, dataJson string) string {
	if key == "" {
		return "错误：变量名不能为空"
	}

	var data EnvVariableData
	if err := json.Unmarshal([]byte(dataJson), &data); err != nil {
		return fmt.Sprintf("数据格式错误：%v", err)
	}

	a.envMutex.Lock()
	defer a.envMutex.Unlock()

	if _, exists := a.envVariables[key]; !exists {
		return "错误：变量不存在"
	}

	a.envVariables[key] = data

	if err := a.saveEnvVariables(); err != nil {
		return fmt.Sprintf("更新失败：%v", err)
	}

	return fmt.Sprintf("环境变量 '%s' 更新成功", key)
}

// PreviewTaskWithVariables 预览任务配置（替换环境变量后）
func (a *App) PreviewTaskWithVariables(taskID string) map[string]interface{} {
	a.cacheMutex.RLock()
	task, exists := a.tasksCache[taskID]
	a.cacheMutex.RUnlock()

	if !exists {
		return map[string]interface{}{
			"error": "任务不存在",
		}
	}

	// 创建任务副本并替换变量
	preview := map[string]interface{}{
		"id":          task.ID,
		"name":        task.Name,
		"url":         a.replaceVariables(task.URL),
		"method":      task.Method,
		"data":        a.replaceVariables(task.Data),
		"headersText": a.replaceVariables(task.HeadersText),
		"headers":     make(map[string]string),
	}

	// 替换headers中的变量
	headers := make(map[string]string)
	for k, v := range task.Headers {
		headers[a.replaceVariables(k)] = a.replaceVariables(v)
	}
	preview["headers"] = headers

	return preview
}

// replaceVariables 替换字符串中的环境变量
func (a *App) replaceVariables(text string) string {
	if text == "" {
		return text
	}

	a.envMutex.RLock()
	defer a.envMutex.RUnlock()

	result := text
	// 使用正则表达式查找 {{VARIABLE_NAME}} 格式的占位符
	// 为了避免无限递归，最多替换10次
	for i := 0; i < 10; i++ {
		oldResult := result
		for key, value := range a.envVariables {
			placeholder := fmt.Sprintf("{{%s}}", key)
			result = strings.ReplaceAll(result, placeholder, value.Value)
		}
		// 如果没有变化，说明替换完成
		if result == oldResult {
			break
		}
	}

	return result
}

// createTaskWithVariables 创建替换了环境变量的任务副本
func (a *App) createTaskWithVariables(task *Task) *Task {
	// 创建任务副本
	taskCopy := *task

	// 替换各个字段中的变量
	taskCopy.URL = a.replaceVariables(task.URL)
	taskCopy.Data = a.replaceVariables(task.Data)
	taskCopy.HeadersText = a.replaceVariables(task.HeadersText)

	// 替换headers中的变量
	taskCopy.Headers = make(map[string]string)
	for k, v := range task.Headers {
		newKey := a.replaceVariables(k)
		newValue := a.replaceVariables(v)
		taskCopy.Headers[newKey] = newValue
	}

	return &taskCopy
}

// createTasksWithSeparatedVariables 创建支持分隔符的任务副本列表
func (a *App) createTasksWithSeparatedVariables(task *Task) []*Task {
	a.envMutex.RLock()
	defer a.envMutex.RUnlock()

	// 查找包含分隔符的环境变量
	var separatedVars []map[string]string
	hasSeperatedVars := false

	for key, envVar := range a.envVariables {
		if envVar.Separator != "" {
			// 分割变量值
			separators := strings.Split(envVar.Separator, ",")
			values := []string{envVar.Value}

			for _, sep := range separators {
				sep = strings.TrimSpace(sep)
				if sep == "" {
					continue
				}

				var newValues []string
				for _, val := range values {
					newValues = append(newValues, strings.Split(val, sep)...)
				}
				values = newValues
			}

			// 清理空值
			var cleanValues []string
			for _, val := range values {
				val = strings.TrimSpace(val)
				if val != "" {
					cleanValues = append(cleanValues, val)
				}
			}

			if len(cleanValues) > 1 {
				hasSeperatedVars = true
				// 为每个分割后的值创建变量映射
				for i, val := range cleanValues {
					if i >= len(separatedVars) {
						separatedVars = append(separatedVars, make(map[string]string))
					}
					separatedVars[i][key] = val
				}
			}
		}
	}

	// 如果没有分隔符变量，返回单个任务
	if !hasSeperatedVars {
		return []*Task{a.createTaskWithVariables(task)}
	}

	// 创建多个任务副本，每个使用不同的变量值
	var tasks []*Task
	for _, varMap := range separatedVars {
		taskCopy := *task

		// 替换各个字段中的变量
		taskCopy.URL = a.replaceVariablesWithMap(task.URL, varMap)
		taskCopy.Data = a.replaceVariablesWithMap(task.Data, varMap)
		taskCopy.HeadersText = a.replaceVariablesWithMap(task.HeadersText, varMap)

		// 替换headers中的变量
		taskCopy.Headers = make(map[string]string)
		for k, v := range task.Headers {
			newKey := a.replaceVariablesWithMap(k, varMap)
			newValue := a.replaceVariablesWithMap(v, varMap)
			taskCopy.Headers[newKey] = newValue
		}

		tasks = append(tasks, &taskCopy)
	}

	return tasks
}

// replaceVariablesWithMap 使用指定的变量映射替换变量
func (a *App) replaceVariablesWithMap(text string, varMap map[string]string) string {
	if text == "" {
		return text
	}

	result := text
	// 首先使用指定的变量映射
	for key, value := range varMap {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// 然后使用其他环境变量
	a.envMutex.RLock()
	defer a.envMutex.RUnlock()

	for key, envVar := range a.envVariables {
		if _, exists := varMap[key]; !exists {
			placeholder := fmt.Sprintf("{{%s}}", key)
			result = strings.ReplaceAll(result, placeholder, envVar.Value)
		}
	}

	return result
}

// evaluateSuccessCondition 评估成功条件（保持向后兼容）
func (a *App) evaluateSuccessCondition(task *Task, resp *http.Response, responseBody string) bool {
	result, _ := a.evaluateSuccessConditionWithDetails(task, resp, responseBody)
	return result
}

// evaluateSuccessConditionWithDetails 评估成功条件并返回详细信息
func (a *App) evaluateSuccessConditionWithDetails(task *Task, resp *http.Response, responseBody string) (bool, *SuccessConditionDetails) {
	// 添加调试日志
	fmt.Printf("=== 成功条件评估调试 ===\n")
	fmt.Printf("启用状态: %v\n", task.SuccessCondition.Enabled)
	fmt.Printf("JSON路径: %s\n", task.SuccessCondition.JsonPath)
	fmt.Printf("操作符: %s\n", task.SuccessCondition.Operator)
	fmt.Printf("期望值: %s\n", task.SuccessCondition.ExpectedValue)
	fmt.Printf("响应体: %s\n", responseBody)

	// 判断是否为字符串基础的条件
	isStringBased := task.SuccessCondition.Operator == "response_contains" ||
		task.SuccessCondition.Operator == "response_not_contains" ||
		task.SuccessCondition.Operator == "response_equals" ||
		task.SuccessCondition.Operator == "response_not_equals"

	details := &SuccessConditionDetails{
		Type:          "json_path",
		JsonPath:      task.SuccessCondition.JsonPath,
		Operator:      task.SuccessCondition.Operator,
		ExpectedValue: task.SuccessCondition.ExpectedValue,
	}

	if isStringBased {
		details.Type = "string_based"
		details.JsonPath = "" // 字符串基础条件不使用JSON路径
	}

	// 如果没有启用自定义成功条件，使用默认的HTTP状态码判断
	if !task.SuccessCondition.Enabled {
		fmt.Printf("未启用自定义成功条件，使用HTTP状态码判断\n")
		result := resp.StatusCode >= 200 && resp.StatusCode < 300
		details.Type = "http_status"
		details.ActualValue = fmt.Sprintf("%d", resp.StatusCode)
		details.Result = result
		details.Reason = "未启用自定义成功条件，使用HTTP状态码判断"
		return result, details
	}

	// 如果是字符串基础的条件，直接对响应体进行判断
	if isStringBased {
		fmt.Printf("使用字符串基础条件判断\n")
		return a.evaluateStringBasedCondition(task, responseBody, details)
	}

	// 如果没有设置JSON路径，使用默认判断
	if task.SuccessCondition.JsonPath == "" {
		fmt.Printf("JSON路径为空，使用HTTP状态码判断\n")
		result := resp.StatusCode >= 200 && resp.StatusCode < 300
		details.Type = "http_status"
		details.ActualValue = fmt.Sprintf("%d", resp.StatusCode)
		details.Result = result
		details.Reason = "JSON路径为空，使用HTTP状态码判断"
		return result, details
	}

	// 如果响应体为空，无法进行JSON路径判断
	if responseBody == "" {
		fmt.Printf("响应体为空，返回false\n")
		details.ActualValue = ""
		details.Result = false
		details.Reason = "响应体为空，无法进行JSON路径判断"
		return false, details
	}

	// 清理响应体：移除BOM和前后空格
	cleanedBody := a.cleanResponseBody(responseBody)
	fmt.Printf("清理后的响应体: %s\n", cleanedBody)

	// 解析JSON响应
	var jsonData interface{}
	if err := json.Unmarshal([]byte(cleanedBody), &jsonData); err != nil {
		// JSON解析失败，显示详细错误信息
		fmt.Printf("JSON解析失败: %v\n", err)
		fmt.Printf("原始响应体长度: %d\n", len(responseBody))
		fmt.Printf("清理后响应体长度: %d\n", len(cleanedBody))
		if len(cleanedBody) > 0 {
			fmt.Printf("响应体前10个字符的字节值: %v\n", []byte(cleanedBody[:min(10, len(cleanedBody))]))
		}
		details.ActualValue = "JSON解析失败"
		details.Result = false
		details.Reason = fmt.Sprintf("JSON解析失败: %v", err)
		return false, details
	}

	// 获取JSON路径对应的值
	value := a.getJsonPathValue(jsonData, task.SuccessCondition.JsonPath)
	if value == nil {
		fmt.Printf("JSON路径 %s 对应的值为nil\n", task.SuccessCondition.JsonPath)
		details.ActualValue = "null"
		details.Result = false
		details.Reason = fmt.Sprintf("JSON路径 %s 对应的值为null", task.SuccessCondition.JsonPath)
		return false, details
	}

	fmt.Printf("JSON路径 %s 对应的值: %v\n", task.SuccessCondition.JsonPath, value)

	// 设置实际值
	details.ActualValue = fmt.Sprintf("%v", value)

	// 根据操作符进行判断
	result := a.evaluateCondition(value, task.SuccessCondition.Operator, task.SuccessCondition.ExpectedValue)
	details.Result = result

	// 设置详细说明
	switch task.SuccessCondition.Operator {
	case "equals":
		details.Reason = fmt.Sprintf("检查 '%s' 是否等于 '%s'", details.ActualValue, details.ExpectedValue)
	case "not_equals":
		details.Reason = fmt.Sprintf("检查 '%s' 是否不等于 '%s'", details.ActualValue, details.ExpectedValue)
	case "contains":
		details.Reason = fmt.Sprintf("检查 '%s' 是否包含 '%s'", details.ActualValue, details.ExpectedValue)
	case "not_contains":
		details.Reason = fmt.Sprintf("检查 '%s' 是否不包含 '%s'", details.ActualValue, details.ExpectedValue)
	default:
		details.Reason = fmt.Sprintf("使用操作符 '%s' 比较 '%s' 和 '%s'", task.SuccessCondition.Operator, details.ActualValue, details.ExpectedValue)
	}

	fmt.Printf("条件判断结果: %v\n", result)
	fmt.Printf("=== 成功条件评估结束 ===\n")
	return result, details
}

// generateConditionFailureDescription 生成成功条件失败的详细描述
func (a *App) generateConditionFailureDescription(details *SuccessConditionDetails) string {
	if details == nil {
		return "成功条件评估失败，无详细信息"
	}

	var description strings.Builder
	description.WriteString("成功条件详情：\n")

	// 条件类型
	switch details.Type {
	case "json_path":
		description.WriteString("- 条件类型：JSON路径判断\n")
		description.WriteString(fmt.Sprintf("- JSON路径：%s\n", details.JsonPath))
	case "string_based":
		description.WriteString("- 条件类型：字符串内容判断\n")
	case "http_status":
		description.WriteString("- 条件类型：HTTP状态码判断\n")
	default:
		description.WriteString(fmt.Sprintf("- 条件类型：%s\n", details.Type))
	}

	// 判断条件
	operatorText := a.getOperatorTextForLog(details.Operator)
	description.WriteString(fmt.Sprintf("- 判断条件：%s\n", operatorText))
	description.WriteString(fmt.Sprintf("- 期望值：\"%s\"\n", details.ExpectedValue))
	description.WriteString(fmt.Sprintf("- 实际值：\"%s\"\n", details.ActualValue))

	// 失败原因
	description.WriteString(fmt.Sprintf("- 失败原因：%s", details.Reason))

	return description.String()
}

// generateHttpErrorDescription 生成HTTP错误的详细描述
func (a *App) generateHttpErrorDescription(statusCode int) string {
	var description strings.Builder
	description.WriteString("HTTP状态错误详情：\n")
	description.WriteString(fmt.Sprintf("- 状态码：%d\n", statusCode))

	if statusCode >= 400 && statusCode < 500 {
		description.WriteString("- 错误类型：客户端错误\n")
		switch statusCode {
		case 400:
			description.WriteString("- 详细说明：请求参数错误，请检查URL、请求头或请求体格式")
		case 401:
			description.WriteString("- 详细说明：未授权访问，请检查认证信息")
		case 403:
			description.WriteString("- 详细说明：访问被禁止，请检查权限设置")
		case 404:
			description.WriteString("- 详细说明：请求的资源不存在，请检查URL是否正确")
		case 405:
			description.WriteString("- 详细说明：请求方法不被允许，请检查HTTP方法")
		case 408:
			description.WriteString("- 详细说明：请求超时，请稍后重试")
		case 429:
			description.WriteString("- 详细说明：请求过于频繁，请降低请求频率")
		default:
			description.WriteString("- 详细说明：客户端请求错误，请检查请求参数")
		}
	} else if statusCode >= 500 {
		description.WriteString("- 错误类型：服务器错误\n")
		switch statusCode {
		case 500:
			description.WriteString("- 详细说明：服务器内部错误，请稍后重试")
		case 502:
			description.WriteString("- 详细说明：网关错误，服务器暂时不可用")
		case 503:
			description.WriteString("- 详细说明：服务不可用，服务器过载或维护中")
		case 504:
			description.WriteString("- 详细说明：网关超时，上游服务器响应超时")
		default:
			description.WriteString("- 详细说明：服务器错误，请稍后重试")
		}
	} else if statusCode >= 300 && statusCode < 400 {
		description.WriteString("- 错误类型：重定向\n")
		description.WriteString("- 详细说明：请求被重定向，可能需要处理跳转逻辑")
	} else {
		description.WriteString("- 错误类型：未知错误\n")
		description.WriteString("- 详细说明：HTTP状态码不在成功范围内（200-299）")
	}

	return description.String()
}

// getOperatorTextForLog 获取操作符的中文文本（用于日志）
func (a *App) getOperatorTextForLog(operator string) string {
	switch operator {
	case "equals":
		return "等于"
	case "not_equals":
		return "不等于"
	case "contains":
		return "包含"
	case "not_contains":
		return "不包含"
	case "response_contains":
		return "响应包含"
	case "response_not_contains":
		return "响应不包含"
	case "response_equals":
		return "响应等于"
	case "response_not_equals":
		return "响应不等于"
	default:
		return operator
	}
}

// evaluateStringBasedCondition 评估字符串基础的成功条件
func (a *App) evaluateStringBasedCondition(task *Task, responseBody string, details *SuccessConditionDetails) (bool, *SuccessConditionDetails) {
	// 清理响应体
	cleanedBody := a.cleanResponseBody(responseBody)
	details.ActualValue = fmt.Sprintf("响应体长度: %d 字符", len(cleanedBody))

	var result bool
	switch task.SuccessCondition.Operator {
	case "response_contains":
		result = strings.Contains(cleanedBody, task.SuccessCondition.ExpectedValue)
		details.Reason = fmt.Sprintf("检查响应体是否包含 '%s'", task.SuccessCondition.ExpectedValue)
	case "response_not_contains":
		result = !strings.Contains(cleanedBody, task.SuccessCondition.ExpectedValue)
		details.Reason = fmt.Sprintf("检查响应体是否不包含 '%s'", task.SuccessCondition.ExpectedValue)
	case "response_equals":
		result = cleanedBody == task.SuccessCondition.ExpectedValue
		details.Reason = fmt.Sprintf("检查响应体是否等于指定内容")
		details.ActualValue = cleanedBody // 对于equals，显示完整内容
	case "response_not_equals":
		result = cleanedBody != task.SuccessCondition.ExpectedValue
		details.Reason = fmt.Sprintf("检查响应体是否不等于指定内容")
		details.ActualValue = cleanedBody // 对于not_equals，显示完整内容
	default:
		result = false
		details.Reason = fmt.Sprintf("未知的字符串基础操作符: %s", task.SuccessCondition.Operator)
	}

	details.Result = result
	fmt.Printf("字符串基础条件判断结果: %v\n", result)
	return result, details
}

// getJsonPathValue 根据JSON路径获取值
func (a *App) getJsonPathValue(data interface{}, path string) interface{} {
	if path == "" {
		return data
	}

	// 分割路径
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			if val, exists := v[part]; exists {
				current = val
			} else {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

// evaluateCondition 评估条件
func (a *App) evaluateCondition(actualValue interface{}, operator, expectedValue string) bool {
	// 将实际值转换为字符串进行比较
	actualStr := fmt.Sprintf("%v", actualValue)

	fmt.Printf("--- 条件判断详情 ---\n")
	fmt.Printf("实际值: '%s'\n", actualStr)
	fmt.Printf("操作符: '%s'\n", operator)
	fmt.Printf("期望值: '%s'\n", expectedValue)

	var result bool
	switch operator {
	case "equals":
		result = actualStr == expectedValue
		fmt.Printf("等于判断: '%s' == '%s' = %v\n", actualStr, expectedValue, result)
	case "not_equals":
		result = actualStr != expectedValue
		fmt.Printf("不等于判断: '%s' != '%s' = %v\n", actualStr, expectedValue, result)
	case "contains":
		result = strings.Contains(actualStr, expectedValue)
		fmt.Printf("包含判断: '%s' contains '%s' = %v\n", actualStr, expectedValue, result)
	case "not_contains":
		result = !strings.Contains(actualStr, expectedValue)
		fmt.Printf("不包含判断: '%s' not contains '%s' = %v\n", actualStr, expectedValue, result)
	default:
		// 未知操作符，使用等于判断
		result = actualStr == expectedValue
		fmt.Printf("未知操作符 '%s'，使用等于判断: '%s' == '%s' = %v\n", operator, actualStr, expectedValue, result)
	}
	fmt.Printf("--- 条件判断结束 ---\n")
	return result
}

// cleanResponseBody 清理响应体，移除BOM和前后空格
func (a *App) cleanResponseBody(responseBody string) string {
	// 转换为字节数组进行处理
	bodyBytes := []byte(responseBody)

	// 检测并移除UTF-8 BOM (EF BB BF)
	if len(bodyBytes) >= 3 && bodyBytes[0] == 0xEF && bodyBytes[1] == 0xBB && bodyBytes[2] == 0xBF {
		fmt.Printf("检测到UTF-8 BOM，正在移除\n")
		bodyBytes = bodyBytes[3:]
	}

	// 检测并移除UTF-16 BE BOM (FE FF)
	if len(bodyBytes) >= 2 && bodyBytes[0] == 0xFE && bodyBytes[1] == 0xFF {
		fmt.Printf("检测到UTF-16 BE BOM，正在移除\n")
		bodyBytes = bodyBytes[2:]
	}

	// 检测并移除UTF-16 LE BOM (FF FE)
	if len(bodyBytes) >= 2 && bodyBytes[0] == 0xFF && bodyBytes[1] == 0xFE {
		fmt.Printf("检测到UTF-16 LE BOM，正在移除\n")
		bodyBytes = bodyBytes[2:]
	}

	// 转换回字符串并去除前后空格
	cleanedBody := strings.TrimSpace(string(bodyBytes))

	// 移除其他可能的不可见字符
	cleanedBody = strings.TrimFunc(cleanedBody, func(r rune) bool {
		// 移除控制字符，但保留换行符和制表符
		return r < 32 && r != '\n' && r != '\r' && r != '\t'
	})

	return cleanedBody
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetVersionInfo 获取应用版本信息
func (a *App) GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   AppVersion,
		Name:      AppName,
		BuildDate: BuildDate,
	}
}
