package main

import (
	"flag"
	"log"
	"strings"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/tomruk/oui"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB
var ouiDb *oui.DB

// flags
var listenAddr = flag.String("listen", ":3000", "address to listen on")
var rootURL = flag.String("root-url", "http://localhost:3000", "root url")
var dbPath = flag.String("db", "test.db", "path to database")

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
	IP              string            `json:"ip"`
	Host            string            `json:"host"`
	OpenServices    []OpenServiceInfo `json:"open_services"`
	DeviceName      string            `json:"device_name"`
	DeviceType      string            `json:"device_type"`
	Screenshots     []string          `json:"screenshots"`
	MacAddress      string            `json:"mac_address"`
	MacManufacturer string            `json:"mac_manufacturer"`
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
	func() {

		hostListenersMutex.RLock()
		defer hostListenersMutex.RUnlock()

		hostsMutex.Lock()
		defer hostsMutex.Unlock()
		host.DetectData()
	}()
	for _, v := range hostListeners {
		select {
		case v <- host:
		default:
		}
	}
	for _, serv := range host.OpenServices {
		if serv.Port == 80 {
			screenshotJobQueue <- ScreenshotJob{
				URL:      "http://" + host.IP,
				HostInfo: host,
			}
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
		Screenshots:  make([]string, 0),
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
	go avahiScanner("cat", "phone_avahi.txt")
	// go arpScanner("arp", "-an")
}

func main() {
	flag.Parse()

	var err error
	db, err = gorm.Open(sqlite.Open(
		*dbPath,
	), &gorm.Config{})
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	db.AutoMigrate(&CachedScreenshot{})
	if err := loadDatabase("oui.txt"); err != nil {
		log.Fatalf("error loading oui database: %v", err)
	}
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Use("/api/ws", func(c *fiber.Ctx) error {
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

		go scanningRoutine()

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
	app.Get("/api/screenshot/:uuid", func(c *fiber.Ctx) error {
		var cs CachedScreenshot
		err := db.Where("uuid = ?", c.Params("uuid")).First(&cs).Error
		if err != nil {
			return c.SendStatus(404)
		}
		return c.Send(cs.Data)
	})

	app.Static("/", "./public")

	startScreenshotWorkers()

	log.Fatal(app.Listen(*listenAddr))

}
