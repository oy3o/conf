package conf

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// checkEnvStrict 检查标记了 env:"strict" 的字段在生产环境是否真的来自环境变量
func checkEnvStrict(appName string, cfg interface{}) error {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = os.Getenv("APP_ENV")
	}
	env = strings.ToLower(env)

	if env != "production" && env != "prod" {
		return nil
	}

	val := reflect.ValueOf(cfg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	return recursiveEnvCheck(appName, val)
}

// resolveKeyName 根据优先级获取字段名称
// 优先级: mapstructure > yaml > json > toml > StructFieldName
func resolveKeyName(field reflect.StructField) string {
	// 1. mapstructure (Viper 默认)
	if tag := field.Tag.Get("mapstructure"); tag != "" {
		name := strings.SplitN(tag, ",", 2)[0]
		if name == "-" {
			return "" // 显式忽略
		}
		if name != "" {
			return name
		}
	}

	// 2. yaml
	if tag := field.Tag.Get("yaml"); tag != "" {
		name := strings.SplitN(tag, ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name != "" {
			return name
		}
	}

	// 3. json
	if tag := field.Tag.Get("json"); tag != "" {
		name := strings.SplitN(tag, ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name != "" {
			return name
		}
	}

	// 4. toml
	if tag := field.Tag.Get("toml"); tag != "" {
		name := strings.SplitN(tag, ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name != "" {
			return name
		}
	}

	// 5. Fallback to Field Name
	return field.Name
}

func recursiveEnvCheck(prefix string, val reflect.Value) error {
	// 处理指针：解引用，如果是 nil 则跳过
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// 0. 跳过未导出字段 (Private fields)
		if !field.IsExported() {
			continue
		}

		// 1. 获取统一的 Key Name
		mapKey := resolveKeyName(field)

		// 如果 resolveKeyName 返回空字符串，说明被显式忽略 ("-")
		if mapKey == "" {
			continue
		}

		// 2. 拼接 Key
		var currentKey string
		if prefix != "" {
			currentKey = strings.ToUpper(prefix + "_" + mapKey)
		} else {
			currentKey = strings.ToUpper(mapKey)
		}

		// 3. 递归处理嵌套结构体 (包含 Struct 和 *Struct)
		derefType := fieldVal.Type()
		if derefType.Kind() == reflect.Ptr {
			derefType = derefType.Elem()
		}

		if derefType.Kind() == reflect.Struct {
			// 递归传递
			if err := recursiveEnvCheck(currentKey, fieldVal); err != nil {
				return err
			}
			continue
		}

		// 4. 检查 env:"strict" 标签
		if tag := field.Tag.Get("env"); tag == "strict" {
			// 必须检查环境变量是否非空
			if os.Getenv(currentKey) == "" {
				return fmt.Errorf("security check failed: field '%s' (tag: '%s') must be set via environment variable '%s' in production", field.Name, mapKey, currentKey)
			}
		}
	}
	return nil
}
