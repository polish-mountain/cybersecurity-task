package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os/exec"
	"strings"
)

func massscanScanner() {
	return
	for {
		proc := exec.Command("../masscan/bin/masscan", "-p", "80", "10.250.192.186/19", "--output-format", "json", "--output-file", "-")
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
				line := scanner.Text()
				line = strings.TrimSpace(line)
				if line == "[" || line == "]" || line == "," {
					continue
				}
				var result MasscanResultRow
				if err := json.Unmarshal([]byte(line), &result); err != nil {
					log.Printf("error unmarshalling json: %v", err)
					continue
				}
				log.Printf("result: %+v", result)
				h := getOrCreateHost(result.IP)
				h.OpenServices = make([]OpenServiceInfo, 0)
				for _, v := range result.Ports {
					alreadyExists := false
					for _, vv := range h.OpenServices {
						if vv.Port == v.Port && vv.Proto == v.Proto {
							alreadyExists = true
							break
						}
					}
					if alreadyExists {
						continue
					}
					h.OpenServices = append(h.OpenServices, OpenServiceInfo{
						Port:  v.Port,
						Proto: v.Proto,
						Title: "",
					})

				}
				updateHost(h)

			}
		}()
		proc.Wait()
		// read from stdout

	}

}
