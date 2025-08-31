// Package handler는 Redis 명령어 처리를 위한 핸들러 시스템을 제공합니다.
// 각 명령어를 개별 핸들러로 분리하여 코드의 가독성과 유지보수성을 향상시킵니다.
//
// 핸들러 패턴의 장점:
//   - 단일 책임 원칙: 각 핸들러는 하나의 명령어만 처리
//   - 확장성: 새로운 명령어 추가가 쉬움
//   - 테스트 용이성: 각 핸들러를 독립적으로 테스트 가능
//   - 코드 분리: 거대한 switch문을 작은 단위로 분해
package handler

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// CommandHandler는 모든 Redis 명령어 핸들러가 구현해야 하는 인터페이스입니다.
//
// 인터페이스 설계 원칙:
//   - 단순함: Execute 메서드 하나만 정의
//   - 일관성: 모든 핸들러가 같은 시그니처 사용
//   - 유연성: interface{} 반환으로 다양한 타입 지원
//
// 반환값 타입:
//   - string: Simple String이나 Bulk String으로 응답
//   - int: Integer로 응답
//   - []string: Array로 응답
//   - nil: Null Bulk String으로 응답
//   - error: 에러 응답
type CommandHandler interface {
	// Execute는 명령어를 실행하고 결과를 반환합니다.
	//
	// 매개변수:
	//   - args: 명령어의 인자들 (명령어 이름 제외)
	//           예: "SET key value" → args = ["key", "value"]
	//   - store: 데이터 저장소 인스턴스
	//
	// 반환값:
	//   - interface{}: 명령어 실행 결과 (타입에 따라 적절한 RESP 형식으로 변환됨)
	//   - error: 실행 중 발생한 에러
	Execute(args []string, store *store.Store) (interface{}, error)
}

// CommandRegistry는 명령어와 해당 핸들러를 매핑하고 관리하는 구조체입니다.
//
// 레지스트리 패턴의 장점:
//   - 중앙 집중식 명령어 관리
//   - 런타임에 명령어 등록/해제 가능
//   - 명령어 존재 여부를 쉽게 확인
//   - 대소문자 구분 없이 명령어 처리
type CommandRegistry struct {
	// handlers는 명령어 이름과 핸들러를 매핑하는 맵입니다.
	// 키는 대문자로 정규화되어 저장됩니다. (예: "ping" → "PING")
	handlers map[string]CommandHandler

	// store는 모든 핸들러가 공유하는 데이터 저장소입니다.
	// 각 핸들러 실행 시 전달됩니다.
	store *store.Store
}

// NewCommandRegistry는 새로운 CommandRegistry 인스턴스를 생성하고
// 기본 명령어 핸들러들을 등록합니다.
//
// 초기화 과정:
//  1. 빈 핸들러 맵 생성
//  2. 저장소 인스턴스 설정
//  3. 기본 명령어 핸들러들 등록 (PING, ECHO, SET, GET, RPUSH)
//
// 매개변수:
//   - store: 모든 핸들러가 사용할 데이터 저장소
//
// 반환값:
//   - *CommandRegistry: 설정된 레지스트리 인스턴스
func NewCommandRegistry(store *store.Store) *CommandRegistry {
	registry := &CommandRegistry{
		handlers: make(map[string]CommandHandler),
		store:    store,
	}

	// 기본 명령어 핸들러들 등록
	// 각 핸들러는 해당 명령어의 비즈니스 로직을 캡슐화합니다.
	registry.Register("PING", &PingHandler{})     // 연결 테스트
	registry.Register("ECHO", &EchoHandler{})     // 메시지 에코
	registry.Register("SET", &SetHandler{})       // 키-값 저장
	registry.Register("GET", &GetHandler{})       // 키로 값 조회
	registry.Register("RPUSH", &RPushHandler{})   // 리스트 끝에 추가
	registry.Register("LPUSH", &LPushHandler{})   // 리스트 앞에 추가
	registry.Register("LRANGE", &LRangeHandler{}) // 리스트 범위 조회

	return registry
}

// Register는 새로운 명령어 핸들러를 등록합니다.
//
// 등록 과정:
//  1. 명령어 이름을 대문자로 정규화
//  2. 핸들러 맵에 저장
//  3. 기존 핸들러가 있으면 덮어씀 (업데이트 가능)
//
// 매개변수:
//   - cmd: 명령어 이름 (대소문자 구분 없음)
//   - handler: 해당 명령어를 처리할 핸들러
//
// 사용 예:
//
//	registry.Register("INCR", &IncrHandler{})
//	registry.Register("llen", &LLenHandler{})  // 소문자도 가능
func (r *CommandRegistry) Register(cmd string, handler CommandHandler) {
	// 명령어 이름을 대문자로 정규화하여 대소문자 구분 없이 처리
	r.handlers[strings.ToUpper(cmd)] = handler
}

// Execute는 명령어를 실행합니다.
//
// 실행 과정:
//  1. 명령어 이름을 대문자로 정규화
//  2. 해당 핸들러 검색
//  3. 핸들러가 존재하면 Execute 호출
//  4. 핸들러가 없으면 에러 반환
//
// 매개변수:
//   - cmd: 실행할 명령어 이름
//   - args: 명령어의 인자들
//
// 반환값:
//   - interface{}: 명령어 실행 결과
//   - error: 실행 중 발생한 에러 (알 수 없는 명령어 포함)
//
// 에러 케이스:
//   - 등록되지 않은 명령어
//   - 핸들러 실행 중 발생한 에러
func (r *CommandRegistry) Execute(cmd string, args []string) (interface{}, error) {
	// 명령어 이름 정규화
	cmdUpper := strings.ToUpper(cmd)

	// 등록된 핸들러 검색
	handler, exists := r.handlers[cmdUpper]
	if !exists {
		// Redis 표준 에러 형식 반환
		return nil, &UnknownCommandError{Command: cmd}
	}

	// 핸들러 실행
	return handler.Execute(args, r.store)
}

// HasCommand는 명령어가 등록되어 있는지 확인합니다.
//
// 매개변수:
//   - cmd: 확인할 명령어 이름 (대소문자 구분 없음)
//
// 반환값:
//   - bool: 명령어가 등록되어 있으면 true, 아니면 false
//
// 사용 예:
//
//	if registry.HasCommand("PING") {
//	    // PING 명령어 사용 가능
//	}
func (r *CommandRegistry) HasCommand(cmd string) bool {
	_, exists := r.handlers[strings.ToUpper(cmd)]
	return exists
}

// GetRegisteredCommands는 등록된 모든 명령어 목록을 반환합니다.
//
// 반환값:
//   - []string: 등록된 명령어 이름들 (대문자)
//
// 사용 예:
//
//	commands := registry.GetRegisteredCommands()
//	fmt.Printf("사용 가능한 명령어: %v", commands)
func (r *CommandRegistry) GetRegisteredCommands() []string {
	commands := make([]string, 0, len(r.handlers))
	for cmd := range r.handlers {
		commands = append(commands, cmd)
	}
	return commands
}

// UnknownCommandError는 알 수 없는 명령어에 대한 에러 타입입니다.
// Redis의 표준 에러 응답 형식을 따릅니다.
type UnknownCommandError struct {
	Command string // 실행을 시도한 명령어 이름
}

// Error는 error 인터페이스를 구현합니다.
// Redis 표준 에러 메시지 형식을 반환합니다.
//
// Redis 에러 메시지 형식:
//
//	-ERR unknown command '<명령어>'
//
// 예시:
//
//	-ERR unknown command 'INVALID'
func (e *UnknownCommandError) Error() string {
	return "-ERR unknown command '" + e.Command + "'"
}
