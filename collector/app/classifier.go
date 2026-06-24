package app

import (
	"strings"
)

// containsAny verifica se a string 's' contém qualquer uma das substrings fornecidas em 'substrs'.
func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs { if strings.Contains(s, sub) { return true } }; return false
}

// ClassifySource classifica a origem de um log com base no nome do aplicativo e no payload, 
// retornando uma categoria como "Firewall", "Windows", "Linux", etc.
func ClassifySource(appName, payload string) string {
	text := strings.ToLower(appName + " " + payload)
	
	if containsAny(text, []string{"pfsense", "opnsense", "fortigate", "sophos", "watchguard", "iptables", "nftables"}) { return "Firewall" }
	if containsAny(text, []string{"microsoft", "winrm", "windows", "eventlog"}) { return "Windows" }
	if containsAny(text, []string{"systemd", "kernel", "cron", "sshd", "rsyslog","wsl"}) { return "Linux" }
	if containsAny(text, []string{"nginx", "apache", "apache2", "httpd", "iis"}) { return "Web" }
	if containsAny(text, []string{"named", "bind", "unbound"}) { return "DNS" }
	if containsAny(text, []string{"dhcpd", "kea", "dnsmasq"}) { return "DHCP" }
	if containsAny(text, []string{"cisco","mtkwlex", "mikrotik", "routeros", "juniper", "aruba", "hp"}) { return "Network" }
	if containsAny(text, []string{"VMICTimeProvider", "VMSMP", "Hyper-V", "vmware", "virtualbox", "qemu"}) { return "Virtualization" }
	
	return "Unknown"
}

/* Classificação simples baseada em palavras-chave no nome do aplicativo e payload. 
Pode ser expandida com ML ou regras mais complexas. */