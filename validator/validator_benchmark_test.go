package validator

import (
	"testing"
)

// 用于基准测试的结构体
type BenchStruct struct {
	Name  string `mapstructure:"name" validate:"required"`
	Age   int    `mapstructure:"age" validate:"gte=18"`
	Email string `mapstructure:"email" validate:"required,email"`
	Role  string `mapstructure:"role" validate:"oneof=admin user"`
}

// 预定义数据
var (
	// 成功数据：完全符合规则
	successData = BenchStruct{
		Name:  "John Doe",
		Age:   25,
		Email: "john@example.com",
		Role:  "admin",
	}

	// 失败数据：4个字段全部验证失败，触发最大开销
	failureData = BenchStruct{
		Name:  "",             // required fail
		Age:   10,             // gte fail
		Email: "invalid-mail", // email fail
		Role:  "guest",        // oneof fail
	}
)

// 1. 基准测试：验证通过 (无错误) - 关闭翻译
func Benchmark_Success_NoTranslation(b *testing.B) {
	v, _ := New() // 不传参，关闭翻译
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = v.Validate(successData)
	}
}

// 2. 基准测试：验证通过 (无错误) - 开启翻译 (EN)
func Benchmark_Success_WithTranslation(b *testing.B) {
	v, _ := New("en") // 开启翻译
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = v.Validate(successData)
	}
}

// 3. 基准测试：验证失败 (产生错误) - 关闭翻译
// 预期：性能较好，因为只做简单的字符串拼接 (tag=param)
func Benchmark_Failure_NoTranslation(b *testing.B) {
	v, _ := New()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = v.Validate(failureData)
	}
}

// 4. 基准测试：验证失败 (产生错误) - 开启翻译 (EN)
// 预期：性能稍差，因为涉及反射查找模板和字符串处理
func Benchmark_Failure_WithTranslation(b *testing.B) {
	v, _ := New("en")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = v.Validate(failureData)
	}
}

// 5. 基准测试：验证失败 (产生错误) - 开启翻译 (ZH)
// 预期：与 EN 类似，用于验证不同语言包是否有差异
func Benchmark_Failure_WithTranslation_ZH(b *testing.B) {
	v, _ := New("zh")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = v.Validate(failureData)
	}
}
