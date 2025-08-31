package handler

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// TestBLPopHandler는 BLPOP 명령어 핸들러를 테스트합니다.
// BLPOP은 blocking left pop으로, 여러 키에서 값을 pop하는 명령어입니다.
func TestBLPopHandler(t *testing.T) {
	handler := &BLPopHandler{}
	dataStore := store.NewStore()

	// === 기본 BLPOP 테스트 ===

	// 테스트 케이스 1: 단일 키에서 BLPOP (값이 있는 경우)
	dataStore.RPUSH("key1", "value1", "value2")
	
	result, err := handler.Execute([]string{"key1", "0"}, dataStore)
	if err != nil {
		t.Fatalf("BLPOP on existing list failed: %v", err)
	}
	
	// 결과는 [key, value] 배열이어야 함
	resultArray, ok := result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}
	
	if len(resultArray) != 2 {
		t.Fatalf("Expected 2-element array, got %d elements", len(resultArray))
	}
	
	if resultArray[0] != "key1" || resultArray[1] != "value1" {
		t.Errorf("Expected [key1, value1], got %v", resultArray)
	}
	
	// 남은 요소 확인
	remaining := dataStore.LRANGE("key1", 0, -1)
	expected := []string{"value2"}
	if !equalStringSlices(remaining, expected) {
		t.Errorf("Expected remaining %v, got %v", expected, remaining)
	}

	// 테스트 케이스 2: 존재하지 않는 키에서 BLPOP (1초 타임아웃)
	result, err = handler.Execute([]string{"nonexistent", "1"}, dataStore)
	if err != nil {
		t.Fatalf("BLPOP on non-existent key should not fail: %v", err)
	}
	
	if _, ok := result.(*NullArray); !ok {
		t.Errorf("Expected NullArray for non-existent key, got %v", result)
	}

	// 테스트 케이스 3: 빈 리스트에서 BLPOP
	dataStore.RPUSH("empty_list", "temp")
	dataStore.LPOP("empty_list", nil) // 리스트를 비움
	
	result, err = handler.Execute([]string{"empty_list", "1"}, dataStore)
	if err != nil {
		t.Fatalf("BLPOP on empty list should not fail: %v", err)
	}
	
	if _, ok := result.(*NullArray); !ok {
		t.Errorf("Expected NullArray for empty list, got %v", result)
	}

	// === 다중 키 BLPOP 테스트 ===

	// 테스트 케이스 4: 여러 키 중 첫 번째 키에 값이 있는 경우
	dataStore.RPUSH("first", "first_value")
	dataStore.RPUSH("second", "second_value")
	
	result, err = handler.Execute([]string{"first", "second", "0"}, dataStore)
	if err != nil {
		t.Fatalf("BLPOP with multiple keys failed: %v", err)
	}
	
	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}
	
	if resultArray[0] != "first" || resultArray[1] != "first_value" {
		t.Errorf("Expected [first, first_value], got %v", resultArray)
	}

	// 테스트 케이스 5: 여러 키 중 두 번째 키에만 값이 있는 경우
	// first 키는 이미 비어있음
	result, err = handler.Execute([]string{"first", "second", "0"}, dataStore)
	if err != nil {
		t.Fatalf("BLPOP with multiple keys (second has value) failed: %v", err)
	}
	
	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}
	
	if resultArray[0] != "second" || resultArray[1] != "second_value" {
		t.Errorf("Expected [second, second_value], got %v", resultArray)
	}

	// 테스트 케이스 6: 여러 키 모두 비어있는 경우
	result, err = handler.Execute([]string{"first", "second", "third", "1"}, dataStore)
	if err != nil {
		t.Fatalf("BLPOP with all empty keys should not fail: %v", err)
	}
	
	if _, ok := result.(*NullArray); !ok {
		t.Errorf("Expected NullArray when all keys are empty, got %v", result)
	}

	// 테스트 케이스 7: 키 순서 우선순위 테스트
	dataStore.RPUSH("priority1", "p1_value")
	dataStore.RPUSH("priority2", "p2_value")
	dataStore.RPUSH("priority3", "p3_value")
	
	// priority2, priority1, priority3 순서로 요청 (priority2가 먼저 처리되어야 함)
	result, err = handler.Execute([]string{"priority2", "priority1", "priority3", "0"}, dataStore)
	if err != nil {
		t.Fatalf("BLPOP priority test failed: %v", err)
	}
	
	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}
	
	if resultArray[0] != "priority2" || resultArray[1] != "p2_value" {
		t.Errorf("Expected priority2 to be processed first, got %v", resultArray)
	}

	// === 타임아웃 파라미터 테스트 ===

	// 테스트 케이스 8: 다양한 타임아웃 값
	dataStore.RPUSH("timeout_test", "timeout_value")
	
	// timeout = 1
	result, err = handler.Execute([]string{"timeout_test", "1"}, dataStore)
	if err != nil {
		t.Fatalf("BLPOP with timeout=1 failed: %v", err)
	}
	
	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}
	
	if resultArray[0] != "timeout_test" || resultArray[1] != "timeout_value" {
		t.Errorf("Expected [timeout_test, timeout_value], got %v", resultArray)
	}

	// 테스트 케이스 9: timeout = 1 (1초 대기)
	result, err = handler.Execute([]string{"empty_timeout", "1"}, dataStore)
	if err != nil {
		t.Fatalf("BLPOP with timeout=0 should not fail: %v", err)
	}
	
	if _, ok := result.(*NullArray); !ok {
		t.Errorf("Expected NullArray for timeout on empty key, got %v", result)
	}

	// === 에러 케이스 테스트 ===

	// 테스트 케이스 10: 인자 부족 (키만 있고 타임아웃 없음)
	result, err = handler.Execute([]string{"key1"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for insufficient arguments")
	}
	
	if _, ok := err.(*WrongNumberOfArgumentsError); !ok {
		t.Errorf("Expected WrongNumberOfArgumentsError, got %T", err)
	}

	// 테스트 케이스 11: 인자 없음
	result, err = handler.Execute([]string{}, dataStore)
	if err == nil {
		t.Fatal("Expected error for no arguments")
	}
	
	if _, ok := err.(*WrongNumberOfArgumentsError); !ok {
		t.Errorf("Expected WrongNumberOfArgumentsError, got %T", err)
	}

	// 테스트 케이스 12: 잘못된 타임아웃 값 (문자열)
	result, err = handler.Execute([]string{"key1", "invalid_timeout"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for invalid timeout")
	}
	
	if _, ok := err.(*InvalidArgumentError); !ok {
		t.Errorf("Expected InvalidArgumentError, got %T", err)
	}

	// 테스트 케이스 13: 음수 타임아웃
	result, err = handler.Execute([]string{"key1", "-1"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for negative timeout")
	}
	
	if _, ok := err.(*InvalidArgumentError); !ok {
		t.Errorf("Expected InvalidArgumentError, got %T", err)
	}

	// === 실제 사용 케이스 테스트 ===

	// 테스트 케이스 14: 작업 큐 시나리오
	// 여러 큐를 모니터링하다가 작업이 들어오면 처리
	dataStore.RPUSH("urgent_queue", "urgent_task1")
	dataStore.RPUSH("normal_queue", "normal_task1")
	dataStore.RPUSH("low_queue", "low_task1")
	
	// 우선순위 순서로 큐 확인: urgent -> normal -> low
	result, err = handler.Execute([]string{"urgent_queue", "normal_queue", "low_queue", "5"}, dataStore)
	if err != nil {
		t.Fatalf("Work queue scenario failed: %v", err)
	}
	
	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}
	
	// urgent_queue에서 작업이 나와야 함
	if resultArray[0] != "urgent_queue" || resultArray[1] != "urgent_task1" {
		t.Errorf("Expected urgent task to be processed first, got %v", resultArray)
	}

	// 테스트 케이스 15: 대용량 키 목록
	// 많은 키를 모니터링하는 경우
	var keys []string
	for i := 0; i < 50; i++ {
		keyName := "bulk_key_" + string(rune('A'+i%26)) // A-Z 반복
		keys = append(keys, keyName)
	}
	
	// 마지막 키에만 값 추가
	lastKey := keys[len(keys)-1]
	dataStore.RPUSH(lastKey, "bulk_value")
	keys = append(keys, "10") // timeout 추가
	
	result, err = handler.Execute(keys, dataStore)
	if err != nil {
		t.Fatalf("Bulk keys test failed: %v", err)
	}
	
	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}
	
	if resultArray[0] != lastKey || resultArray[1] != "bulk_value" {
		t.Errorf("Expected [%s, bulk_value], got %v", lastKey, resultArray)
	}

	// 테스트 케이스 16: 빈 문자열 값 처리
	dataStore.RPUSH("empty_value_test", "", "non_empty")
	
	result, err = handler.Execute([]string{"empty_value_test", "0"}, dataStore)
	if err != nil {
		t.Fatalf("BLPOP with empty string value failed: %v", err)
	}
	
	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}
	
	if resultArray[0] != "empty_value_test" || resultArray[1] != "" {
		t.Errorf("Expected [empty_value_test, ''], got %v", resultArray)
	}
	
	// 두 번째 값 확인
	result, err = handler.Execute([]string{"empty_value_test", "0"}, dataStore)
	if err != nil {
		t.Fatalf("BLPOP second value failed: %v", err)
	}
	
	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}
	
	if resultArray[0] != "empty_value_test" || resultArray[1] != "non_empty" {
		t.Errorf("Expected [empty_value_test, non_empty], got %v", resultArray)
	}
}