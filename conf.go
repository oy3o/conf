package conf

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/mcuadros/go-defaults"
	"github.com/oy3o/conf/validator"
	"github.com/spf13/viper"
)

// MustLoad 加载配置，失败则 panic
func MustLoad[T any](appName string, opts ...Option) *T {
	cfg, err := Load[T](appName, opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// Load 加载并验证配置
func Load[T any](appName string, opts ...Option) (*T, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	var cfg T

	// 1. 设置结构体默认值 (Tag: default)
	defaults.SetDefaults(&cfg)

	// 2. 初始化 Viper
	v := viper.New()
	v.SetConfigName(o.fileName)
	v.SetConfigType(o.fileType)
	for _, path := range o.searchPaths {
		v.AddConfigPath(path)
	}

	// 3. 绑定环境变量
	// 规则: appName="myapp", field="db.host" -> "MYAPP_DB_HOST"
	v.SetEnvPrefix(appName)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 4. 读取文件 (忽略文件未找到错误，支持纯 Env 运行)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}

	// 5. 解析到结构体 (严格模式：防止拼写错误)
	if err := v.Unmarshal(&cfg, func(c *mapstructure.DecoderConfig) {
		c.TagName = "mapstructure"
		c.ErrorUnused = true // 关键：配置文件有多余字段直接报错
	}); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// 6. 生产环境来源检查 (Env Strict)
	if err := checkEnvStrict(appName, &cfg); err != nil {
		return nil, err
	}

	// 7. 数据内容验证 (集成新 Validator)
	val, err := validator.New(o.locale) // 初始化验证器
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}

	// 执行验证 (混合模式：自动识别 Interface 或 Tag)
	if err := val.Validate(&cfg); err != nil {
		return nil, err // 直接返回 validator 的友好错误信息
	}

	return &cfg, nil
}
