package core

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/slinxlink/node/internal/database"
	"github.com/slinxlink/node/internal/util"
)

const singBoxReleaseURL = "https://api.github.com/repos/MoewSama/node/releases/latest"

type singBoxRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type CoreUpdateResult struct {
	HasUpdate     bool   `json:"has_update"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
}

// CheckCoreUpdate 检查 sing-box 核心是否有新版本
func CheckCoreUpdate() (*CoreUpdateResult, error) {
	current := Default.Version()

	// 获取最新 release
	resp, err := http.Get(singBoxReleaseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var release singBoxRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	// 从资产名中提取 sing-box 版本号（sing-box-版本号_linux_arch.gz）
	arch := runtime.GOARCH
	prefix := "sing-box-"
	suffix := fmt.Sprintf("_linux_%s.gz", arch)
	latest := ""
	for _, a := range release.Assets {
		if strings.HasPrefix(a.Name, prefix) && strings.HasSuffix(a.Name, suffix) {
			// 去掉前缀和后缀，中间就是版本号
			ver := strings.TrimPrefix(a.Name, prefix)
			ver = strings.TrimSuffix(ver, suffix)
			latest = ver
			break
		}
	}

	return &CoreUpdateResult{
		HasUpdate:      latest != "" && latest != current,
		CurrentVersion: current,
		LatestVersion:  latest,
	}, nil
}

// UpdateCoreBin 下载并更新 sing-box 核心
// force=true 时跳过版本比对，同版本也重新下载覆盖（用于重装/换构建 tag 的核心）
func UpdateCoreBin(force bool) error {
	// 获取最新 release
	resp, err := http.Get(singBoxReleaseURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var release singBoxRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return err
	}

	// 找到对应架构的资产（sing-box-版本号_linux_arch.gz）
	arch := runtime.GOARCH
	prefix := fmt.Sprintf("sing-box-")
	suffix := fmt.Sprintf("_linux_%s.gz", arch)
	var downloadURL string
	var targetVersion string
	for _, a := range release.Assets {
		if strings.HasPrefix(a.Name, prefix) && strings.HasSuffix(a.Name, suffix) {
			downloadURL = a.BrowserDownloadURL
			targetVersion = strings.TrimSuffix(strings.TrimPrefix(a.Name, prefix), suffix)
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("未找到资产: %s*%s", prefix, suffix)
	}

	// 非 force 时跳过同版本重装
	if !force && targetVersion != "" && targetVersion == Default.Version() {
		return fmt.Errorf("核心已是最新版本 %s（如需重装请使用强制更新）", targetVersion)
	}

	// 获取 core 配置的 BinPath
	var coreRecord database.Core
	database.DB.First(&coreRecord)
	binPath := coreRecord.BinPath
	if binPath == "" {
		binPath = "bin/sing-box"
	}

	util.Info("[core] 开始更新 sing-box 核心: %s", release.TagName)

	// 下载 .gz 文件
	tmpDir, err := os.MkdirTemp("", "singbox-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// 从下载 URL 中提取文件名
	gzName := downloadURL[strings.LastIndex(downloadURL, "/")+1:]
	gzPath := filepath.Join(tmpDir, gzName)
	if err := downloadFile(downloadURL, gzPath); err != nil {
		util.Error("[core] 下载 sing-box 失败: %v", err)
		return fmt.Errorf("下载失败: %w", err)
	}

	binPathAbs, err := filepath.Abs(binPath)
	if err != nil {
		return err
	}

	// 停止核心（正在运行中文件被占用无法覆盖）
	wasRunning := Default.Status() == "running"
	if wasRunning {
		util.Info("[core] 核心运行中，先停止...")
		if err := Default.Stop(); err != nil {
			return fmt.Errorf("停止核心失败: %w", err)
		}
	}

	// 解压 .gz 覆盖二进制
	if err := extractGz(gzPath, binPathAbs); err != nil {
		util.Error("[core] 解压 sing-box 失败: %v", err)
		return fmt.Errorf("解压失败: %w", err)
	}

	// 设置执行权限
	if err := os.Chmod(binPathAbs, 0755); err != nil {
		return err
	}

	// 重新启动核心
	if wasRunning {
		util.Info("[core] 重新启动核心...")
		if err := Default.Start(); err != nil {
			return fmt.Errorf("启动核心失败: %w", err)
		}
	}

	util.Info("[core] sing-box 核心更新成功: %s", release.TagName)
	return nil
}

func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// 解压 .gz 到目标目录，覆盖同名文件
func extractGz(gzPath, binPath string) error {
	f, err := os.Open(gzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	out, err := os.Create(binPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, gzReader)
	return err
}