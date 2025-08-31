// Package protocol_test는 RESP 프로토콜 구현에 대한 테스트를 포함합니다.
// Go의 표준 테스트 방식을 따릅니다: 테스트 함수는 Test로 시작하고 *testing.T를 받습니다.
package protocol

import (
	"bufio"   // 테스트 입력을 위한 버퍼링된 리더 생성
	"bytes"   // 테스트 출력을 위한 버퍼 생성
	"strings" // 문자열을 Reader로 변환
	"testing" // Go의 표준 테스트 패키지
)

// TestParseSimpleString은 Simple String 타입의 파싱을 테스트합니다.
// 테스트 케이스: +OK\r\n → "OK"
//
// 테스트 목적:
//   - Simple String 형식을 올바르게 파싱하는지 확인
//   - + 시작 문자를 제거하고 \r\n을 제거하는지 확인
//
// Simple String은 Redis에서 간단한 상태 메시지를 전달할 때 사용됩니다.
func TestParseSimpleString(t *testing.T) {
	// 테스트 입력: RESP Simple String 형식
	input := "+OK\r\n"

	// 문자열을 Reader로 변환하고 Parser 생성
	reader := bufio.NewReader(strings.NewReader(input))
	parser := NewParser(reader)

	// 파싱 실행
	result, err := parser.Parse()
	if err != nil {
		// 파싱 오류가 발생하면 테스트 즉시 실패
		t.Fatalf("unexpected error: %v", err)
	}

	// 결과 검증: "OK" 문자열이 반환되어야 함
	if result != "OK" {
		t.Errorf("expected 'OK', got %v", result)
	}
}

// TestParseBulkString은 Bulk String 타입의 다양한 케이스를 테스트합니다.
//
// 테스트하는 케이스:
//  1. 정상 문자열: $5\r\nhello\r\n → "hello"
//  2. null 문자열: $-1\r\n → nil (Redis의 null 값)
//  3. 빈 문자열: $0\r\n\r\n → ""
//
// Table-driven test 패턴:
//   - Go의 권장 테스트 패턴으로, 여러 테스트 케이스를 테이블로 관리
//   - t.Run()을 사용하여 각 케이스를 서브테스트로 실행
//   - 실패 시 어떤 케이스가 실패했는지 명확히 표시
func TestParseBulkString(t *testing.T) {
	// 테스트 케이스 테이블 정의
	tests := []struct {
		name     string      // 테스트 케이스 이름 (t.Run에서 사용)
		input    string      // RESP 형식 입력
		expected interface{} // 기대하는 결과값
	}{
		{
			name:     "normal bulk string",
			input:    "$5\r\nhello\r\n",
			expected: "hello",
		},
		{
			name:     "null bulk string",
			input:    "$-1\r\n",
			expected: nil, // Redis에서 키가 없을 때 반환하는 값
		},
		{
			name:     "empty bulk string",
			input:    "$0\r\n\r\n",
			expected: "", // 빈 문자열도 유효한 값
		},
	}

	// 각 테스트 케이스를 순회하며 실행
	for _, tt := range tests {
		// 서브테스트로 실행 (실패 시 케이스 이름이 표시됨)
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			parser := NewParser(reader)

			result, err := parser.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestParseArray는 Array 타입의 파싱을 테스트합니다.
// 테스트 케이스: *2\r\n$4\r\nPING\r\n$4\r\ntest\r\n → ["PING", "test"]
//
// 테스트 목적:
//   - 배열 크기를 올바르게 파싱하는지 확인
//   - 각 요소를 재귀적으로 파싱하는지 확인
//   - Bulk String 요소들을 올바르게 처리하는지 확인
//
// 실제 Redis 명령어 예시:
//   - PING test 명령어는 이와 같은 형식으로 전송됨
func TestParseArray(t *testing.T) {
	// 테스트 입력: 2개의 Bulk String을 포함하는 배열
	// *2 = 배열에 2개 요소
	// $4\r\nPING = 첫 번째 요소 (4바이트 문자열 "PING")
	// $4\r\ntest = 두 번째 요소 (4바이트 문자열 "test")
	input := "*2\r\n$4\r\nPING\r\n$4\r\ntest\r\n"
	reader := bufio.NewReader(strings.NewReader(input))
	parser := NewParser(reader)

	// 파싱 실행
	result, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 타입 체크: 결과가 []interface{} 타입인지 확인
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", result)
	}

	// 배열 크기 검증
	if len(arr) != 2 {
		t.Errorf("expected array length 2, got %d", len(arr))
	}

	// 첫 번째 요소 검증
	if arr[0] != "PING" {
		t.Errorf("expected first element 'PING', got %v", arr[0])
	}

	// 두 번째 요소 검증
	if arr[1] != "test" {
		t.Errorf("expected second element 'test', got %v", arr[1])
	}
}

// TestParseInteger는 Integer 타입의 파싱을 테스트합니다.
// 테스트 케이스: :42\r\n → 42
//
// 테스트 목적:
//   - 정수를 올바르게 파싱하는지 확인
//   - int64 타입으로 반환하는지 확인
//
// Integer는 Redis에서 다음과 같은 경우에 사용됩니다:
//   - RPUSH의 반환값 (리스트 길이)
//   - INCR/DECR의 반환값 (증가/감소된 값)
//   - EXISTS의 반환값 (1 또는 0)
func TestParseInteger(t *testing.T) {
	// 테스트 입력: RESP Integer 형식
	input := ":42\r\n"
	reader := bufio.NewReader(strings.NewReader(input))
	parser := NewParser(reader)

	// 파싱 실행
	result, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 타입 체크: 결과가 int64 타입인지 확인
	num, ok := result.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", result)
	}

	// 값 검증
	if num != 42 {
		t.Errorf("expected 42, got %d", num)
	}
}

// TestWriteSimpleString은 Simple String 작성 기능을 테스트합니다.
// 테스트 케이스: "OK" → "+OK\r\n"
//
// 테스트 목적:
//   - Simple String 형식을 올바르게 생성하는지 확인
//   - + 시작 문자와 \r\n 종료 문자가 추가되는지 확인
//
// bytes.Buffer 사용:
//   - 테스트에서는 네트워크 연결 대신 메모리 버퍼를 사용
//   - 출력을 문자열로 쉽게 검증 가능
func TestWriteSimpleString(t *testing.T) {
	// 테스트용 버퍼 생성 (네트워크 연결 대신 사용)
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	// Simple String 작성
	err := writer.WriteSimpleString("OK")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 결과 검증: RESP 형식이 올바른지 확인
	expected := "+OK\r\n"
	if buf.String() != expected {
		// %q를 사용하여 특수 문자(\r\n)를 볼 수 있게 출력
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestWriteBulkString은 Bulk String 작성 기능의 다양한 케이스를 테스트합니다.
//
// 테스트하는 케이스:
//  1. 정상 문자열: "hello" → "$5\r\nhello\r\n"
//  2. null 문자열: nil → "$-1\r\n" (GET에서 키가 없을 때)
//  3. 빈 문자열: "" → "$0\r\n\r\n"
//
// WriteBulkString의 특징:
//   - 포인터를 받아서 nil 처리 가능
//   - 길이를 명시하여 바이너리 안전
func TestWriteBulkString(t *testing.T) {
	// Table-driven test 패턴 사용
	tests := []struct {
		name     string  // 테스트 케이스 이름
		input    *string // 입력 문자열 포인터 (nil 가능)
		expected string  // 기대하는 RESP 형식 출력
	}{
		{
			name:     "normal bulk string",
			input:    stringPtr("hello"),
			expected: "$5\r\nhello\r\n",
		},
		{
			name:     "null bulk string",
			input:    nil, // Redis에서 null 값을 표현
			expected: "$-1\r\n",
		},
		{
			name:     "empty bulk string",
			input:    stringPtr(""),
			expected: "$0\r\n\r\n", // 길이 0이지만 유효한 문자열
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := NewWriter(&buf)

			err := writer.WriteBulkString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}

// TestWriteInteger는 Integer 작성 기능을 테스트합니다.
// 테스트 케이스: 42 → ":42\r\n"
//
// 테스트 목적:
//   - Integer 형식을 올바르게 생성하는지 확인
//   - : 시작 문자와 \r\n 종료 문자가 추가되는지 확인
//
// Integer는 주로 명령어의 반환값으로 사용됩니다:
//   - RPUSH: 리스트의 새 길이 반환
//   - DEL: 삭제된 키의 개수 반환
func TestWriteInteger(t *testing.T) {
	// 테스트용 버퍼 생성
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	// Integer 작성
	err := writer.WriteInteger(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 결과 검증
	expected := ":42\r\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestWriteArray는 Array 작성 기능을 테스트합니다.
// 테스트 케이스: ["PING", "test"] → "*2\r\n$4\r\nPING\r\n$4\r\ntest\r\n"
//
// 테스트 목적:
//   - 배열 크기를 올바르게 명시하는지 확인
//   - 각 요소를 Bulk String으로 올바르게 작성하는지 확인
//
// Array는 주로 다음과 같은 경우에 사용됩니다:
//   - LRANGE: 리스트 요소들 반환
//   - KEYS: 키 목록 반환
//   - MGET: 여러 값들 반환
func TestWriteArray(t *testing.T) {
	// 테스트용 버퍼 생성
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	// 문자열 배열 작성
	err := writer.WriteArray([]string{"PING", "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 결과 검증
	// *2 = 2개 요소
	// $4\r\nPING = 첫 번째 요소 (Bulk String)
	// $4\r\ntest = 두 번째 요소 (Bulk String)
	expected := "*2\r\n$4\r\nPING\r\n$4\r\ntest\r\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// stringPtr는 문자열의 포인터를 반환하는 헬퍼 함수입니다.
// 테스트에서 문자열 포인터가 필요할 때 사용합니다.
//
// Go에서는 리터럴 문자열의 주소를 직접 가져올 수 없기 때문에
// 이런 헬퍼 함수가 필요합니다.
//
// 예시:
//
//	stringPtr("hello") → &"hello"
func stringPtr(s string) *string {
	return &s
}
