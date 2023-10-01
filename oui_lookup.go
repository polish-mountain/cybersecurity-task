package main

import (
	"bufio"
	"os"
	"strings"
)

type CompanyInfo struct {
	Abbreviation string
	FullName     string
}

var macDatabase map[string]CompanyInfo

func loadDatabase(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	macDatabase = make(map[string]CompanyInfo)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue // skip invalid lines
		}

		mac := strings.TrimSpace(parts[0])
		abbreviation := strings.TrimSpace(parts[1])
		fullName := strings.TrimSpace(parts[2])

		macDatabase[mac] = CompanyInfo{
			Abbreviation: abbreviation,
			FullName:     fullName,
		}
	}

	return scanner.Err()
}

func lookupMac(mac string) (CompanyInfo, bool) {
	if len(mac) < 8 {
		return CompanyInfo{}, false
	}
	info, exists := macDatabase[strings.ToUpper(mac[:8])]
	return info, exists
}
