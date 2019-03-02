package options

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/spf13/viper"
)

var m sync.RWMutex

// Get an option
func Get(k string) string {
	m.RLock()
	defer m.RUnlock()

	return v.GetString(k)
}

// GetInt option
func GetInt(k string) int {
	m.RLock()
	defer m.RUnlock()

	return v.GetInt(k)
}

const (
	// Filename for config file
	Filename = "options"
	// Format for config file
	Format            = "toml"
	defaultSavePath   = "hinatazaka"
	configPathWindows = `AppData\Roaming\hinatazaka\`
	configPathUnix    = ".config/hinatazaka/"
	defaultUserAgent  = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36`
	defaultChromePort = 32719
)

// ConfigPath is the path where track list and config file are kept
var ConfigPath string

var v = viper.New()

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("options.init:", err)
		os.Exit(1)
	}

	// set defaults
	v.SetDefault("user_agent", defaultUserAgent)
	v.SetDefault("chrome_port", defaultChromePort)

	v.SetConfigType(Format)
	v.SetConfigName(Filename)

	var configPath string
	if runtime.GOOS == "windows" {
		configPath = filepath.Join(home, configPathWindows)
	} else {
		configPath = filepath.Join(home, configPathUnix)
	}
	if err != nil {
		fmt.Println("options.init:", err)
		os.Exit(1)
	}

	savePath := filepath.Join(home, defaultSavePath)

	ConfigPath = configPath

	if err := os.MkdirAll(ConfigPath, 0700); err != nil {
		fmt.Println("error:", err)
		return
	}

	v.SetDefault("save_to", savePath)
	v.AddConfigPath(configPath)

	if err := v.ReadInConfig(); err != nil {
		p := filepath.Join(configPath, Filename+"."+Format)
		if err := v.WriteConfigAs(p); err != nil {
			panic(err)
		}
		fmt.Println("[ok] wrote new config file")
	}
}
