package storage

import (
	"archive/zip"
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/healthManager"
	"elake-api-gateway/internal/logger"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/ip2location/ip2location-go"
	"go.uber.org/zap"
)

// filePrefix IP数据库文件前缀
const filePrefix = "IPDB-"

// fileSuffix IP数据库文件后缀
const fileSuffix = ".bin"

// IPDB IP数据库包装器
type IPDB struct {
	ipdb     *ip2location.DB
	mu       sync.RWMutex
	filePath string
}

// downloadCounter 下载次数计数器
var downloadCounter struct {
	mu   sync.Mutex
	day  string
	used int
}

// Get 返回 IPDB 句柄
func (w *IPDB) Get() *ip2location.DB {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.ipdb
}

// InitIPDB 初始化 IPDB
func InitIPDB() *IPDB {
	cfg := config.Get()
	dataPath := cfg.Gateway.DataPath
	latest, err := FindLatestFile(dataPath)
	if err != nil {
		logger.Log.Error("扫描 IPDB 目录失败", zap.Error(err))
		return nil
	}
	// 如果没有文件, 首次下载
	if latest == "" {
		logger.Log.Info("IPDB 不存在")
		newFile, err := DownloadNewVersion()
		if err != nil {
			logger.Log.Error("下载失败", zap.Error(err))
			return nil
		}
		latest = filepath.Join(dataPath, newFile)
	}
	// 加载数据库
	db, err := ip2location.OpenDB(latest)
	if err != nil {
		logger.Log.Error("加载 IPDB 失败", zap.Error(err))
		return nil
	}
	wrapper := &IPDB{
		ipdb:     db,
		filePath: latest,
	}
	// 测试用
	//logger.Log.Info("触发 IPDB 自动更新")
	//if err := Update(wrapper); err != nil {
	//	logger.Log.Error("更新失败", zap.Error(err))
	//}
	go StartUpdateTask(wrapper)
	healthManager.Global().Register(
		"IPDB",
		cfg.Health.IPDBHealthCheckFailThreshold,
		cfg.Health.IPDBHealthCheckOKThreshold,
	)
	// 健康检查
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.Health.IPDBHealthCheckInterval) * time.Second)
		for range ticker.C {
			ipdb := wrapper.Get()
			if ipdb == nil {
				healthManager.Global().Report("IPDB", errors.New("ipdb nil"))
				continue
			}
			_, err := ipdb.Get_all("8.8.8.8")
			healthManager.Global().Report("IPDB", err)
		}
	}()
	return wrapper
}

// FindLatestFile 查找目录中最新的 IPDB 文件
func FindLatestFile(dir string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var list []string
	for _, f := range files {
		name := f.Name()
		if strings.HasPrefix(name, filePrefix) &&
			strings.HasSuffix(name, fileSuffix) {
			list = append(list, name)
		}
	}
	if len(list) == 0 {
		return "", nil
	}
	sort.Strings(list)
	return filepath.Join(dir, list[len(list)-1]), nil
}

// StartUpdateTask 启动 IPDB 更新任务
func StartUpdateTask(wrapper *IPDB) {
	// 启动时先清理一次临时文件
	go CleanupTempFiles()
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		time.Sleep(time.Until(next))
		// 每天清理一次临时文件
		go CleanupTempFiles()
		if ShouldUpdate(wrapper.GetFilePath()) {
			logger.Log.Info("触发 IPDB 自动更新")
			if err := Update(wrapper); err != nil {
				logger.Log.Error("更新失败", zap.Error(err))
			}
			// 更新完成后清理临时文件
			go CleanupTempFiles()
		}
	}
}

// CleanupTempFiles 清理 IPDB 产生的临时文件
func CleanupTempFiles() {
	// 清理临时目录中的 IPDB 临时文件
	cfg := config.Get()
	tempPath := cfg.Gateway.TempPath
	if err := CleanupFilesInDir(tempPath, func(name string) bool {
		// 清理 IPDB_ 开头的zip文件和 IPDB -开头的bin文件
		return strings.HasPrefix(name, filePrefix) && strings.HasSuffix(name, ".zip") ||
			strings.HasPrefix(name, filePrefix) && strings.HasSuffix(name, fileSuffix)
	}); err != nil {
		logger.Log.Error("清理临时文件失败", zap.Error(err))
	}
	// 清理数据目录中的过期 IPDB 文件, 只保留最新的一个
	dataPath := cfg.Gateway.DataPath
	if err := CleanupOldIPDBFiles(dataPath); err != nil {
		logger.Log.Error("清理过期 IPDB 文件失败", zap.Error(err))
	}
}

// CleanupFilesInDir 清理目录中符合条件的文件
func CleanupFilesInDir(dir string, filter func(string) bool) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.IsDir() && filter(f.Name()) {
			filePath := filepath.Join(dir, f.Name())
			if err := os.Remove(filePath); err != nil {
				logger.Log.Warn("删除文件失败", zap.String("file", filePath), zap.Error(err))
			}
		}
	}
	return nil
}

// CleanupOldIPDBFiles 清理数据目录中的过期 IPDB 文件, 只保留最新的一个
func CleanupOldIPDBFiles(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var ipdbFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), filePrefix) && strings.HasSuffix(f.Name(), fileSuffix) {
			ipdbFiles = append(ipdbFiles, f.Name())
		}
	}
	if len(ipdbFiles) <= 1 {
		return nil
	}
	// 按文件名排序, 最新的文件在最后
	sort.Strings(ipdbFiles)
	// 保留最后一个文件, 删除其他文件
	for i := 0; i < len(ipdbFiles)-1; i++ {
		filePath := filepath.Join(dir, ipdbFiles[i])
		if err := os.Remove(filePath); err != nil {
			logger.Log.Warn("删除过期 IPDB 文件失败", zap.String("file", filePath), zap.Error(err))
		}
	}
	return nil
}

// GetFilePath 返回当前 IPDB 文件路径
func (w *IPDB) GetFilePath() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.filePath
}

// ExtractYearMonth 从文件名提取年月
func ExtractYearMonth(filePath string) (string, error) {
	base := filepath.Base(filePath)
	base = strings.TrimSuffix(base, fileSuffix)
	base = strings.TrimPrefix(base, filePrefix)
	// 处理带有时间戳的文件名, 例如 20260303_123456789
	if len(base) < 8 {
		return "", fmt.Errorf("invalid file name")
	}
	// 提取前8个字符作为日期部分
	datePart := base[:8]
	// 验证日期部分是否为数字
	for _, c := range datePart {
		if c < 0 || c > 9 {
			return "", fmt.Errorf("invalid file name")
		}
	}
	return datePart[:6], nil
}

// ShouldUpdate 判断是否需要更新
func ShouldUpdate(currentFile string) bool {
	now := time.Now()
	// 只在每月 3, 4, 5 号尝试, 错过这几天就不搜了, 避免每天轮询
	if now.Day() < 3 || now.Day() > 5 {
		return false
	}
	currentYM, err := ExtractYearMonth(currentFile)
	nowYM := now.Format("200601")
	if err == nil && currentYM == nowYM {
		return false
	}
	// 检查今天是否还有剩余重试次数
	downloadCounter.mu.Lock()
	defer downloadCounter.mu.Unlock()
	today := now.Format("20060102")
	if downloadCounter.day == today && downloadCounter.used >= config.Get().IPDB.MaxDownloadAttempts {
		// 如果今天的 5 次机会已经用完了, 等明天凌晨 3 点
		return false
	}
	return true
}

// Update 更新 IPDB
func Update(wrapper *IPDB) error {
	newFile, err := DownloadNewVersion()
	if err != nil {
		return err
	}
	cfg := config.Get()
	dataPath := cfg.Gateway.DataPath
	fullPath := filepath.Join(dataPath, newFile)
	newDB, err := ip2location.OpenDB(fullPath)
	if err != nil {
		return err
	}
	wrapper.mu.Lock()
	oldDB := wrapper.ipdb
	oldFile := wrapper.filePath
	wrapper.ipdb = newDB
	wrapper.filePath = fullPath
	wrapper.mu.Unlock()
	if oldDB != nil {
		oldDB.Close()
	}
	// 删除旧文件
	if oldFile != "" && oldFile != fullPath {
		_ = os.Remove(oldFile)
	}
	return nil
}

// DownloadNewVersion 下载新版本
func DownloadNewVersion() (string, error) {
	cfg := config.Get().IPDB
	for i := 1; i <= cfg.MaxDownloadAttempts; i++ {
		if i > 1 {
			retryTime := time.Duration(i*i) * time.Second
			logger.Log.Info(fmt.Sprintf("%v 秒后, 第 %d 次尝试重新下载 IPDB", retryTime, i))
			time.Sleep(retryTime)
		}
		if !AllowDownloadToday() {
			return "", fmt.Errorf("今日下载次数已用尽")
		}
		fileName, err := DownloadAndVerifyOnce()
		if err == nil {
			return fileName, nil
		}
		// 检查是否是DNS相关错误
		if IsDNSError(err) {
			wait := time.Duration(i*i*5) * time.Second
			logger.Log.Warn("DNS 解析失败，准备重试",
				zap.Int("attempt", i),
				zap.Duration("wait", wait),
				zap.Error(err),
			)
			time.Sleep(wait)
			continue
		}
		logger.Log.Error("下载或校验失败", zap.Int("attempt", i), zap.Error(err))
		// 如果是下载次数限制错误, 直接返回不再重试
		if err.Error() == "下载次数已达上限, 请24小时后再试" {
			return "", err
		}
	}
	return "", fmt.Errorf("超过最大下载次数 %d 次", cfg.MaxDownloadAttempts)
}

// AllowDownloadToday 检查今日是否已下载超过最大次数
func AllowDownloadToday() bool {
	cfg := config.Get().IPDB
	today := time.Now().Format("20060102")
	downloadCounter.mu.Lock()
	defer downloadCounter.mu.Unlock()
	if downloadCounter.day != today {
		downloadCounter.day = today
		downloadCounter.used = 0
	}
	if downloadCounter.used >= cfg.MaxDownloadAttempts {
		return false
	}
	downloadCounter.used++
	return true
}

// DownloadAndVerifyOnce 下载并校验一次
func DownloadAndVerifyOnce() (string, error) {
	cfg := config.Get()
	tempPath := cfg.Gateway.TempPath
	dataPath := cfg.Gateway.DataPath
	dateStr := time.Now().Format("20060102")
	fileName := filePrefix + dateStr + fileSuffix
	tempZip := filepath.Join(
		tempPath,
		fmt.Sprintf("%s%d.zip", filePrefix, time.Now().UnixNano()),
	)
	tempBin := filepath.Join(tempPath, fileName)
	url := fmt.Sprintf(
		"https://www.ip2location.com/download/?token=%s&file=%s",
		cfg.IPDB.Token,
		"DB11LITEBINIPV6",
	)
	// 下载
	if err := DownloadFile(url, tempZip); err != nil {
		return "", err
	}
	// zip 校验
	if err := ValidateZip(tempZip); err != nil {
		_ = os.Remove(tempZip)
		return "", err
	}
	// zip 解压
	if err := UnzipAndExtractBIN(tempZip, tempBin); err != nil {
		_ = os.Remove(tempZip)
		return "", err
	}
	// bin 校验
	testDB, err := ip2location.OpenDB(tempBin)
	if err != nil {
		_ = os.Remove(tempZip)
		_ = os.Remove(tempBin)
		return "", err
	}
	testDB.Close()
	// 移动文件到数据目录
	finalPath := filepath.Join(dataPath, fileName)
	// 先尝试直接重命名
	if err := os.Rename(tempBin, finalPath); err != nil {
		// 如果是因为目标文件已存在, 尝试先删除目标文件
		if os.IsExist(err) {
			if removeErr := os.Remove(finalPath); removeErr == nil {
				// 删除成功后再次尝试重命名
				if renameErr := os.Rename(tempBin, finalPath); renameErr == nil {
					_ = os.Remove(tempZip)
					return fileName, nil
				}
			}
		}
		// 如果删除失败(可能是因为文件被占用), 使用临时文件名
		tempFinalName := filePrefix + dateStr + "_" + fmt.Sprintf("%d", time.Now().UnixNano()) + fileSuffix
		tempFinalPath := filepath.Join(dataPath, tempFinalName)
		if err := os.Rename(tempBin, tempFinalPath); err != nil {
			_ = os.Remove(tempZip)
			return "", err
		}
		_ = os.Remove(tempZip)
		return tempFinalName, nil
	}
	_ = os.Remove(tempZip)
	return fileName, nil
}

// DownloadFile 下载文件
func DownloadFile(url, dest string) error {
	// 创建客户端
	client := grab.NewClient()
	client.HTTPClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
	req, err := grab.NewRequest(dest, url)
	req.HTTPRequest.Header.Set("User-Agent", "Mozilla/5.0")
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	resp := client.Do(req)
	t := time.NewTicker(3 * time.Second)
	done := false
	for !done {
		select {
		case <-t.C:
			logger.Log.Info(fmt.Sprintf(
				"IPDB 下载进度: %.2f%%(%.2f KB/s | %d MB)",
				resp.Progress()*100,
				resp.BytesPerSecond()/1024,
				resp.Size()/1024/1024,
			))
		case <-resp.Done:
			t.Stop()
			done = true
		}
	}
	if err := resp.Err(); err != nil {
		_ = os.Remove(dest)
		return err
	}
	// 检查下载的文件是否是HTML错误页面
	if isHTML, err := IsDownloadLimitError(dest); err != nil {
		_ = os.Remove(dest)
		return fmt.Errorf("检查下载文件失败: %v", err)
	} else if isHTML {
		_ = os.Remove(dest)
		// 设置今日下载次数为最大值, 避免重试
		SetDownloadLimitReached()
		return fmt.Errorf("下载次数已达上限, 请24小时后再试")
	}
	logger.Log.Info("IPDB 下载完成")
	return nil
}

// IsDNSError 检查错误是否是DNS相关错误
func IsDNSError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "Temporary failure in name resolution")
}

// IsDownloadLimitError 检查下载的文件是否是HTML错误页面(下载次数限制)
func IsDownloadLimitError(path string) (bool, error) {
	// 读取文件前几个字节检查是否是HTML
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	// 检查文件内容是否包含下载限制错误信息
	content := string(data)
	if strings.Contains(content, "THIS FILE CAN ONLY BE DOWNLOADED 5 TIMES WITHIN 24 HOURS") {
		return true, nil
	}
	return false, nil
}

// SetDownloadLimitReached 设置今日下载次数已达上限
func SetDownloadLimitReached() {
	cfg := config.Get().IPDB
	today := time.Now().Format("20060102")
	downloadCounter.mu.Lock()
	defer downloadCounter.mu.Unlock()
	downloadCounter.day = today
	downloadCounter.used = cfg.MaxDownloadAttempts
	logger.Log.Warn("IPDB 下载次数已达上限, 已设置今日下载次数为最大值")
}

// ValidateZip 校验 ZIP 文件是否包含指定的 BIN 文件
func ValidateZip(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("ZIP 文件损坏: %v", err)
	}
	defer func(r *zip.ReadCloser) {
		_ = r.Close()
	}(r)
	const targetName = "IP2LOCATION-LITE-DB11.IPV6.BIN"
	for _, f := range r.File {
		if strings.EqualFold(f.Name, targetName) {
			return nil
		}
	}
	return fmt.Errorf("ZIP 内未找到目标 BIN 文件")
}

// UnzipAndExtractBIN 解压 ZIP 文件并提取指定的 BIN 文件
func UnzipAndExtractBIN(srcZip, destBin string) error {
	logger.Log.Info("正在解压 IPDB")
	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer func(r *zip.ReadCloser) {
		_ = r.Close()
	}(r)
	const targetName = "IP2LOCATION-LITE-DB11.IPV6.BIN"
	for _, f := range r.File {
		if strings.EqualFold(f.Name, targetName) {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			out, err := os.Create(destBin)
			if err != nil {
				_ = rc.Close()
				return err
			}
			_, err = io.Copy(out, rc)
			if closeErr := rc.Close(); closeErr != nil {
				logger.Log.Error("关闭 ZIP 内文件失败", zap.Error(closeErr))
			}
			if closeErr := out.Close(); closeErr != nil {
				logger.Log.Error("关闭输出文件失败", zap.Error(closeErr))
			}
			return err
		}
	}
	return fmt.Errorf("ZIP 中未找到目标文件")
}
