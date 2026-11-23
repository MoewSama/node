package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/slinxlink/node/internal/database"
)

// TestSaveInboundMasquerade 覆盖 Hysteria2 伪装保存时的参数校验：
// 伪装对象必须带 type，参数不完整时应在保存阶段就返回 400，
// 而不是把坏配置写进库后在核心校验阶段静默失败。
func TestSaveInboundMasquerade(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := database.Init(); err != nil {
		t.Fatalf("init database: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/inbound", SaveInbound)

	post := func(t *testing.T, body map[string]any) (int, string) {
		t.Helper()
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/inbound", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}

	base := func() map[string]any {
		return map[string]any{
			"Protocol":          "hysteria",
			"Port":              8443,
			"Enable":            true,
			"TLSType":           "tls",
			"Certs":             "[1]",
			"MasqueradeEnabled": true,
			"MasqueradeType":    "default",
		}
	}

	t.Run("静态文件目录为空", func(t *testing.T) {
		body := base()
		body["MasqueradeType"] = "file"
		body["Port"] = 9001
		if code, resp := post(t, body); code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d: %s", code, resp)
		}
	})

	t.Run("静态文件目录为相对路径", func(t *testing.T) {
		body := base()
		body["MasqueradeType"] = "file"
		body["MasqueradePath"] = "www/html"
		body["Port"] = 9002
		if code, resp := post(t, body); code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d: %s", code, resp)
		}
	})

	t.Run("反向代理地址缺少协议", func(t *testing.T) {
		body := base()
		body["MasqueradeType"] = "proxy"
		body["MasqueradeURL"] = "www.example.com"
		body["Port"] = 9003
		if code, resp := post(t, body); code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d: %s", code, resp)
		}
	})

	t.Run("自定义内容状态码越界", func(t *testing.T) {
		body := base()
		body["MasqueradeType"] = "string"
		body["MasqueradeCode"] = 42
		body["MasqueradeBody"] = "<html></html>"
		body["Port"] = 9004
		if code, resp := post(t, body); code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d: %s", code, resp)
		}
	})

	t.Run("安全类型非 TLS 时拒绝伪装", func(t *testing.T) {
		body := base()
		body["TLSType"] = "reality"
		body["MasqueradeType"] = "proxy"
		body["MasqueradeURL"] = "https://www.example.com"
		body["Port"] = 9005
		if code, resp := post(t, body); code != http.StatusBadRequest {
			t.Fatalf("期望 400，实际 %d: %s", code, resp)
		}
	})

	t.Run("合法配置可以保存", func(t *testing.T) {
		for i, body := range []map[string]any{
			func() map[string]any {
				b := base()
				b["MasqueradeType"] = "proxy"
				b["MasqueradeURL"] = "https://www.example.com"
				b["RewriteHost"] = true
				b["Port"] = 9101
				return b
			}(),
			func() map[string]any {
				b := base()
				b["MasqueradeType"] = "file"
				b["MasqueradePath"] = "/var/www/html"
				b["Port"] = 9102
				return b
			}(),
			func() map[string]any {
				b := base()
				b["MasqueradeType"] = "string"
				b["MasqueradeCode"] = 200
				b["MasqueradeBody"] = "<html>hi</html>"
				b["Port"] = 9103
				return b
			}(),
		} {
			if code, resp := post(t, body); code != http.StatusOK {
				t.Fatalf("case %d: 期望 200，实际 %d: %s", i, code, resp)
			}
		}

		var saved []database.Inbound
		database.DB.Find(&saved)
		if len(saved) != 3 {
			t.Fatalf("期望保存 3 条入站，实际 %d", len(saved))
		}
		for _, ib := range saved {
			if !ib.MasqueradeEnabled || ib.MasqueradeType == "" || ib.MasqueradeType == "default" {
				t.Fatalf("伪装字段未正确落库: %+v", ib)
			}
		}
	})
}
