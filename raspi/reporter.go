package main

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "sync"
    "time"

    jwt "github.com/dgrijalva/jwt-go"
    MQTT "github.com/eclipse/paho.mqtt.golang"
)

type ReporterConfig struct {
    Tags string `default:"room=raspi-grove"`
}

type Data struct {
	Name  string
	Value float64
	Tags  string
}

type Reporter struct {
	cfg  ReporterConfig
    mu sync.Mutex
	data []Data
}

func NewReporter(cfg ReporterConfig) *Reporter {
	return &Reporter{cfg: cfg}
}

var (
    deviceID = flag.String("device", "", "Cloud IoT Core Device ID")
    bridge = struct {
        host *string
        port *string
    }{
        flag.String("mqtt_host", "mqtt.googleapis.com", "MQTT Bridge Host"),
        flag.String("mqtt_port", "8883", "MQTT Bridge Port"),
    }
    projectID = flag.String("project", "", "GCP Project ID")
    registryID = flag.String("registry", "", "Cloud IoT Registry ID (short from)")
    region = flag.String("region", "", "GCP Region")
    certsCA = flag.String("ca_certs", "", "Dowload https://pki.google.com/roots.pem")
    privateKey = flag.String("private_key", "", "Path to private key file")
    measurement = flag.String("influx_db", "Raspberry_Pi_Zero_W", "InfluxDB Measurement to store into")
    format = flag.String("format", "line", "Data format: influxdb line protocol")
)

func (r *Reporter) Run(ctx context.Context) error {
    log.Println("[main] Entered")

    log.Println("[main] Flags")
    flag.Parse()

    log.Println("[main] Loading Google's roots")
    certpool := x509.NewCertPool()
    pemCerts, err := ioutil.ReadFile(*certsCA)
    if err == nil {
        certpool.AppendCertsFromPEM(pemCerts)
    }
    log.Println("[main] Creating TLS Config")

    config := &tls.Config{
        RootCAs:            certpool,
        ClientAuth:         tls.NoClientCert,
        ClientCAs:          nil,
        InsecureSkipVerify: true,
        Certificates:       []tls.Certificate{},
        MinVersion:         tls.VersionTLS12,
    }

    clientID := fmt.Sprintf("projects/%v/locations/%v/registries/%v/devices/%v",
        *projectID,
        *region,
        *registryID,
        *deviceID,
    )

    log.Println("[main] Creating MQTT Client Options")
    opts := MQTT.NewClientOptions()

    broker := fmt.Sprintf("ssl://%v:%v", *bridge.host, *bridge.port)
    log.Printf("[main] Broker '%v'", broker)

    opts.AddBroker(broker)
    opts.SetClientID(clientID).SetTLSConfig(config)

    opts.SetUsername("unused")

    token := jwt.New(jwt.SigningMethodRS256)
    token.Claims = jwt.StandardClaims{
        Audience:  *projectID,
        IssuedAt:  time.Now().Unix(),
        ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
    }

    log.Println("[main] Load Private Key")
    keyBytes, err := ioutil.ReadFile(*privateKey)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("[main] Parse Private Key")
    key, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("[main] Sign String")
    tokenString, err := token.SignedString(key)
    if err != nil {
        log.Fatal(err)
    }

    opts.SetPassword(tokenString)


    // Incoming
    opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
        fmt.Printf("[handler] Topic: %v\n", msg.Topic())
        fmt.Printf("[handler] Payload: %v\n", msg.Payload())
    })
    log.Println("[main] MQTT Client Connecting")
    client := MQTT.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        log.Fatal(token.Error())
    }

    topic := struct {
        config    string
        telemetry string
    }{
        config:    fmt.Sprintf("/devices/%v/config", *deviceID),
        telemetry: fmt.Sprintf("/devices/%v/events", *deviceID),
    }

    log.Println("[main] Creating Subscription")
    client.Subscribe(topic.config, 0, nil)

    // MQTT
    t := time.NewTicker(time.Second)
    defer t.Stop()
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-t.C:
            r.mu.Lock()
            mString := ""
            for _, d := range r.data {
                /*tags := d.Tags
                if tags != "" {
                    tags = "," + tags
                }*/
                // Tag
                addr := getLocalIp()
                switch addr { // Add raspberry's IP address here
                case "192.168.1.13":
                    r.cfg.Tags = "room=Bureau"
                default:
                    r.cfg.Tags = "room=raspi-grove"
                }
                // Data
                if mString == ""{
                    mString = fmt.Sprintf("%s,%s %s=%0.2f %d", *measurement, r.cfg.Tags, d.Name, d.Value, time.Now().UnixNano())
                } else {
                    mString = fmt.Sprintf("%s\n%s,%s %s=%0.2f %d", mString, *measurement, r.cfg.Tags, d.Name, d.Value, time.Now().UnixNano())
                }
                log.Printf("[main] %s: %.02f\n", d.Name, d.Value)
                time.Sleep(time.Second)
            }
            r.data = r.data[:0]
            r.mu.Unlock()

            if len(mString) > 0 {
                log.Printf("[main] Sending\n")
                token := client.Publish(
                    topic.telemetry,
                    0,
                    false,
                    mString)
                token.WaitTimeout(5 * time.Second)
                // Debug
                fmt.Printf("%s\n", mString)
            }
        }
    }
}

    // HTTP
	/*t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			r.mu.Lock()
			msg := fmt.Sprintf("alive,%s value=1\n", r.cfg.Tags)
			for _, d := range r.data {
				tags := d.Tags
				if tags != "" {
					tags = "," + tags
				}
				msg += fmt.Sprintf("%s,%s%s value=%s\n", d.Name, r.cfg.Tags, tags, d.Value)
			}
			r.data = r.data[:0]
			r.mu.Unlock()
			fmt.Println("sending", msg)
			resp, err := http.Post(r.cfg.URL, "application/x-www-form-urlencoded", bytes.NewBufferString(msg))
			if err != nil {
				fmt.Println("err", err)
			} else {
				defer resp.Body.Close()
				buf, _ := ioutil.ReadAll(resp.Body)
				fmt.Println("response", resp.StatusCode, "-", string(buf))
			}
		}
	}*/
func (r *Reporter) Report(name, tags string, value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
    r.data = append(r.data, Data{Name: name, Tags: tags, Value: value})
}
