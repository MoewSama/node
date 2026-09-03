package sub

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/slinxlink/node/internal/database"
	tpl "github.com/slinxlink/node/internal/sub/template"
	"github.com/slinxlink/node/internal/util"
)

// ── 订阅入口 ─────────────────────────────────────────────────────────

// cfdBinding 返回当前启用中的 cloudflared 入站的 Origin 域名与绑定的 vless 入站端口。
// 未配置 Origin 或绑定端口时 origin/bindPort 均为零值，订阅不生成 CF 节点。
func cfdBinding() (origin string, bindPort int) {
	var cfd database.Inbound
	if database.DB.Where("protocol = ? AND enable = ? AND cf_origin != '' AND cf_bind_port > 0", "cloudflared", true).First(&cfd).Error != nil {
		return "", 0
	}
	return cfd.CfOrigin, cfd.CfBindPort
}

func Sub(token string) string {
	user, inbounds := getUser(token)
	if user == nil {
		return ""
	}

	host := getHost()
	origin, bindPort := cfdBinding()
	var uris []string
	for _, inbound := range inbounds {
		if !inbound.HideInSub {
			uri := dispatch(*user, inbound, host)
			if uri != "" {
				uris = append(uris, uri)
			}
		}
		// CF 隧道节点不受 HideInSub 影响：绑定的隐藏入站只藏明文直连，CF 节点照常下发
		if origin != "" && bindPort == inbound.Port && inbound.Protocol == "vless" && inbound.Transport == "websocket" {
			if uri := vlessCF(*user, inbound, origin); uri != "" {
				uris = append(uris, uri)
			}
		}
	}
	return strings.Join(uris, "\n")
}

func Clash(token string) (string, string) {
	user, inbounds := getUser(token)
	if user == nil {
		return "", ""
	}

	host := getHost()
	origin, bindPort := cfdBinding()
	var proxies []string
	for _, inbound := range inbounds {
		if !inbound.HideInSub {
			proxy := dispatchClash(*user, inbound, host)
			if proxy != "" {
				proxies = append(proxies, proxy)
			}
		}
		if origin != "" && bindPort == inbound.Port && inbound.Protocol == "vless" && inbound.Transport == "websocket" {
			if proxy := vlessClashCF(*user, inbound, origin); proxy != "" {
				proxies = append(proxies, proxy)
			}
		}
	}

	name := util.SanitizeFileName(user.Name)
	if name == "" {
		name = "SLINX"
	}
	return tpl.RenderClash(proxies), name
}

func Surge(token string) (string, string) {
	user, inbounds := getUser(token)
	if user == nil {
		return "", ""
	}

	host := getHost()
	var proxies []string
	var names []string
	for _, inbound := range inbounds {
		if inbound.HideInSub {
			continue
		}
		proxy := dispatchSurge(*user, inbound, host)
		if proxy != "" {
			proxies = append(proxies, proxy)
			names = append(names, inbound.Name)
		}
	}

	name := util.SanitizeFileName(user.Name)
	if name == "" {
		name = "SLINX"
	}
	return tpl.RenderSurge(proxies, names), name
}

// ── 信息查询 ─────────────────────────────────────────────────────────

type Data struct {
	User     database.User      `json:"user"`
	Inbounds []database.Inbound `json:"inbounds"`
	Uris     []string           `json:"uris"`
	Urls     []string           `json:"urls"`
	Jsons    []string           `json:"jsons"`
}

func Info(token string) *Data {
	var user database.User
	if database.DB.Where("token = ? AND enable = ?", token, true).First(&user).Error != nil {
		return nil
	}

	var ids []int
	if err := json.Unmarshal([]byte(user.Inbounds), &ids); err != nil {
		return nil
	}

	var inbounds []database.Inbound
	database.DB.Where("id IN ? AND enable = ?", ids, true).Find(&inbounds)

	host := getHost()
	origin, bindPort := cfdBinding()
	var uris []string
	var jsons []string
	for _, inbound := range inbounds {
		if !inbound.HideInSub {
			uri := dispatch(user, inbound, host)
			if uri != "" {
				uris = append(uris, uri)
			}
			jsons = append(jsons, Json(user, inbound, "singbox"))
		}
		if origin != "" && bindPort == inbound.Port && inbound.Protocol == "vless" && inbound.Transport == "websocket" {
			if uri := vlessCF(user, inbound, origin); uri != "" {
				uris = append(uris, uri)
			}
			jsons = append(jsons, vlessSingBoxCF(user, inbound, origin))
		}
	}

	return &Data{
		User:     user,
		Inbounds: inbounds,
		Uris:     uris,
		Urls:     Url(token),
		Jsons:    jsons,
	}
}

func Url(token string) []string {
	var config database.Config
	database.DB.First(&config)

	host := config.Domain
	if host == "" {
		host = config.IPv4
	}

	scheme := "http"
	if config.Domain != "" {
		scheme = "https"
	}

	base := fmt.Sprintf("%s://%s:%d", scheme, host, config.SubPort)
	sub := fmt.Sprintf("%s%s/%s", base, config.SubPath, token)

	return []string{
		sub,
		sub + "/clash",
		sub + "/surge",
	}
}

func Uri(user database.User, inbound database.Inbound) string {
	host := getHost()
	return dispatch(user, inbound, host)
}

func Json(user database.User, inbound database.Inbound, format string) string {
	host := getHost()
	switch format {
	case "singbox":
		outbound := dispatchSingBox(user, inbound, host)
		if outbound == "" {
			return ""
		}
		return tpl.RenderSingBox([]string{outbound})
	default:
		return ""
	}
}

// ── CF 隧道单节点 ────────────────────────────────────────────────────
// 若该入站被 cloudflared 绑定，返回其 CF 隧道节点，否则返回空字符串。
// 不受 HideInSub 影响：绑定的隐藏入站只藏明文直连，CF 节点照常可取。
func CfUri(user database.User, inbound database.Inbound) string {
	origin, bindPort := cfdBinding()
	if origin == "" || bindPort != inbound.Port || inbound.Protocol != "vless" || inbound.Transport != "websocket" {
		return ""
	}
	return vlessCF(user, inbound, origin)
}

func CfJson(user database.User, inbound database.Inbound) string {
	origin, bindPort := cfdBinding()
	if origin == "" || bindPort != inbound.Port || inbound.Protocol != "vless" || inbound.Transport != "websocket" {
		return ""
	}
	return vlessSingBoxCF(user, inbound, origin)
}

// ── 协议分发 ─────────────────────────────────────────────────────────

func dispatch(user database.User, inbound database.Inbound, host string) string {
	switch inbound.Protocol {
	case "vless":
		return vless(user.UUID, host, inbound)
	case "vmess":
		return vmess(user.UUID, host, inbound)
	case "hysteria":
		return hysteria(user.Password, host, inbound)
	case "trojan":
		return trojan(user.Password, host, inbound)
	case "tuic":
		return tuic(user.UUID, user.Password, host, inbound)
	case "anytls":
		return anytls(user.Password, host, inbound)
	default:
		return ""
	}
}

func dispatchClash(user database.User, inbound database.Inbound, host string) string {
	switch inbound.Protocol {
	case "vless":
		return vlessClash(user.UUID, host, inbound)
	case "vmess":
		return vmessClash(user.UUID, host, inbound)
	case "hysteria":
		return hysteriaClash(user.Password, host, inbound)
	case "trojan":
		return trojanClash(user.Password, host, inbound)
	case "tuic":
		return tuicClash(user.UUID, user.Password, host, inbound)
	case "anytls":
		return anytlsClash(user.Password, host, inbound)
	default:
		return ""
	}
}

func dispatchSurge(user database.User, inbound database.Inbound, host string) string {
	switch inbound.Protocol {
	case "vmess":
		return vmessSurge(user.UUID, host, inbound)
	case "hysteria":
		return hysteriaSurge(user.Password, host, inbound)
	case "trojan":
		return trojanSurge(user.Password, host, inbound)
	case "tuic":
		return tuicSurge(user.UUID, user.Password, host, inbound)
	case "anytls":
		return anytlsSurge(user.Password, host, inbound)
	default:
		return ""
	}
}

func dispatchSingBox(user database.User, inbound database.Inbound, host string) string {
	switch inbound.Protocol {
	case "vless":
		return vlessSingBox(user.UUID, host, inbound)
	case "vmess":
		return vmessSingBox(user.UUID, host, inbound)
	case "hysteria":
		return hysteriaSingBox(user.Password, host, inbound)
	case "trojan":
		return trojanSingBox(user.Password, host, inbound)
	case "tuic":
		return tuicSingBox(user.UUID, user.Password, host, inbound)
	case "anytls":
		return anytlsSingBox(user.Password, host, inbound)
	default:
		return ""
	}
}

// ── 内部工具 ─────────────────────────────────────────────────────────

// formatHost 处理服务器地址：IPv6 地址加方括号，其他原样返回。
// 用于 URI 分享链接中 host:port 的拼接。
func formatHost(host string) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return "[" + host + "]"
	}
	return host
}

func getHost() string {
	var config database.Config
	database.DB.First(&config)
	if config.Domain != "" {
		return config.Domain
	}
	return config.IPv4
}

func getUser(token string) (*database.User, []database.Inbound) {
	var user database.User
	if database.DB.Where("token = ? AND enable = ?", token, true).First(&user).Error != nil {
		return nil, nil
	}
	var ids []int
	if err := json.Unmarshal([]byte(user.Inbounds), &ids); err != nil {
		return nil, nil
	}
	var inbounds []database.Inbound
	database.DB.Where("id IN ? AND enable = ?", ids, true).Find(&inbounds)
	return &user, inbounds
}

func extractECHConfig(pem string) string {
	lines := strings.Split(pem, "\n")
	var content []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-----") {
			continue
		}
		content = append(content, line)
	}
	return strings.Join(content, "")
}
