package handler

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// TestPingHandler는 PING 명령어 핸들러를 테스트합니다.
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