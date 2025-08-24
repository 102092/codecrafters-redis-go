package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit
var storage = make(map[string]string) // 전역 변수로 데이터 저장소 선언

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn)
	}
}

func parseRESP(reader *bufio.Reader) (interface{}, error) {
	// 첫 바이트로 타입 판별
	typeByte, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	switch typeByte {
	case '+': // Simple String
		return readSimpleString(reader)
	case '*': // Array
		return readArray(reader)
	case '$': // Bulk String
		return readBulkString(reader)
	default:
		return nil, fmt.Errorf("unknown RESP type: %c", typeByte)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		value, err := parseRESP(reader)
		if err != nil {
			fmt.Println("Error parsing response: ", err.Error())
			return
		}

		if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
			if cmd, ok := arr[0].(string); ok {
				switch strings.ToUpper(cmd) {
				case "PING":
					conn.Write([]byte("+PONG\r\n"))

				case "ECHO":
					if len(arr) > 1 {
						if msg, ok := arr[1].(string); ok {
							conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(msg), msg)))
						}

					}
				case "SET":
					if len(arr) >= 2 {
						key := arr[1].(string)
						value := arr[2].(string)

						storage[key] = value
						conn.Write([]byte("+OK\r\n"))
					}
				case "GET":
					if len(arr) >= 2 {
						key := arr[1].(string)

						value, exists := storage[key]

						if exists {
							conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)))
						} else {
							conn.Write([]byte("$-1\r\n")) // null bulk string
						}
					}
				}
			}
		}
	}
}

// Simple String 읽기: +OK\r\n
func readSimpleString(reader *bufio.Reader) (string, error) {
	line, err := readLine(reader)
	if err != nil {
		return "", err
	}
	return line,
		nil
}

// Array 읽기: *2\r\n$4\r\nPING\r\n$4\r\ntest\r\n
func readArray(reader *bufio.Reader) ([]interface{}, error) {
	line, err := readLine(reader)
	if err != nil {
		return nil, err
	}

	count, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return nil, err
	}

	// Null array
	if count == -1 {
		return nil, nil
	}

	// 각 요소를 재귀적으로 파싱
	result := make([]interface{}, count)
	for i := int64(0); i < count; i++ {
		value, err := parseRESP(reader)
		if err != nil {
			return nil, err
		}
		result[i] = value
	}

	return result, nil
}

// Bulk String 읽기: $6\r\nfoobar\r\n 또는 $-1\r\n (nil)
func readBulkString(reader *bufio.Reader) (interface{}, error) {
	line, err := readLine(reader)
	if err != nil {
		return nil, err
	}

	length, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return nil, err
	}

	// Null bulk string
	if length == -1 {
		return nil, nil
	}

	// 지정된 길이만큼 읽기
	buf := make([]byte, length+2) // +2 for \r\n
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return nil, err
	}

	// \r\n 제거
	return string(buf[:length]), nil
}

// \r\n까지 한 줄 읽기
func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// \r\n 제거
	if len(line) >= 2 && line[len(line)-2] == '\r' {
		return line[:len(line)-2], nil
	}

	return line[:len(line)-1], nil
}
