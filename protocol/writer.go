// Package protocol은 Redis의 RESP(REdis Serialization Protocol) 프로토콜을 처리합니다.
package protocol

import (
	"fmt" // 포맷팅된 문자열 생성을 위해 사용 (Sprintf 등)
	"io"  // Writer 인터페이스를 위해 사용
)

// Writer는 RESP 프로토콜 형식으로 데이터를 작성하는 구조체입니다.
// Redis 클라이언트에게 응답을 보낼 때 사용됩니다.
type Writer struct {
	// writer는 실제 데이터를 쓰는 인터페이스
	// 주로 net.Conn(네트워크 연결)이나 bytes.Buffer(테스트용)가 사용됨
	writer io.Writer
}

// NewWriter는 새로운 Writer 인스턴스를 생성합니다.
// 매개변수:
//   - w: 데이터를 쓸 io.Writer (예: TCP 연결, 버퍼 등)
//
// 반환값:
//   - 생성된 Writer 포인터
func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

// WriteSimpleString은 Simple String 형식으로 문자열을 작성합니다.
// 형식: +<문자열>\r\n
// 예시: "OK" → "+OK\r\n"
//
// 사용 예:
//   - 명령 성공 응답: +OK\r\n
//   - PING 응답: +PONG\r\n
//
// 주의사항:
//   - 문자열에 \r이나 \n이 포함되면 안 됨 (단순 문자열만 가능)
//   - 바이너리 안전하지 않음
func (w *Writer) WriteSimpleString(s string) error {
	// + 시작 문자와 \r\n 종료 문자를 추가하여 작성
	_, err := w.writer.Write([]byte(fmt.Sprintf("+%s\r\n", s)))
	return err
}

// WriteBulkString은 Bulk String 형식으로 문자열을 작성합니다.
// 형식: $<길이>\r\n<데이터>\r\n
// 예시:
//   - "hello" → "$5\r\nhello\r\n"
//   - "" → "$0\r\n\r\n" (빈 문자열)
//   - nil → "$-1\r\n" (null bulk string)
//
// 사용 예:
//   - GET 명령어의 응답 (값이 있을 때)
//   - ECHO 명령어의 응답
//   - 바이너리 안전한 데이터 전송
//
// 매개변수:
//   - s: 문자열 포인터 (nil일 수 있음)
//     nil은 Redis의 null 값을 표현 (예: 키가 없을 때)
func (w *Writer) WriteBulkString(s *string) error {
	// nil 처리: Redis의 null bulk string
	if s == nil {
		// $-1\r\n은 null을 나타내는 특별한 형식
		_, err := w.writer.Write([]byte("$-1\r\n"))
		return err
	}

	// 정상 문자열: 길이를 먼저 보내고 데이터를 보냄
	// 길이는 바이트 수 기준 (UTF-8 문자열의 경우 len()이 바이트 수 반환)
	_, err := w.writer.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(*s), *s)))
	return err
}

// WriteInteger는 Integer 형식으로 정수를 작성합니다.
// 형식: :<정수>\r\n
// 예시:
//   - 42 → ":42\r\n"
//   - -100 → ":-100\r\n"
//   - 0 → ":0\r\n"
//
// 사용 예:
//   - RPUSH의 반환값 (리스트 길이)
//   - LLEN의 반환값 (리스트 길이)
//   - INCR의 반환값 (증가된 값)
//   - 성공/실패 코드 (1/0/-1 등)
//
// 매개변수:
//   - n: 작성할 정수값
func (w *Writer) WriteInteger(n int) error {
	// : 시작 문자와 \r\n 종료 문자를 추가하여 작성
	_, err := w.writer.Write([]byte(fmt.Sprintf(":%d\r\n", n)))
	return err
}

// WriteArray는 Array 형식으로 문자열 배열을 작성합니다.
// 형식: *<요소개수>\r\n<요소1><요소2>...
// 예시: ["PING", "test"] → "*2\r\n$4\r\nPING\r\n$4\r\ntest\r\n"
//
// 사용 예:
//   - LRANGE의 반환값 (리스트 요소들)
//   - KEYS의 반환값 (키 목록)
//   - 여러 값을 반환하는 모든 명령어
//
// 동작 과정:
//  1. 먼저 배열 크기를 작성 (*2\r\n)
//  2. 각 요소를 Bulk String으로 작성
//
// 매개변수:
//   - arr: 작성할 문자열 배열
func (w *Writer) WriteArray(arr []string) error {
	// 먼저 배열 크기를 명시 (*<개수>\r\n)
	_, err := w.writer.Write([]byte(fmt.Sprintf("*%d\r\n", len(arr))))
	if err != nil {
		return err
	}

	// 각 요소를 Bulk String 형식으로 작성
	// RESP 배열의 요소는 주로 Bulk String을 사용
	for _, s := range arr {
		if err := w.WriteBulkString(&s); err != nil {
			return err
		}
	}
	return nil
}

// WriteOK는 표준 OK 응답을 작성하는 헬퍼 함수입니다.
// 출력: +OK\r\n
//
// 사용되는 명령어:
//   - SET: 값 설정 성공 시
//   - DEL: 키 삭제 성공 시
//   - FLUSHDB: 데이터베이스 초기화 성공 시
//   - 기타 성공적으로 수행된 명령어들
func (w *Writer) WriteOK() error {
	return w.WriteSimpleString("OK")
}

// WritePONG은 PING 명령에 대한 표준 응답을 작성하는 헬퍼 함수입니다.
// 출력: +PONG\r\n
//
// 사용 예:
//   - PING 명령어에 대한 응답
//   - 연결 상태 확인 (헬스 체크)
//   - 클라이언트-서버 간 네트워크 지연 측정
//
// Redis PING 명령어:
//   - PING (no args) → PONG
//   - PING "hello" → "hello" (인자가 있으면 그대로 반환)
func (w *Writer) WritePONG() error {
	return w.WriteSimpleString("PONG")
}
