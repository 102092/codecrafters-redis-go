// Package handler는 Redis의 List 타입 명령어들을 구현합니다.
// List는 순서가 있는 문자열들의 컬렉션으로, 양쪽 끝에서 삽입/삭제가 가능합니다.
package handler

import (
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// RPushHandler는 RPUSH 명령어를 처리하는 핸들러입니다.
type RPushHandler struct{}

func (h *RPushHandler) Execute(args []string, store *store.Store) (interface{}, error) {
	// 최소 인자 개수 검증 (key + 최소 1개 값)
	if len(args) < 2 {
		return nil, &WrongNumberOfArgumentsError{Command: "rpush"}
	}

	key := args[0]
	values := args[1:] // 첫 번째 인자 이후의 모든 값들

	// 저장소의 RPUSH 메서드 호출
	// variadic 파라미터로 여러 값을 한 번에 전달
	newLength := store.RPUSH(key, values...)

	// 새로운 리스트 길이를 Integer로 반환
	// Redis RPUSH는 항상 정수를 반환함
	return newLength, nil
}

// LRangeHandler는 LRANGE 명령어를 처리하는 핸들러입니다.
type LRangeHandler struct{}

func (h *LRangeHandler) Execute(args []string, store *store.Store) (interface{}, error) {
	// 정확한 인자 개수 검증 (key, start, stop)
	if len(args) != 3 {
		return nil, &WrongNumberOfArgumentsError{Command: "lrange"}
	}

	key := args[0]

	// start 인덱스 파싱
	start, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, &InvalidArgumentError{
			Message: "value is not an integer or out of range",
		}
	}

	// stop 인덱스 파싱
	stop, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, &InvalidArgumentError{
			Message: "value is not an integer or out of range",
		}
	}

	// 저장소에서 지정된 범위의 요소들 조회
	// LRANGE 로직(인덱스 검증, 음수 처리 등)은 Store에서 처리
	elements := store.LRANGE(key, start, stop)

	// 결과 배열 반환
	// []string 타입은 main.go의 writeResponse에서 Array로 변환됨
	return elements, nil
}

// LPushHandler는 LPUSH 명령어를 처리하는 핸들러입니다.
type LPushHandler struct{}

func (h *LPushHandler) Execute(args []string, store *store.Store) (interface{}, error) {
	// 최소 인자 개수 검증 (key + 최소 1개 값)
	// Redis와 동일한 에러 메시지 형식 준수
	if len(args) < 2 {
		return nil, &WrongNumberOfArgumentsError{Command: "lpush"}
	}

	// 키와 값들 분리
	key := args[0]
	values := args[1:] // 슬라이스 참조 (메모리 복사 없음)

	// 저장소의 LPUSH 메서드 호출
	// variadic parameter 패턴으로 모든 값을 한 번에 전달
	// 원자적 연산 보장 (중간 실패 없음)
	newLength := store.LPUSH(key, values...)

	// 새로운 리스트 길이를 Integer로 반환
	// Redis LPUSH는 항상 정수를 반환함 (RESP Integer 타입)
	return newLength, nil
}

// LLenHandler는 LLEN 명령어를 처리하는 핸들러입니다.
type LLenHandler struct{}

func (h *LLenHandler) Execute(args []string, store *store.Store) (interface{}, error) {
	// 정확한 인자 개수 검증 (key 하나만 필요)
	if len(args) != 1 {
		return nil, &WrongNumberOfArgumentsError{Command: "llen"}
	}

	key := args[0]

	// 저장소에서 리스트 길이 조회
	// Store.LLEN은 키가 없으면 0, 있으면 실제 길이 반환
	length := store.LLEN(key)

	// 길이를 Integer로 반환
	// Redis LLEN은 항상 정수를 반환함 (RESP Integer 타입)
	return length, nil
}

// LPopHandler는 LPOP 명령어를 처리하는 핸들러입니다.
type LPopHandler struct{}

// Execute는 LPOP 명령어를 실행합니다.
// Redis 6.2+ 구문: LPOP key [count]
func (h *LPopHandler) Execute(args []string, store *store.Store) (interface{}, error) {
	// 인자 개수 검증 (key 또는 key + count)
	if len(args) < 1 || len(args) > 2 {
		return nil, &WrongNumberOfArgumentsError{Command: "lpop"}
	}

	key := args[0]
	var count *int = nil

	// count 인자가 있는 경우 파싱
	if len(args) == 2 {
		countValue, err := strconv.Atoi(args[1])
		if err != nil {
			return nil, &InvalidArgumentError{
				Message: "value is not an integer or out of range",
			}
		}
		count = &countValue
	}

	// 저장소에서 왼쪽 끝 요소(들) 제거 및 반환
	result := store.LPOP(key, count)

	// count에 따라 반환 타입 처리
	if count == nil {
		// 단일 요소 모드: *string 반환값을 적절히 처리
		if ptr, ok := result.(*string); ok {
			if ptr == nil {
				return nil, nil
			}
			return *ptr, nil
		} else if result == nil {
			return nil, nil
		}
	}

	// count 지정 모드: []string 그대로 반환
	return result, nil
}

// BLPopHandler는 BLPOP 명령어를 처리하는 핸들러입니다.
type BLPopHandler struct{}

// Execute는 BLPOP 명령어를 실행합니다.
// Redis 구문: BLPOP key [key ...] timeout
func (h *BLPopHandler) Execute(args []string, store *store.Store) (interface{}, error) {
	// 인자 개수 검증 (최소 2개: key + timeout)
	if len(args) < 2 {
		return nil, &WrongNumberOfArgumentsError{Command: "blpop"}
	}

	// 마지막 인자는 timeout
	timeoutStr := args[len(args)-1]
	keys := args[:len(args)-1]

	// timeout 파싱
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		return nil, &InvalidArgumentError{
			Message: "timeout is not a float or out of range",
		}
	}

	// timeout이 음수이면 에러
	if timeout < 0 {
		return nil, &InvalidArgumentError{
			Message: "timeout is negative",
		}
	}

	// Store의 blocking BLPOP 메소드 호출
	result := store.BLPOPBlocking(keys, timeout)

	// 결과가 있으면 [key, value] 배열로 반환
	if result != nil {
		return []string{result.Key, result.Value}, nil
	}

	// 타임아웃이 발생하여 nil이 반환됨
	return nil, nil
}

// TODO: 향후 구현할 List 명령어들
//
// RPopHandler - RPOP key
//   - 리스트의 오른쪽 끝에서 요소 제거하고 반환
//
// LIndexHandler - LINDEX key index
//   - 지정된 인덱스의 요소 반환
//   - 음수 인덱스 지원 (-1은 마지막 요소)
//
// 구현 시 고려사항:
//   1. 키가 존재하지 않는 경우 처리
//   2. 키가 List 타입이 아닌 경우 에러 처리
//   3. 인덱스 범위 검증
//   4. 원자적 연산 보장
