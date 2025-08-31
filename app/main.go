package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/protocol"
	"github.com/codecrafters-io/redis-starter-go/store"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	// Create a single store instance for all connections
	dataStore := store.NewStore()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn, dataStore)
	}
}

func handleConnection(conn net.Conn, dataStore *store.Store) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	parser := protocol.NewParser(reader)
	writer := protocol.NewWriter(conn)

	for {
		value, err := parser.Parse()
		if err != nil {
			fmt.Println("Error parsing response: ", err.Error())
			return
		}

		if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
			if cmd, ok := arr[0].(string); ok {
				switch strings.ToUpper(cmd) {
				case "PING":
					writer.WritePONG()

				case "ECHO":
					if len(arr) > 1 {
						if msg, ok := arr[1].(string); ok {
							writer.WriteBulkString(&msg)
						}
					}
				case "SET":
					if len(arr) >= 5 {
						key := arr[1].(string)
						value := arr[2].(string)
						unit := arr[3].(string)
						expiry := arr[4].(string)

						if strings.ToUpper(unit) == "PX" {
							px, _ := strconv.Atoi(expiry)
							dataStore.SET(key, value, &px)
							writer.WriteOK()
							continue
						}
					} else if len(arr) >= 2 {
						key := arr[1].(string)
						value := arr[2].(string)

						dataStore.SET(key, value, nil)
						writer.WriteOK()
					}
				case "GET":
					if len(arr) >= 2 {
						key := arr[1].(string)
						value := dataStore.GET(key)

						writer.WriteBulkString(value)
					}
				case "RPUSH":
					if len(arr) >= 3 {
						key := arr[1].(string)
						values := make([]string, 0, len(arr)-2)

						for i := 2; i < len(arr); i++ {
							if val, ok := arr[i].(string); ok {
								values = append(values, val)
							}
						}

						length := dataStore.RPUSH(key, values...)
						writer.WriteInteger(length)
					}
				}
			}
		}
	}
}
