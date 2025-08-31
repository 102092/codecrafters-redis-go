// Package handler는 PING과 ECHO 명령어 핸들러를 구현합니다.
// 이 두 명령어는 연결 테스트와 메시지 에코에 사용되는 기본적인 명령어들입니다.
package handler

import (
	"github.com/codecrafters-io/redis-starter-go/store"
)

// PingHandler는 PING 명령어를 처리하는 핸들러입니다.
//
// PING 명령어의 역할:
//   - 클라이언트와 서버 간의 연결 상태 확인 (헬스 체크)
//   - 네트워크 지연 측정 (RTT, Round Trip Time)
//   - 서버가 살아있는지 확인 (keepalive)
//
// Redis PING 명령어 사양:
//   - PING (인자 없음) → "PONG" 반환
//   - PING <메시지> → <메시지> 그대로 반환 (Redis 2.8+)
//
// 예시:
//
//	클라이언트: PING
//	서버: +PONG\r\n
//
//	클라이언트: PING "hello world"
//	서버: $11\r\nhello world\r\n
type PingHandler struct{}

// Execute는 PING 명령어를 실행합니다.
//
// PING 동작 로직:
//  1. 인자가 없으면 → "PONG" 문자열 반환 (Simple String)
//  2. 인자가 있으면 → 첫 번째 인자를 그대로 반환 (Bulk String)
//  3. 데이터 저장소는 사용하지 않음 (상태 없는 명령어)
//
// 매개변수:
//   - args: 명령어 인자들
//   - 빈 슬라이스 → 기본 PONG 응답
//   - 1개 이상 → 첫 번째 인자를 에코
//   - store: 사용하지 않음 (nil이어도 무관)
//
// 반환값:
//   - interface{}: "PONG" (string) 또는 에코할 메시지 (string)
//   - error: 항상 nil (PING은 실패할 수 없음)
//
// 성능 특성:
//   - O(1) 시간 복잡도
//   - 메모리 사용량 최소
//   - I/O 없음
func (h *PingHandler) Execute(args []string, store *store.Store) (interface{}, error) {
	// 인자가 없는 경우: 기본 PONG 응답
	if len(args) == 0 {
		return "PONG", nil
	}

	// 인자가 있는 경우: 첫 번째 인자를 에코
	// Redis는 여러 인자가 있어도 첫 번째만 사용
	return args[0], nil
}

// EchoHandler는 ECHO 명령어를 처리하는 핸들러입니다.
//
// ECHO 명령어의 역할:
//   - 클라이언트가 보낸 메시지를 그대로 반환
//   - 네트워크 연결 및 프로토콜 테스트
//   - 디버깅 및 개발 도구로 활용
//
// Redis ECHO 명령어 사양:
//   - ECHO <메시지> → <메시지> 그대로 반환
//   - 인자가 없으면 에러
//   - 여러 인자가 있으면 첫 번째만 사용
//
// 예시:
//
//	클라이언트: ECHO "Hello Redis"
//	서버: $11\r\nHello Redis\r\n
//
// PING과 ECHO의 차이점:
//   - PING: 인자 없어도 OK, 기본값 "PONG"
//   - ECHO: 인자 필수, 기본값 없음
type EchoHandler struct{}

// Execute는 ECHO 명령어를 실행합니다.
//
// ECHO 동작 로직:
//  1. 인자 개수 확인 (최소 1개 필요)
//  2. 첫 번째 인자를 그대로 반환
//  3. 데이터 저장소는 사용하지 않음
//
// 매개변수:
//   - args: 명령어 인자들 (최소 1개 필요)
//   - store: 사용하지 않음
//
// 반환값:
//   - interface{}: 에코할 메시지 (string)
//   - error: 인자가 없으면 에러
//
// 에러 케이스:
//   - 인자가 없는 경우: "wrong number of arguments" 에러
//
// Redis 표준 에러 메시지 형식:
//
//	-ERR wrong number of arguments for 'echo' command
func (h *EchoHandler) Execute(args []string, store *store.Store) (interface{}, error) {
	// 인자 개수 검증
	if len(args) == 0 {
		return nil, &WrongNumberOfArgumentsError{Command: "echo"}
	}

	// 첫 번째 인자를 그대로 반환
	return args[0], nil
}

// WrongNumberOfArgumentsError는 명령어 인자 개수가 잘못된 경우의 에러입니다.
// Redis의 표준 에러 응답 형식을 따릅니다.
type WrongNumberOfArgumentsError struct {
	Command string // 에러가 발생한 명령어 이름
}

// Error는 error 인터페이스를 구현합니다.
// Redis 표준 에러 메시지 형식을 반환합니다.
//
// Redis 에러 메시지 형식:
//
//	-ERR wrong number of arguments for '<명령어>' command
//
// 예시:
//
//	-ERR wrong number of arguments for 'echo' command
//	-ERR wrong number of arguments for 'set' command
func (e *WrongNumberOfArgumentsError) Error() string {
	return "-ERR wrong number of arguments for '" + e.Command + "' command"
}
