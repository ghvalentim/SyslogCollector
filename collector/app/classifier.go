package app

import (
	"strings"
)


func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs { if strings.Contains(s, sub) { return true } }; return false
}

func ClassifySource(appName, payload string) string {
	text := strings.ToLower(appName + " " + payload)
	
	if containsAny(text, []string{"pfsense", "opnsense", "fortigate", "sophos", "watchguard", "iptables", "nftables"}) { return "Firewall" }
	if containsAny(text, []string{"microsoft", "winrm", "windows", "eventlog"}) { return "Windows" }
	if containsAny(text, []string{"systemd", "kernel", "cron", "sshd", "rsyslog","wsl"}) { return "Linux" }
	if containsAny(text, []string{"nginx", "apache", "apache2", "httpd", "iis"}) { return "Web" }
	if containsAny(text, []string{"named", "bind", "unbound"}) { return "DNS" }
	if containsAny(text, []string{"dhcpd", "kea", "dnsmasq"}) { return "DHCP" }
	if containsAny(text, []string{"cisco", "mikrotik", "routeros", "juniper", "aruba", "hp"}) { return "Network" }
	
	return "Unknown"
}

/* Classificação simples baseada em palavras-chave no nome do aplicativo e payload. 
Pode ser expandida com ML ou regras mais complexas. */