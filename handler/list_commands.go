// Package handler는 Redis의 List 타입 명령어들을 구현합니다.
// List는 순서가 있는 문자열들의 컬렉션으로, 양쪽 끝에서 삽입/삭제가 가능합니다.
package handler

import (
	"github.com/codecrafters-io/redis-starter-go/store"
)

// RPushHandler는 RPUSH 명령어를 처리하는 핸들러입니다.
//
// RPUSH 명령어의 역할:
//   - 리스트의 오른쪽 끝(tail)에 하나 이상의 값을 추가
//   - 키가 존재하지 않으면 새 리스트 생성 후 추가
//   - 원자적 연산: 모든 값이 한 번에 추가됨
//
// Redis RPUSH 명령어 사양:
//   - RPUSH key value [value ...] → (integer) 새로운 리스트 길이
//   - 키가 없으면 빈 리스트 생성 후 추가
//   - 리스트가 아닌 타입의 키에 사용하면 에러 (현재 미구현)
//
// 예시:
//
//	RPUSH mylist "hello" → :1\r\n
//	RPUSH mylist "world" "foo" → :3\r\n
//	RPUSH newlist "first" → :1\r\n (새 리스트 생성)
//
// 시간 복잡도:
//   - O(1) - 각 추가되는 요소당
//   - O(N) - N개 요소 추가 시
//
// 공간 복잡도: O(N) - 추가되는 요소 수에 비례
//
// Redis List의 특징:
//   - 중복 값 허용
//   - 순서 보장 (삽입 순서)
//   - 인덱스 접근 가능 (0부터 시작)
//   - 양쪽 끝에서 빠른 삽입/삭제
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

// TODO: 향후 구현할 List 명령어들
//
// LRangeHandler - LRANGE key start stop
//   - 리스트의 지정된 범위 요소들을 반환
//   - 예: LRANGE mylist 0 -1 (모든 요소)
//   - 반환: Array of Bulk Strings
//
// LPushHandler - LPUSH key value [value ...]
//   - 리스트의 왼쪽 끝(head)에 값 추가
//   - RPUSH의 반대 방향
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
