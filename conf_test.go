package conf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ----------------------------------------------------------------
// 定义测试用的配置结构体
// ----------------------------------------------------------------

type Database struct {
	Host string `mapstructure:"host" validate:"required"`
	Port int    `mapstructure:"port" validate:"min=1024" default:"3306"`
	// 生产环境必须来自 Env
	Password string `mapstructure:"password" env:"strict"`
}

type TestConfig struct {
	AppName  string   `mapstructure:"app_name" default:"TestApp"`
	Debug    bool     `mapstructure:"debug"`
	Database Database `mapstructure:"database"`
}

type StrictConfig struct {
	// 测试带参数的 tag 解析
	Password string `mapstructure:"password,omitempty" env:"strict"`
	// 测试指针递归
	Sub *StrictSub `mapstructure:"sub"`
}

type StrictSub struct {
	ApiKey string `mapstructure:"api_key" env:"strict"`
}

// ----------------------------------------------------------------
// 辅助函数
// ----------------------------------------------------------------

// createConfigFile 在临时目录创建一个配置文件，返回目录路径
func createConfigFile(t *testing.T, fileName, content string) string {
	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	return dir
}

// ----------------------------------------------------------------
// 测试用例
// ----------------------------------------------------------------

func TestLoad_BasicDefaults(t *testing.T) {
	content := `
database:
  host: "localhost"
`
	configDir := createConfigFile(t, "config.yaml", content)

	cfg, err := Load[TestConfig]("myapp", WithSearchPaths(configDir))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 验证默认值
	if cfg.AppName != "TestApp" {
		t.Errorf("Expected AppName 'TestApp', got '%s'", cfg.AppName)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Expected DB Port 3306, got %d", cfg.Database.Port)
	}
}

func TestLoad_ConfigFileOverride(t *testing.T) {
	// 场景：配置文件覆盖默认值
	content := `
app_name: "MyCoolApp"
database:
  host: "127.0.0.1"
  port: 5432
`
	configDir := createConfigFile(t, "config.yaml", content)

	cfg, err := Load[TestConfig]("myapp", WithSearchPaths(configDir))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.AppName != "MyCoolApp" {
		t.Errorf("Expected AppName 'MyCoolApp', got '%s'", cfg.AppName)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Expected DB Port 5432, got %d", cfg.Database.Port)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	// 场景：环境变量覆盖配置文件
	content := `
database:
  host: "127.0.0.1"
`
	configDir := createConfigFile(t, "config.yaml", content)

	// 设置环境变量
	os.Setenv("MYAPP_DATABASE_HOST", "192.168.1.1")
	defer os.Unsetenv("MYAPP_DATABASE_HOST")

	cfg, err := Load[TestConfig]("myapp", WithSearchPaths(configDir))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Database.Host != "192.168.1.1" {
		t.Errorf("Expected Host '192.168.1.1', got '%s'", cfg.Database.Host)
	}
}

func TestLoad_Validation(t *testing.T) {
	// 场景：验证规则失败 (Port < 1024)
	// 注意：Host 必填，但因为没有默认值，空字符串也会触发 required 错误
	content := `
database:
  port: 80
`
	configDir := createConfigFile(t, "config.yaml", content)

	// 测试英文错误
	_, err := Load[TestConfig]("myapp",
		WithSearchPaths(configDir),
		WithLocale("en"),
	)

	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	errMsg := err.Error()

	// Host 缺失 (required)
	if !strings.Contains(errMsg, "database.host") || !strings.Contains(errMsg, "required") {
		t.Errorf("Expected required error for host, got: %s", errMsg)
	}
	// Port 范围错误 (min)
	if !strings.Contains(errMsg, "database.port") || (!strings.Contains(errMsg, "greater") && !strings.Contains(errMsg, "min")) {
		t.Errorf("Expected min error for port, got: %s", errMsg)
	}
}

func TestLoad_Validation_Chinese(t *testing.T) {
	// 场景：中文错误提示
	configDir := createConfigFile(t, "config.yaml", "")

	_, err := Load[TestConfig]("myapp",
		WithSearchPaths(configDir),
		WithLocale("zh"),
	)

	if err == nil {
		t.Fatal("Expected validation error")
	}

	if !strings.Contains(err.Error(), "为必填字段") {
		t.Errorf("Expected Chinese error message, got: %s", err.Error())
	}
}

func TestLoad_StrictEnv(t *testing.T) {
	// 定义测试用的临时文件
	content := `
database:
  host: "localhost"
`
	configDir := createConfigFile(t, "config.yaml", content)

	t.Run("Production Missing Env", func(t *testing.T) {
		// 设置为生产环境
		os.Setenv("GO_ENV", "production")
		defer os.Unsetenv("GO_ENV")

		// 确保目标环境变量为空
		os.Unsetenv("MYAPP_DATABASE_PASSWORD")

		_, err := Load[TestConfig]("myapp", WithSearchPaths(configDir))
		if err == nil {
			t.Fatal("Expected strict env error in production, got nil")
		}

		// [说明] 此时错误信息应包含全大写的 MYAPP_DATABASE_PASSWORD
		expected := "must be set via environment variable 'MYAPP_DATABASE_PASSWORD'"
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("Expected error containing '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("Production With Env", func(t *testing.T) {
		os.Setenv("GO_ENV", "production")
		defer os.Unsetenv("GO_ENV")

		// 设置必要的环境变量 (确保这里是大写)
		os.Setenv("MYAPP_DATABASE_PASSWORD", "secret123")
		defer os.Unsetenv("MYAPP_DATABASE_PASSWORD")

		_, err := Load[TestConfig]("myapp", WithSearchPaths(configDir))
		if err != nil {
			t.Fatalf("Expected no error when env is set, got %v", err)
		}
	})

	t.Run("Dev Environment Skip Check", func(t *testing.T) {
		os.Setenv("GO_ENV", "dev")
		defer os.Unsetenv("GO_ENV")

		os.Unsetenv("MYAPP_DATABASE_PASSWORD")

		// 开发环境应该忽略 strict 检查
		_, err := Load[TestConfig]("myapp", WithSearchPaths(configDir))
		if err != nil {
			t.Fatalf("Expected no strict error in dev, got %v", err)
		}
	})
}

func TestLoad_JsonFile(t *testing.T) {
	// 场景：测试 Option 更改文件类型
	content := `{
		"app_name": "JsonApp",
		"database": {
			"host": "json-host"
		}
	}`
	configDir := createConfigFile(t, "settings.json", content)

	cfg, err := Load[TestConfig]("myapp",
		WithSearchPaths(configDir),
		WithFileName("settings"),
		WithFileType("json"),
	)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.AppName != "JsonApp" {
		t.Errorf("Expected AppName 'JsonApp', got %s", cfg.AppName)
	}
}

func TestMustLoad_Panic(t *testing.T) {
	// 场景：验证 MustLoad 会 Panic
	configDir := createConfigFile(t, "config.yaml", "database:\n  port: 10") // invalid port

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected MustLoad to panic")
		}
	}()

	MustLoad[TestConfig]("myapp", WithSearchPaths(configDir))
}

func TestStrictEnv_Complex(t *testing.T) {
	os.Setenv("GO_ENV", "production")
	defer os.Unsetenv("GO_ENV")

	// 场景 1: 缺少 Password (应该报错，且 key 不含 omitempty)
	t.Run("Missing Password", func(t *testing.T) {
		os.Unsetenv("MYAPP_PASSWORD")
		cfg := &StrictConfig{Sub: &StrictSub{ApiKey: "123"}}
		err := checkEnvStrict("myapp", cfg)
		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "MYAPP_PASSWORD") {
			t.Errorf("Error should contain MYAPP_PASSWORD, got: %v", err)
		}
		if strings.Contains(err.Error(), "OMITEMPTY") {
			t.Error("Error key should not contain tag options like OMITEMPTY")
		}
	})

	// 场景 2: 缺少嵌套指针里的 ApiKey
	t.Run("Missing Nested Ptr Env", func(t *testing.T) {
		os.Setenv("MYAPP_PASSWORD", "pass") // 满足第一层
		os.Unsetenv("MYAPP_SUB_API_KEY")    // 缺失第二层

		cfg := &StrictConfig{Sub: &StrictSub{}} // 指针不为 nil
		err := checkEnvStrict("myapp", cfg)
		if err == nil {
			t.Fatal("Expected error for nested pointer strict field")
		}
		if !strings.Contains(err.Error(), "MYAPP_SUB_API_KEY") {
			t.Errorf("Expected error for MYAPP_SUB_API_KEY, got: %v", err)
		}
	})
}

// ----------------------------------------------------------------
// 测试混合验证模式 (SelfValidatable)
// ----------------------------------------------------------------

type FastConfig struct {
	APIKey string `mapstructure:"api_key"`
}

// 实现 SelfValidatable 接口
func (f *FastConfig) Validate() error {
	if len(f.APIKey) != 32 {
		return fmt.Errorf("api_key must be 32 chars")
	}
	return nil
}

func TestLoad_HybridValidation(t *testing.T) {
	configDir := createConfigFile(t, "config.yaml", `api_key: "too_short"`)

	_, err := Load[FastConfig]("myapp", WithSearchPaths(configDir))
	if err == nil {
		t.Fatal("Expected validation error from interface")
	}

	if !strings.Contains(err.Error(), "api_key must be 32 chars") {
		t.Errorf("Expected custom interface error, got: %s", err.Error())
	}
}

// ----------------------------------------------------------------
// 测试多标签支持 (Mapstructure / Json / Yaml)
// ----------------------------------------------------------------

type MultiTagConfig struct {
	// 只有 json 标签
	Host string `json:"db_host" env:"strict"`
	// 只有 yaml 标签
	Port int `yaml:"db_port" validate:"min=1024"`
	// 混合标签 (测试 mapstructure 优先)
	User string `mapstructure:"db_user" json:"user" yaml:"u" env:"strict"`
}

func TestLoad_MultiTags_EnvStrict(t *testing.T) {
	// 模拟生产环境
	os.Setenv("GO_ENV", "production")
	defer os.Unsetenv("GO_ENV")

	configDir := createConfigFile(t, "config.yaml", "")

	t.Run("Fail When Env Missing", func(t *testing.T) {
		os.Unsetenv("MYAPP_DB_HOST") // json tag: db_host
		os.Unsetenv("MYAPP_DB_USER") // mapstructure tag: db_user

		_, err := Load[MultiTagConfig]("myapp", WithSearchPaths(configDir))
		if err == nil {
			t.Fatal("Expected strict env error")
		}
		// 应该识别 json 标签生成的 key
		if !strings.Contains(err.Error(), "MYAPP_DB_HOST") {
			t.Errorf("Error should refer to json tag key MYAPP_DB_HOST, got: %v", err)
		}
	})

	t.Run("Success When Env Present", func(t *testing.T) {
		os.Setenv("MYAPP_DB_HOST", "localhost") // Matches json:"db_host"
		os.Setenv("MYAPP_DB_USER", "root")      // Matches mapstructure:"db_user"
		defer os.Unsetenv("MYAPP_DB_HOST")
		defer os.Unsetenv("MYAPP_DB_USER")

		_, err := Load[MultiTagConfig]("myapp", WithSearchPaths(configDir))
		// 期待报错字段为 "db_port" (来自 yaml 标签)，而不是 "Port"
		if !strings.Contains(err.Error(), "db_port") {
			t.Errorf("Expected error message to use yaml tag 'db_port', got: %s", err.Error())
		}
	})
}

func TestValidator_MultiTag_FieldNames(t *testing.T) {
	_, err := Load[MultiTagConfig]("myapp",
		WithSearchPaths(createConfigFile(t, "conf.yaml", "")),
		WithLocale("en"), // 使用英文方便检查 Key
	)

	if err == nil {
		t.Fatal("Expected validation error")
	}

	// 期待报错字段为 "db_port" (来自 yaml 标签)，而不是 "Port"
	if !strings.Contains(err.Error(), "db_port") {
		t.Errorf("Expected error message to use yaml tag 'db_port', got: %s", err.Error())
	}
}
