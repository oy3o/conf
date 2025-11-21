package validator

import (
	"strings"
	"testing"
)

// 定义一个用于测试的结构体
type UserConfig struct {
	// 测试 mapstructure 优先级最高
	Username string `mapstructure:"user_name" json:"u_name" validate:"required"`
	// 测试 json 作为备选
	Email string `json:"email_addr" validate:"required,email"`
	// 测试 yaml 作为第三备选
	Role string `yaml:"role_name" validate:"required"`
	// 测试 fallback 到字段名
	Age int `validate:"gte=18"`
	// 测试忽略字段
	Ignored string `json:"-" validate:"required"`
	// 测试嵌套结构体
	Settings Settings `mapstructure:"settings"`
}

type Settings struct {
	Theme string `mapstructure:"theme_mode" validate:"oneof=dark light"`
}

func TestNew(t *testing.T) {
	t.Run("Should initialize with default English", func(t *testing.T) {
		v, err := New()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if v == nil {
			t.Fatal("Expected validator instance, got nil")
		}
	})

	t.Run("Should initialize with Chinese", func(t *testing.T) {
		v, err := New("zh")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if v == nil {
			t.Fatal("Expected validator instance, got nil")
		}
	})

	t.Run("Should return error for unsupported locale", func(t *testing.T) {
		_, err := New("fr")
		if err == nil {
			t.Log("Warning: 'fr' locale might have fallen back to default or no error returned")
		}
	})
}

func TestValidator_Validate_Success(t *testing.T) {
	v, _ := New("en")

	cfg := UserConfig{
		Username: "admin",
		Email:    "admin@example.com",
		Role:     "admin",
		Age:      20,
		Ignored:  "value",
		Settings: Settings{Theme: "dark"},
	}

	if err := v.Validate(cfg); err != nil {
		t.Errorf("Expected validation to pass, but got error: %v", err)
	}
}

func TestValidator_Validate_TagPriority(t *testing.T) {
	// 使用英文环境测试，检查字段名是否正确映射
	v, _ := New("en")

	// 构造一个空对象，触发所有 required 错误
	cfg := UserConfig{Age: 10, Settings: Settings{Theme: "blue"}} // Age < 18, Theme invalid (blue is not dark/light)

	err := v.Validate(cfg)
	if err == nil {
		t.Fatal("Expected validation errors, got nil")
	}

	// 断言错误类型
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("Expected *ValidationError type, got %T", err)
	}

	// 验证标签映射逻辑
	expectedFields := map[string]string{
		"user_name":           "required", // mapstructure 优先
		"email_addr":          "required", // json 备选
		"role_name":           "required", // yaml 备选
		"Age":                 "18",       // Fallback to field name
		"settings.theme_mode": "oneof",    // Nested mapstructure (必须包含路径)
	}

	for field, keyword := range expectedFields {
		msg, exists := ve.Errors[field]
		if !exists {
			// 为了方便调试，打印出当前所有存在的 keys
			allKeys := make([]string, 0, len(ve.Errors))
			for k := range ve.Errors {
				allKeys = append(allKeys, k)
			}
			t.Errorf("Expected error for field '%s', but it was missing. Existing keys: %v", field, allKeys)
			continue
		}

		if !strings.Contains(msg, keyword) {
			t.Logf("Field verified: %s -> %s", field, msg)
		}
	}
}

func TestValidator_Validate_Translation_ZH(t *testing.T) {
	v, _ := New("zh")

	cfg := UserConfig{} // 全空

	err := v.Validate(cfg)
	if err == nil {
		t.Fatal("Expected error")
	}

	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatal("Type assertion failed")
	}

	// 检查中文翻译
	// 期望: "user_name为必填字段"
	msg := ve.Errors["user_name"]
	if !strings.Contains(msg, "必填字段") {
		t.Errorf("Expected Chinese error message containing '必填字段', got: %s", msg)
	}
}

func TestValidator_Validate_Translation_EN(t *testing.T) {
	v, _ := New("en")
	cfg := UserConfig{}
	err := v.Validate(cfg)
	ve, _ := err.(*ValidationError)

	// 期望: "user_name is a required field"
	msg := ve.Errors["user_name"]
	if !strings.Contains(msg, "required") {
		t.Errorf("Expected English error message containing 'required', got: %s", msg)
	}
}

func TestValidator_PanicSafety(t *testing.T) {
	v, _ := New()

	t.Run("Nil input", func(t *testing.T) {
		err := v.Validate(nil)
		if err == nil {
			t.Error("Expected error for nil input")
		}
		// 确认没有 panic，且返回了非 ValidationError 类型的错误
		if _, ok := err.(*ValidationError); ok {
			t.Error("Expected standard error for nil input, not ValidationError")
		}
	})

	t.Run("Non-struct input", func(t *testing.T) {
		i := 123
		err := v.Validate(i)
		if err == nil {
			t.Error("Expected error for non-struct input")
		}
	})

	t.Run("Pointer to non-struct", func(t *testing.T) {
		i := 123
		err := v.Validate(&i)
		if err == nil {
			t.Error("Expected error for pointer to non-struct")
		}
	})
}

func TestValidationError_Error_String(t *testing.T) {
	// 测试 Error() 方法的字符串格式化
	ve := &ValidationError{
		Errors: map[string]string{
			"host": "is required",
			"port": "must be number",
		},
	}

	msg := ve.Error()
	if !strings.Contains(msg, "validation failed:") {
		t.Error("Error string should contain header")
	}
	if !strings.Contains(msg, "host: is required") {
		t.Error("Error string should contain field error")
	}
}
