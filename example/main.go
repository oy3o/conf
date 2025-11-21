package main

import (
	"fmt"

	"github.com/oy3o/conf"
)

type DBConfig struct {
	// 支持 Viper 标签，同时使用 validate 校验
	Host string `mapstructure:"host" validate:"required"`
	Port int    `mapstructure:"port" validate:"min=1024"`
	// 生产环境必须来自 Env
	Password string `mapstructure:"password" validate:"required" env:"strict"`
}

type AppConfig struct {
	Name string   `mapstructure:"name" default:"MyService"`
	DB   DBConfig `mapstructure:"db"`
}

// 假设这是高频调用的请求对象，我们手动实现验证以提升性能
type LoginRequest struct {
	User string
	Pass string
}

// 实现 SelfValidatable 接口
func (r *LoginRequest) Validate() error {
	if r.User == "" {
		return fmt.Errorf("user required")
	}
	return nil
}

func main() {
	// 模拟生产环境测试 strict 检查
	// os.Setenv("GO_ENV", "production")
	// os.Setenv("MYAPP_DB_PASSWORD", "123456") // 如果注释掉这行，MustLoad 会 panic

	// 1. 加载配置 (使用中文报错)
	cfg := conf.MustLoad[AppConfig]("myapp",
		conf.WithLocale("zh"),
		conf.WithFileType("yaml"),
	)

	fmt.Printf("App Loaded: %s, DB Port: %d\n", cfg.Name, cfg.DB.Port)

	// -------------------------------------------------------
	// 2. 复用验证器进行业务验证 (可选)
	// 如果你想在业务代码里也用这个高性能验证器，可以单独初始化
	// -------------------------------------------------------
	/*
	   val, _ := validator.New("zh")
	   req := LoginRequest{User: ""}
	   if err := val.Validate(&req); err != nil {
	       fmt.Println("API Validation Error:", err)
	   }
	*/
}
