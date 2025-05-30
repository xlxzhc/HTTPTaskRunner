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
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Task represents a scheduled HTTP request task
type Task struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	Method         string   `json:"method"`
	Cookie         string   `json:"cookie"`
	Headers        string   `json:"headers"`
	Data           string   `json:"data"`
	UseVirtualIP   bool     `json:"useVirtualIP"`
	Times          int      `json:"times"`
	Threads        int      `json:"threads"`
	ScheduledTime  string   `json:"scheduledTime"`
	CronExpression string   `json:"cronExpression"`
	DelayMin       int      `json:"delayMin"`
	DelayMax       int      `json:"delayMax"`
	Tags           []string `json:"tags"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
	IsRunning      bool     `json:"isRunning"`
}

// TaskProgress represents the progress of a running task
type TaskProgress struct {
	CurrentRequest int       `json:"currentRequest"`
	TotalRequests  int       `json:"totalRequests"`
	StartTime      time.Time `json:"startTime"`
	ElapsedTime    int64     `json:"elapsedTime"` // in milliseconds
	DelayInfo      []int     `json:"delayInfo"`   // Array of delays used
}

// TaskLogs represents logs for a task
type TaskLogs map[string][]string

// TaskLogsResult represents the result of loading task logs
type TaskLogsResult struct {
	Logs TaskLogs `json:"logs"`
	Path string   `json:"path"`
}

// DeleteLogsResult represents the result of deleting old logs
type DeleteLogsResult struct {
	Count int    `json:"count"`
	Path  string `json:"path"`
}

// App struct
type App struct {
	ctx           context.Context
	Tasks         map[string]*Task
	RunningTasks  map[string]context.CancelFunc
	TaskProgress  map[string]*TaskProgress
	TaskLogs      TaskLogs
	cronScheduler *cron.Cron
	cronJobs      map[string]cron.EntryID
}

// NewApp creates a new App application struct
func NewApp() *App {
	cronScheduler := cron.New(cron.WithSeconds())
	cronScheduler.Start()

	return &App{
		Tasks:         make(map[string]*Task),
		RunningTasks:  make(map[string]context.CancelFunc),
		TaskProgress:  make(map[string]*TaskProgress),
		TaskLogs:      make(TaskLogs),
		cronScheduler: cronScheduler,
		cronJobs:      make(map[string]cron.EntryID),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.loadTasksFromDisk()
	a.loadTaskLogsFromDisk()

	// Restart any tasks with cron expressions
	for id, task := range a.Tasks {
		if task.CronExpression != "" {
			a.scheduleCronTask(id, task.CronExpression)
		}
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.cronScheduler != nil {
		a.cronScheduler.Stop()
	}

	for taskID, cancelFunc := range a.RunningTasks {
		cancelFunc()
		delete(a.RunningTasks, taskID)
	}

	// Save task logs before shutdown
	a.saveTaskLogsToDisk()
}

// GenerateTaskID generates a unique ID for a task
func (a *App) GenerateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

// SendRequest sends an HTTP request to the specified URL
func (a *App) SendRequest(url string, method string, ck string, data string, headers string, useVirtualIP bool) string {
	if url == "" {
		return "URL cannot be empty"
	}

	// Create a client with timeout
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	// Create request
	req, err := http.NewRequest(strings.ToUpper(method), url, strings.NewReader(data))
	if err != nil {
		return fmt.Sprintf("Error creating request: %s", err.Error())
	}

	// Add Content-Type header if not present in custom headers
	if !strings.Contains(strings.ToLower(headers), "content-type:") {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Process and filter headers
	var filteredHeaderLines []string
	var hasAcceptEncodingGzip bool

	// Add custom headers
	if headers != "" {
		headerLines := strings.Split(headers, "\n")
		for _, line := range headerLines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Check if it's an Accept-Encoding header with gzip
			if strings.HasPrefix(strings.ToLower(line), "accept-encoding:") &&
				strings.Contains(strings.ToLower(line), "gzip") {
				hasAcceptEncodingGzip = true
				continue // Skip this header
			}

			// Add the header to filtered list
			filteredHeaderLines = append(filteredHeaderLines, line)

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				req.Header.Set(key, value)
			}
		}
	}

	// Update the headers string to exclude the filtered out headers
	headers = strings.Join(filteredHeaderLines, "\n")

	// Add cookie header if provided
	if ck != "" {
		req.Header.Set("Cookie", ck)
	}

	// Add a random Chinese IP if useVirtualIP is true
	if useVirtualIP {
		ip := generateChineseIP()
		req.Header.Set("X-Forwarded-For", ip)
		req.Header.Set("Forwarded-For", ip)
	}

	// Log if we removed an Accept-Encoding header
	if hasAcceptEncodingGzip {
		fmt.Printf("Removed Accept-Encoding header containing gzip\n")
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error sending request: %s", err.Error())
	}
	defer resp.Body.Close()

	// Read response
	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response: %s", err.Error())
	}

	// Try to decode JSON Unicode escape sequences
	responseStr := buf.String()
	decodedStr := decodeUnicodeJSON(responseStr)

	return decodedStr
}

// ExecuteCardPackage executes the card package operation multiple times with multi-threading
func (a *App) ExecuteCardPackage(url string, method string, ck string, data string, headers string, useVirtualIP bool, times int, threads int, delayMin int, delayMax int) string {
	// Create a simple progress tracker
	progress := &TaskProgress{
		CurrentRequest: 0,
		TotalRequests:  times,
		StartTime:      time.Now(),
		ElapsedTime:    0,
		DelayInfo:      make([]int, 0, times),
	}

	// Execute using the context-aware function with a background context
	return a.ExecuteCardPackageWithContext(
		context.Background(),
		url,
		method,
		ck,
		data,
		headers,
		useVirtualIP,
		times,
		threads,
		delayMin,
		delayMax,
		progress,
	)
}

// SaveTask saves a task configuration
func (a *App) SaveTask(name string, url string, method string, ck string, data string,
	headers string, useVirtualIP bool, times int, threads int,
	scheduledTime string, cronExpression string, delayMin int, delayMax int, tags []string) string {

	now := time.Now().Format(time.RFC3339)
	taskID := a.GenerateTaskID()

	task := &Task{
		ID:             taskID,
		Name:           name,
		URL:            url,
		Method:         method,
		Cookie:         ck,
		Headers:        headers,
		Data:           data,
		UseVirtualIP:   useVirtualIP,
		Times:          times,
		Threads:        threads,
		ScheduledTime:  scheduledTime,
		CronExpression: cronExpression,
		DelayMin:       delayMin,
		DelayMax:       delayMax,
		Tags:           tags,
		CreatedAt:      now,
		UpdatedAt:      now,
		IsRunning:      false,
	}

	a.Tasks[taskID] = task

	// If a cron expression is provided, schedule the task
	if cronExpression != "" {
		err := a.scheduleCronTask(taskID, cronExpression)
		if err != nil {
			return fmt.Sprintf("Task saved with ID: %s, but cron scheduling failed: %s", taskID, err.Error())
		}
	}

	// Save tasks to disk
	err := a.saveTasksToDisk()
	if err != nil {
		return fmt.Sprintf("Error saving task: %s", err.Error())
	}

	return fmt.Sprintf("Task saved successfully with ID: %s", taskID)
}

// GetAllTasks returns all saved tasks
func (a *App) GetAllTasks() map[string]*Task {
	return a.Tasks
}

// GetTaskByID returns a specific task by ID
func (a *App) GetTaskByID(id string) *Task {
	return a.Tasks[id]
}

// UpdateTask updates an existing task
func (a *App) UpdateTask(id string, name string, url string, method string, ck string,
	data string, headers string, useVirtualIP bool, times int, threads int,
	scheduledTime string, cronExpression string, delayMin int, delayMax int, tags []string) string {

	task, exists := a.Tasks[id]
	if !exists {
		return fmt.Sprintf("Task with ID %s not found", id)
	}

	// If task is running, we can't update it
	if task.IsRunning {
		return fmt.Sprintf("Task %s is currently running and cannot be updated", id)
	}

	// Update task properties
	task.Name = name
	task.URL = url
	task.Method = method
	task.Cookie = ck
	task.Data = data
	task.Headers = headers
	task.UseVirtualIP = useVirtualIP
	task.Times = times
	task.Threads = threads
	task.ScheduledTime = scheduledTime

	// Handle cron expression changes
	if task.CronExpression != cronExpression {
		// Remove old cron job if it exists
		if oldJobID, exists := a.cronJobs[id]; exists && task.CronExpression != "" {
			a.cronScheduler.Remove(oldJobID)
			delete(a.cronJobs, id)
		}

		// Set new cron expression
		task.CronExpression = cronExpression

		// Schedule new cron job if needed
		if cronExpression != "" {
			err := a.scheduleCronTask(id, cronExpression)
			if err != nil {
				return fmt.Sprintf("Task updated, but cron scheduling failed: %s", err.Error())
			}
		}
	}

	task.DelayMin = delayMin
	task.DelayMax = delayMax
	task.Tags = tags
	task.UpdatedAt = time.Now().Format(time.RFC3339)

	// Save tasks to disk
	err := a.saveTasksToDisk()
	if err != nil {
		return fmt.Sprintf("Error updating task: %s", err.Error())
	}

	return fmt.Sprintf("Task %s updated successfully", id)
}

// DeleteTask deletes a task by ID
func (a *App) DeleteTask(id string) string {
	task, exists := a.Tasks[id]
	if !exists {
		return fmt.Sprintf("Task with ID %s not found", id)
	}

	// If the task is running, stop it
	if task.IsRunning {
		a.StopTask(id)
	}

	// If the task has a cron job, remove it
	if jobID, hasCron := a.cronJobs[id]; hasCron {
		a.cronScheduler.Remove(jobID)
		delete(a.cronJobs, id)
	}

	// Remove task from maps
	delete(a.Tasks, id)
	delete(a.TaskProgress, id)

	// Save tasks to disk
	err := a.saveTasksToDisk()
	if err != nil {
		return fmt.Sprintf("Error deleting task: %s", err.Error())
	}

	return fmt.Sprintf("Task %s deleted successfully", id)
}

// ExportTasks exports all tasks to a JSON file
func (a *App) ExportTasks(filepath string) string {
	tasksJSON, err := json.MarshalIndent(a.Tasks, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error exporting tasks: %s", err.Error())
	}

	err = os.WriteFile(filepath, tasksJSON, 0644)
	if err != nil {
		return fmt.Sprintf("Error writing to file: %s", err.Error())
	}

	return fmt.Sprintf("Tasks exported successfully to %s", filepath)
}

// ImportTasks imports tasks from a JSON file
func (a *App) ImportTasks(filepath string) string {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %s", err.Error())
	}

	var importedTasks map[string]*Task
	err = json.Unmarshal(data, &importedTasks)
	if err != nil {
		return fmt.Sprintf("Error parsing tasks: %s", err.Error())
	}

	// Merge imported tasks with existing ones
	for id, task := range importedTasks {
		a.Tasks[id] = task
	}

	// Save the updated tasks to disk
	err = a.saveTasksToDisk()
	if err != nil {
		return fmt.Sprintf("Error saving imported tasks: %s", err.Error())
	}

	return fmt.Sprintf("Imported %d tasks successfully", len(importedTasks))
}

// GetTasksPath returns the path to the tasks data file
func (a *App) getTasksPath() string {
	// Store tasks in the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	// Create the data directory if it doesn't exist
	dataDir := filepath.Join(homeDir, ".myui")
	os.MkdirAll(dataDir, 0755)

	return filepath.Join(dataDir, "tasks.json")
}

// saveTasksToDisk saves all tasks to a file
func (a *App) saveTasksToDisk() error {
	tasksPath := a.getTasksPath()
	tasksJSON, err := json.MarshalIndent(a.Tasks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tasksPath, tasksJSON, 0644)
}

// loadTasksFromDisk loads all tasks from a file
func (a *App) loadTasksFromDisk() {
	tasksPath := a.getTasksPath()

	// If the file doesn't exist, just return (no tasks saved yet)
	if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
		return
	}

	data, err := os.ReadFile(tasksPath)
	if err != nil {
		fmt.Printf("Error loading tasks: %s\n", err.Error())
		return
	}

	err = json.Unmarshal(data, &a.Tasks)
	if err != nil {
		fmt.Printf("Error parsing tasks: %s\n", err.Error())
		return
	}
}

// decodeUnicodeJSON attempts to decode Unicode escape sequences in JSON strings
func decodeUnicodeJSON(input string) string {
	// Check if it looks like JSON
	if !strings.HasPrefix(strings.TrimSpace(input), "{") && !strings.HasPrefix(strings.TrimSpace(input), "[") {
		return input
	}

	// Try to decode as raw JSON first to handle Unicode escapes
	var result interface{}
	err := json.Unmarshal([]byte(input), &result)
	if err != nil {
		// If we can't parse it as JSON, return the original string
		return input
	}

	// Re-encode with indentation for better readability
	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return input
	}

	return string(prettyJSON)
}

// Chinese IP ranges
var chineseIPRanges = []struct {
	start string
	end   string
}{
	{"1.0.1.0", "1.0.3.255"},         // China
	{"1.1.0.0", "1.1.255.255"},       // China
	{"1.2.0.0", "1.2.255.255"},       // China
	{"1.4.1.0", "1.4.127.255"},       // China
	{"1.8.0.0", "1.8.255.255"},       // China
	{"1.12.0.0", "1.15.255.255"},     // China
	{"1.24.0.0", "1.31.255.255"},     // China
	{"14.0.0.0", "14.0.7.855"},       // China
	{"14.1.0.0", "14.127.255.255"},   // China
	{"27.0.0.0", "27.127.255.255"},   // China
	{"36.0.0.0", "36.255.255.255"},   // China
	{"39.0.0.0", "39.255.255.255"},   // China
	{"42.0.0.0", "42.255.255.255"},   // China
	{"49.0.0.0", "49.255.255.255"},   // China
	{"58.0.0.0", "58.255.255.255"},   // China
	{"59.0.0.0", "59.255.255.255"},   // China
	{"60.0.0.0", "60.255.255.255"},   // China
	{"61.0.0.0", "61.255.255.255"},   // China
	{"101.0.0.0", "101.255.255.255"}, // China
	{"103.0.0.0", "103.255.255.255"}, // China
	{"106.0.0.0", "106.255.255.255"}, // China
	{"110.0.0.0", "110.255.255.255"}, // China
	{"111.0.0.0", "111.255.255.255"}, // China
	{"112.0.0.0", "112.255.255.255"}, // China
	{"113.0.0.0", "113.255.255.255"}, // China
	{"114.0.0.0", "114.255.255.255"}, // China
	{"115.0.0.0", "115.255.255.255"}, // China
	{"116.0.0.0", "116.255.255.255"}, // China
	{"117.0.0.0", "117.255.255.255"}, // China
	{"118.0.0.0", "118.255.255.255"}, // China
	{"119.0.0.0", "119.255.255.255"}, // China
	{"120.0.0.0", "120.255.255.255"}, // China
	{"121.0.0.0", "121.255.255.255"}, // China
	{"122.0.0.0", "122.255.255.255"}, // China
	{"123.0.0.0", "123.255.255.255"}, // China
	{"124.0.0.0", "124.255.255.255"}, // China
	{"125.0.0.0", "125.255.255.255"}, // China
	{"175.0.0.0", "175.255.255.255"}, // China
	{"180.0.0.0", "180.255.255.255"}, // China
	{"182.0.0.0", "182.255.255.255"}, // China
	{"183.0.0.0", "183.255.255.255"}, // China
	{"202.0.0.0", "202.255.255.255"}, // China
	{"203.0.0.0", "203.255.255.255"}, // China
	{"210.0.0.0", "210.255.255.255"}, // China
	{"211.0.0.0", "211.255.255.255"}, // China
	{"218.0.0.0", "218.255.255.255"}, // China
	{"220.0.0.0", "220.255.255.255"}, // China
	{"221.0.0.0", "221.255.255.255"}, // China
	{"222.0.0.0", "222.255.255.255"}, // China
	{"223.0.0.0", "223.255.255.255"}, // China
}

// Helper function to generate a random Chinese IP address
func generateChineseIP() string {
	rand.Seed(time.Now().UnixNano())

	// Select a random Chinese IP range
	ipRange := chineseIPRanges[rand.Intn(len(chineseIPRanges))]

	// Parse start and end IP
	startParts := strings.Split(ipRange.start, ".")
	endParts := strings.Split(ipRange.end, ".")

	// Convert to integers
	var startIP [4]int
	var endIP [4]int
	for i := 0; i < 4; i++ {
		fmt.Sscanf(startParts[i], "%d", &startIP[i])
		fmt.Sscanf(endParts[i], "%d", &endIP[i])
	}

	// Generate random IP within range
	var ip [4]int
	for i := 0; i < 4; i++ {
		if startIP[i] == endIP[i] {
			ip[i] = startIP[i]
		} else {
			ip[i] = startIP[i] + rand.Intn(endIP[i]-startIP[i]+1)
		}
	}

	return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
}

// scheduleCronTask schedules a task to run on a cron schedule
func (a *App) scheduleCronTask(taskID string, cronExpr string) error {
	// If the task is already scheduled, remove the old schedule
	if jobID, exists := a.cronJobs[taskID]; exists {
		a.cronScheduler.Remove(jobID)
		delete(a.cronJobs, taskID)
	}

	// Add the new cron job
	jobID, err := a.cronScheduler.AddFunc(cronExpr, func() {
		// Get the task
		task, exists := a.Tasks[taskID]
		if !exists {
			return
		}

		// Execute the task if it's not already running
		if !task.IsRunning {
			go a.ExecuteTask(taskID)
		}
	})

	if err != nil {
		return err
	}

	// Store the job ID
	a.cronJobs[taskID] = jobID
	return nil
}

// ExecuteTask executes a task by ID (update to log responses)
func (a *App) ExecuteTask(taskID string) string {
	task, exists := a.Tasks[taskID]
	if !exists {
		return fmt.Sprintf("Task with ID %s not found", taskID)
	}

	if _, running := a.RunningTasks[taskID]; running {
		return fmt.Sprintf("Task %s is already running", task.Name)
	}

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Store the cancel function
	a.RunningTasks[taskID] = cancel

	// Mark task as running
	task.IsRunning = true

	// Create a progress tracker
	progress := &TaskProgress{
		CurrentRequest: 0,
		TotalRequests:  task.Times,
		StartTime:      time.Now(),
		ElapsedTime:    0,
		DelayInfo:      make([]int, 0, task.Times),
	}

	a.TaskProgress[taskID] = progress

	// Log task start
	a.addTaskLog(taskID, fmt.Sprintf("开始执行任务: %s", task.Name))

	// Log cron expression if present
	if task.CronExpression != "" {
		a.addTaskLog(taskID, fmt.Sprintf("定时规则: %s", task.CronExpression))
	}

	// Execute the task in a goroutine
	go func() {
		defer func() {
			// Clean up when done
			delete(a.RunningTasks, taskID)
			task.IsRunning = false

			// Add completion log
			elapsedMs := time.Since(progress.StartTime).Milliseconds()
			a.addTaskLog(taskID, fmt.Sprintf("任务完成，总耗时: %dms, 总请求数: %d", elapsedMs, progress.CurrentRequest))

			// Save task logs
			a.saveTaskLogsToDisk()
		}()

		result := a.ExecuteCardPackageWithContext(
			ctx,
			task.URL,
			task.Method,
			task.Cookie,
			task.Data,
			task.Headers,
			task.UseVirtualIP,
			task.Times,
			task.Threads,
			task.DelayMin,
			task.DelayMax,
			progress,
		)

		// Log the result
		fmt.Printf("Task %s completed: %s\n", task.Name, result)
		a.addTaskLog(taskID, fmt.Sprintf("执行结果: %s", truncateString(result, 500)))
	}()

	return fmt.Sprintf("Started task: %s", task.Name)
}

// StopTask stops a running task
func (a *App) StopTask(taskID string) string {
	cancel, exists := a.RunningTasks[taskID]
	if !exists {
		return fmt.Sprintf("Task with ID %s is not running", taskID)
	}

	// Cancel the task
	cancel()

	// Mark the task as not running
	if task, ok := a.Tasks[taskID]; ok {
		task.IsRunning = false
	}

	// Remove from running tasks
	delete(a.RunningTasks, taskID)

	return fmt.Sprintf("Task %s stopped", taskID)
}

// GetTaskProgress returns the progress of a running task
func (a *App) GetTaskProgress(taskID string) *TaskProgress {
	return a.TaskProgress[taskID]
}

// TestTask tests a task with a single request (update to log response)
func (a *App) TestTask(taskID string) string {
	task, exists := a.Tasks[taskID]
	if !exists {
		return fmt.Sprintf("Task with ID %s not found", taskID)
	}

	// Log the test start
	a.addTaskLog(taskID, fmt.Sprintf("测试任务: %s", task.Name))

	// Send a single request
	response := a.SendRequest(
		task.URL,
		task.Method,
		task.Cookie,
		task.Data,
		task.Headers,
		task.UseVirtualIP,
	)

	// Log the response
	a.addTaskLog(taskID, fmt.Sprintf("测试响应: %s", truncateString(response, 500)))

	// Save task logs
	a.saveTaskLogsToDisk()

	return response
}

// ExecuteCardPackageWithContext executes the card package operation with context (update to log responses)
func (a *App) ExecuteCardPackageWithContext(
	ctx context.Context,
	url string,
	method string,
	ck string,
	data string,
	headers string,
	useVirtualIP bool,
	times int,
	threads int,
	delayMin int,
	delayMax int,
	progress *TaskProgress,
) string {
	if url == "" {
		return "URL cannot be empty"
	}

	if times <= 0 {
		return "Times must be greater than 0"
	}

	if threads <= 0 {
		threads = 1
	}

	// Limit threads to a reasonable number
	if threads > 100 {
		threads = 100
	}

	// Create a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup
	wg.Add(threads)

	// Create a channel for results
	resultCh := make(chan string, times)

	// Create a channel for tasks
	taskCh := make(chan int, times)

	// Fill the task channel with task indices
	for i := 0; i < times; i++ {
		taskCh <- i
	}
	close(taskCh)

	// Track start time
	startTime := time.Now()

	// Create worker goroutines
	for i := 0; i < threads; i++ {
		go func(workerID int) {
			defer wg.Done()

			for taskIndex := range taskCh {
				// Check if context is cancelled
				select {
				case <-ctx.Done():
					return
				default:
					// Continue execution
				}

				// Apply random delay if specified
				var delay int
				if delayMax > delayMin && delayMin >= 0 {
					delay = rand.Intn(delayMax-delayMin+1) + delayMin
					if delay > 0 {
						time.Sleep(time.Duration(delay) * time.Millisecond)
					}
				}

				// Send request
				response := a.SendRequest(url, method, ck, data, headers, useVirtualIP)

				// Get task ID from context if available
				if taskID, ok := ctx.Value("taskID").(string); ok && taskID != "" {
					// Log the response for the task
					a.addTaskLog(taskID, fmt.Sprintf("请求 %d 响应: %s", taskIndex+1, truncateString(response, 200)))
				}

				// Update progress
				progress.CurrentRequest++
				if delay > 0 {
					progress.DelayInfo = append(progress.DelayInfo, delay)
				}
				progress.ElapsedTime = time.Since(startTime).Milliseconds()

				// Send result to channel
				resultCh <- fmt.Sprintf("Request %d: %s", taskIndex+1, response)
			}
		}(i)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(resultCh)

	// Collect results
	var results []string
	for result := range resultCh {
		results = append(results, result)
	}

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)
	elapsedMs := elapsedTime.Milliseconds()

	// Calculate average delay if any delays were applied
	var avgDelay float64
	if len(progress.DelayInfo) > 0 {
		var totalDelay int
		for _, d := range progress.DelayInfo {
			totalDelay += d
		}
		avgDelay = float64(totalDelay) / float64(len(progress.DelayInfo))
	}

	// Format summary
	var summary string
	if len(progress.DelayInfo) > 0 {
		summary = fmt.Sprintf("Completed %d requests in %dms (%.2f req/s). Average delay: %.2fms",
			times, elapsedMs, float64(times)/(float64(elapsedMs)/1000), avgDelay)
	} else {
		summary = fmt.Sprintf("Completed %d requests in %dms (%.2f req/s)",
			times, elapsedMs, float64(times)/(float64(elapsedMs)/1000))
	}

	return summary
}

// ExportTasksByIDs exports selected tasks to a JSON file
func (a *App) ExportTasksByIDs(filepath string, taskIDs []string) string {
	if len(taskIDs) == 0 {
		return "No tasks selected for export"
	}

	// Create a map with only the selected tasks
	selectedTasks := make(map[string]*Task)
	for _, id := range taskIDs {
		if task, exists := a.Tasks[id]; exists {
			selectedTasks[id] = task
		}
	}

	if len(selectedTasks) == 0 {
		return "None of the selected task IDs exist"
	}

	// Export the selected tasks
	tasksJSON, err := json.MarshalIndent(selectedTasks, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error exporting tasks: %s", err.Error())
	}

	err = os.WriteFile(filepath, tasksJSON, 0644)
	if err != nil {
		return fmt.Sprintf("Error writing to file: %s", err.Error())
	}

	return fmt.Sprintf("Exported %d tasks successfully to %s", len(selectedTasks), filepath)
}

// ExportTasksByTags exports tasks with the specified tags to a JSON file
func (a *App) ExportTasksByTags(filepath string, tags []string) string {
	if len(tags) == 0 {
		return "No tags specified for filtering"
	}

	// Create a map with only the tasks that have the specified tags
	selectedTasks := make(map[string]*Task)
	for id, task := range a.Tasks {
		// Check if the task has any of the specified tags
		for _, taskTag := range task.Tags {
			for _, filterTag := range tags {
				if taskTag == filterTag {
					selectedTasks[id] = task
					break
				}
			}
			// If we've already added this task, no need to check more tags
			if _, added := selectedTasks[id]; added {
				break
			}
		}
	}

	if len(selectedTasks) == 0 {
		return "No tasks found with the specified tags"
	}

	// Export the selected tasks
	tasksJSON, err := json.MarshalIndent(selectedTasks, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error exporting tasks: %s", err.Error())
	}

	err = os.WriteFile(filepath, tasksJSON, 0644)
	if err != nil {
		return fmt.Sprintf("Error writing to file: %s", err.Error())
	}

	return fmt.Sprintf("Exported %d tasks with the specified tags to %s", len(selectedTasks), filepath)
}

// OpenSaveFileDialog opens a save file dialog
func (a *App) OpenSaveFileDialog(title string, defaultFilename string, filter string) (string, error) {
	return wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "JSON Files",
				Pattern:     filter,
			},
		},
	})
}

// OpenFileDialog opens a file dialog
func (a *App) OpenFileDialog(title string, filter string) (string, error) {
	return wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: title,
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "JSON Files",
				Pattern:     filter,
			},
		},
	})
}

// getLogsPath returns the path to the logs directory
func (a *App) getLogsPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return filepath.Join(homeDir, ".myui", "logs")
}

// EnsureLogDirectory ensures the logs directory exists
func (a *App) EnsureLogDirectory() error {
	logsPath := a.getLogsPath()
	return os.MkdirAll(logsPath, 0755)
}

// SaveTaskLogs saves task logs to disk
func (a *App) SaveTaskLogs(logs TaskLogs) *TaskLogsResult {
	a.TaskLogs = logs
	err := a.saveTaskLogsToDisk()
	if err != nil {
		fmt.Printf("Error saving task logs: %v\n", err)
	}

	return &TaskLogsResult{
		Logs: a.TaskLogs,
		Path: a.getLogsPath(),
	}
}

// saveTaskLogsToDisk saves task logs to disk
func (a *App) saveTaskLogsToDisk() error {
	logsPath := a.getLogsPath()

	// Create logs directory if it doesn't exist
	err := os.MkdirAll(logsPath, 0755)
	if err != nil {
		return err
	}

	// Save each task's logs to its own file
	for taskID, logs := range a.TaskLogs {
		if len(logs) == 0 {
			continue
		}

		taskLogPath := filepath.Join(logsPath, taskID+".log")
		err := os.WriteFile(taskLogPath, []byte(strings.Join(logs, "\n")), 0644)
		if err != nil {
			fmt.Printf("Error saving logs for task %s: %v\n", taskID, err)
		}
	}

	return nil
}

// LoadTaskLogs loads task logs from disk
func (a *App) LoadTaskLogs() *TaskLogsResult {
	a.loadTaskLogsFromDisk()

	return &TaskLogsResult{
		Logs: a.TaskLogs,
		Path: a.getLogsPath(),
	}
}

// loadTaskLogsFromDisk loads task logs from disk
func (a *App) loadTaskLogsFromDisk() {
	logsPath := a.getLogsPath()

	// Create logs directory if it doesn't exist
	err := os.MkdirAll(logsPath, 0755)
	if err != nil {
		fmt.Printf("Error creating logs directory: %v\n", err)
		return
	}

	// List log files
	files, err := os.ReadDir(logsPath)
	if err != nil {
		fmt.Printf("Error reading logs directory: %v\n", err)
		return
	}

	// Load each task's logs
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".log") {
			continue
		}

		taskID := strings.TrimSuffix(file.Name(), ".log")
		taskLogPath := filepath.Join(logsPath, file.Name())

		content, err := os.ReadFile(taskLogPath)
		if err != nil {
			fmt.Printf("Error reading log file %s: %v\n", taskLogPath, err)
			continue
		}

		if len(content) > 0 {
			a.TaskLogs[taskID] = strings.Split(string(content), "\n")
		}
	}
}

// ClearTaskLogs clears logs for a task or all tasks
func (a *App) ClearTaskLogs(taskID string) error {
	logsPath := a.getLogsPath()

	if taskID == "all" {
		// Clear all logs
		a.TaskLogs = make(TaskLogs)

		// Delete all log files
		files, err := os.ReadDir(logsPath)
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".log") {
				continue
			}

			err := os.Remove(filepath.Join(logsPath, file.Name()))
			if err != nil {
				fmt.Printf("Error deleting log file %s: %v\n", file.Name(), err)
			}
		}
	} else {
		// Clear logs for a specific task
		delete(a.TaskLogs, taskID)

		// Delete the log file
		taskLogPath := filepath.Join(logsPath, taskID+".log")
		err := os.Remove(taskLogPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// DeleteOldTaskLogs deletes logs older than the specified number of days
func (a *App) DeleteOldTaskLogs(days int) *DeleteLogsResult {
	if days <= 0 {
		days = 7 // Default to 7 days
	}

	cutoffDate := time.Now().AddDate(0, 0, -days)
	deletedCount := 0

	// Process each task's logs
	for taskID, logs := range a.TaskLogs {
		if len(logs) == 0 {
			continue
		}

		filteredLogs := make([]string, 0, len(logs))

		for _, log := range logs {
			// Try to extract date from log entry (format: [YYYY-MM-DD] [HH:MM:SS.SSS] ...)
			dateMatch := strings.Split(log, "]")
			if len(dateMatch) < 2 {
				// Keep entries without valid date format
				filteredLogs = append(filteredLogs, log)
				continue
			}

			dateStr := strings.TrimPrefix(dateMatch[0], "[")
			logDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				// Keep entries with invalid date format
				filteredLogs = append(filteredLogs, log)
				continue
			}

			// Keep logs newer than cutoff date
			if !logDate.Before(cutoffDate) {
				filteredLogs = append(filteredLogs, log)
			}
		}

		deletedCount += len(logs) - len(filteredLogs)

		if len(filteredLogs) > 0 {
			a.TaskLogs[taskID] = filteredLogs
		} else {
			delete(a.TaskLogs, taskID)
		}
	}

	// Save changes to disk
	a.saveTaskLogsToDisk()

	return &DeleteLogsResult{
		Count: deletedCount,
		Path:  a.getLogsPath(),
	}
}

// addTaskLog adds a log entry for a task
func (a *App) addTaskLog(taskID string, message string) {
	if a.TaskLogs == nil {
		a.TaskLogs = make(TaskLogs)
	}

	if _, exists := a.TaskLogs[taskID]; !exists {
		a.TaskLogs[taskID] = make([]string, 0)
	}

	// Format: [YYYY-MM-DD] [HH:MM:SS.SSS] message
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04:05.000")

	logEntry := fmt.Sprintf("[%s] [%s] %s", dateStr, timeStr, message)
	a.TaskLogs[taskID] = append(a.TaskLogs[taskID], logEntry)
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}
