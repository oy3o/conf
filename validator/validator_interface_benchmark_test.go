package validator

import (
	"errors"
	"testing"
)

// -------------------------------------------------------
// 1. 反射方式的结构体 (Reflect)
// -------------------------------------------------------
type UserReflect struct {
	Name string `validate:"required"`
	Age  int    `validate:"gte=18"`
}

// -------------------------------------------------------
// 2. 接口方式的结构体 (Interface / Direct Call)
// -------------------------------------------------------
type UserInterface struct {
	Name string
	Age  int
}

// 实现 SelfValidatable 接口
// 逻辑与上面的 Tag 保持一致：必填 + 大于等于18
func (u *UserInterface) Validate() error {
	if len(u.Name) == 0 {
		return errors.New("name is required")
	}
	if u.Age < 18 {
		return errors.New("age must be >= 18")
	}
	return nil
}

// -------------------------------------------------------
// 准备数据
// -------------------------------------------------------
var (
	reflectSuccess = UserReflect{Name: "Admin", Age: 20}
	reflectFail    = UserReflect{Name: "", Age: 10}

	interfaceSuccess = &UserInterface{Name: "Admin", Age: 20}
	interfaceFail    = &UserInterface{Name: "", Age: 10}
)

// -------------------------------------------------------
// Benchmarks
// -------------------------------------------------------

// 1. 反射验证 - 成功场景
func Benchmark_Reflect_Success(b *testing.B) {
	v, _ := New() // 关闭翻译以测试纯性能
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = v.Validate(reflectSuccess)
	}
}

// 2. 接口验证 - 成功场景 (期待极速)
func Benchmark_Interface_Success(b *testing.B) {
	v, _ := New()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 注意：Validate 内部会有一次类型断言 i.(SelfValidatable)
		_ = v.Validate(interfaceSuccess)
	}
}

// 3. 反射验证 - 失败场景
func Benchmark_Reflect_Failure(b *testing.B) {
	v, _ := New()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = v.Validate(reflectFail)
	}
}

// 4. 接口验证 - 失败场景 (期待极速)
func Benchmark_Interface_Failure(b *testing.B) {
	v, _ := New()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = v.Validate(interfaceFail)
	}
}

// 5. 对照组：完全不经过 Validator 封装，直接调用方法
// 用于测试 Validator 封装中的类型断言(Type Assertion)损耗
func Benchmark_Pure_Direct_Call(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = interfaceSuccess.Validate()
	}
}
