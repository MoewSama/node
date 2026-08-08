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

	latest := strings.TrimPrefix(release.TagName, "v")

	return &CoreUpdateResult{
		HasUpdate:      latest != "" && latest != current,
		CurrentVersion: current,
		LatestVersion:  latest,
	}, nil
}

// UpdateCoreBin 下载并更新 sing-box 核心
func UpdateCoreBin() error {
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

	// 找到对应架构的资产
	arch := runtime.GOARCH
	assetName := fmt.Sprintf("sing-box_linux_%s.gz", arch)
	var downloadURL string
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("未找到资产: %s", assetName)
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

	gzPath := filepath.Join(tmpDir, assetName)
	if err := downloadFile(downloadURL, gzPath); err != nil {
		util.Error("[core] 下载 sing-box 失败: %v", err)
		return fmt.Errorf("下载失败: %w", err)
	}

	binPathAbs, err := filepath.Abs(binPath)
	if err != nil {
		return err
	}

	// 解压 .gz 覆盖二进制
	if err := extractGz(gzPath, binPathAbs); err != nil {
		util.Error("[core] 解压 sing-box 失败: %v", err)
		return fmt.Errorf("解压失败: %w", err)
	}

	// 检查解压出的二进制是否存在
	if _, err := os.Stat(binPathAbs); err != nil {
		return fmt.Errorf("解压后找不到二进制: %w", err)
	}

	// 设置执行权限
	if err := os.Chmod(binPathAbs, 0755); err != nil {
		return err
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