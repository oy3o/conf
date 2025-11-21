package conf

type options struct {
	searchPaths []string
	fileType    string
	fileName    string
	locale      string // zh, en, or ""
}

type Option func(*options)

func defaultOptions() *options {
	return &options{
		searchPaths: []string{".", "./config"},
		fileType:    "yaml",
		fileName:    "config",
		locale:      "zh", // 默认开启中文，对国内开发友好
	}
}

// WithSearchPaths 指定配置文件的搜索路径
func WithSearchPaths(paths ...string) Option {
	return func(o *options) {
		o.searchPaths = paths
	}
}

// WithFileType 指定文件类型 (yaml, json, toml)
func WithFileType(t string) Option {
	return func(o *options) {
		o.fileType = t
	}
}

// WithFileName 指定文件名 (默认 config)
func WithFileName(n string) Option {
	return func(o *options) {
		o.fileName = n
	}
}

// WithLocale 指定验证错误语言 ("zh", "en", ""=关闭)
func WithLocale(locale string) Option {
	return func(o *options) {
		o.locale = locale
	}
}
