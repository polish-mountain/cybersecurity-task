package main

import (
	"bufio"
	"log"
	"os/exec"
	"strings"
	"time"
)

func arpScanner(name string, args ...string) {
	for {
		proc := exec.Command(name, args...)
		out, err := proc.StdoutPipe()
		if err != nil {
			log.Printf("error creating stdout pipe: %v", err)
			return
		}
		if err := proc.Start(); err != nil {
			log.Printf("error starting scan: %v", err)
			return
		}
		go func() {
			scanner := bufio.NewScanner(out)
			for scanner.Scan() {
				segments := strings.Split(scanner.Text(), " ")
				// parse segments
				if len(segments) < 4 {
					continue
				}
				ipAddr := segments[1]
				macAddr := segments[3]
				ipAddr = strings.ReplaceAll(ipAddr, "(", "")
				ipAddr = strings.ReplaceAll(ipAddr, ")", "")

				// log.Printf("result: %#v", segments)
				host := getOrCreateHost(ipAddr)
				ouiResult, ok := lookupMac(macAddr)
				if !ok {
					// log.Printf("error querying oui database: %v", err)

				} else {
					host.MacManufacturer = ouiResult.Abbreviation
				}
				host.MacAddress = macAddr
				updateHost(host)

			}
		}()
		proc.Wait()
		// read from stdout
		time.Sleep(5 * time.Second)

	}
}
