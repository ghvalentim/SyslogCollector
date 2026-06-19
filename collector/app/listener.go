package app

import (
	"log"
	"net"
	"strings"
)

func InitListener() {
	go startUDPServer()
	go startTCPServer()
}

func startUDPServer() {
	addr := net.UDPAddr{Port: 514, IP: net.ParseIP("0.0.0.0")}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil { log.Fatalf("Erro UDP: %v", err) }
	defer conn.Close()
	buf := make([]byte, 8192)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err == nil { logChan <- LogJob{SourceIP: remoteAddr.IP.String(), Protocol: "UDP", Payload: string(buf[:n])} }
	}
}

func startTCPServer() {
	listener, err := net.Listen("tcp", ":514")
	if err != nil { log.Fatalf("Erro TCP: %v", err) }
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err == nil { go handleTCPConnection(conn) }
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 8192)
	for {
		n, err := conn.Read(buf)
		if err != nil { break }
		ip := strings.Split(conn.RemoteAddr().String(), ":")[0]
		logChan <- LogJob{SourceIP: ip, Protocol: "TCP", Payload: string(buf[:n])}
	}
}

/* Listener simples para receber logs via UDP e TCP na porta 514.
Cada mensagem recebida é enviada para a Worker Pool através do canal logChan.
Pode ser expandido para suportar TLS, autenticação, ou outros protocolos. */