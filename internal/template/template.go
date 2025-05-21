package template

import (
	"html/template"
	"log"
	"sync"
)

// 模板缓存
var (
	templates     = make(map[string]*template.Template)
	templateMutex sync.RWMutex
)

// InitTemplates 初始化并预加载所有模板
func InitTemplates() {
	// 定义模板函数
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
	}

	// 列出所有需要加载的模板
	templateFiles := map[string]string{
		"home":   "templates/home.tmpl",
		"upload": "templates/upload.tmpl",
		"login":  "templates/login.tmpl",
		"admin":  "templates/admin.tmpl",
	}

	// 加载每个模板
	for name, file := range templateFiles {
		// 创建带有函数的模板
		tmpl := template.New(file).Funcs(funcMap)

		// 解析模板文件
		tmpl, err := tmpl.ParseFiles(file)
		if err != nil {
			log.Fatalf("Failed to parse template %s: %v", file, err)
		}

		templateMutex.Lock()
		templates[name] = tmpl
		templateMutex.Unlock()

		log.Printf("Template loaded: %s", name)
	}
}

// GetTemplate 根据名称获取模板
func GetTemplate(name string) (*template.Template, bool) {
	templateMutex.RLock()
	defer templateMutex.RUnlock()

	tmpl, ok := templates[name]
	return tmpl, ok
}
