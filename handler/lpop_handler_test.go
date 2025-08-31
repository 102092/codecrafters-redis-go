package handler

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// TestLPopHandler는 LPOP 명령어 핸들러를 테스트합니다.
// 단일 요소 제거와 multiple elements 제거 기능을 모두 테스트합니다.
func TestLPopHandler(t *testing.T) {
	handler := &LPopHandler{}
	dataStore := store.NewStore()

	// === 단일 요소 LPOP 테스트 ===

	// 테스트 케이스 1: 존재하지 않는 키
	result, err := handler.Execute([]string{"nonexistent"}, dataStore)
	if err != nil {
		t.Fatalf("LPOP on non-existent key should not fail: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil for non-existent key, got %v", result)
	}

	// 테스트 케이스 2: 단일 요소 리스트
	dataStore.RPUSH("single", "only_one")
	result, err = handler.Execute([]string{"single"}, dataStore)
	if err != nil {
		t.Fatalf("LPOP on single element list failed: %v", err)
	}
	if result != "only_one" {
		t.Errorf("Expected 'only_one', got %v", result)
	}

	// 키가 삭제되었는지 확인
	length := dataStore.LLEN("single")
	if length != 0 {
		t.Errorf("Key should be deleted after popping last element, but LLEN is %d", length)
	}

	// 테스트 케이스 3: 다중 요소 리스트에서 단일 LPOP
	dataStore.RPUSH("multi", "first", "second", "third")
	result, err = handler.Execute([]string{"multi"}, dataStore)
	if err != nil {
		t.Fatalf("LPOP on multi element list failed: %v", err)
	}
	if result != "first" {
		t.Errorf("Expected 'first', got %v", result)
	}

	// 남은 요소들 확인
	remaining := dataStore.LRANGE("multi", 0, -1)
	expected := []string{"second", "third"}
	if !equalStringSlices(remaining, expected) {
		t.Errorf("Expected %v, got %v", expected, remaining)
	}

	// === 다중 요소 LPOP 테스트 (Redis 6.2+ 기능) ===

	// 테스트 케이스 4: LPOP key count (여러 요소 제거)
	dataStore.RPUSH("multicount", "a", "b", "c", "d", "e")
	result, err = handler.Execute([]string{"multicount", "3"}, dataStore)
	if err != nil {
		t.Fatalf("LPOP with count failed: %v", err)
	}

	// 결과는 []string이어야 함
	resultArray, ok := result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}

	expectedArray := []string{"a", "b", "c"}
	if !equalStringSlices(resultArray, expectedArray) {
		t.Errorf("Expected %v, got %v", expectedArray, resultArray)
	}

	// 남은 요소들 확인
	remaining = dataStore.LRANGE("multicount", 0, -1)
	expected = []string{"d", "e"}
	if !equalStringSlices(remaining, expected) {
		t.Errorf("Expected remaining %v, got %v", expected, remaining)
	}

	// 테스트 케이스 5: count가 리스트 길이보다 클 때
	dataStore.RPUSH("overcount", "x", "y")
	result, err = handler.Execute([]string{"overcount", "5"}, dataStore)
	if err != nil {
		t.Fatalf("LPOP with count > length failed: %v", err)
	}

	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}

	expectedArray = []string{"x", "y"}
	if !equalStringSlices(resultArray, expectedArray) {
		t.Errorf("Expected %v, got %v", expectedArray, resultArray)
	}

	// 키가 삭제되었는지 확인
	length = dataStore.LLEN("overcount")
	if length != 0 {
		t.Errorf("Key should be deleted after popping all elements, but LLEN is %d", length)
	}

	// 테스트 케이스 6: 존재하지 않는 키에 count 적용
	result, err = handler.Execute([]string{"nonexistent2", "3"}, dataStore)
	if err != nil {
		t.Fatalf("LPOP with count on non-existent key should not fail: %v", err)
	}

	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}

	if len(resultArray) != 0 {
		t.Errorf("Expected empty array for non-existent key, got %v", resultArray)
	}

	// 테스트 케이스 7: count = 0
	dataStore.RPUSH("zerocount", "a", "b", "c")
	result, err = handler.Execute([]string{"zerocount", "0"}, dataStore)
	if err != nil {
		t.Fatalf("LPOP with count=0 failed: %v", err)
	}

	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}

	if len(resultArray) != 0 {
		t.Errorf("Expected empty array for count=0, got %v", resultArray)
	}

	// 원래 리스트가 변경되지 않았는지 확인
	length = dataStore.LLEN("zerocount")
	if length != 3 {
		t.Errorf("List should be unchanged after count=0, but LLEN is %d", length)
	}

	// 테스트 케이스 8: 음수 count
	result, err = handler.Execute([]string{"zerocount", "-1"}, dataStore)
	if err != nil {
		t.Fatalf("LPOP with negative count failed: %v", err)
	}

	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}

	if len(resultArray) != 0 {
		t.Errorf("Expected empty array for negative count, got %v", resultArray)
	}

	// 테스트 케이스 9: LPUSH 후 LPOP count (스택 동작)
	dataStore.LPUSH("stackcount", "bottom", "middle", "top")
	result, err = handler.Execute([]string{"stackcount", "2"}, dataStore)
	if err != nil {
		t.Fatalf("LPOP with count after LPUSH failed: %v", err)
	}

	resultArray, ok = result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}

	expectedArray = []string{"top", "middle"}
	if !equalStringSlices(resultArray, expectedArray) {
		t.Errorf("Expected %v, got %v", expectedArray, resultArray)
	}

	// === 에러 테스트 ===

	// 테스트 케이스 10: 잘못된 인자 개수 (인자 없음)
	result, err = handler.Execute([]string{}, dataStore)
	if err == nil {
		t.Fatal("Expected error for no arguments")
	}

	if _, ok := err.(*WrongNumberOfArgumentsError); !ok {
		t.Errorf("Expected WrongNumberOfArgumentsError, got %T", err)
	}

	// 테스트 케이스 11: 잘못된 인자 개수 (인자 과다)
	result, err = handler.Execute([]string{"key", "count", "extra"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for too many arguments")
	}

	if _, ok := err.(*WrongNumberOfArgumentsError); !ok {
		t.Errorf("Expected WrongNumberOfArgumentsError, got %T", err)
	}

	// 테스트 케이스 12: 잘못된 count 값 (문자열)
	result, err = handler.Execute([]string{"key", "invalid"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for invalid count")
	}

	if _, ok := err.(*InvalidArgumentError); !ok {
		t.Errorf("Expected InvalidArgumentError, got %T", err)
	}
}