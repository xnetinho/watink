package services

import (
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

// Regressão do 500 "sem corpo": se o RabbitMQ estava fora do ar quando o business subiu, Connect() falha, o
// main.go só registra um aviso e segue — e s.channel fica nil para sempre. Todo publish virava nil pointer
// dereference (panic recuperado pelo Gin = 500 sem payload). Tem de ser um erro normal, não um panic.
func TestPublishCommand_WithoutConnection_ReturnsErrorInsteadOfPanicking(t *testing.T) {
	svc := NewRabbitMQProvider("amqp://127.0.0.1:1/")
	if err := svc.Connect(); err == nil {
		t.Fatal("o teste precisa de um broker inalcançável")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PublishCommand entrou em pânico em vez de devolver erro: %v", r)
		}
	}()
	err := svc.PublishCommand("wbot.t.1.message.send.text", map[string]interface{}{"type": "message.send.text"})
	if err == nil {
		t.Fatal("sem conexão o publish deve falhar")
	}
	if !errors.Is(err, ErrRabbitMQNotConnected) {
		t.Fatalf("erro = %v, esperado ErrRabbitMQNotConnected", err)
	}
}

// O broker sobe DEPOIS do business (ordem de subida de um Swarm): a conexão tem de se estabelecer sozinha,
// iniciar os consumidores uma única vez e o publish passar a funcionar sem reiniciar o serviço. O "broker
// que aparece depois" é um proxy TCP que só começa a escutar tarde, na porta que o serviço já tenta.
func TestConnectWithRetry_BrokerAppearsLater(t *testing.T) {
	real := os.Getenv("AMQP_TEST_HOSTPORT")
	if real == "" {
		t.Skip("AMQP_TEST_HOSTPORT não definido (ex.: localhost:56720)")
	}
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := probe.Addr().String()
	probe.Close()

	svc := NewRabbitMQProvider("amqp://guest:guest@" + addr + "/")
	started := make(chan struct{}, 4)
	if svc.ConnectWithRetry(func() { started <- struct{}{} }) {
		t.Fatal("ainda não há broker, não devia conectar de primeira")
	}
	if err := svc.PublishCommand("wbot.t.1.message.send.text", map[string]interface{}{"type": "x"}); !errors.Is(err, ErrRabbitMQNotConnected) {
		t.Fatalf("antes de conectar o publish deve falhar com ErrRabbitMQNotConnected: %v", err)
	}

	// Deixa passar mais de uma tentativa (a cada 5 s) com o broker ainda fora: só um loop que persiste conecta depois.
	time.Sleep(11 * time.Second)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				up, err := net.Dial("tcp", real)
				if err != nil {
					c.Close()
					return
				}
				go io.Copy(up, c)
				io.Copy(c, up)
				c.Close()
				up.Close()
			}()
		}
	}()

	select {
	case <-started:
	case <-time.After(20 * time.Second):
		t.Fatal("a conexão em segundo plano não se estabeleceu")
	}
	if err := svc.PublishCommand("wbot.t.1.message.send.text", map[string]interface{}{"type": "x"}); err != nil {
		t.Fatalf("depois de conectar o publish deve funcionar: %v", err)
	}
	select {
	case <-started:
		t.Fatal("onConnected rodou mais de uma vez")
	case <-time.After(6 * time.Second):
	}
}
