package handler

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// TestSetHandler는 SET 명령어 핸들러를 테스트합니다.
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