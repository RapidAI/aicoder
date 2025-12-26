//go:build windows

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func init() {
	hideConsole()
}

func hideConsole() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	user32 := syscall.NewLazyDLL("user32.dll")

	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	showWindow := user32.NewProc("ShowWindow")

	if getConsoleWindow.Find() == nil && showWindow.Find() == nil {
		hwnd, _, _ := getConsoleWindow.Call()
		if hwnd != 0 {
			showWindow.Call(hwnd, 0) // SW_HIDE = 0
		}
	}
}

func (a *App) platformStartup() {
	hideConsole()
}

func (a *App) updatePathForNode() {
	nodePath := `C:\Program Files\nodejs`
	npmPath := filepath.Join(os.Getenv("AppData"), "npm")

	currentPath := os.Getenv("PATH")
	newPath := currentPath
	
	// Check and add Node.js path
	if _, err := os.Stat(nodePath); err == nil {
		if !strings.Contains(strings.ToLower(currentPath), strings.ToLower(nodePath)) {
			newPath = nodePath + string(os.PathListSeparator) + newPath
		}
	}
	
	// Check and add npm global bin path
	if _, err := os.Stat(npmPath); err == nil {
		if !strings.Contains(strings.ToLower(currentPath), strings.ToLower(npmPath)) {
			newPath = npmPath + string(os.PathListSeparator) + newPath
		}
	}

	if newPath != currentPath {
		os.Setenv("PATH", newPath)
		a.log("Updated PATH environment variable for the current process.")
	}
}

func (a *App) CheckEnvironment() {
	go func() {
		a.log("Checking Node.js installation...")

		// Check for node
		nodeCmd := exec.Command("node", "--version")
		nodeCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := nodeCmd.Run(); err != nil {
			a.log("Node.js not found. Downloading and installing...")
			if err := a.installNodeJS(); err != nil {
				a.log("Failed to install Node.js: " + err.Error())
				return
			}
			a.log("Node.js installed successfully.")
		} else {
			a.log("Node.js is installed.")
		}

		// Update path for the current process anyway to ensure npm is found
		a.updatePathForNode()

		npmPath := "npm"
		if _, err := exec.LookPath("npm"); err != nil {
			npmPath = `C:\Program Files\nodejs\npm.cmd`
		}

		a.log("Checking Claude Code...")

		claudeCheckCmd := exec.Command("claude", "--version")
		claudeCheckCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		claudeExists := claudeCheckCmd.Run() == nil

		if !claudeExists {
			a.log("Claude Code not found. Installing...")
			cmdStr := fmt.Sprintf("%s install -g @anthropic-ai/claude-code", npmPath)
			a.log("Running command: " + cmdStr)
			
			installCmd := exec.Command(npmPath, "install", "-g", "@anthropic-ai/claude-code")
			installCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

			if out, err := installCmd.CombinedOutput(); err != nil {
				a.log("Failed to install Claude Code: " + string(out))
			} else {
				a.log("Claude Code installed successfully. Refreshing environment...")
				a.updatePathForNode()
			}
		} else {
			a.log("Claude Code found. Checking for updates...")
			cmdStr := fmt.Sprintf("%s install -g @anthropic-ai/claude-code", npmPath)
			a.log("Running command: " + cmdStr)

			installCmd := exec.Command(npmPath, "install", "-g", "@anthropic-ai/claude-code")
			installCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if out, err := installCmd.CombinedOutput(); err != nil {
				a.log("Failed to update Claude Code: " + string(out))
			} else {
				a.log("Claude Code updated successfully.")
				a.updatePathForNode()
			}
		}

		a.log("Environment check complete.")
		runtime.EventsEmit(a.ctx, "env-check-done")
	}()
}

func (a *App) installNodeJS() error {
	arch := os.Getenv("PROCESSOR_ARCHITECTURE")
	nodeArch := "x64"
	if arch == "ARM64" || os.Getenv("PROCESSOR_ARCHITEW6432") == "ARM64" {
		nodeArch = "arm64"
	}

	// Using a more recent version
	nodeVersion := "22.14.0"
	fileName := fmt.Sprintf("node-v%s-%s.msi", nodeVersion, nodeArch)
	
	downloadURL := fmt.Sprintf("https://nodejs.org/dist/v%s/%s", nodeVersion, fileName)
	if strings.HasPrefix(strings.ToLower(a.CurrentLanguage), "zh") && nodeArch != "arm64" {
		// Use a mirror in China for faster download (only for x64 as arm64 might not be synced)
		downloadURL = fmt.Sprintf("https://mirrors.tuna.tsinghua.edu.cn/nodejs-release/v%s/%s", nodeVersion, fileName)
	}

	a.log(fmt.Sprintf("Downloading Node.js %s for %s...", nodeVersion, nodeArch))

	// Pre-check if the file exists and is accessible
	client := &http.Client{Timeout: 10 * time.Second}
	headReq, _ := http.NewRequest("HEAD", downloadURL, nil)
	headReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	headResp, err := client.Do(headReq)
	if err != nil || headResp.StatusCode != http.StatusOK {
		status := "Unknown"
		if headResp != nil {
			status = headResp.Status
		}
		return fmt.Errorf("Node.js installer is not accessible (Status: %s). Please check your internet connection or mirror availability.", status)
	}
	headResp.Body.Close()

	tempDir := os.TempDir()
	msiPath := filepath.Join(tempDir, fileName)

	if err := a.downloadFile(msiPath, downloadURL); err != nil {
		return fmt.Errorf("error downloading Node.js installer: %w", err)
	}
	defer os.Remove(msiPath)

	a.log("Installing Node.js (this may take a moment, please grant administrator permission if prompted)...")
	// Use /passive for basic UI or /qn for completely silent.
	// Adding ALLUSERS=1 to ensure it's in the standard path.
	cmd := exec.Command("msiexec", "/i", msiPath, "/passive", "ALLUSERS=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error installing Node.js: %s\n%s", err, string(output))
	}

	// Wait a bit for the system to finalize the installation
	time.Sleep(2 * time.Second)

	return nil
}

func (a *App) downloadFile(filepath string, url string) error {
	a.log(fmt.Sprintf("Requesting URL: %s", url))
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	// Add User-Agent to avoid 403 Forbidden from some mirrors/CDNs
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Use a client with timeout for the connection phase
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error during download: %v. Please check your internet connection or firewall settings.", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s. The file might not be available on this server.", resp.Status)
	}

	size := resp.ContentLength
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer out.Close()

	var downloaded int64
	buffer := make([]byte, 32768)
	lastReport := time.Now()

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			out.Write(buffer[:n])
			downloaded += int64(n)
			if size > 0 && time.Since(lastReport) > 500*time.Millisecond {
				percent := float64(downloaded) / float64(size) * 100
				a.log(fmt.Sprintf("Downloading Node.js (%.1f%%): %d/%d bytes", percent, downloaded, size))
				lastReport = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("interrupted download: %v. The connection was lost during data transfer.", err)
		}
	}

	return nil
}

func (a *App) restartApp() {
	executable, err := os.Executable()
	if err != nil {
		a.log("Failed to get executable path: " + err.Error())
		return
	}

	cmd := exec.Command("cmd", "/c", "start", "", executable)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		a.log("Failed to restart: " + err.Error())
	} else {
		runtime.Quit(a.ctx)
	}
}

func (a *App) LaunchClaude(yoloMode bool, projectDir string) {
	args := []string{"/c", "start", "cmd.exe", "/k", "claude"}
	if yoloMode {
		args = append(args, "--dangerously-skip-permissions")
	}
	
	cmd := exec.Command("cmd.exe", args...)
	if projectDir != "" {
		cmd.Dir = projectDir
	}
	
	cmd.Env = os.Environ()
	
	if err := cmd.Start(); err != nil {
		a.log("Failed to launch Claude: " + err.Error())
	}
}

func (a *App) syncToSystemEnv(config AppConfig) {
	var selectedModel *ModelConfig
	for _, m := range config.Models {
		if m.ModelName == config.CurrentModel {
			selectedModel = &m
			break
		}
	}

	if selectedModel == nil {
		return
	}

	baseUrl := getBaseUrl(selectedModel)

	// Set environment variables for the current process immediately
	os.Setenv("ANTHROPIC_AUTH_TOKEN", selectedModel.ApiKey)
	os.Setenv("ANTHROPIC_BASE_URL", baseUrl)

	// Set persistent environment variables on Windows in a goroutine because setx is slow
	go func() {
		cmd1 := exec.Command("setx", "ANTHROPIC_AUTH_TOKEN", selectedModel.ApiKey)
		cmd1.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd1.Run()

		cmd2 := exec.Command("setx", "ANTHROPIC_BASE_URL", baseUrl)
		cmd2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd2.Run()
	}()
}
