package app

// HumanizeFacility converte o código de facility num nome legível, como "Kernel", "User", "Mail", etc.
func HumanizeFacility(facStr string) string {
	facMap := map[string]string{
		"0": "Kernel", "1": "User", "2": "Mail", "3": "Daemon", "4": "Auth", "5": "Syslog",
		"6": "LPR", "7": "News", "8": "UUCP", "9": "Cron", "10": "AuthPriv", "11": "FTP",
		"12": "NTP", "13": "Security", "14": "Console", "15": "Clock", "16": "Local0",
		"17": "Local1", "18": "Local2", "19": "Local3", "20": "Local4", "21": "Local5",
		"22": "Local6", "23": "Local7",
	}
	if name, exists := facMap[facStr]; exists { return name }
	return "Unknown"
}

/* Classificação simples baseada em palavras-chave no nome do aplicativo e payload. 
Pode ser expandida com ML ou regras mais complexas. */