//go:build linux

package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func autoStartPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "autostart", "krab.desktop"), nil
}

func AutoStartEnabled() (bool, error) {
	path, err := autoStartPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func SetAutoStart(enabled bool) error {
	path, err := autoStartPath()
	if err != nil {
		return err
	}
	if !enabled {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return err
	}
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`").Replace(executable)
	content := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=Krab\nExec=\"%s\"\nTerminal=false\nX-GNOME-Autostart-enabled=true\n", escaped)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}
