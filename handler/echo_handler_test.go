package handler

import (
	"strings"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// TestEchoHandler는 ECHO 명령어 핸들러를 테스트합니다.
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