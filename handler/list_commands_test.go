package handler

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// TestRPushHandler는 RPUSH 명령어 핸들러를 테스트합니다.
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

// TestLPushHandler는 LPUSH 명령어 핸들러를 테스트합니다.
func TestLPushHandler(t *testing.T) {
	handler := &LPushHandler{}
	dataStore := store.NewStore()

	// 테스트 케이스 1: 새 리스트에 단일 값 추가
	result, err := handler.Execute([]string{"newlist", "first"}, dataStore)
	if err != nil {
		t.Fatalf("LPUSH on new list failed: %v", err)
	}
	if result != 1 {
		t.Errorf("Expected length 1, got %v", result)
	}

	// 실제 저장된 값 검증
	actualList := dataStore.LRANGE("newlist", 0, -1)
	expected := []string{"first"}
	if !equalStringSlices(actualList, expected) {
		t.Errorf("Expected %v, got %v", expected, actualList)
	}

	// 테스트 케이스 2: 기존 리스트에 값 추가
	result, err = handler.Execute([]string{"newlist", "second"}, dataStore)
	if err != nil {
		t.Fatalf("LPUSH to existing list failed: %v", err)
	}
	if result != 2 {
		t.Errorf("Expected length 2, got %v", result)
	}

	// 순서 확인: "second"가 앞에 와야 함
	actualList = dataStore.LRANGE("newlist", 0, -1)
	expected = []string{"second", "first"}
	if !equalStringSlices(actualList, expected) {
		t.Errorf("Expected %v, got %v", expected, actualList)
	}

	// 테스트 케이스 3: 다중 값 추가
	result, err = handler.Execute([]string{"multilist", "a", "b", "c"}, dataStore)
	if err != nil {
		t.Fatalf("LPUSH with multiple values failed: %v", err)
	}
	if result != 3 {
		t.Errorf("Expected length 3, got %v", result)
	}

	// Redis LPUSH의 실제 동작: 역순!
	actualList = dataStore.LRANGE("multilist", 0, -1)
	expected = []string{"c", "b", "a"}
	if !equalStringSlices(actualList, expected) {
		t.Errorf("Expected %v, got %v", expected, actualList)
	}

	// 테스트 케이스 4: 에러 케이스
	result, err = handler.Execute([]string{"onlykey"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for insufficient args")
	}

	if _, ok := err.(*WrongNumberOfArgumentsError); !ok {
		t.Errorf("Expected WrongNumberOfArgumentsError, got %T", err)
	}

	result, err = handler.Execute([]string{}, dataStore)
	if err == nil {
		t.Fatal("Expected error for no args")
	}

	// 테스트 케이스 5: RPUSH와 LPUSH 비교
	rpushHandler := &RPushHandler{}
	rpushHandler.Execute([]string{"rpush_test", "1", "2", "3"}, dataStore)
	rpushResult := dataStore.LRANGE("rpush_test", 0, -1)

	handler.Execute([]string{"lpush_test", "1", "2", "3"}, dataStore)
	lpushResult := dataStore.LRANGE("lpush_test", 0, -1)

	if equalStringSlices(rpushResult, lpushResult) {
		t.Error("RPUSH and LPUSH should produce different results")
	}

	expectedRpush := []string{"1", "2", "3"}
	expectedLpush := []string{"3", "2", "1"}

	if !equalStringSlices(rpushResult, expectedRpush) {
		t.Errorf("RPUSH result: expected %v, got %v", expectedRpush, rpushResult)
	}
	if !equalStringSlices(lpushResult, expectedLpush) {
		t.Errorf("LPUSH result: expected %v, got %v", expectedLpush, lpushResult)
	}
}

// TestLRangeHandler는 LRANGE 명령어 핸들러를 테스트합니다.
func TestLRangeHandler(t *testing.T) {
	handler := &LRangeHandler{}
	dataStore := store.NewStore()

	// 테스트 데이터 준비
	dataStore.RPUSH("testlist", "first", "second", "third", "fourth", "fifth")

	// 테스트 케이스 1: 기본 범위 조회
	result, err := handler.Execute([]string{"testlist", "0", "2"}, dataStore)
	if err != nil {
		t.Fatalf("LRANGE 0 2 failed: %v", err)
	}

	expected := []string{"first", "second", "third"}
	if !equalStringSlices(result.([]string), expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}

	// 테스트 케이스 2: 음수 인덱스
	result, err = handler.Execute([]string{"testlist", "-3", "-1"}, dataStore)
	if err != nil {
		t.Fatalf("LRANGE -3 -1 failed: %v", err)
	}

	expected = []string{"third", "fourth", "fifth"}
	if !equalStringSlices(result.([]string), expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}

	// 테스트 케이스 3: 전체 리스트
	result, err = handler.Execute([]string{"testlist", "0", "-1"}, dataStore)
	if err != nil {
		t.Fatalf("LRANGE 0 -1 failed: %v", err)
	}

	expected = []string{"first", "second", "third", "fourth", "fifth"}
	if !equalStringSlices(result.([]string), expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}

	// 테스트 케이스 4: 범위 초과
	result, err = handler.Execute([]string{"testlist", "10", "20"}, dataStore)
	if err != nil {
		t.Fatalf("LRANGE 10 20 failed: %v", err)
	}

	if len(result.([]string)) != 0 {
		t.Errorf("Expected empty slice, got %v", result)
	}

	// 테스트 케이스 5: 존재하지 않는 키
	result, err = handler.Execute([]string{"nonexistent", "0", "10"}, dataStore)
	if err != nil {
		t.Fatalf("LRANGE on non-existent key failed: %v", err)
	}

	if len(result.([]string)) != 0 {
		t.Errorf("Expected empty slice for non-existent key, got %v", result)
	}

	// 테스트 케이스 6: 에러 케이스
	result, err = handler.Execute([]string{"testlist", "0"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for insufficient args")
	}

	result, err = handler.Execute([]string{"testlist", "0", "1", "2"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for too many args")
	}

	result, err = handler.Execute([]string{"testlist", "notanumber", "1"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for invalid start index")
	}

	result, err = handler.Execute([]string{"testlist", "0", "notanumber"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for invalid stop index")
	}

	// 테스트 케이스 7: 역순 인덱스
	result, err = handler.Execute([]string{"testlist", "3", "1"}, dataStore)
	if err != nil {
		t.Fatalf("LRANGE 3 1 failed: %v", err)
	}

	if len(result.([]string)) != 0 {
		t.Errorf("Expected empty slice for reversed range, got %v", result)
	}

	// 테스트 케이스 8: 단일 요소 조회
	result, err = handler.Execute([]string{"testlist", "2", "2"}, dataStore)
	if err != nil {
		t.Fatalf("LRANGE 2 2 failed: %v", err)
	}

	expected = []string{"third"}
	if !equalStringSlices(result.([]string), expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// TestLLenHandler는 LLEN 명령어 핸들러를 테스트합니다.
func TestLLenHandler(t *testing.T) {
	handler := &LLenHandler{}
	dataStore := store.NewStore()

	// 테스트 케이스 1: 존재하지 않는 키
	result, err := handler.Execute([]string{"nonexistent"}, dataStore)
	if err != nil {
		t.Fatalf("LLEN on non-existent key should not fail: %v", err)
	}
	if result != 0 {
		t.Errorf("Expected 0 for non-existent key, got %v", result)
	}

	// 테스트 케이스 2: 단일 요소 리스트
	dataStore.RPUSH("single", "only_one")
	result, err = handler.Execute([]string{"single"}, dataStore)
	if err != nil {
		t.Fatalf("LLEN on single element list failed: %v", err)
	}
	if result != 1 {
		t.Errorf("Expected 1 for single element list, got %v", result)
	}

	// 테스트 케이스 3: 다중 요소 리스트
	dataStore.RPUSH("multi", "a", "b", "c", "d", "e")
	result, err = handler.Execute([]string{"multi"}, dataStore)
	if err != nil {
		t.Fatalf("LLEN on multi element list failed: %v", err)
	}
	if result != 5 {
		t.Errorf("Expected 5 for multi element list, got %v", result)
	}

	// 테스트 케이스 4: 동적 리스트 변화
	dynamicKey := "dynamic"
	result, _ = handler.Execute([]string{dynamicKey}, dataStore)
	if result != 0 {
		t.Errorf("Initial state should be 0, got %v", result)
	}

	dataStore.RPUSH(dynamicKey, "item1")
	result, _ = handler.Execute([]string{dynamicKey}, dataStore)
	if result != 1 {
		t.Errorf("After 1 RPUSH should be 1, got %v", result)
	}

	dataStore.RPUSH(dynamicKey, "item2", "item3")
	result, _ = handler.Execute([]string{dynamicKey}, dataStore)
	if result != 3 {
		t.Errorf("After adding 2 more should be 3, got %v", result)
	}

	dataStore.LPUSH(dynamicKey, "front1", "front2")
	result, _ = handler.Execute([]string{dynamicKey}, dataStore)
	if result != 5 {
		t.Errorf("After LPUSH 2 more should be 5, got %v", result)
	}

	// 테스트 케이스 5: 에러 케이스
	result, err = handler.Execute([]string{}, dataStore)
	if err == nil {
		t.Fatal("Expected error for no arguments")
	}

	if _, ok := err.(*WrongNumberOfArgumentsError); !ok {
		t.Errorf("Expected WrongNumberOfArgumentsError, got %T", err)
	}

	result, err = handler.Execute([]string{"key1", "key2"}, dataStore)
	if err == nil {
		t.Fatal("Expected error for too many arguments")
	}

	// 테스트 케이스 6: 대용량 리스트
	largeKey := "large"
	expectedSize := 1000
	for i := 0; i < expectedSize; i++ {
		dataStore.RPUSH(largeKey, "item")
	}

	result, err = handler.Execute([]string{largeKey}, dataStore)
	if err != nil {
		t.Fatalf("LLEN on large list failed: %v", err)
	}
	if result != expectedSize {
		t.Errorf("Expected %d for large list, got %v", expectedSize, result)
	}
}