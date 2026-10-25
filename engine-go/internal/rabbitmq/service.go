package rabbitmq

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alltomatos/watinkdev/engine-go/internal/safego"
	"github.com/streadway/amqp"
)

const (
	// amqpPreconditionFailed é o código AMQP 406 (argumentos da fila diferentes dos da fila existente).
	amqpPreconditionFailed = 406

	commandsExchange = "wbot.commands"
	eventsExchange   = "wbot.events"

	// Reentregas de um comando recuperável antes de ir para a fila de falhas.
	maxCommandRetries = 3
	retryBaseBackoff  = 2 * time.Second
	retryMaxBackoff   = 30 * time.Second

	reconnectInitial = 2 * time.Second
	reconnectMax     = 30 * time.Second

	// Espera máxima pela confirmação do broker antes de tratar o evento como pendente.
	confirmTimeout = 10 * time.Second
)

// ErrNotConnected indica que não há conexão com o broker neste momento.
var ErrNotConnected = errors.New("rabbitmq: sem conexão com o broker")

// ErrUnrecoverable marca uma falha que não adianta repetir (comando mal formado, sessão inexistente...). O consumidor
// manda o comando direto para a fila de falhas em vez de reprocessá-lo.
var ErrUnrecoverable = errors.New("falha irrecuperável")

// Unrecoverable embrulha err como irrecuperável.
func Unrecoverable(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrUnrecoverable, err)
}

// consumerSpec descreve um consumidor para que o supervisor possa recriá-lo depois de uma queda.
type consumerSpec struct {
	queue      string
	keys       []string
	concurrent bool
	handler    func(amqp.Delivery) error
}

// RabbitMQService mantém a conexão do engine com o broker.
//
//   - Publicação: canal próprio com confirmações do broker e saída durável (outbox em disco). Um evento só sai do disco
//     depois que o broker o confirma; com o broker fora, ele espera e é reenviado quando a conexão volta.
//   - Consumo: um canal por consumidor, refeito sozinho depois de uma queda do canal ou da conexão.
//   - Comandos: Ack só no sucesso; falha recuperável é reentregue com limite e backoff; irrecuperável ou esgotada vai
//     para a fila de falhas (DLQ). Um pânico no tratador vira falha, nunca derruba o engine.
type RabbitMQService struct {
	url string

	mu      sync.Mutex
	conn    *amqp.Connection
	pubCh   *amqp.Channel
	confirm chan amqp.Confirmation

	consumers []consumerSpec
	closed    bool
	connected chan struct{} // fechado enquanto há conexão

	out         *outbox
	supervisors sync.Once
	pubMu       sync.Mutex // serializa publicação + espera de confirmação
	drain       sync.Mutex // um reenvio da outbox por vez
	wakeCh      chan struct{}
}

// NewRabbitMQService cria o serviço lendo AMQP_URL do ambiente (padrão: guest@localhost) e a pasta da saída durável de
// ENGINE_OUTBOX_DIR (padrão: o diretório temporário do sistema, o que NÃO sobrevive a recriar o contêiner: em produção
// monte um volume).
func NewRabbitMQService() *RabbitMQService {
	url := os.Getenv("AMQP_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}
	return &RabbitMQService{url: url, wakeCh: make(chan struct{}, 1)}
}

func outboxPath() string {
	dir := os.Getenv("ENGINE_OUTBOX_DIR")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "watink-engine")
	}
	return filepath.Join(dir, "events.outbox")
}

// Connect abre a conexão, declara as trocas e recria os consumidores já registrados. Chamar de novo depois de uma queda
// é seguro: o supervisor faz isso sozinho.
func (s *RabbitMQService) Connect() error {
	if err := s.openOutbox(); err != nil {
		return err
	}
	if err := s.dial(); err != nil {
		return err
	}
	s.startSupervisorsOnce()
	return nil
}

func (s *RabbitMQService) openOutbox() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.out != nil {
		return nil
	}
	o, err := openOutbox(outboxPath())
	if err != nil {
		return err
	}
	s.out = o
	if n := o.Len(); n > 0 {
		log.Printf("[RabbitMQ] %d evento(s) pendente(s) da execução anterior serão reenviados", n)
	}
	return nil
}

// dial estabelece conexão, canal de publicação com confirmações e trocas. Não reabre consumidores (ver resubscribe).
func (s *RabbitMQService) dial() error {
	conn, err := amqp.Dial(s.url)
	if err != nil {
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("rabbitmq channel: %w", err)
	}
	for _, ex := range []string{commandsExchange, eventsExchange} {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			_ = conn.Close()
			return fmt.Errorf("exchange %s: %w", ex, err)
		}
	}
	if err := ch.Confirm(false); err != nil {
		_ = conn.Close()
		return fmt.Errorf("rabbitmq confirm mode: %w", err)
	}
	confirm := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	s.mu.Lock()
	s.conn, s.pubCh, s.confirm = conn, ch, confirm
	if s.connected == nil {
		s.connected = make(chan struct{})
	}
	select {
	case <-s.connected:
	default:
		close(s.connected)
	}
	s.mu.Unlock()
	log.Println("[RabbitMQ] connected")
	return nil
}

// startSupervisorsOnce liga, uma vez por serviço, o vigia da conexão e o reenviador da saída durável.
func (s *RabbitMQService) startSupervisorsOnce() {
	s.supervisors.Do(func() {
		safego.Go("rabbitmq.watchConnection", s.watchConnection)
		safego.Go("rabbitmq.drainOutbox", s.drainLoop)
		s.wake()
	})
}

// watchConnection reconecta com backoff sempre que a conexão cai e recria os consumidores.
func (s *RabbitMQService) watchConnection() {
	for {
		s.mu.Lock()
		conn, closed := s.conn, s.closed
		s.mu.Unlock()
		if closed {
			return
		}
		if conn != nil {
			<-conn.NotifyClose(make(chan *amqp.Error, 1))
			s.mu.Lock()
			if s.closed {
				s.mu.Unlock()
				return
			}
			s.connected = make(chan struct{}) // volta a "sem conexão"
			s.mu.Unlock()
			log.Println("[RabbitMQ] conexão perdida, reconectando...")
		}
		backoff := reconnectInitial
		for {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return
			}
			if err := s.dial(); err != nil {
				log.Printf("[RabbitMQ] reconexão falhou (%v); nova tentativa em %v", err, backoff)
				time.Sleep(backoff)
				if backoff < reconnectMax {
					backoff *= 2
				}
				continue
			}
			s.resubscribeAll()
			s.wake()
			break
		}
	}
}

// ConsumeCommands registra um consumidor serial (uma entrega por vez) de comandos. O tratador devolve o erro: nil = Ack,
// erro = reentrega com limite ou DLQ.
func (s *RabbitMQService) ConsumeCommands(queueName string, routingKeys []string, handler func(amqp.Delivery) error) error {
	return s.addConsumer(consumerSpec{queue: queueName, keys: routingKeys, handler: handler})
}

// ConsumeCommandsConcurrent é o consumidor das chamadas: canal próprio e uma goroutine por entrega, sem laço serial.
// "Atender" ou "Encerrar" não podem esperar atrás do envio de mensagens de nenhuma empresa.
func (s *RabbitMQService) ConsumeCommandsConcurrent(queueName string, routingKeys []string, handler func(amqp.Delivery) error) error {
	return s.addConsumer(consumerSpec{queue: queueName, keys: routingKeys, concurrent: true, handler: handler})
}

func (s *RabbitMQService) addConsumer(spec consumerSpec) error {
	s.mu.Lock()
	s.consumers = append(s.consumers, spec)
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return ErrNotConnected
	}
	return s.startConsumer(spec)
}

func (s *RabbitMQService) resubscribeAll() {
	s.mu.Lock()
	specs := append([]consumerSpec(nil), s.consumers...)
	s.mu.Unlock()
	for _, spec := range specs {
		if err := s.startConsumer(spec); err != nil {
			log.Printf("[RabbitMQ] não foi possível retomar o consumidor %q: %v", spec.queue, err)
		}
	}
}

func (s *RabbitMQService) dlqName(queue string) string { return queue + ".dlq" }

// startConsumer abre um canal exclusivo, declara a fila (com a DLQ) e começa a consumir. Quando o canal fecha, o
// consumidor termina e watchConnection o recria assim que houver conexão.
func (s *RabbitMQService) startConsumer(spec consumerSpec) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return ErrNotConnected
	}
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("channel %s: %w", spec.queue, err)
	}
	if err := ch.Qos(10, 0, false); err != nil {
		log.Printf("[RabbitMQ] QoS warning (%s): %v", spec.queue, err)
	}
	dlq := s.dlqName(spec.queue)
	if _, err := ch.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		return fmt.Errorf("dlq %s: %w", dlq, err)
	}
	args := amqp.Table{"x-dead-letter-exchange": "", "x-dead-letter-routing-key": dlq}
	var parked []amqp.Delivery
	if _, err := ch.QueueDeclare(spec.queue, true, false, false, false, args); err != nil {
		var ae *amqp.Error
		if !errors.As(err, &ae) || ae.Code != amqpPreconditionFailed {
			_ = ch.Close()
			return fmt.Errorf("queue %s: %w", spec.queue, err)
		}
		// A fila já existe sem a fila de falhas (versão antiga do engine) e o broker não deixa mudar os argumentos de
		// uma fila existente. Apaga e recria; os comandos que estavam parados nela são guardados e republicados.
		ch, parked, err = s.recreateQueue(conn, spec, args)
		if err != nil {
			return err
		}
	}
	for _, key := range spec.keys {
		if err := ch.QueueBind(spec.queue, key, commandsExchange, false, nil); err != nil {
			_ = ch.Close()
			return fmt.Errorf("bind %s: %w", key, err)
		}
	}
	for _, d := range parked {
		if err := ch.Publish(commandsExchange, d.RoutingKey, false, false, amqp.Publishing{
			ContentType: d.ContentType, DeliveryMode: amqp.Persistent, Body: d.Body, Headers: d.Headers, Timestamp: time.Now(),
		}); err != nil {
			log.Printf("[RabbitMQ] comando %s da fila antiga %q NÃO foi republicado: %v", d.RoutingKey, spec.queue, err)
		}
	}
	if len(parked) > 0 {
		log.Printf("[RabbitMQ] fila %q recriada com fila de falhas; %d comando(s) parado(s) republicado(s)", spec.queue, len(parked))
	}
	msgs, err := ch.Consume(spec.queue, "", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("consume %s: %w", spec.queue, err)
	}
	safego.Go("amqp.consumer."+spec.queue, func() {
		for d := range msgs {
			if spec.concurrent {
				d := d
				safego.Go("amqp.handle."+spec.queue, func() { s.handleDelivery(ch, spec, d) })
				continue
			}
			s.handleDelivery(ch, spec, d)
		}
		log.Printf("[RabbitMQ] consumidor %q encerrado (canal fechado)", spec.queue)
	})
	return nil
}

// maxParked limita quantos comandos parados na fila antiga são guardados em memória durante a recriação.
const maxParked = 10000

// recreateQueue esvazia (para memória), apaga e recria a fila com os argumentos novos. Devolve um canal novo, porque o
// erro de pré-condição fecha o canal anterior.
func (s *RabbitMQService) recreateQueue(conn *amqp.Connection, spec consumerSpec, args amqp.Table) (*amqp.Channel, []amqp.Delivery, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, fmt.Errorf("channel %s: %w", spec.queue, err)
	}
	var parked []amqp.Delivery
	for len(parked) < maxParked {
		d, ok, err := ch.Get(spec.queue, false)
		if err != nil || !ok {
			break
		}
		parked = append(parked, d)
	}
	if _, err := ch.QueueDelete(spec.queue, false, false, false); err != nil {
		_ = ch.Close()
		return nil, nil, fmt.Errorf("apagar fila antiga %s: %w", spec.queue, err)
	}
	if _, err := ch.QueueDeclare(spec.queue, true, false, false, false, args); err != nil {
		_ = ch.Close()
		return nil, nil, fmt.Errorf("recriar fila %s: %w", spec.queue, err)
	}
	return ch, parked, nil
}

// handleDelivery roda o tratador contendo pânico e decide Ack, reentrega ou DLQ.
func (s *RabbitMQService) handleDelivery(ch *amqp.Channel, spec consumerSpec, d amqp.Delivery) {
	err := safego.Run("amqp.handler."+spec.queue, func() error { return spec.handler(d) })
	if err == nil {
		if aerr := d.Ack(false); aerr != nil {
			log.Printf("[RabbitMQ] Ack falhou (%s): %v", spec.queue, aerr)
		}
		return
	}
	s.failDelivery(ch, spec, d, err)
}

func retryCount(d amqp.Delivery) int {
	switch v := d.Headers["x-retry-count"].(type) {
	case int32:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// failDelivery reentrega um comando recuperável (com limite e backoff) ou o envia para a DLQ.
func (s *RabbitMQService) failDelivery(ch *amqp.Channel, spec consumerSpec, d amqp.Delivery, cause error) {
	n := retryCount(d)
	if errors.Is(cause, ErrUnrecoverable) || n >= maxCommandRetries {
		log.Printf("[RabbitMQ] comando %s vai para a fila de falhas (tentativa %d): %v", d.RoutingKey, n, cause)
		if err := d.Nack(false, false); err != nil {
			log.Printf("[RabbitMQ] Nack falhou (%s): %v", spec.queue, err)
		}
		return
	}
	backoff := retryBaseBackoff << uint(n)
	if backoff > retryMaxBackoff {
		backoff = retryMaxBackoff
	}
	log.Printf("[RabbitMQ] comando %s falhou (%v); nova tentativa %d/%d em %v", d.RoutingKey, cause, n+1, maxCommandRetries, backoff)
	time.Sleep(backoff)

	headers := amqp.Table{}
	for k, v := range d.Headers {
		headers[k] = v
	}
	headers["x-retry-count"] = int32(n + 1)
	if err := ch.Publish(commandsExchange, d.RoutingKey, false, false, amqp.Publishing{
		ContentType: d.ContentType, DeliveryMode: amqp.Persistent, Body: d.Body, Headers: headers, Timestamp: time.Now(),
	}); err != nil {
		log.Printf("[RabbitMQ] não foi possível reenfileirar (%v); enviando à fila de falhas", err)
		_ = d.Nack(false, false)
		return
	}
	if err := d.Ack(false); err != nil {
		log.Printf("[RabbitMQ] Ack após reenfileirar falhou (%s): %v", spec.queue, err)
	}
}

// PublishEvent grava o evento na saída durável e tenta entregá-lo ao broker. Devolve erro só se não conseguiu gravar em
// disco; se o broker estiver fora, o evento fica guardado e é reenviado quando a conexão voltar.
func (s *RabbitMQService) PublishEvent(routingKey string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := s.openOutbox(); err != nil {
		return err
	}
	s.mu.Lock()
	out := s.out
	s.mu.Unlock()
	if _, err := out.Add(routingKey, body); err != nil {
		return err
	}
	s.wake()
	return nil
}

func (s *RabbitMQService) wake() {
	select {
	case s.wakeCh <- struct{}{}:
	default:
	}
}

// drainLoop reenvia, em ordem, o que está pendente na saída durável.
func (s *RabbitMQService) drainLoop() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.wakeCh:
		case <-t.C:
		}
		s.mu.Lock()
		closed := s.closed
		s.mu.Unlock()
		if closed {
			return
		}
		s.drainOnce()
	}
}

// drainOnce publica todos os pendentes e tira do disco cada um que o broker confirmar. Para no primeiro que falhar, para
// manter a ordem de chegada.
func (s *RabbitMQService) drainOnce() {
	s.drain.Lock()
	defer s.drain.Unlock()
	s.mu.Lock()
	out := s.out
	s.mu.Unlock()
	if out == nil {
		return
	}
	for _, e := range out.Pending() {
		if err := s.publishConfirmed(e); err != nil {
			if !errors.Is(err, ErrNotConnected) {
				log.Printf("[RabbitMQ] evento %s não confirmado (fica pendente): %v", e.RoutingKey, err)
			}
			return
		}
		if err := out.Confirm(e.Seq); err != nil {
			log.Printf("[RabbitMQ] não foi possível tirar o evento %d da saída durável: %v", e.Seq, err)
			return
		}
	}
}

// publishConfirmed publica e espera a confirmação do broker (ou erro/timeout).
func (s *RabbitMQService) publishConfirmed(e outboxEntry) error {
	s.pubMu.Lock()
	defer s.pubMu.Unlock()
	s.mu.Lock()
	ch, confirm := s.pubCh, s.confirm
	s.mu.Unlock()
	if ch == nil {
		return ErrNotConnected
	}
	if err := ch.Publish(eventsExchange, e.RoutingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         e.Body,
		Timestamp:    time.Now(),
		MessageId:    fmt.Sprintf("%d", e.Seq),
	}); err != nil {
		return fmt.Errorf("%w: %v", ErrNotConnected, err)
	}
	select {
	case c, ok := <-confirm:
		if !ok || !c.Ack {
			return errors.New("broker não confirmou a publicação")
		}
		return nil
	case <-time.After(confirmTimeout):
		return errors.New("tempo esgotado esperando a confirmação do broker")
	}
}

// Pending informa quantos eventos aguardam confirmação (para métricas e testes).
func (s *RabbitMQService) Pending() int {
	s.mu.Lock()
	out := s.out
	s.mu.Unlock()
	if out == nil {
		return 0
	}
	return out.Len()
}

// Close fecha canal e conexão. Os eventos pendentes ficam no disco para a próxima execução.
func (s *RabbitMQService) Close() error {
	s.mu.Lock()
	s.closed = true
	ch, conn, out := s.pubCh, s.conn, s.out
	s.mu.Unlock()
	s.wake()

	if ch != nil {
		_ = ch.Close()
	}
	var err error
	if conn != nil {
		err = conn.Close()
	}
	if out != nil {
		_ = out.Close()
	}
	return err
}
