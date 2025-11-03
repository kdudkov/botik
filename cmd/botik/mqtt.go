package main

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	tokenTimeout = time.Millisecond * 500
)

type MqttConfig interface {
	MQTTServer() string
	MQTTUser() string
	MQTTPassword() string
	MQTTClientID() string
}

type MqttClient struct {
	mqttConnected int32
	client        mqtt.Client
	counter       int32
	sendQueue     chan *Message
	logger        *slog.Logger
	config        MqttConfig
	cb            func(topic string, payload []byte)
}

type Message struct {
	Topic   string
	Payload string
	Qos     byte
}

func NewMqttClient(logger *slog.Logger, c MqttConfig, cb func(topic string, payload []byte)) *MqttClient {
	cl := &MqttClient{
		sendQueue: make(chan *Message, 100),
		logger:    logger.With(slog.String("logger", "mqtt")),
		config:    c,
		cb:        cb,
	}
	
	cl.setup()

	return cl
}

func (m *MqttClient) setup() {
	opts := mqtt.NewClientOptions().
		AddBroker(m.config.MQTTServer()).
		SetConnectTimeout(time.Second * 3).
		SetWriteTimeout(time.Second * 3).
		SetAutoReconnect(true).
		SetClientID(m.config.MQTTClientID()).
		SetUsername(m.config.MQTTUser()).
		SetPassword(m.config.MQTTPassword()).
		SetOnConnectHandler(m.onConnected).
		SetConnectionLostHandler(m.onDisconnected).
		SetDefaultPublishHandler(m.onReceive)

	m.client = mqtt.NewClient(opts)
}

func (m *MqttClient) setConnected(t bool) {
	if t {
		atomic.StoreInt32(&m.mqttConnected, 1)
	} else {
		atomic.StoreInt32(&m.mqttConnected, 0)
	}
}

func (m *MqttClient) isConnected() bool {
	return atomic.LoadInt32(&m.mqttConnected) == 1
}

func (m *MqttClient) Run(ctx context.Context) {
	m.Connect()
	
	for {
		select {
		case <-ctx.Done():
			m.logger.Info("stopping sender")
			return
		case msg := <-m.sendQueue:
			token := m.client.Publish(msg.Topic, msg.Qos, false, []byte(msg.Payload))
			if !token.WaitTimeout(tokenTimeout) {
				m.logger.Error("send timeout")
				break
			}

			if token.Error() != nil {
				m.logger.Error("publish error", "error", token.Error())
			}
		}
	}
}

func (m *MqttClient) tryConnect() error {
	if m.isConnected() {
		return nil
	}

	m.logger.Info("connecting...")

	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		m.logger.Error("Connect error", "error", token.Error())
		return token.Error()
	}

	return nil
}

func (m *MqttClient) Connect() {
	if m.isConnected() {
		return
	}

	timeout := time.Second
	for {
		if err := m.tryConnect(); err == nil {
			return
		}
		time.Sleep(timeout)
		if timeout < time.Second*30 {
			timeout *= 2
		}
	}
}

func (m *MqttClient) onConnected(_ mqtt.Client) {
	m.setConnected(true)
	m.logger.Info("MQTT connected")

	if token := m.client.Subscribe("frigate/reviews", 0, nil); token.Wait() && token.Error() != nil {
		m.logger.Error("subscribe error", "error", token.Error())
		m.client.Disconnect(10)
	}
}

func (m *MqttClient) onDisconnected(_ mqtt.Client, err error) {
	m.setConnected(false)
	m.logger.Info("MQTT disconnected", slog.Any("error", err))
	time.AfterFunc(time.Second, m.Connect)
}

func (m *MqttClient) onReceive(_ mqtt.Client, msg mqtt.Message) {
	m.cb(msg.Topic(), msg.Payload())
}

func (m *MqttClient) Send(topic string, payload string, qos byte) bool {
	if !m.isConnected() {
		return false
	}

	select {
	case m.sendQueue <- &Message{Topic: topic, Payload: payload, Qos: qos}:
		return true
	default:
		m.logger.Warn("sendQueue is full")
		return false
	}
}
