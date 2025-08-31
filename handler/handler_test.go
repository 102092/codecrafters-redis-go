// Package handler_test는 핸들러 시스템에 대한 종합적인 테스트를 제공합니다.
// Go의 권장 테스트 패턴을 따르며, 각 핸들러의 동작을 검증합니다.
package handler

import (
	"strings"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// TestPingHandler는 PING 명령어 핸들러를 테스트합니다.
//
// 테스트하는 케이스:
//  1. 인자 없는 PING → "PONG" 반환
//  2. 메시지와 함께하는 PING → 메시지 에코
//
// PING 명령어의 특징:
//   - 실패할 수 없는 명령어 (항상 성공)
//   - 데이터 저장소를 사용하지 않음
//   - 연결 상태 확인용
func TestPingHandler(t *testing.T) {
	handler := &PingHandler{}
	store := store.NewStore()

	// 테스트 케이스 1: 인자 없는 PING
	result, err := handler.Execute([]string{}, store)
	if err != nil {
		t.Fatalf("PING without args failed: %v", err)
	}
	if result != "PONG" {
		t.Errorf("Expected 'PONG', got %v", result)
	}

	// 테스트 케이스 2: 메시지와 함께하는 PING
	message := "hello world"
	result, err = handler.Execute([]string{message}, store)
	if err != nil {
		t.Fatalf("PING with message failed: %v", err)
	}
	if result != message {
		t.Errorf("Expected %q, got %v", message, result)
	}

	// 테스트 케이스 3: 여러 인자가 있는 경우 (첫 번째만 사용)
	result, err = handler.Execute([]string{"first", "second", "third"}, store)
	if err != nil {
		t.Fatalf("PING with multiple args failed: %v", err)
	}
	if result != "first" {
		t.Errorf("Expected 'first', got %v", result)
	}
}

// TestEchoHandler는 ECHO 명령어 핸들러를 테스트합니다.
//
// 테스트하는 케이스:
//  1. 정상적인 ECHO → 메시지 에코
//  2. 인자 없는 ECHO → 에러 반환
//  3. 여러 인자가 있는 ECHO → 첫 번째 인자만 에코
//
// ECHO와 PING의 차이점:
//   - ECHO는 인자가 필수
//   - PING은 인자가 선택적
func TestEchoHandler(t *testing.T) {
	handler := &EchoHandler{}
	store := store.NewStore()

	// 테스트 케이스 1: 정상적인 ECHO
	message := "Hello Redis"
	result, err := handler.Execute([]string{message}, store)
	if err != nil {
		t.Fatalf("ECHO failed: %v", err)
	}
	if result != message {
		t.Errorf("Expected %q, got %v", message, result)
	}

	// 테스트 케이스 2: 인자 없는 ECHO (에러 케이스)
	result, err = handler.Execute([]string{}, store)
	if err == nil {
		t.Fatal("Expected error for ECHO without args")
	}

	// 에러 메시지 검증
	expectedError := "wrong number of arguments for 'echo' command"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}

	// 테스트 케이스 3: 여러 인자 (첫 번째만 사용)
	result, err = handler.Execute([]string{"first", "second"}, store)
	if err != nil {
		t.Fatalf("ECHO with multiple args failed: %v", err)
	}
	if result != "first" {
		t.Errorf("Expected 'first', got %v", result)
	}
}

// TestSetHandler는 SET 명령어 핸들러를 테스트합니다.
//
// 테스트하는 케이스:
//  1. 기본 SET 명령어
//  2. TTL이 있는 SET 명령어 (PX 옵션)
//  3. 잘못된 인자 개수
//  4. 잘못된 TTL 값
//
// SET 명령어의 특징:
//   - 항상 "OK" 반환 (성공 시)
//   - 기존 값 덮어쓰기
//   - TTL 옵션 지원
func TestSetHandler(t *testing.T) {
	handler := &SetHandler{}
	dataStore := store.NewStore()

	// 테스트 케이스 1: 기본 SET
	result, err := handler.Execute([]string{"mykey", "myvalue"}, dataStore)
	if err != nil {
		t.Fatalf("SET failed: %v", err)
	}
	if result != "OK" {
		t.Errorf("Expected 'OK', got %v", result)
	}

	// 저장이 되었는지 확인
	value := dataStore.GET("mykey")
	if value == nil || *value != "myvalue" {
		t.Errorf("Value not stored correctly, got %v", value)
	}

	// 테스트 케이스 2: TTL이 있는 SET
	result, err = handler.Execute([]string{"tempkey", "tempvalue", "PX", "1000"}, dataStore)
	if err != nil {
		t.Fatalf("SET with TTL failed: %v", err)
	}
	if result != "OK" {
		t.Errorf("Expected 'OK', got %v", result)
	}

	// 테스트 케이스 3: 인자 부족 (에러 케이스)
	result, err = handler.Execute([]string{"onlykey"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for insufficient args")
	}

	// 테스트 케이스 4: 잘못된 TTL 값 (에러 케이스)
	result, err = handler.Execute([]string{"key", "value", "PX", "notanumber"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for invalid TTL")
	}

	// 테스트 케이스 5: 알 수 없는 옵션 (에러 케이스)
	result, err = handler.Execute([]string{"key", "value", "UNKNOWN", "123"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for unknown option")
	}
}

// TestGetHandler는 GET 명령어 핸들러를 테스트합니다.
//
// 테스트하는 케이스:
//  1. 존재하는 키 조회
//  2. 존재하지 않는 키 조회
//  3. 잘못된 인자 개수
//
// GET 명령어의 특징:
//   - 값이 있으면 문자열 반환
//   - 값이 없으면 nil 반환
//   - 만료된 값은 자동 삭제
func TestGetHandler(t *testing.T) {
	handler := &GetHandler{}
	dataStore := store.NewStore()

	// 테스트 데이터 설정
	dataStore.SET("existingkey", "existingvalue", nil)

	// 테스트 케이스 1: 존재하는 키 조회
	result, err := handler.Execute([]string{"existingkey"}, dataStore)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	if result != "existingvalue" {
		t.Errorf("Expected 'existingvalue', got %v", result)
	}

	// 테스트 케이스 2: 존재하지 않는 키 조회
	result, err = handler.Execute([]string{"nonexistentkey"}, dataStore)
	if err != nil {
		t.Fatalf("GET for non-existent key failed: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}

	// 테스트 케이스 3: 인자 부족 (에러 케이스)
	result, err = handler.Execute([]string{}, dataStore)
	if err == nil {
		t.Fatal("Expected error for no args")
	}

	// 테스트 케이스 4: 인자 과다 (에러 케이스)
	result, err = handler.Execute([]string{"key1", "key2"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for too many args")
	}
}

// TestRPushHandler는 RPUSH 명령어 핸들러를 테스트합니다.
//
// 테스트하는 케이스:
//  1. 새 리스트에 단일 값 추가
//  2. 기존 리스트에 여러 값 추가
//  3. 잘못된 인자 개수
//
// RPUSH 명령어의 특징:
//   - 리스트 길이를 정수로 반환
//   - 키가 없으면 새 리스트 생성
//   - 여러 값을 한 번에 추가 가능
func TestRPushHandler(t *testing.T) {
	handler := &RPushHandler{}
	dataStore := store.NewStore()

	// 테스트 케이스 1: 새 리스트에 단일 값 추가
	result, err := handler.Execute([]string{"newlist", "first"}, dataStore)
	if err != nil {
		t.Fatalf("RPUSH failed: %v", err)
	}
	if result != 1 {
		t.Errorf("Expected length 1, got %v", result)
	}

	// 테스트 케이스 2: 기존 리스트에 여러 값 추가
	result, err = handler.Execute([]string{"newlist", "second", "third"}, dataStore)
	if err != nil {
		t.Fatalf("RPUSH with multiple values failed: %v", err)
	}
	if result != 3 {
		t.Errorf("Expected length 3, got %v", result)
	}

	// 테스트 케이스 3: 인자 부족 (에러 케이스)
	result, err = handler.Execute([]string{"onlykey"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for insufficient args")
	}

	// 테스트 케이스 4: 빈 인자 (에러 케이스)
	result, err = handler.Execute([]string{}, dataStore)
	if err == nil {
		t.Fatal("Expected error for no args")
	}
}

// TestCommandRegistry는 명령어 레지스트리 시스템을 테스트합니다.
//
// 테스트하는 케이스:
//  1. 레지스트리 초기화
//  2. 명령어 등록 및 실행
//  3. 대소문자 구분 없는 명령어 처리
//  4. 알 수 없는 명령어 처리
//  5. 등록된 명령어 목록 조회
//
// 레지스트리의 특징:
//   - 중앙 집중식 명령어 관리
//   - 대소문자 구분 없음
//   - 런타임 명령어 등록 가능
func TestCommandRegistry(t *testing.T) {
	dataStore := store.NewStore()
	registry := NewCommandRegistry(dataStore)

	// 테스트 케이스 1: 기본 명령어들이 등록되었는지 확인
	expectedCommands := []string{"PING", "ECHO", "SET", "GET", "RPUSH"}
	for _, cmd := range expectedCommands {
		if !registry.HasCommand(cmd) {
			t.Errorf("Command %s not registered", cmd)
		}
	}

	// 테스트 케이스 2: 명령어 실행 (대문자)
	result, err := registry.Execute("PING", []string{})
	if err != nil {
		t.Fatalf("PING execution failed: %v", err)
	}
	if result != "PONG" {
		t.Errorf("Expected 'PONG', got %v", result)
	}

	// 테스트 케이스 3: 명령어 실행 (소문자) - 대소문자 구분 없음
	result, err = registry.Execute("ping", []string{})
	if err != nil {
		t.Fatalf("ping (lowercase) execution failed: %v", err)
	}
	if result != "PONG" {
		t.Errorf("Expected 'PONG', got %v", result)
	}

	// 테스트 케이스 4: 혼합 케이스
	result, err = registry.Execute("PiNg", []string{})
	if err != nil {
		t.Fatalf("PiNg (mixed case) execution failed: %v", err)
	}
	if result != "PONG" {
		t.Errorf("Expected 'PONG', got %v", result)
	}

	// 테스트 케이스 5: 알 수 없는 명령어 (에러 케이스)
	result, err = registry.Execute("UNKNOWN", []string{})
	if err == nil {
		t.Fatal("Expected error for unknown command")
	}

	// UnknownCommandError 타입 확인
	if _, ok := err.(*UnknownCommandError); !ok {
		t.Errorf("Expected UnknownCommandError, got %T", err)
	}

	// 에러 메시지 확인
	expectedError := "unknown command 'UNKNOWN'"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}

	// 테스트 케이스 6: 새로운 핸들러 등록
	registry.Register("CUSTOM", &PingHandler{}) // 테스트용으로 PingHandler 재사용

	if !registry.HasCommand("CUSTOM") {
		t.Error("Custom command not registered")
	}

	result, err = registry.Execute("CUSTOM", []string{})
	if err != nil {
		t.Fatalf("Custom command execution failed: %v", err)
	}

	// 테스트 케이스 7: 등록된 명령어 목록 조회
	commands := registry.GetRegisteredCommands()
	if len(commands) < len(expectedCommands) {
		t.Errorf("Expected at least %d commands, got %d", len(expectedCommands), len(commands))
	}

	// CUSTOM 명령어가 목록에 있는지 확인
	found := false
	for _, cmd := range commands {
		if cmd == "CUSTOM" {
			found = true
			break
		}
	}
	if !found {
		t.Error("CUSTOM command not in registered commands list")
	}
}

// TestErrorTypes는 다양한 에러 타입들을 테스트합니다.
//
// 테스트하는 에러 타입:
//  1. UnknownCommandError
//  2. WrongNumberOfArgumentsError
//  3. InvalidArgumentError
//
// 에러 메시지는 Redis 표준 형식을 따릅니다.
func TestErrorTypes(t *testing.T) {
	// 테스트 케이스 1: UnknownCommandError
	unknownErr := &UnknownCommandError{Command: "BADCMD"}
	expectedMsg := "-ERR unknown command 'BADCMD'"
	if unknownErr.Error() != expectedMsg {
		t.Errorf("Expected %q, got %q", expectedMsg, unknownErr.Error())
	}

	// 테스트 케이스 2: WrongNumberOfArgumentsError
	wrongArgsErr := &WrongNumberOfArgumentsError{Command: "set"}
	expectedMsg = "-ERR wrong number of arguments for 'set' command"
	if wrongArgsErr.Error() != expectedMsg {
		t.Errorf("Expected %q, got %q", expectedMsg, wrongArgsErr.Error())
	}

	// 테스트 케이스 3: InvalidArgumentError
	invalidArgErr := &InvalidArgumentError{Message: "syntax error"}
	expectedMsg = "-ERR syntax error"
	if invalidArgErr.Error() != expectedMsg {
		t.Errorf("Expected %q, got %q", expectedMsg, invalidArgErr.Error())
	}
}

// 벤치마크 테스트들

// BenchmarkPingHandler는 PING 핸들러의 성능을 측정합니다.
// PING은 가장 빠른 명령어 중 하나이므로 기준점으로 사용됩니다.
func BenchmarkPingHandler(b *testing.B) {
	handler := &PingHandler{}
	store := store.NewStore()
	args := []string{}

	// 벤치마크 루프
	for i := 0; i < b.N; i++ {
		_, _ = handler.Execute(args, store)
	}
}

// BenchmarkCommandRegistry는 레지스트리를 통한 명령어 실행 성능을 측정합니다.
// 실제 사용 시나리오와 유사한 벤치마크입니다.
func BenchmarkCommandRegistry(b *testing.B) {
	dataStore := store.NewStore()
	registry := NewCommandRegistry(dataStore)

	// 벤치마크 루프
	for i := 0; i < b.N; i++ {
		_, _ = registry.Execute("PING", []string{})
	}
}
