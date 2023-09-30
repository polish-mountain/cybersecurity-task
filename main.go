package main

import (
	"flag"
	"log"
	"strings"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// flags
var listenAddr = flag.String("listen", ":3000", "address to listen on")

var isScanningMutex sync.Mutex
var isScanning bool = false

type MasscanResultRow struct {
	IP        string `json:"ip"`
	Timestamp string `json:"timestamp"`
	Ports     []struct {
		Port   int    `json:"port"`
		Proto  string `json:"proto"`
		Status string `json:"status"`
		Reason string `json:"reason"`
		TTL    int    `json:"ttl"`
	} `json:"ports"`
}

type OpenServiceInfo struct {
	Port  int    `json:"port"`
	Proto string `json:"proto"`
	Title string `json:"title"`
}

type HostInfo struct {
	IP           string            `json:"ip"`
	Host         string            `json:"host"`
	OpenServices []OpenServiceInfo `json:"open_services"`
	DeviceName   string            `json:"device_name"`
	DeviceType   string            `json:"device_type"`
}

func (h *HostInfo) DetectData() {
	if strings.Contains(h.DeviceName, "MacBook") ||
		strings.Contains(h.DeviceName, "LAPTOP-") ||
		strings.Contains(h.DeviceName, "Pavilion") ||
		strings.Contains(h.DeviceName, "-LAPTOP") ||
		strings.Contains(h.DeviceName, "_LAPTOP") {
		h.DeviceType = "laptop"
	} else if strings.Contains(h.DeviceName, "DESKTOP-") || strings.Contains(h.DeviceName, "Mac mini") || strings.Contains(h.DeviceName, "iMac") {
		h.DeviceType = "desktop"
	} else if strings.Contains(h.DeviceName, "iPhone") || strings.Contains(h.DeviceName, "Redmi") || strings.Contains(h.DeviceName, "POCO") {
		h.DeviceType = "phone"
	} else if strings.Contains(h.DeviceName, "iPad") {
		h.DeviceType = "tablet"
	} else if strings.Contains(h.DeviceName, "MBP") {
		h.DeviceType = "laptop"
	}

}

var hosts = make([]*HostInfo, 0)
var hostsMutex sync.RWMutex = sync.RWMutex{}
var hostListenersMutex sync.RWMutex = sync.RWMutex{}
var hostListeners = make([]chan *HostInfo, 0)

func addHostListener(ch chan *HostInfo) func() {
	hostListenersMutex.Lock()
	defer hostListenersMutex.Unlock()
	hostListeners = append(hostListeners, ch)
	return func() {
		hostListenersMutex.Lock()
		defer hostListenersMutex.Unlock()
		for i, v := range hostListeners {
			if v == ch {
				hostListeners = append(hostListeners[:i], hostListeners[i+1:]...)
				return
			}
		}
	}
}

func updateHost(host *HostInfo) {
	hostListenersMutex.RLock()
	defer hostListenersMutex.RUnlock()
	hostsMutex.Lock()
	defer hostsMutex.Unlock()
	host.DetectData()
	for _, v := range hostListeners {
		select {
		case v <- host:
		default:
		}
	}
}

func getOrCreateHost(ip string) *HostInfo {
	hostsMutex.Lock()
	defer hostsMutex.Unlock()
	for _, v := range hosts {
		if v.IP == ip {
			return v
		}
	}
	host := &HostInfo{
		IP:           ip,
		OpenServices: make([]OpenServiceInfo, 0),
	}
	hosts = append(hosts, host)
	return host
}

func scanningRoutine() {
	isScanningMutex.Lock()
	if isScanning {
		isScanningMutex.Unlock()
		return
	}
	isScanning = true
	isScanningMutex.Unlock()
	go massscanScanner()
	go avahiScanner("avahi-browse", "-apr")
	go avahiScanner("cat", "laptop_avahi.txt")
}

func main() {
	flag.Parse()
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Use("/api/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/api/hosts", func(c *fiber.Ctx) error {
		hostsMutex.RLock()
		defer hostsMutex.RUnlock()
		return c.JSON(hosts)
	})

	app.Get("/api/ws", websocket.New(func(c *websocket.Conn) {

		// websocket.Conn bindings https://pkg.go.dev/github.com/fasthttp/websocket?tab=doc#pkg-index
		readChan := make(chan *HostInfo)
		unsubscribe := addHostListener(readChan)
		defer unsubscribe()
		for {
			select {
			case host := <-readChan:
				if err := c.WriteJSON(host); err != nil {
					log.Printf("error writing to websocket: %v", err)
					return
				}
			}
		}

	}))

	app.Static("/", "./public")
	go scanningRoutine()

	log.Fatal(app.Listen(*listenAddr))

}
