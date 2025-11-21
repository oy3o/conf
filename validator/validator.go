package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

// SelfValidatable 定义了自验证接口
type SelfValidatable interface {
	Validate() error
}

// Validator 封装结构体
type Validator struct {
	validate *validator.Validate
	trans    ut.Translator
}

// New 初始化验证器
func New(locale ...string) (*Validator, error) {
	v := validator.New()

	// 1. 注册自定义 Tag Name 获取函数
	// 统一逻辑：mapstructure > yaml > json > toml > FieldName
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		// Priority 1: mapstructure
		if tag := fld.Tag.Get("mapstructure"); tag != "" {
			name := strings.SplitN(tag, ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}

		// Priority 2: yaml
		if tag := fld.Tag.Get("yaml"); tag != "" {
			name := strings.SplitN(tag, ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}

		// Priority 3: json
		if tag := fld.Tag.Get("json"); tag != "" {
			name := strings.SplitN(tag, ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}

		// Priority 4: toml
		if tag := fld.Tag.Get("toml"); tag != "" {
			name := strings.SplitN(tag, ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}

		return fld.Name
	})

	// 2. 语言包处理 (保持不变)
	if len(locale) == 0 || locale[0] == "" {
		return &Validator{validate: v, trans: nil}, nil
	}

	lang := locale[0]
	zhT := zh.New()
	enT := en.New()
	uni := ut.New(enT, zhT, enT)

	trans, ok := uni.GetTranslator(lang)
	if !ok {
		// 找不到语言时，默认回退到英文，避免报错
		trans, _ = uni.GetTranslator("en")
	}

	var err error
	switch lang {
	case "zh":
		err = zh_translations.RegisterDefaultTranslations(v, trans)
	default:
		err = en_translations.RegisterDefaultTranslations(v, trans)
	}
	if err != nil {
		return nil, err
	}

	return &Validator{validate: v, trans: trans}, nil
}

type ValidationError struct {
	Errors map[string]string
}

func (e *ValidationError) Error() string {
	var msgs []string
	for field, msg := range e.Errors {
		msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
	}
	return fmt.Sprintf("validation failed:\n - %s", strings.Join(msgs, "\n - "))
}

// Validate 执行验证 (保持不变)
func (v *Validator) Validate(i interface{}) error {
	if sv, ok := i.(SelfValidatable); ok {
		return sv.Validate()
	}

	err := v.validate.Struct(i)
	if err == nil {
		return nil
	}

	if _, ok := err.(*validator.InvalidValidationError); ok {
		return fmt.Errorf("invalid validation error: %w", err)
	}

	validationErrors := err.(validator.ValidationErrors)
	translatedErrors := make(map[string]string)

	for _, e := range validationErrors {
		// 处理 Namespace (去除结构体前缀)
		namespace := e.Namespace()
		if i := strings.Index(namespace, "."); i != -1 {
			namespace = namespace[i+1:]
		}

		if v.trans != nil {
			translatedErrors[namespace] = e.Translate(v.trans)
		} else {
			if e.Param() != "" {
				translatedErrors[namespace] = fmt.Sprintf("%s=%s", e.Tag(), e.Param())
			} else {
				translatedErrors[namespace] = e.Tag()
			}
		}
	}

	return &ValidationError{Errors: translatedErrors}
}
