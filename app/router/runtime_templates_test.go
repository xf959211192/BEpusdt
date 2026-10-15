package router

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/v03413/bepusdt/app/log"
	"github.com/v03413/bepusdt/app/model"
)

// 构建必须包含生成的管理页面，防止编译通过却在启动时因模板缺失退出。
func TestProductionTemplatesAndLocalHealth(t *testing.T) {
	root := t.TempDir()
	if err := model.Init(filepath.Join(root, "runtime.db"), ""); err != nil {
		t.Fatal(err)
	}
	defer model.Close()
	if err := log.Init(filepath.Join(root, "logs")); err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	engine := Handler()
	request := httptest.NewRequest("GET", "/internal/health", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatalf("健康检查失败：%d", response.Code)
	}
	for _, field := range []string{"api_token", "api_auth_token", "rpc_endpoint", "password"} {
		if strings.Contains(response.Body.String(), field) {
			t.Fatal("监控响应包含敏感配置字段")
		}
	}
}
