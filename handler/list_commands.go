// Package handler는 Redis의 List 타입 명령어들을 구현합니다.
// List는 순서가 있는 문자열들의 컬렉션으로, 양쪽 끝에서 삽입/삭제가 가능합니다.
package handler

import (
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// RPushHandler는 RPUSH 명령어를 처리하는 핸들러입니다.
type RPushHandler struct{}

// Execute는 RPUSH 명령어를 실행합니다.
//
// RPUSH 동작 로직:
//  1. 인자 개수 검증 (최소 2개: key, value1, [value2, ...])
//  2. 키와 추가할 값들 추출
//  3. 저장소의 RPUSH 메서드 호출
//  4. 새로운 리스트 길이 반환
//
// 매개변수:
//   - args: 명령어 인자들
//   - args[0]: 리스트 키 이름
//   - args[1:]: 추가할 값들 (1개 이상)
//   - store: 데이터 저장소
//
// 반환값:
//   - interface{}: 새로운 리스트의 길이 (int)
//   - error: 인자가 부족한 경우
//
// 에러 케이스:
//   - 인자가 2개 미만 (키와 최소 하나의 값 필요)
//
// 동작 예시:
//
//	초기 상태: mylist = ["a", "b"]
//	RPUSH mylist "c" "d" 실행
//	결과: mylist = ["a", "b", "c", "d"], 반환값: 4
//
// 빈 리스트 처리:
//
//	초기 상태: newlist 키 없음
//	RPUSH newlist "first" 실행
//	결과: newlist = ["first"], 반환값: 1
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
//
// LRANGE 명령어의 역할:
//   - 리스트의 지정된 범위의 요소들을 조회하여 배열로 반환
//   - 인덱스는 0부터 시작하며 음수 인덱스도 지원
//   - 존재하지 않는 키는 빈 배열 반환
//   - 읽기 전용 연산 (리스트 내용 변경 없음)
//
// Redis LRANGE 명령어 사양:
//   - LRANGE key start stop → *<count>\r\n<elements>
//   - start와 stop은 모두 포함되는 범위
//   - 인덱스가 범위를 벗어나면 유효한 범위로 자동 조정
//
// 인덱스 규칙:
//   - 양수: 0부터 시작 (0=첫번째, 1=두번째, ...)
//   - 음수: 뒤에서부터 (-1=마지막, -2=뒤에서 두번째, ...)
//   - 범위 초과: 자동으로 리스트 범위 내로 조정
//
// 예시:
//
//	리스트 mylist = ["a", "b", "c", "d", "e"]
//	LRANGE mylist 0 2    → ["a", "b", "c"] (인덱스 0, 1, 2)
//	LRANGE mylist 1 -1   → ["b", "c", "d", "e"] (두번째부터 마지막)
//	LRANGE mylist -3 -1  → ["c", "d", "e"] (뒤에서 3개)
//	LRANGE mylist 10 20  → [] (범위 초과로 빈 배열)
//	LRANGE nonexistent 0 10 → [] (존재하지 않는 키)
//
// 시간 복잡도:
//   - O(S+N) - S는 시작 인덱스까지의 오프셋, N은 반환할 요소 수
//   - 대부분의 경우 O(N)으로 간주
//
// 공간 복잡도: O(N) - 반환할 요소 개수에 비례
//
// RESP 프로토콜 응답:
//   - Array 타입으로 응답 (*<count>\r\n)
//   - 각 요소는 Bulk String ($<len>\r\n<data>\r\n)
//   - 빈 배열은 *0\r\n
type LRangeHandler struct{}

// Execute는 LRANGE 명령어를 실행합니다.
//
// LRANGE 동작 로직:
//  1. 인자 개수 검증 (정확히 3개: key, start, stop)
//  2. start와 stop 인덱스를 정수로 파싱
//  3. 저장소의 LRANGE 메서드 호출
//  4. 결과 배열 반환
//
// 인자 검증 과정:
//   - 인자 개수가 3개가 아니면 에러
//   - start와 stop이 정수가 아니면 에러
//   - 인덱스 범위 검증은 Store에서 처리
//
// 매개변수:
//   - args: 명령어 인자들
//   - args[0]: 리스트 키 이름
//   - args[1]: 시작 인덱스 (문자열, 정수로 파싱됨)
//   - args[2]: 끝 인덱스 (문자열, 정수로 파싱됨)
//   - store: 데이터 저장소
//
// 반환값:
//   - interface{}: 요소들의 배열 ([]string)
//   - error: 인자 개수 불일치 또는 인덱스 파싱 실패
//
// 에러 케이스:
//   - 인자가 3개가 아닌 경우 (key, start, stop 필요)
//   - start 또는 stop이 정수가 아닌 경우
//
// 동작 예시:
//
//	리스트 상태: mylist = ["hello", "world", "foo", "bar"]
//	LRANGE mylist 1 2 실행
//	결과: ["world", "foo"], 에러: nil
//
//	존재하지 않는 키:
//	LRANGE nonexistent 0 10 실행
//	결과: [], 에러: nil
//
// Redis 호환성:
//   - Redis와 동일한 인덱스 처리 방식
//   - 음수 인덱스 완벽 지원
//   - 범위 초과 시 자동 조정
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

// Execute는 LPUSH 명령어를 실행합니다.
//
// **LPUSH 동작 로직:**
//  1. 인자 개수 검증 (최소 2개: key, value1, [value2, ...])
//  2. 키와 추가할 값들 분리
//  3. 저장소의 LPUSH 메서드 호출
//  4. 새로운 리스트 길이 반환
//
// **인자 처리 방식:**
//   - args[0]: 리스트 키 이름
//   - args[1:]: 추가할 값들 (순서 중요!)
//   - variadic parameter로 Store.LPUSH에 전달
//
// **매개변수:**
//   - args: 명령어 인자들
//   - args[0]: 리스트 키 이름
//   - args[1:]: 추가할 값들 (1개 이상, 순서 보장)
//   - store: 데이터 저장소 인스턴스
//
// **반환값:**
//   - interface{}: 새로운 리스트의 길이 (int)
//   - error: 인자가 부족한 경우 WrongNumberOfArgumentsError
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

// TODO: 향후 구현할 List 명령어들
//
// LPopHandler - LPOP key
//   - 리스트의 왼쪽 끝에서 요소 제거하고 반환
//   - 스택 또는 큐 구현에 사용
//
// RPopHandler - RPOP key
//   - 리스트의 오른쪽 끝에서 요소 제거하고 반환
//
// LLenHandler - LLEN key
//   - 리스트의 길이 반환
//   - 키가 없으면 0 반환
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
