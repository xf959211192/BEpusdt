package router

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLocalHealthRejectsNonLoopback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	registerLocalHealth(engine)
	request := httptest.NewRequest("GET", "/internal/health", nil)
	request.RemoteAddr = "192.0.2.10:32000"
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != 403 {
		t.Fatal("监控接口未拒绝外部来源")
	}
}
