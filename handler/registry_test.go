package handler

import (
	"strings"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// TestCommandRegistry는 명령어 레지스트리 시스템을 테스트합니다.
func TestCommandRegistry(t *testing.T) {
	dataStore := store.NewStore()
	registry := NewCommandRegistry(dataStore)

	// 테스트 케이스 1: 기본 명령어들이 등록되었는지 확인
	expectedCommands := []string{"PING", "ECHO", "SET", "GET", "RPUSH", "LPUSH", "LRANGE", "LLEN", "LPOP"}
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