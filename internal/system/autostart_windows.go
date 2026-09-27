//go:build windows

package system

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const autoStartRegistryPath = `Software\Microsoft\Windows\CurrentVersion\Run`
const autoStartValueName = "Krab"

func AutoStartEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, autoStartRegistryPath, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	defer key.Close()
	_, _, err = key.GetStringValue(autoStartValueName)
	if err == nil {
		return true, nil
	}
	if err == registry.ErrNotExist {
		return false, nil
	}
	return false, err
}

func SetAutoStart(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, autoStartRegistryPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue(autoStartValueName); err != nil && err != registry.ErrNotExist {
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
	return key.SetStringValue(autoStartValueName, fmt.Sprintf(`"%s"`, executable))
}
