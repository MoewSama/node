package template

import (
	"os"
	"strings"

	"github.com/slinxlink/node/internal/util"
)

const rulesetURL = "https://raw.githubusercontent.com/slinxlink/ruleset/main"

const singBoxTemplate = `{
    "dns": {
        "servers": [
            {
                "type": "local",
                "tag": "dns"
            }
        ],
        "final": "dns"
    },
    "inbounds": [
        {
            "type": "tun",
            "tag": "tun-in",
            "address": ["172.19.0.1/30", "fdfe:dcba:9876::1/126"],
            "auto_route": true,
            "strict_route": true
        }
    ],
    "outbounds": [
        // __OUTBOUNDS__
        {
            "type": "direct",
            "tag": "direct"
        },
        {
            "type": "block",
            "tag": "block"
        }
    ],
    "route": {
        "final": "proxy",
        "rules": [
            {
                "action": "sniff"
            },
            {
                "protocol": "dns",
                "action": "hijack-dns"
            },
            {
                "ip_is_private": true,
                "outbound": "direct"
            },
            {
                "rule_set": [
                    "ai",
                    "media",
                    "social",
                    "game",
                    "google"
                ],
                "outbound": "proxy"
            },
            {
                "rule_set": [
                    "apple",
                    "microsoft",
                    "cn",
                    "geoip-cn"
                ],
                "outbound": "direct"
            }
        ],
        "default_domain_resolver": "dns",
        "rule_set": [
            {
                "tag": "ai",
                "type": "remote",
                "format": "binary",
                "url": "__RULESET_URL__/ruleset/ai.srs"
            },
            {
                "tag": "media",
                "type": "remote",
                "format": "binary",
                "url": "__RULESET_URL__/ruleset/media.srs"
            },
            {
                "tag": "social",
                "type": "remote",
                "format": "binary",
                "url": "__RULESET_URL__/ruleset/social.srs"
            },
            {
                "tag": "game",
                "type": "remote",
                "format": "binary",
                "url": "__RULESET_URL__/ruleset/game.srs"
            },
            {
                "tag": "apple",
                "type": "remote",
                "format": "binary",
                "url": "__RULESET_URL__/ruleset/apple.srs"
            },
            {
                "tag": "google",
                "type": "remote",
                "format": "binary",
                "url": "__RULESET_URL__/ruleset/google.srs"
            },
            {
                "tag": "microsoft",
                "type": "remote",
                "format": "binary",
                "url": "__RULESET_URL__/ruleset/microsoft.srs"
            },
            {
                "tag": "cn",
                "type": "remote",
                "format": "binary",
                "url": "__RULESET_URL__/geo/site/cn.srs"
            },
            {
                "tag": "geoip-cn",
                "type": "remote",
                "format": "binary",
                "url": "__RULESET_URL__/geo/ip/cn.srs"
            }
        ]
    }
}`

func GenerateSingBox() {
	if err := os.MkdirAll("data/ruleset", 0755); err != nil {
		util.Error("[singbox] 创建目录失败: %v", err)
		return
	}
	if err := os.WriteFile(SingBoxTemplatePath, []byte(singBoxTemplate), 0644); err != nil {
		util.Error("[singbox] 模板写入失败: %v", err)
		return
	}
	util.Info("[singbox] 模板已生成")
}

func RenderSingBox(outbounds []string) string {
	data, err := os.ReadFile(SingBoxTemplatePath)
	if err != nil {
		util.Error("[singbox] 读取模板失败: %v", err)
		return ""
	}

	outboundBlock := strings.Join(outbounds, ",\n    ")
	conf := strings.Replace(string(data), "// __OUTBOUNDS__", outboundBlock+",", 1)
	conf = strings.ReplaceAll(conf, "__RULESET_URL__", rulesetURL)
	return conf
}
