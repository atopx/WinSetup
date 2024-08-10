package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Location map[string]string `toml:"location"`
	Target   []*TargetItem     `toml:"target"`
}

type TargetItem struct {
	Id   string `toml:"id"`
	name string
	Path string `toml:"path"`
	Link string `toml:"link"`
	// Version 使用指定的版本；默认为最新版本
	Version string `toml:"version"`
	// NoUpgrade 如果已安装的版本已存在，则跳过升级
	NoUpgrade bool `toml:"no-upgrade"`
	// IgnoreSecurityHash 忽略安装程序哈希检查失败
	IgnoreSecurityHash bool `toml:"ignore-security-hash"`
	// UninstallPrevious 升级期间卸载以前版本的程序包
	UninstallPrevious bool `toml:"uninstall-previous"`
	// skip-dependencies 跳过处理包依赖项和 Windows 功能
	SkipDependencies bool `toml:"skip-dependencies"`
}

var cfg = new(Config)

func GetInstalledMap() map[string]bool {
	data, err := exec.Command("pwsh", "-Command", "Get-WinGetPackage | Select ID").CombinedOutput()
	if err != nil {
		log.Fatalf("get installed app failed: %s", err)
	}
	list := strings.Split(string(bytes.TrimSpace(data)), "\n")[2:]
	installedApps := make(map[string]bool, len(list))
	for _, line := range list {
		id := strings.ToLower(strings.TrimSpace(line))
		installedApps[id] = true
	}
	return installedApps
}

func MustPathExists(path string) (err error) {
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModePerm)
	}
	return err
}

func CleanEmptyDir(dir string) {
	var files []fs.DirEntry
	files, err := os.ReadDir(dir)
	if err != nil {
		slog.Error(fmt.Sprintf("clean empty dir [%s] failed: %s", dir, err.Error()))
		return
	}
	if len(files) == 0 {
		_ = os.RemoveAll(dir)
	}
}

func installer(item *TargetItem) error {
	location := strings.Join([]string{strings.TrimRight(cfg.Location[item.Link], "\\"), item.Path}, "\\")
	if err := MustPathExists(location); err != nil {
		return fmt.Errorf("check path error: %w", err)
	}
	defer CleanEmptyDir(location)
	command := fmt.Sprintf("winget install --id %s -l %s --verbose", item.Id, location)
	if item.IgnoreSecurityHash {
		command += " --ignore-security-hash"
	}
	if item.NoUpgrade {
		command += " --no-upgrade"
	}
	if item.Version != "" {
		command += " --version" + item.Version
	}
	if item.UninstallPrevious {
		command += " --uninstall-previous"
	}
	if item.SkipDependencies {
		command += " --skip-dependencies"
	}
	item.name = item.Id[strings.LastIndex(item.Id, ".")+1:]
	slog.Info(fmt.Sprintf("[%s] start install: %s\n", item.name, command))
	execCmd := exec.Command("pwsh", "-Command", command)
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("[%s] error creating StdoutPipe for command: %w", item.name, err)
	}

	if err = execCmd.Start(); err != nil {
		return fmt.Errorf("[%s] error starting command: %w", item.name, err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fmt.Printf("[%s] %s\n", item.name, scanner.Text())
	}

	if err = execCmd.Wait(); err != nil {
		return fmt.Errorf("[%s] install failed", item.name)
	}

	return nil
}

func Installer(queue <-chan *TargetItem, wg *sync.WaitGroup) {
	defer wg.Done()
	for item := range queue {
		if err := installer(item); err != nil {
			slog.Error(err.Error())
		}
	}
}

func main() {

	confPath := flag.String("c", "config.toml", "配置路径, 默认当前路径的`config.toml`")
	concurrent := flag.Int("n", 4, "并发安装的协程数量, 默认`4`")
	flag.Parse()

	data, err := os.ReadFile(*confPath)
	if err != nil {
		log.Fatalf("load the configuration file failed: %s", err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("configuration file format error: %s", err)
	}
	installedApps := GetInstalledMap()

	wg := new(sync.WaitGroup)
	wg.Add(*concurrent)

	queue := make(chan *TargetItem, *concurrent)

	for range *concurrent {
		go Installer(queue, wg)
	}

	for _, target := range cfg.Target {
		id := strings.ToLower(target.Id)
		if installedApps[id] {
			slog.Info(fmt.Sprintf("skip installed app: %s\n", target.Id))
			continue
		}
		if target.Path == "" {
			target.Path = strings.ReplaceAll(id, ".", "\\")
		}
		queue <- target
	}
	close(queue)
	wg.Wait()
}
