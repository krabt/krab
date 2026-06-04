package keeper

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ProcessConfig 进程配置
type ProcessConfig struct {
	Name         string            `json:"name" yaml:"name"`
	Command      string            `json:"command" yaml:"command"`
	Args         []string          `json:"args" yaml:"args"`
	WorkDir      string            `json:"workdir" yaml:"workdir"`
	AutoRestart  bool              `json:"auto_restart" yaml:"auto_restart"`
	Enabled      bool              `json:"enabled" yaml:"enabled"`
	Environment  map[string]string `json:"environment" yaml:"environment"`
	User         string            `json:"user" yaml:"user"`
	MaxRestarts  int               `json:"max_restarts" yaml:"max_restarts"`
	RestartDelay int               `json:"restart_delay" yaml:"restart_delay"` // 重启延迟秒数
	Description  string            `json:"description" yaml:"description"`
}

// Config 总配置
type Config struct {
	Processes []ProcessConfig `json:"processes" yaml:"processes"`
}

// ProcessStatus 进程状态
type ProcessStatus struct {
	Config       ProcessConfig `json:"config"`
	PID          int           `json:"pid"`
	Status       string        `json:"status"` // running, stopped, error, disabled
	StartTime    time.Time     `json:"start_time"`
	Restarts     int           `json:"restarts"`
	LastError    string        `json:"last_error"`
	LastExitCode int           `json:"last_exit_code"`
	Output       []string      `json:"output"` // 最近的输出日志
}

// ProcessInfo 进程运行信息
type ProcessInfo struct {
	Cmd     *exec.Cmd
	Cancel  context.CancelFunc
	Context context.Context
}

// ProcessManager 进程管理器
type ProcessManager struct {
	processes    map[string]*ProcessStatus
	commands     map[string]*ProcessInfo
	mutex        sync.RWMutex
	config       *Config
	configPath   string
	lastModified time.Time
	logChan      chan logEntry
	stopLogChan  chan struct{}
}

// logEntry 日志条目
type logEntry struct {
	name    string
	message string
	isError bool
}

// NewProcessManager 创建新的进程管理器
func NewProcessManager(configPath string) *ProcessManager {
	pm := &ProcessManager{
		processes:   make(map[string]*ProcessStatus),
		commands:    make(map[string]*ProcessInfo),
		configPath:  configPath,
		logChan:     make(chan logEntry, 256),
		stopLogChan: make(chan struct{}),
	}
	// 启动日志处理goroutine
	go pm.logWorker()
	return pm
}

// logWorker 后台处理日志，避免频繁锁定
func (pm *ProcessManager) logWorker() {
	for {
		select {
		case entry := <-pm.logChan:
			pm.mutex.Lock()
			if status, exists := pm.processes[entry.name]; exists {
				logLine := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), entry.message)
				// 使用环形缓冲优化
				if len(status.Output) >= 50 {
					status.Output = status.Output[1:]
				}
				status.Output = append(status.Output, logLine)
			}
			pm.mutex.Unlock()
		case <-pm.stopLogChan:
			return
		}
	}
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *Config {
	return &Config{
		Processes: []ProcessConfig{
			{
				Name:         "example-service",
				Command:      "/bin/echo",
				Args:         []string{"Hello World"},
				WorkDir:      "/tmp",
				AutoRestart:  true,
				Enabled:      false,
				Environment:  map[string]string{"ENV": "production"},
				User:         "",
				MaxRestarts:  10,
				RestartDelay: 5,
				Description:  "示例服务 - 请修改配置文件",
			},
		},
	}
}

func (pm *ProcessManager) SetConfig(config *Config) {
	pm.config = config
}
func (pm *ProcessManager) SetStatus() {
	for _, psCfg := range pm.config.Processes {
		pm.processes[psCfg.Name] = &ProcessStatus{
			Config: psCfg,
			Status: "stopped",
			Output: make([]string, 0, 50),
		}
	}

}

// StartProcess 启动进程
func (pm *ProcessManager) StartProcess(name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	status, exists := pm.processes[name]
	if !exists {
		return fmt.Errorf("进程 %s 不存在", name)
	}

	if status.Status == "running" {
		return fmt.Errorf("进程 %s 已经在运行", name)
	}

	if !status.Config.Enabled {
		return fmt.Errorf("进程 %s 已被禁用", name)
	}

	config := status.Config

	// 检查可执行文件是否存在
	execPath := config.Command
	if !filepath.IsAbs(execPath) {
		// 如果不是绝对路径，在 PATH 中查找
		if _, err := exec.LookPath(execPath); err != nil {
			status.Status = "error"
			status.LastError = fmt.Sprintf("命令不存在: %s", execPath)
			pm.addLog(name, fmt.Sprintf("ERROR: 命令不存在: %s", execPath))
			return fmt.Errorf("命令不存在: %s", execPath)
		}
	} else {
		if _, err := os.Stat(execPath); os.IsNotExist(err) {
			status.Status = "error"
			status.LastError = fmt.Sprintf("可执行文件不存在: %s", execPath)
			pm.addLog(name, fmt.Sprintf("ERROR: 可执行文件不存在: %s", execPath))
			return fmt.Errorf("可执行文件不存在: %s", execPath)
		}
	}

	// 检查重启次数限制
	if status.Restarts >= config.MaxRestarts {
		status.Status = "disabled"
		status.Config.AutoRestart = false
		pm.addLog(name, fmt.Sprintf("ERROR: 重启次数过多 (%d次)，已禁用自动重启", status.Restarts))
		return fmt.Errorf("进程 %s 重启次数过多，已禁用", name)
	}

	// 创建上下文用于进程控制
	ctx, cancel := context.WithCancel(context.Background())

	// 构建命令
	var cmd *exec.Cmd
	if needsSudo(config.Command, config.User) {
		// 使用 sudo 启动
		args := buildSudoArgs(config)
		cmd = exec.CommandContext(ctx, "sudo", args...)
	} else {
		// 过滤掉空参数
		var filteredArgs []string
		for _, arg := range config.Args {
			if arg != "" {
				filteredArgs = append(filteredArgs, arg)
			}
		}
		cmd = exec.CommandContext(ctx, config.Command, filteredArgs...)
	}

	// 设置工作目录
	if config.WorkDir != "" {
		cmd.Dir = config.WorkDir
	}

	// 设置环境变量
	if len(config.Environment) > 0 {
		env := os.Environ()
		for key, value := range config.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	// 设置进程组，便于管理子进程
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	// 捕获输出
	cmd.Stdout = &logWriter{name: name, pm: pm, isStdout: true}
	cmd.Stderr = &logWriter{name: name, pm: pm, isStdout: false}

	// 启动进程
	err := cmd.Start()
	if err != nil {
		cancel()
		status.Status = "error"
		status.LastError = err.Error()
		pm.addLog(name, fmt.Sprintf("ERROR: 启动失败: %v", err))
		return fmt.Errorf("启动进程 %s 失败: %v", name, err)
	}

	// 保存进程信息
	pm.commands[name] = &ProcessInfo{
		Cmd:     cmd,
		Cancel:  cancel,
		Context: ctx,
	}

	status.PID = cmd.Process.Pid
	status.Status = "running"
	status.StartTime = time.Now()
	status.LastError = ""

	pm.addLog(name, fmt.Sprintf("INFO: 进程启动成功，PID: %d", status.PID))

	// 监控进程状态
	go pm.monitorProcess(name)

	log.Printf("进程 %s 启动成功，PID: %d", name, status.PID)
	return nil
}

// buildSudoArgs 构建 sudo 命令参数
func buildSudoArgs(config ProcessConfig) []string {
	args := []string{}

	// 如果指定了用户，添加-u 参数
	if config.User != "" {
		args = append(args, "-u", config.User)
	}

	args = append(args, config.Command)

	// 添加进程参数
	for _, arg := range config.Args {
		if arg != "" {
			args = append(args, arg)
		}
	}

	return args
}

// StopProcess 停止进程
func (pm *ProcessManager) StopProcess(name string) error {
	pm.mutex.Lock()

	status, exists := pm.processes[name]
	if !exists {
		pm.mutex.Unlock()
		return fmt.Errorf("进程 %s 不存在", name)
	}

	procInfo, cmdExists := pm.commands[name]
	// 允许 stopping 状态进入，防止重复调用 Stop 时报“没有运行”的错，同时隔绝 Start
	if !cmdExists || (status.Status != "running" && status.Status != "stopping") {
		pm.mutex.Unlock()
		return fmt.Errorf("进程 %s 没有运行", name)
	}

	// 如果已经在停止中了，直接返回，防止重复触发停止逻辑
	if status.Status == "stopping" {
		pm.mutex.Unlock()
		log.Printf("进程 %s 正在停止中，请勿重复操作", name)
		return nil
	}

	// 1. 关键点：先标记为 stopping 状态
	status.Status = "stopping"
	pm.addLog(name, "INFO: 正在停止进程...")

	// 取消上下文触发优雅退出
	procInfo.Cancel()
	pm.mutex.Unlock() // 2. 关键点：立刻释放锁！不要带着锁去等待 IO

	// 在锁外面执行等待逻辑
	done := make(chan error, 1)
	go func() {
		done <- procInfo.Cmd.Wait()
	}()

	timeout := 5 * time.Second
	var killed bool

	// 等待进程退出
	select {
	case <-done:
		// 进程已经自然退出
	case <-time.After(timeout):
		// 超时，强制杀死进程组
		if procInfo.Cmd.Process != nil {
			// 注意：这里需要考虑系统兼容性（Windows 不支持负数 PID 杀进程组）
			_ = syscall.Kill(-procInfo.Cmd.Process.Pid, syscall.SIGKILL)
			<-done // 确保 Wait() 协程回收
			killed = true
		}
	}

	// 3. 退出完毕后，重新加锁更新最终状态
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	delete(pm.commands, name)
	status.Status = "stopped"
	status.PID = 0

	if killed {
		pm.addLog(name, fmt.Sprintf("WARNING: 进程未在 %v 内退出，已强制终止", timeout))
	} else {
		pm.addLog(name, "INFO: 进程已手动停止")
	}

	log.Printf("进程 %s 已停止", name)
	return nil
}

// RestartProcess 重启进程
func (pm *ProcessManager) RestartProcess(name string) error {
	// 先停止进程
	err := pm.StopProcess(name)
	if err != nil && !strings.Contains(err.Error(), "没有运行") {
		return err
	}

	// 等待指定时间后重启
	pm.mutex.RLock()
	delay := 2
	if status, exists := pm.processes[name]; exists {
		delay = status.Config.RestartDelay
	}
	pm.mutex.RUnlock()

	time.Sleep(time.Duration(delay) * time.Second)
	return pm.StartProcess(name)
}

// EnableAutoRestart 启用自动重启
func (pm *ProcessManager) EnableAutoRestart(name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	status, exists := pm.processes[name]
	if !exists {
		return fmt.Errorf("进程 %s 不存在", name)
	}

	status.Config.AutoRestart = true
	status.Config.Enabled = true
	status.Restarts = 0 // 重置重启计数
	if status.Status == "disabled" {
		status.Status = "stopped"
	}

	pm.addLog(name, "INFO: 已启用自动重启并重置重启计数")
	return nil
}

// monitorProcess 监控进程状态
func (pm *ProcessManager) monitorProcess(name string) {
	pm.mutex.RLock()
	procInfo, exists := pm.commands[name]
	if !exists {
		pm.mutex.RUnlock()
		return
	}
	cmd := procInfo.Cmd
	pm.mutex.RUnlock()

	err := cmd.Wait()

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	status := pm.processes[name]
	delete(pm.commands, name)

	// 获取退出状态码
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}

		// 如果是被取消的上下文，说明是正常停止
		if err == context.Canceled {
			pm.addLog(name, "INFO: 进程正常停止")
			log.Printf("进程 %s 正常停止", name)
		} else {
			status.LastError = err.Error()
			pm.addLog(name, fmt.Sprintf("ERROR: 进程异常退出: %v (退出码: %d)", err, exitCode))
			log.Printf("进程 %s 异常退出: %v (退出码: %d)", name, err, exitCode)
		}
	} else {
		pm.addLog(name, "INFO: 进程正常退出")
		log.Printf("进程 %s 正常退出", name)
	}

	status.Status = "stopped"
	status.PID = 0
	status.LastExitCode = exitCode

	// 只有在异常退出时才增加重启计数
	if err != nil && err != context.Canceled {
		status.Restarts++

		// 如果重启次数过多，禁用自动重启
		if status.Restarts >= status.Config.MaxRestarts {
			log.Printf("进程 %s 重启次数过多(%d次)，禁用自动重启", name, status.Restarts)
			status.Config.AutoRestart = false
			status.Status = "disabled"
			pm.addLog(name, fmt.Sprintf("WARNING: 重启次数过多 (%d次)，已禁用自动重启", status.Restarts))
			return
		}

		// 自动重启
		if status.Config.AutoRestart && status.Config.Enabled {
			restartDelay := status.Config.RestartDelay
			pm.addLog(name, fmt.Sprintf("INFO: %d秒后自动重启 (第%d次重启)", restartDelay, status.Restarts))
			log.Printf("%d秒后自动重启进程 %s (第%d次重启)", restartDelay, name, status.Restarts)

			// 保存一份配置副本，避免竞态条件
			configName := name
			// 使用 goroutine 避免阻塞
			go func(processName string, delay int) {
				time.Sleep(time.Duration(delay) * time.Second)
				err := pm.StartProcess(processName)
				if err != nil {
					log.Printf("自动重启进程 %s 失败: %v", processName, err)
				}
			}(configName, restartDelay)
		}
	}
}

// addLog 添加日志 - 使用通道避免长时间持有锁
func (pm *ProcessManager) addLog(name, message string) {
	select {
	case pm.logChan <- logEntry{name: name, message: message, isError: false}:
	default:
		// 如果通道满，丢弃最旧的日志以防止堵塞
		<-pm.logChan
		pm.logChan <- logEntry{name: name, message: message, isError: false}
	}
	// 同时记录到标准日志
	log.Printf("%s: %s", name, message)
}

// logWriter 用于捕获进程输出
type logWriter struct {
	name     string
	pm       *ProcessManager
	isStdout bool
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	line := strings.TrimSpace(string(p))
	if line == "" {
		return len(p), nil
	}

	// 构建日志信息
	prefix := "STDOUT"
	if !lw.isStdout {
		prefix = "STDERR"
	}
	message := fmt.Sprintf("%s: %s", prefix, line)

	// 使用通道异步处理日志，避免持有锁
	select {
	case lw.pm.logChan <- logEntry{name: lw.name, message: message, isError: !lw.isStdout}:
	default:
		// 如果通道满，防止阻塞进程输出
	}

	return len(p), nil
}

// needsSudo 检查是否需要 sudo 权限
func needsSudo(command, user string) bool {
	// 如果指定了用户，需要 sudo
	if user != "" {
		return true
	}

	// 检查文件权限或者根据路径判断
	if strings.HasPrefix(command, "/opt/") || strings.HasPrefix(command, "/usr/") {
		return true
	}

	// 检查文件所有者
	if info, err := os.Stat(command); err == nil {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			// 如果文件属于 root 用户
			return stat.Uid == 0
		}
	}

	return false
}

// GetProcesses 获取所有进程状态
func (pm *ProcessManager) GetProcesses() map[string]*ProcessStatus {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	result := make(map[string]*ProcessStatus)
	for k, v := range pm.processes {
		// 创建副本避免并发问题
		statusCopy := *v
		result[k] = &statusCopy
	}
	return result
}

// GetProcesses 获取所有进程状态
func (pm *ProcessManager) GetProcesseByName(svcName string) string {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.processes[svcName].Status
}
