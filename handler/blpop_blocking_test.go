package handler

import (
	"sync"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/store"
)

// TestBLPopBlocking은 실제 blocking 동작을 테스트합니다.
func TestBLPopBlocking(t *testing.T) {
	handler := &BLPopHandler{}
	dataStore := store.NewStore()

	// === 실제 blocking 테스트 ===

	// 테스트 케이스 1: 짧은 타임아웃 후 값 추가
	t.Run("ShortTimeoutWithValueAdded", func(t *testing.T) {
		var wg sync.WaitGroup
		var result interface{}
		var err error

		// 고루틴에서 BLPOP 실행 (1초 타임아웃)
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err = handler.Execute([]string{"test_key", "1"}, dataStore)
		}()

		// 잠시 대기 후 값 추가
		time.Sleep(200 * time.Millisecond)
		dataStore.RPUSH("test_key", "test_value")

		// 결과 대기
		wg.Wait()

		if err != nil {
			t.Fatalf("BLPOP should not fail: %v", err)
		}

		resultArray, ok := result.([]string)
		if !ok {
			t.Fatalf("Expected []string result, got %T", result)
		}

		if len(resultArray) != 2 || resultArray[0] != "test_key" || resultArray[1] != "test_value" {
			t.Errorf("Expected [test_key, test_value], got %v", resultArray)
		}
	})

	// 테스트 케이스 2: 타임아웃 발생
	t.Run("TimeoutOccurs", func(t *testing.T) {
		start := time.Now()
		result, err := handler.Execute([]string{"empty_key", "1"}, dataStore)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("BLPOP should not fail on timeout: %v", err)
		}

		if result != nil {
			t.Errorf("Expected nil result on timeout, got %v", result)
		}

		// 대략 1초 정도 걸려야 함 (오차 허용)
		if duration < 900*time.Millisecond || duration > 1200*time.Millisecond {
			t.Errorf("Expected ~1s timeout, got %v", duration)
		}
	})

	// 테스트 케이스 3: 여러 클라이언트가 같은 키를 대기
	t.Run("MultipleWaitersOnSameKey", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]interface{}, 3)
		errors := make([]error, 3)

		// 3개의 고루틴이 같은 키를 대기
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				results[index], errors[index] = handler.Execute([]string{"multi_wait", "2"}, dataStore)
			}(i)
		}

		// 잠시 대기 후 값 하나만 추가
		time.Sleep(200 * time.Millisecond)
		dataStore.RPUSH("multi_wait", "shared_value")

		// 결과 대기
		wg.Wait()

		// 하나의 고루틴만 값을 받아야 하고, 나머지는 타임아웃
		successCount := 0
		timeoutCount := 0

		for i := 0; i < 3; i++ {
			if errors[i] != nil {
				t.Fatalf("BLPOP %d should not fail: %v", i, errors[i])
			}

			if results[i] != nil {
				successCount++
				resultArray, ok := results[i].([]string)
				if !ok {
					t.Fatalf("Expected []string result, got %T", results[i])
				}
				if len(resultArray) != 2 || resultArray[0] != "multi_wait" || resultArray[1] != "shared_value" {
					t.Errorf("Expected [multi_wait, shared_value], got %v", resultArray)
				}
			} else {
				timeoutCount++
			}
		}

		if successCount != 1 {
			t.Errorf("Expected exactly 1 success, got %d", successCount)
		}

		if timeoutCount != 2 {
			t.Errorf("Expected exactly 2 timeouts, got %d", timeoutCount)
		}
	})

	// 테스트 케이스 4: 여러 키를 모니터링하다가 하나에 값 추가
	t.Run("MultipleKeysOneGetsValue", func(t *testing.T) {
		var wg sync.WaitGroup
		var result interface{}
		var err error

		// key1, key2, key3을 모니터링
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err = handler.Execute([]string{"key1", "key2", "key3", "2"}, dataStore)
		}()

		// key2에 값 추가
		time.Sleep(200 * time.Millisecond)
		dataStore.RPUSH("key2", "key2_value")

		wg.Wait()

		if err != nil {
			t.Fatalf("BLPOP should not fail: %v", err)
		}

		resultArray, ok := result.([]string)
		if !ok {
			t.Fatalf("Expected []string result, got %T", result)
		}

		if len(resultArray) != 2 || resultArray[0] != "key2" || resultArray[1] != "key2_value" {
			t.Errorf("Expected [key2, key2_value], got %v", resultArray)
		}
	})

	// 테스트 케이스 5: 순서 우선순위 테스트 (blocking 환경에서)
	t.Run("KeyPriorityInBlocking", func(t *testing.T) {
		var wg sync.WaitGroup
		var result interface{}
		var err error

		// priority_low, priority_high 순서로 모니터링
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err = handler.Execute([]string{"priority_low", "priority_high", "2"}, dataStore)
		}()

		// 두 키에 동시에 값 추가 (low가 먼저 추가되지만 실제 처리는 우선순위 순서)
		time.Sleep(200 * time.Millisecond)
		dataStore.RPUSH("priority_low", "low_value")
		dataStore.RPUSH("priority_high", "high_value")

		wg.Wait()

		if err != nil {
			t.Fatalf("BLPOP should not fail: %v", err)
		}

		resultArray, ok := result.([]string)
		if !ok {
			t.Fatalf("Expected []string result, got %T", result)
		}

		// priority_low가 먼저 지정되었으므로 low_value가 반환되어야 함
		if len(resultArray) != 2 || resultArray[0] != "priority_low" || resultArray[1] != "low_value" {
			t.Errorf("Expected [priority_low, low_value], got %v", resultArray)
		}
	})

	// 테스트 케이스 6: LPUSH로 값 추가 시 알림
	t.Run("LPUSHTriggersWaiters", func(t *testing.T) {
		var wg sync.WaitGroup
		var result interface{}
		var err error

		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err = handler.Execute([]string{"lpush_test", "2"}, dataStore)
		}()

		// LPUSH로 값 추가
		time.Sleep(200 * time.Millisecond)
		dataStore.LPUSH("lpush_test", "lpush_value")

		wg.Wait()

		if err != nil {
			t.Fatalf("BLPOP should not fail: %v", err)
		}

		resultArray, ok := result.([]string)
		if !ok {
			t.Fatalf("Expected []string result, got %T", result)
		}

		if len(resultArray) != 2 || resultArray[0] != "lpush_test" || resultArray[1] != "lpush_value" {
			t.Errorf("Expected [lpush_test, lpush_value], got %v", resultArray)
		}
	})

	// 테스트 케이스 7: 연속된 blocking 요청
	t.Run("ConsecutiveBlockingRequests", func(t *testing.T) {
		// 첫 번째 요청
		var wg1 sync.WaitGroup
		var result1 interface{}
		var err1 error

		wg1.Add(1)
		go func() {
			defer wg1.Done()
			result1, err1 = handler.Execute([]string{"consecutive", "2"}, dataStore)
		}()

		time.Sleep(100 * time.Millisecond)
		dataStore.RPUSH("consecutive", "first_value")
		wg1.Wait()

		if err1 != nil {
			t.Fatalf("First BLPOP should not fail: %v", err1)
		}

		resultArray1, ok := result1.([]string)
		if !ok {
			t.Fatalf("Expected []string result, got %T", result1)
		}

		if len(resultArray1) != 2 || resultArray1[1] != "first_value" {
			t.Errorf("Expected first_value, got %v", resultArray1)
		}

		// 두 번째 요청 (바로 이어서)
		var wg2 sync.WaitGroup
		var result2 interface{}
		var err2 error

		wg2.Add(1)
		go func() {
			defer wg2.Done()
			result2, err2 = handler.Execute([]string{"consecutive", "2"}, dataStore)
		}()

		time.Sleep(100 * time.Millisecond)
		dataStore.RPUSH("consecutive", "second_value")
		wg2.Wait()

		if err2 != nil {
			t.Fatalf("Second BLPOP should not fail: %v", err2)
		}

		resultArray2, ok := result2.([]string)
		if !ok {
			t.Fatalf("Expected []string result, got %T", result2)
		}

		if len(resultArray2) != 2 || resultArray2[1] != "second_value" {
			t.Errorf("Expected second_value, got %v", resultArray2)
		}
	})
}

// TestBLPopInfiniteWait는 timeout=0 (무한 대기) 모드를 테스트합니다.
func TestBLPopInfiniteWait(t *testing.T) {
	handler := &BLPopHandler{}
	dataStore := store.NewStore()

	// 테스트 케이스 1: timeout=0, 값이 있는 경우 (즉시 반환)
	dataStore.RPUSH("immediate", "immediate_value")
	
	start := time.Now()
	result, err := handler.Execute([]string{"immediate", "0"}, dataStore)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("BLPOP should not fail: %v", err)
	}

	resultArray, ok := result.([]string)
	if !ok {
		t.Fatalf("Expected []string result, got %T", result)
	}

	if len(resultArray) != 2 || resultArray[0] != "immediate" || resultArray[1] != "immediate_value" {
		t.Errorf("Expected [immediate, immediate_value], got %v", resultArray)
	}

	// 값이 있으면 즉시 반환되어야 함
	if duration > 100*time.Millisecond {
		t.Errorf("Expected immediate return, took %v", duration)
	}

	// 테스트 케이스 2: timeout=0, 무한 대기 후 값 추가
	t.Run("InfiniteWaitWithValueAdded", func(t *testing.T) {
		var wg sync.WaitGroup
		var result interface{}
		var err error

		// 고루틴에서 BLPOP 실행 (무한 대기)
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err = handler.Execute([]string{"infinite_wait", "0"}, dataStore)
		}()

		// 잠시 대기 후 값 추가
		time.Sleep(200 * time.Millisecond)
		dataStore.RPUSH("infinite_wait", "infinite_value")

		// 결과 대기
		wg.Wait()

		if err != nil {
			t.Fatalf("BLPOP should not fail: %v", err)
		}

		resultArray, ok := result.([]string)
		if !ok {
			t.Fatalf("Expected []string result, got %T", result)
		}

		if len(resultArray) != 2 || resultArray[0] != "infinite_wait" || resultArray[1] != "infinite_value" {
			t.Errorf("Expected [infinite_wait, infinite_value], got %v", resultArray)
		}
	})
}