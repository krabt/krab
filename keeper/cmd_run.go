package keeper

import (
	"fmt"
	"os/exec"
	"strings"
)

// CheckSudoPermission 检查当前是否已有 sudo 权限（不会弹出密码框）
func CheckSudoPermission() bool {
	cmd := exec.Command("sudo", "-n", "true")
	err := cmd.Run()
	return err == nil
}

// RunWithSudo 使用图形界面请求管理员权限执行命令
func RunWithSudo(shellCmd string) (string, error) {
	escapedCmd := strings.ReplaceAll(shellCmd, `"`, `\"`)
	script := fmt.Sprintf(
		`do shell script "%s" with administrator privileges`,
		escapedCmd,
	)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// RunCommandWithPrivilege 根据权限决定是否弹出密码窗口
func RunCommandWithPrivilege(shellCmd string) (string, error) {
	// 如果已有 sudo 权限，直接执行
	if CheckSudoPermission() {
		cmd := exec.Command("bash", "-c", shellCmd)
		output, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(output)), err
	}

	// 否则弹出 macOS 密码输入框
	return RunWithSudo(shellCmd)
}

func SetProxy(port string) {
	cmd := fmt.Sprintf("networksetup -setwebproxy Wi-Fi 127.0.0.1 %v && networksetup -setsecurewebproxy Wi-Fi 127.0.0.1 %s && networksetup -setwebproxystate Wi-Fi on && networksetup -setsecurewebproxystate Wi-Fi on", port, port)
	result, err := RunCommandWithPrivilege(cmd)
	if err != nil {
		fmt.Printf("设置系统代理失败, msg: %v, err %v", result, err)
	} else {
		fmt.Print("设置系统代理成功")
	}

}

func UnsetProxy() {
	cmd := "networksetup -setwebproxystate Wi-Fi off && networksetup -setsecurewebproxystate Wi-Fi off"
	result, err := RunCommandWithPrivilege(cmd)
	if err != nil {
		fmt.Printf("取消系统代理失败, msg: %v, err %v", result, err)
	} else {
		fmt.Print("取消设置系统代理成功")
	}
}
