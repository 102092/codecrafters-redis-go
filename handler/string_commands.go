// Package handler는 Redis의 String 타입 명령어들을 구현합니다.
// String 타입은 Redis의 가장 기본적인 데이터 타입으로 키-값 저장에 사용됩니다.
package handler

import (
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// SetHandler는 SET 명령어를 처리하는 핸들러입니다.
//
// SET 명령어의 역할:
//   - 키에 문자열 값을 저장
//   - 기존 값이 있으면 덮어씀
//   - TTL(Time To Live) 옵션 지원
//
// Redis SET 명령어 사양:
//   - SET key value → OK
//   - SET key value PX milliseconds → OK (만료 시간 설정)
//   - SET key value EX seconds → OK (현재 미구현)
//
// 예시:
//
//	SET mykey "Hello World" → +OK\r\n
//	SET session:123 "user_data" PX 30000 → +OK\r\n (30초 후 만료)
//
// 시간 복잡도: O(1)
// 공간 복잡도: O(1)
type SetHandler struct{}

// Execute는 SET 명령어를 실행합니다.
//
// SET 동작 로직:
//  1. 인자 개수 검증 (최소 2개: key, value)
//  2. 기본 SET: key, value 저장
//  3. 옵션 처리: PX (밀리초 TTL) 지원
//  4. 저장소에 값 저장
//  5. "OK" 응답 반환
//
// 지원하는 인자 패턴:
//   - [key, value]: 기본 SET
//   - [key, value, "PX", milliseconds]: TTL과 함께 SET
//
// 매개변수:
//   - args: 명령어 인자들
//   - args[0]: 키 이름
//   - args[1]: 저장할 값
//   - args[2]: "PX" (선택적)
//   - args[3]: 밀리초 단위 TTL (선택적)
//   - store: 데이터 저장소
//
// 반환값:
//   - interface{}: "OK" 문자열
//   - error: 인자가 부족하거나 잘못된 경우
//
// 에러 케이스:
//   - 인자가 2개 미만
//   - TTL 값이 숫자가 아님
//   - 알 수 없는 옵션
func (h *SetHandler) Execute(args []string, store *store.Store) (interface{}, error) {
	// 최소 인자 개수 검증 (key, value)
	if len(args) < 2 {
		return nil, &WrongNumberOfArgumentsError{Command: "set"}
	}

	key := args[0]
	value := args[1]

	// TTL 옵션 처리
	var ttlMs *int

	// 옵션이 있는 경우 (PX milliseconds)
	if len(args) >= 4 {
		option := strings.ToUpper(args[2])

		switch option {
		case "PX":
			// 밀리초 단위 TTL 파싱
			ms, err := strconv.Atoi(args[3])
			if err != nil {
				return nil, &InvalidArgumentError{
					Message: "value is not an integer or out of range",
				}
			}
			ttlMs = &ms

		default:
			// 지원하지 않는 옵션
			return nil, &InvalidArgumentError{
				Message: "syntax error",
			}
		}
	} else if len(args) == 3 {
		// 인자가 3개인 경우: 잘못된 형식
		return nil, &InvalidArgumentError{
			Message: "syntax error",
		}
	}

	// 저장소에 값 저장
	// TTL이 있으면 만료 시간과 함께, 없으면 영구 저장
	store.SET(key, value, ttlMs)

	// SET 명령어는 항상 "OK" 반환
	return "OK", nil
}

// GetHandler는 GET 명령어를 처리하는 핸들러입니다.
//
// GET 명령어의 역할:
//   - 키에 저장된 문자열 값을 조회
//   - 키가 없으면 null 반환
//   - 만료된 키는 자동으로 삭제되고 null 반환
//
// Redis GET 명령어 사양:
//   - GET key → value (키가 존재하는 경우)
//   - GET key → (nil) (키가 없거나 만료된 경우)
//
// 예시:
//
//	GET mykey → $11\r\nHello World\r\n
//	GET nonexistent → $-1\r\n (null bulk string)
//
// 시간 복잡도: O(1)
// 공간 복잡도: O(1)
type GetHandler struct{}

// Execute는 GET 명령어를 실행합니다.
//
// GET 동작 로직:
//  1. 인자 개수 검증 (정확히 1개: key)
//  2. 저장소에서 키 조회
//  3. 값이 있으면 반환, 없으면 nil 반환
//  4. 만료된 값은 store.GET에서 자동 처리
//
// 매개변수:
//   - args: 명령어 인자들
//   - args[0]: 조회할 키 이름
//   - store: 데이터 저장소
//
// 반환값:
//   - interface{}: 저장된 값 (string) 또는 nil
//   - error: 인자가 잘못된 경우
//
// 에러 케이스:
//   - 인자가 1개가 아닌 경우
//
// 특별한 반환값:
//   - nil: 키가 존재하지 않거나 만료됨 → Null Bulk String ($-1\r\n)
//   - string: 실제 저장된 값 → Bulk String ($<len>\r\n<value>\r\n)
func (h *GetHandler) Execute(args []string, store *store.Store) (interface{}, error) {
	// 정확한 인자 개수 검증
	if len(args) != 1 {
		return nil, &WrongNumberOfArgumentsError{Command: "get"}
	}

	key := args[0]

	// 저장소에서 값 조회
	// store.GET은 만료 확인과 자동 삭제를 수행
	value := store.GET(key)

	// 포인터가 nil이면 키가 없거나 만료됨
	if value == nil {
		return nil, nil // nil 반환 → Null Bulk String
	}

	// 실제 값 반환 → Bulk String
	return *value, nil
}

// InvalidArgumentError는 명령어 인자가 잘못된 경우의 에러입니다.
// 인자 개수는 맞지만 값이나 형식이 잘못된 경우 사용합니다.
type InvalidArgumentError struct {
	Message string // 구체적인 에러 메시지
}

// Error는 error 인터페이스를 구현합니다.
//
// Redis 에러 메시지 형식:
//
//	-ERR <메시지>
//
// 예시:
//
//	-ERR value is not an integer or out of range
//	-ERR syntax error
func (e *InvalidArgumentError) Error() string {
	return "-ERR " + e.Message
}
