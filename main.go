package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Msg struct {
	MAC string `json:"mac"`
}

func main() {
	// ----- ENV variables -----
	mqttAddr := os.Getenv("MQTT_ADDR")
	if mqttAddr == "" {
		log.Fatal("MQTT_ADDR environment variable not set")
	}

	mqttUser := os.Getenv("MQTT_USER")
	mqttPass := os.Getenv("MQTT_PASS")

	topic := os.Getenv("MQTT_TOPIC")
	if topic == "" {
		topic = "wol/trigger"
	}

	// ----- MQTT options -----
	opts := mqtt.NewClientOptions().
		AddBroker(mqttAddr).
		SetClientID("wol-agent").
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(10 * time.Second)

	if mqttUser != "" {
		opts.SetUsername(mqttUser)
		opts.SetPassword(mqttPass)
	}

	// ----- Re-subscribe on every connect -----
	opts.OnConnect = func(c mqtt.Client) {
		slog.Info("Connected", "broker", mqttAddr)

		token := c.Subscribe(topic, 0, msgHandler)
		token.Wait()
		if token.Error() != nil {
			slog.Error("Subscribe failed", "error", token.Error())
		} else {
			slog.Info("Subscribed", "topic", topic)
		}
	}

	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		slog.Warn("Connection lost", "error", err)
	}

	// ----- Connect -----
	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if token.Error() != nil {
		log.Fatal(token.Error())
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	slog.Info("Shutting down...")

	client.Disconnect(250) // ms to wait for in-flight messages
}

// ----- Message handler -----
func msgHandler(_ mqtt.Client, msg mqtt.Message) {
	var m Msg
	if err := json.Unmarshal(msg.Payload(), &m); err != nil {
		slog.Error("Invalid JSON payload", "error", err)
		return
	}

	mac := strings.ToUpper(strings.TrimSpace(m.MAC))
	slog.Info("Received WOL request", "mac", mac)

	pkt, err := NewMagicPacket(mac)
	if err != nil {
		slog.Error("Invalid MAC", "error", err)
		return
	}

	// Send broadcast to port 9
	if err := pkt.Send("255.255.255.255:9"); err != nil {
		slog.Error("Sending", "error", err)
	} else {
		slog.Info("WOL sent", "mac", mac)
	}
}
