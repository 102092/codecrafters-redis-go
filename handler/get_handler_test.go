package handler

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// TestGetHandler는 GET 명령어 핸들러를 테스트합니다.
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