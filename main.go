package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"strings"

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

	// ----- MQTT connect -----
	opts := mqtt.NewClientOptions().
		AddBroker(mqttAddr).
		SetClientID("wol-agent")

	if mqttUser != "" {
		opts.SetUsername(mqttUser)
		opts.SetPassword(mqttPass)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	slog.Info("Connected", "broker", mqttAddr)
	slog.Info("Listening", "topic", "wol/trigger")

	client.Subscribe("wol/trigger", 0, func(_ mqtt.Client, msg mqtt.Message) {
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
	})

	select {} // block forever
}
