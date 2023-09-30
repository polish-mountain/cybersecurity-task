package main

import (
	"strconv"
	"strings"
)

// func echoE(input string) string {
// 	cmd := exec.Command("echo", "-e", input)
// 	out, err := cmd.Output()
// 	if err != nil {
// 		log.Printf("error running echo: %v", err)
// 		return input
// 	}
// 	return string(out)
// }

func echoE(input string) string {
	var output strings.Builder
	i := 0
	for i < len(input) {
		if input[i] == '\\' {
			if i+1 < len(input) {
				switch input[i+1] {
				case 'a':
					output.WriteByte('\a')
					i += 2
				case 'b':
					output.WriteByte('\b')
					i += 2
				case 'f':
					output.WriteByte('\f')
					i += 2
				case 'n':
					output.WriteByte('\n')
					i += 2
				case 'r':
					output.WriteByte('\r')
					i += 2
				case 't':
					output.WriteByte('\t')
					i += 2
				case 'v':
					output.WriteByte('\v')
					i += 2
				case '\\':
					output.WriteByte('\\')
					i += 2
				case '"':
					output.WriteByte('"')
					i += 2
				case '\'':
					output.WriteByte('\'')
					i += 2
				case 'x': // Handle hex
					if i+2 < len(input) {
						start := i + 2
						end := start
						for end < len(input) && ((input[end] >= '0' && input[end] <= '9') || (input[end] >= 'a' && input[end] <= 'f') || (input[end] >= 'A' && input[end] <= 'F')) && (end-start) < 3 {
							end++
						}
						hex := input[start:end]
						val, err := strconv.ParseInt(hex, 16, 8)
						if err == nil {
							output.WriteByte(byte(val))
						} else {
							// If there's an error, just write the original sequence.
							output.WriteString("\\x" + hex)
						}
						i = end
					} else { // If incomplete hex sequence
						output.WriteString("\\x")
						i += 2
					}
				default:

					if '0' <= input[i+1] && input[i+1] <= '9' {
						start := i + 1
						end := start
						for end < len(input) && '0' <= input[end] && input[end] <= '9' && (end-start) < 4 {
							end++
						}
						octal := input[start:end]
						val, err := strconv.ParseInt(octal, 10, 8)
						if err == nil {
							output.WriteByte(byte(val))
						} else {
							// // If there's an error (shouldn't happen), just write the original sequence.
							// output.WriteString("\\" + octal)
						}
						i = end
					} else { // Escapes of random characters
						output.WriteByte(input[i+1])
						i += 2
					}
				}
			} else { // If '\' is the last character
				output.WriteByte('\\')
				i++
			}
		} else {
			output.WriteByte(input[i])
			i++
		}
	}
	return output.String()
}
