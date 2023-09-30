package main

import (
	"bufio"
	"log"
	"os/exec"
	"strings"
)

func avahiScanner(name string, args ...string) {
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
				segments := strings.Split(scanner.Text(), ";")
				// parse segments
				if len(segments) < 8 {
					continue
				}
				deviceName := echoE(segments[3])
				ipAddr := segments[7]

				if len(deviceName) < 3 {
					continue
				}
				// log.Printf("result: %#v", segments)
				host := getOrCreateHost(ipAddr)
				host.DeviceName = deviceName
				updateHost(host)

			}
		}()
		proc.Wait()
		// read from stdout
		return
	}
}
