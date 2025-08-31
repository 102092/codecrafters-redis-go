package main

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/handler"
	"github.com/codecrafters-io/redis-starter-go/protocol"
	"github.com/codecrafters-io/redis-starter-go/store"
)

func main() {
	// Redis 서버 시작 로그
	fmt.Println("Starting Redis server on port 6379...")

	// TCP 리스너 생성
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	// 데이터 저장소 생성
	dataStore := store.NewStore()

	// 명령어 핸들러 레지스트리 생성
	// 모든 Redis 명령어들이 여기에 등록됩니다
	registry := handler.NewCommandRegistry(dataStore)

	fmt.Println("Redis server ready to accept connections")

	// 클라이언트 연결 수락 루프
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		// 각 연결을 별도의 고루틴에서 처리
		// 동시에 여러 클라이언트 연결을 처리할 수 있음
		go handleConnection(conn, registry)
	}
}

// handleConnection은 클라이언트 연결을 처리하는 핵심 함수입니다.
// 각 클라이언트 연결마다 별도의 고루틴에서 실행되어 동시성을 지원합니다.
//
// 연결 처리 과정:
//  1. RESP 프로토콜 파서와 라이터 초기화
//  2. 클라이언트 명령어 수신 대기
//  3. 명령어 파싱 및 핸들러로 위임
//  4. 결과를 RESP 형식으로 응답
//  5. 에러 발생 시 연결 종료
//
// 매개변수:
//   - conn: 클라이언트와의 네트워크 연결
//   - registry: 명령어 핸들러 레지스트리
func handleConnection(conn net.Conn, registry *handler.CommandRegistry) {
	// 연결 종료 보장 (defer로 확실히 정리)
	defer conn.Close()

	// RESP 프로토콜 처리를 위한 파서와 라이터 초기화
	reader := bufio.NewReader(conn)
	parser := protocol.NewParser(reader)
	writer := protocol.NewWriter(conn)

	// 클라이언트 명령어 처리 루프
	// 연결이 끊어질 때까지 계속 명령어를 수신하고 처리
	for {
		// RESP 프로토콜로 전송된 명령어 파싱
		value, err := parser.Parse()
		if err != nil {
			// 연결 끊김, 잘못된 프로토콜 등의 에러
			fmt.Printf("Connection error: %v\n", err)
			return
		}

		// 파싱된 데이터가 배열이고 비어있지 않은지 확인
		// Redis 명령어는 항상 배열 형태로 전송됨
		// 예: ["SET", "key", "value"] 또는 ["GET", "key"]
		if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
			// 첫 번째 요소가 명령어 이름
			if cmdName, ok := arr[0].(string); ok {
				// 명령어 인자들 추출 (명령어 이름 제외)
				args := make([]string, 0, len(arr)-1)
				for i := 1; i < len(arr); i++ {
					if arg, ok := arr[i].(string); ok {
						args = append(args, arg)
					}
				}

				// 핸들러 레지스트리를 통해 명령어 실행
				// 각 명령어별 비즈니스 로직은 개별 핸들러에서 처리
				result, err := registry.Execute(cmdName, args)

				if err != nil {
					// 명령어 실행 중 에러 발생
					// Redis 표준 에러 응답 형식으로 전송
					writer.WriteSimpleString(err.Error())
				} else {
					// 명령어 실행 성공: 결과 타입에 따라 적절한 RESP 형식으로 응답
					writeResponse(writer, result)
				}
			} else {
				// 명령어 이름이 문자열이 아닌 경우 (프로토콜 오류)
				writer.WriteSimpleString("-ERR invalid command format")
			}
		} else {
			// 배열이 아니거나 빈 배열인 경우 (프로토콜 오류)
			writer.WriteSimpleString("-ERR invalid request format")
		}
	}
}

// writeResponse는 명령어 실행 결과를 적절한 RESP 형식으로 응답하는 함수입니다.
// Go의 타입 시스템을 활용하여 결과 타입에 따라 올바른 RESP 형식을 선택합니다.
//
// 지원하는 응답 타입:
//   - nil: Null Bulk String ($-1\r\n)
//   - string: Bulk String ($<len>\r\n<data>\r\n) 또는 Simple String (+<data>\r\n)
//   - int: Integer (:<num>\r\n)
//   - []string: Array (*<count>\r\n<elements>...)
//
// 매개변수:
//   - writer: RESP 응답을 작성할 Writer
//   - result: 명령어 실행 결과 (다양한 타입 가능)
func writeResponse(writer *protocol.Writer, result interface{}) {
	switch v := result.(type) {
	case nil:
		// nil 값: Redis의 null 응답 (키가 없는 경우 등)
		writer.WriteBulkString(nil)

	case string:
		// 문자열: 대부분의 값 응답
		// 특별한 응답들은 Simple String으로, 일반 값들은 Bulk String으로 처리
		if v == "OK" || v == "PONG" {
			// 상태 응답은 Simple String으로
			writer.WriteSimpleString(v)
		} else {
			// 일반 값은 Bulk String으로 (바이너리 안전)
			writer.WriteBulkString(&v)
		}

	case int:
		// 정수: RPUSH 등의 반환값
		writer.WriteInteger(v)

	case []string:
		// 문자열 배열: LRANGE 등의 반환값
		writer.WriteArray(v)

	case *handler.NullArray:
		// BLPOP timeout시 null array (*-1\r\n) 응답
		writer.WriteNullArray()

	default:
		// 예상하지 못한 타입: 개발 중 디버깅용
		fmt.Printf("Warning: unexpected result type %T: %v\n", result, result)
		writer.WriteSimpleString("-ERR internal server error")
	}
}
