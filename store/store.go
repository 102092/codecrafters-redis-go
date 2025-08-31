package store

import (
	"sync"
	"time"
)

// ValueWithTTL represents a value with an expiration time
type ValueWithTTL struct {
	Value    string
	ExpireAt time.Time
}

// BlockingWaiter represents a client waiting for a blocking operation
type BlockingWaiter struct {
	Keys     []string           // Keys this waiter is monitoring
	Response chan *BLPopResult  // Channel to send result
	Timeout  time.Duration      // How long to wait
	StartTime time.Time         // When the wait started
}

// Store manages key-value storage with optional TTL support
type Store struct {
	storage       map[string]string       // Regular key-value storage
	expireStorage map[string]ValueWithTTL // Storage with TTL
	listStorage   map[string][]string     // List storage
	
	// Blocking operation support
	mu            sync.RWMutex                    // Protects all blocking operations
	waiters       map[string][]*BlockingWaiter   // Key -> list of waiters
	waiterCleanup chan *BlockingWaiter           // Channel for cleanup
}

// NewStore creates a new Store instance
func NewStore() *Store {
	store := &Store{
		storage:       make(map[string]string),
		expireStorage: make(map[string]ValueWithTTL),
		listStorage:   make(map[string][]string),
		waiters:       make(map[string][]*BlockingWaiter),
		waiterCleanup: make(chan *BlockingWaiter, 100),
	}
	
	// Start cleanup goroutine for expired waiters
	go store.cleanupWaiters()
	
	return store
}

// SET implements Redis SET command
// Supports both regular SET and SET with PX (milliseconds expiry)
func (s *Store) SET(key, value string, px *int) { // TODO handle different time unit
	if px != nil {
		// SET with expiry
		expireAt := time.Now().Add(time.Duration(*px) * time.Millisecond)
		s.expireStorage[key] = ValueWithTTL{
			Value:    value,
			ExpireAt: expireAt,
		}
		// Remove from regular storage if exists
		delete(s.storage, key)
	} else {
		// Regular SET without expiry
		s.storage[key] = value
		// Remove from expire storage if exists
		delete(s.expireStorage, key)
	}
}

// GET implements Redis GET command
// Returns nil if key doesn't exist or has expired
func (s *Store) GET(key string) *string {
	// Check expire storage first
	if obj, exists := s.expireStorage[key]; exists {
		now := time.Now()
		if obj.ExpireAt.Before(now) {
			// Key has expired, delete it
			delete(s.expireStorage, key)
			return nil
		}
		return &obj.Value
	}

	// Check regular storage
	if value, exists := s.storage[key]; exists {
		return &value
	}

	// Key not found
	return nil
}

// RPUSH는 Redis RPUSH 명령어를 구현합니다.
// 리스트의 오른쪽 끝(뒤쪽)에 하나 이상의 값을 추가합니다.
//
// 동작 방식:
//   - 키가 없으면 새로운 리스트 생성 후 값 추가
//   - 키가 있으면 기존 리스트 끝에 값 추가
//   - 여러 값을 한 번에 추가 가능
//
// 매개변수:
//   - key: 리스트 키
//   - values: 추가할 값들 (가변 인자)
//
// 반환값:
//   - int: 추가 후 리스트의 총 길이
//
// 시간 복잡도: O(N) (N은 추가할 값의 개수)
func (s *Store) RPUSH(key string, values ...string) int {
	list, exists := s.listStorage[key]
	if !exists {
		list = []string{}
	}

	list = append(list, values...)
	s.listStorage[key] = list

	// 새 값이 추가되었으므로 대기 중인 클라이언트들에게 알림
	s.notifyWaiters(key)

	return len(list)
}

// LRANGE는 Redis LRANGE 명령어를 구현합니다.
// 리스트의 지정된 범위의 요소들을 조회합니다.
//
// 인덱스 규칙:
//   - 0부터 시작 (첫 번째 요소가 인덱스 0)
//   - 음수 인덱스 지원 (-1은 마지막 요소, -2는 뒤에서 두 번째)
//   - 범위를 벗어난 인덱스는 자동으로 조정됨
//
// 매개변수:
//   - key: 조회할 리스트의 키
//   - start: 시작 인덱스 (포함)
//   - stop: 끝 인덱스 (포함)
//
// 반환값:
//   - []string: 지정된 범위의 요소들 (빈 슬라이스 가능)
//
// 예시:
//   - LRANGE mylist 0 2   → 인덱스 0, 1, 2 요소들
//   - LRANGE mylist 1 -1  → 인덱스 1부터 마지막까지
//   - LRANGE mylist -3 -1 → 뒤에서 3번째부터 마지막까지
//
// 시간 복잡도: O(S+N) (S는 시작 위치까지의 오프셋, N은 반환할 요소 수)
func (s *Store) LRANGE(key string, start, stop int) []string {
	// 키가 존재하지 않으면 빈 슬라이스 반환
	list, exists := s.listStorage[key]
	if !exists {
		return []string{}
	}

	// 리스트가 비어있으면 빈 슬라이스 반환
	length := len(list)
	if length == 0 {
		return []string{}
	}

	// 음수 인덱스를 양수로 변환
	// -1은 length-1 (마지막 요소), -2는 length-2 등
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// 인덱스가 범위를 벗어났을 때 조정
	if start < 0 {
		start = 0 // 리스트 시작으로 조정
	}

	if start >= length {
		return []string{} // 시작점이 리스트 끝을 넘어서면 빈 결과
	}

	if stop >= length {
		stop = length - 1 // 리스트 끝으로 조정
	}

	if stop < start {
		return []string{} // stop이 start보다 앞에 있으면 빈 결과
	}

	// 범위에 해당하는 부분 슬라이스 반환
	// Go 슬라이스는 [start:stop+1] 형태로 사용 (stop+1은 제외)
	return list[start : stop+1]
}

// LPUSH는 Redis LPUSH 명령어를 구현합니다.
// 리스트의 왼쪽 끝(head)에 하나 이상의 값을 추가합니다.
//
// 동작 방식:
//   - 키가 없으면 새로운 리스트 생성 후 값 추가
//   - 키가 있으면 기존 리스트 앞쪽에 값 추가 (prepend)
//   - 여러 값을 한 번에 추가 가능 (원자적 연산)
//
// 매개변수:
//   - key: 리스트 키
//   - values: 추가할 값들 (가변 인자, 왼쪽부터 순서대로 추가)
//
// 반환값:
//   - int: 추가 후 리스트의 총 길이
//
// 예시:
//
//	초기: []
//	LPUSH key "a" "b" "c" → ["a", "b", "c"] (길이: 3)
//	LPUSH key "d" → ["d", "a", "b", "c"] (길이: 4)
//
// 시간 복잡도: O(N+M) (N=기존 크기, M=추가할 요소 수)
// 공간 복잡도: O(N+M) (새 슬라이스 할당)
func (s *Store) LPUSH(key string, values ...string) int {
	// 기존 리스트 조회 (없으면 빈 슬라이스)
	existingList, exists := s.listStorage[key]
	if !exists {
		existingList = []string{}
	}

	// Redis LPUSH key "a" "b" "c"의 실제 동작:
	//   1. "a" 추가 → [...기존요소들, "a"]
	//   2. "b" 추가 (앞쪽에) → ["b", ...기존요소들, "a"]
	//   3. "c" 추가 (앞쪽에) → ["c", "b", ...기존요소들, "a"]
	//
	// 따라서 values를 역순으로 하나씩 앞에 추가해야 함

	// 새로운 슬라이스 생성 (capacity 최적화)
	newLength := len(values) + len(existingList)
	newList := make([]string, 0, newLength)

	// values를 역순으로 추가
	for i := len(values) - 1; i >= 0; i-- {
		newList = append(newList, values[i])
	}

	// 기존 요소들을 뒤에 추가
	newList = append(newList, existingList...)

	// 저장소 업데이트
	s.listStorage[key] = newList

	// 새 값이 추가되었으므로 대기 중인 클라이언트들에게 알림
	s.notifyWaiters(key)

	return newLength
}

// LLEN은 Redis LLEN 명령어를 구현합니다.
// 리스트의 길이(요소 개수)를 반환합니다.
//
// 동작 방식:
//   - 키가 존재하지 않으면 0 반환 (Redis 표준 동작)
//   - 키가 존재하면 리스트의 요소 개수 반환
//   - 빈 리스트도 0 반환
//
// 매개변수:
//   - key: 길이를 조회할 리스트 키
//
// 반환값:
//   - int: 리스트의 길이 (0 이상의 정수)
//
// 예시:
//   - 키가 없음 → 0
//   - 빈 리스트 [] → 0
//   - ["a", "b", "c"] → 3
//
// 시간 복잡도: O(1)
// 공간 복잡도: O(1) (추가 메모리 할당 없음)
func (s *Store) LLEN(key string) int {
	// 리스트 존재 여부 확인
	list, exists := s.listStorage[key]
	if !exists {
		// 키가 존재하지 않으면 0 반환 (Redis 표준 동작)
		return 0
	}

	return len(list)
}

// LPOP은 Redis LPOP 명령어를 구현합니다.
// 리스트의 왼쪽 끝(head)에서 요소를 제거하고 반환합니다.
//
// 매개변수:
//   - key: 리스트 키
//   - count: 제거할 요소 개수 (옵셔널, nil이면 1개)
//
// 반환값:
//   - interface{}: count에 따라 *string 또는 []string 반환
//   - count가 nil: *string (단일 요소 또는 nil)
//   - count가 지정됨: []string (빈 배열 가능)
//
// 예시:
//   - LPOP key → "a" (단일 요소)
//   - LPOP key 2 → ["a", "b"] (여러 요소)
//   - LPOP key 10 → ["a", "b", "c"] (count > 길이일 때 모든 요소)
//
// 시간 복잡도: O(N) (N=제거할 요소 개수)
// 공간 복잡도: O(N) (새 슬라이스 할당)
func (s *Store) LPOP(key string, count *int) interface{} {
	// 리스트 존재 여부 확인
	list, exists := s.listStorage[key]
	if !exists {
		// 키가 존재하지 않는 경우
		if count == nil {
			return nil // 단일 요소 모드: nil 반환
		}
		return []string{} // 다중 요소 모드: 빈 배열 반환
	}

	// 빈 리스트인 경우
	if len(list) == 0 {
		if count == nil {
			return nil // 단일 요소 모드: nil 반환
		}
		return []string{} // 다중 요소 모드: 빈 배열 반환
	}

	// count가 nil이면 단일 요소 제거 (기존 동작)
	if count == nil {
		firstElement := list[0]

		// 리스트에 요소가 하나뿐이면 키를 완전히 삭제
		if len(list) == 1 {
			delete(s.listStorage, key)
			return &firstElement
		}

		// 첫 번째 요소를 제외한 나머지로 새 슬라이스 생성
		newList := make([]string, len(list)-1)
		copy(newList, list[1:])
		s.listStorage[key] = newList

		return &firstElement
	}

	// count가 지정된 경우 (다중 요소 제거)
	actualCount := *count

	// count가 0 이하인 경우 빈 배열 반환
	if actualCount <= 0 {
		return []string{}
	}

	// 실제 제거할 요소 개수 결정 (리스트 길이와 count 중 작은 값)
	removeCount := actualCount
	if removeCount > len(list) {
		removeCount = len(list)
	}

	// 제거할 요소들 추출
	removedElements := make([]string, removeCount)
	copy(removedElements, list[:removeCount])

	// 리스트에서 모든 요소를 제거하는 경우 키 삭제
	if removeCount >= len(list) {
		delete(s.listStorage, key)
		return removedElements
	}

	// 일부 요소만 제거하는 경우 나머지 요소들로 새 슬라이스 생성
	remainingElements := make([]string, len(list)-removeCount)
	copy(remainingElements, list[removeCount:])
	s.listStorage[key] = remainingElements

	return removedElements
}

// BLPopResult는 BLPOP 명령어의 반환 결과를 나타냅니다.
type BLPopResult struct {
	Key   string // 값이 제거된 키
	Value string // 제거된 값
}

// BLPOP은 Redis BLPOP 명령어를 구현합니다.
// 여러 키에서 왼쪽 끝 요소를 blocking 방식으로 제거하고 반환합니다.
//
// Redis BLPOP 동작 방식:
//   - 키들을 순서대로 확인하여 비어있지 않은 첫 번째 리스트에서 요소 제거
//   - 모든 키가 비어있거나 존재하지 않으면 nil 반환 (non-blocking 모드)
//   - 반환값: BLPopResult{Key: "키이름", Value: "제거된값"} 또는 nil
//
// 매개변수:
//   - keys: 확인할 키들의 목록
//
// 반환값:
//   - *BLPopResult: 제거된 키와 값 (nil이면 모든 리스트가 비어있음)
//
// 예시:
//   - BLPOP key1 key2 key3 → key1에서 값 제거: {Key: "key1", Value: "value"}
//   - 모든 키가 비어있음 → nil
//
// 시간 복잡도: O(N) (N=확인할 키의 개수)
// 공간 복잡도: O(1) (결과 구조체만 할당)
//
// 참고: 현재는 non-blocking 모드로 구현됨. 
// 실제 blocking 기능은 handler 레이어에서 구현됩니다.
func (s *Store) BLPOP(keys []string) *BLPopResult {
	// 키들을 순서대로 확인
	for _, key := range keys {
		// 각 키에 대해 LPOP 시도 (count = nil로 단일 요소 제거)
		result := s.LPOP(key, nil)
		
		// nil이 아니면 값이 있다는 의미
		if result != nil {
			// LPOP은 count가 nil일 때 *string을 반환
			if valuePtr, ok := result.(*string); ok && valuePtr != nil {
				return &BLPopResult{
					Key:   key,
					Value: *valuePtr,
				}
			}
		}
	}
	
	// 모든 키가 비어있거나 존재하지 않음
	return nil
}

// cleanupWaiters는 만료된 대기자들을 정리하는 고루틴입니다.
func (s *Store) cleanupWaiters() {
	for waiter := range s.waiterCleanup {
		s.mu.Lock()
		// Remove waiter from all keys it was monitoring
		for _, key := range waiter.Keys {
			waiters := s.waiters[key]
			for i, w := range waiters {
				if w == waiter {
					// Remove from slice
					s.waiters[key] = append(waiters[:i], waiters[i+1:]...)
					break
				}
			}
			// Clean up empty waiter lists
			if len(s.waiters[key]) == 0 {
				delete(s.waiters, key)
			}
		}
		s.mu.Unlock()
		
		// Close the response channel to signal timeout
		close(waiter.Response)
	}
}

// notifyWaiters는 키에 새 값이 추가되었을 때 대기자들에게 알림을 보냅니다.
func (s *Store) notifyWaiters(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	waiters := s.waiters[key]
	if len(waiters) == 0 {
		return
	}
	
	// FIFO: 가장 먼저 대기한 waiter가 값을 받음
	waiter := waiters[0]
	
	// Remove this waiter from ALL keys it was waiting for
	for _, waitKey := range waiter.Keys {
		keyWaiters := s.waiters[waitKey]
		for i, w := range keyWaiters {
			if w == waiter {
				s.waiters[waitKey] = append(keyWaiters[:i], keyWaiters[i+1:]...)
				break
			}
		}
		// Clean up empty waiter list
		if len(s.waiters[waitKey]) == 0 {
			delete(s.waiters, waitKey)
		}
	}
	
	// Try to get a value respecting the waiter's original key priority
	result := s.BLPOP(waiter.Keys)
	if result != nil {
		// Send the result
		select {
		case waiter.Response <- result:
			// Success
		default:
			// Channel might be closed, ignore
		}
		
		// Remove this waiter from other keys it was monitoring
		for _, otherKey := range waiter.Keys {
			if otherKey == key {
				continue
			}
			otherWaiters := s.waiters[otherKey]
			for i, w := range otherWaiters {
				if w == waiter {
					s.waiters[otherKey] = append(otherWaiters[:i], otherWaiters[i+1:]...)
					break
				}
			}
			if len(s.waiters[otherKey]) == 0 {
				delete(s.waiters, otherKey)
			}
		}
	}
}

// BLPOPBlocking은 실제 blocking 기능을 가진 BLPOP을 구현합니다.
func (s *Store) BLPOPBlocking(keys []string, timeoutSeconds float64) *BLPopResult {
	// 먼저 non-blocking으로 시도
	result := s.BLPOP(keys)
	if result != nil {
		return result
	}
	
	// timeout 설정 (0이면 무한 대기)
	var timeout time.Duration
	var useTimeout bool
	if timeoutSeconds > 0 {
		timeout = time.Duration(timeoutSeconds * float64(time.Second))
		useTimeout = true
	}
	
	// 대기자 생성
	waiter := &BlockingWaiter{
		Keys:      keys,
		Response:  make(chan *BLPopResult, 1),
		Timeout:   timeout,
		StartTime: time.Now(),
	}
	
	// 모든 키에 대기자 등록
	s.mu.Lock()
	for _, key := range keys {
		s.waiters[key] = append(s.waiters[key], waiter)
	}
	s.mu.Unlock()
	
	// 타임아웃 고루틴 시작 (timeout > 0인 경우만)
	if useTimeout {
		go func() {
			time.Sleep(timeout)
			select {
			case s.waiterCleanup <- waiter:
				// Cleanup initiated
			default:
				// Cleanup channel full, waiter might already be processed
			}
		}()
	}
	
	// 결과를 기다림
	if useTimeout {
		// 타임아웃이 있는 경우
		select {
		case result = <-waiter.Response:
			return result
		case <-time.After(timeout + 100*time.Millisecond):
			// 추가 타임아웃으로 안전장치
			return nil
		}
	} else {
		// 무한 대기 (timeout=0)
		result = <-waiter.Response
		return result
	}
}
