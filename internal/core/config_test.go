package core

import (
	"encoding/json"
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/slinxlink/node/internal/database"
)

// TestHysteria2MasqueradeType 回归测试：
// sing-box 的 masquerade 对象形式必须带 type（file / proxy / string），
// 缺 type 会被核心判为 unknown masquerade type 直接拒绝启动。
// 这里同时校验面板输出与 sing-box 解析器的实际接受结果。
func TestHysteria2MasqueradeType(t *testing.T) {
	cases := []struct {
		name string
		ib   database.Inbound
		want string // 期望 sing-box 解析出的 masquerade.type，空表示不配置
	}{
		{
			name: "proxy",
			ib: database.Inbound{
				Protocol: "hysteria", Port: 443, MasqueradeEnabled: true,
				MasqueradeType: "proxy", MasqueradeURL: "https://www.example.com", RewriteHost: true,
			},
			want: "proxy",
		},
		{
			name: "file",
			ib: database.Inbound{
				Protocol: "hysteria", Port: 443, MasqueradeEnabled: true,
				MasqueradeType: "file", MasqueradePath: "/var/www/html",
			},
			want: "file",
		},
		{
			name: "string",
			ib: database.Inbound{
				Protocol: "hysteria", Port: 443, MasqueradeEnabled: true,
				MasqueradeType: "string", MasqueradeCode: 200, MasqueradeBody: "<html>hi</html>",
			},
			want: "string",
		},
		{
			name: "default 不写 masquerade",
			ib: database.Inbound{
				Protocol: "hysteria", Port: 443, MasqueradeEnabled: true,
				MasqueradeType: "default",
			},
			want: "",
		},
		{
			name: "空类型按 default 处理",
			ib: database.Inbound{
				Protocol: "hysteria", Port: 443, MasqueradeEnabled: true,
			},
			want: "",
		},
		{
			name: "未开启伪装",
			ib: database.Inbound{
				Protocol: "hysteria", Port: 443, MasqueradeType: "proxy", MasqueradeURL: "https://www.example.com",
			},
			want: "",
		},
		{
			name: "proxy 地址非法时回退 404，不产出坏配置",
			ib: database.Inbound{
				Protocol: "hysteria", Port: 443, MasqueradeEnabled: true,
				MasqueradeType: "proxy", MasqueradeURL: "example.com",
			},
			want: "",
		},
		{
			name: "file 目录为空时回退 404",
			ib: database.Inbound{
				Protocol: "hysteria", Port: 443, MasqueradeEnabled: true,
				MasqueradeType: "file",
			},
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ic, err := buildInbound(tc.ib, nil)
			if err != nil {
				t.Fatalf("buildInbound: %v", err)
			}
			raw, err := json.Marshal(ic)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var opts option.Hysteria2InboundOptions
			if err := json.Unmarshal(raw, &opts); err != nil {
				t.Fatalf("sing-box 解析失败: %v, config=%s", err, raw)
			}
			got := ""
			if opts.Masquerade != nil {
				got = opts.Masquerade.Type
			}
			if got != tc.want {
				t.Fatalf("masquerade.type = %q, want %q, config=%s", got, tc.want, raw)
			}
		})
	}
}
