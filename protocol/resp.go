// Package protocol은 Redis의 RESP(REdis Serialization Protocol) 프로토콜을 처리합니다.
// RESP는 Redis 클라이언트와 서버 간의 통신에 사용되는 텍스트 기반 프로토콜입니다.
package protocol

import (
	"bufio"   // 버퍼링된 I/O를 제공하여 효율적인 읽기/쓰기를 지원
	"fmt"     // 포맷팅된 I/O 함수들 (에러 메시지 생성 등)
	"io"      // 기본 I/O 인터페이스와 함수들
	"strconv" // 문자열과 다른 타입 간의 변환 (문자열을 숫자로 변환 등)
)

// Parser는 RESP 프로토콜 형식의 데이터를 파싱하는 구조체입니다.
// Redis 클라이언트로부터 받은 명령어를 해석할 때 사용됩니다.
type Parser struct {
	// reader는 네트워크 연결에서 데이터를 버퍼링하여 읽습니다.
	// 버퍼링을 통해 시스템 콜 횟수를 줄여 성능을 향상시킵니다.
	reader *bufio.Reader
}

// NewParser는 새로운 Parser 인스턴스를 생성합니다.
// 매개변수:
//   - reader: TCP 연결 등에서 데이터를 읽을 bufio.Reader
//
// 반환값:
//   - 생성된 Parser 포인터
func NewParser(reader *bufio.Reader) *Parser {
	return &Parser{reader: reader}
}

// Parse는 RESP 프로토콜 데이터를 파싱하는 메인 함수입니다.
// RESP 데이터 타입을 식별하고 적절한 파싱 함수를 호출합니다.
//
// RESP 데이터 타입:
//   - '+': Simple String (간단한 문자열, 예: +OK\r\n)
//   - '$': Bulk String (길이가 명시된 문자열, 예: $5\r\nhello\r\n)
//   - '*': Array (배열, 예: *2\r\n$4\r\nPING\r\n$4\r\ntest\r\n)
//   - ':': Integer (정수, 예: :1000\r\n)
//   - '-': Error (에러, 예: -ERR unknown command\r\n) - 현재 미구현
//
// 반환값:
//   - interface{}: 파싱된 데이터 (string, []interface{}, int64 등)
//   - error: 파싱 중 발생한 에러
func (p *Parser) Parse() (interface{}, error) {
	// 첫 번째 바이트를 읽어서 데이터 타입을 판별합니다
	typeByte, err := p.reader.ReadByte()
	if err != nil {
		return nil, err
	}

	// RESP 프로토콜 명세: https://redis.io/docs/latest/develop/reference/protocol-spec/
	// 타입 바이트에 따라 적절한 파싱 함수를 호출합니다
	switch typeByte {
	case '+':
		// Simple String: 줄바꿈까지의 짧은 문자열 (주로 상태 응답용)
		return p.readSimpleString()
	case '$':
		// Bulk String: 바이너리 안전 문자열 (길이가 명시됨)
		return p.readBulkString()
	case '*':
		// Array: 다른 RESP 타입들의 배열
		return p.readArray()
	case ':':
		// Integer: 부호있는 64비트 정수
		return p.readInteger()
	default:
		// 알 수 없는 타입은 에러 반환
		return nil, fmt.Errorf("unknown RESP type: %c", typeByte)
	}
}

// readSimpleString은 Simple String 타입을 파싱합니다.
// 형식: +<문자열>\r\n
// 예시: +OK\r\n → "OK"
//
// Simple String 특징:
//   - 줄바꿈 문자를 포함할 수 없음
//   - 주로 짧은 상태 메시지에 사용 (OK, PONG 등)
//   - 바이너리 안전하지 않음
func (p *Parser) readSimpleString() (string, error) {
	// \r\n까지의 한 줄을 읽습니다
	line, err := p.readLine()
	if err != nil {
		return "", err
	}
	return line, nil
}

// readBulkString은 Bulk String 타입을 파싱합니다.
// 형식: $<길이>\r\n<데이터>\r\n
// 예시:
//   - $5\r\nhello\r\n → "hello"
//   - $0\r\n\r\n → "" (빈 문자열)
//   - $-1\r\n → nil (null bulk string)
//
// Bulk String 특징:
//   - 바이너리 안전 (모든 바이트 값 포함 가능)
//   - 길이가 먼저 명시되어 정확한 바이트 수만큼 읽음
//   - null 값 표현 가능 ($-1)
func (p *Parser) readBulkString() (interface{}, error) {
	// 첫 줄에서 문자열 길이를 읽습니다
	line, err := p.readLine()
	if err != nil {
		return nil, err
	}

	// 문자열을 정수로 변환 (10진수, 64비트)
	length, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return nil, err
	}

	// -1은 null bulk string을 의미 (Redis의 nil 값)
	if length == -1 {
		return nil, nil
	}

	// 지정된 길이 + 2바이트(\r\n) 만큼의 버퍼 생성
	buf := make([]byte, length+2)
	// 정확히 필요한 바이트 수만큼 읽기 (부분 읽기 방지)
	_, err = io.ReadFull(p.reader, buf)
	if err != nil {
		return nil, err
	}

	// \r\n을 제외한 실제 데이터만 반환
	return string(buf[:length]), nil
}

// readArray는 Array 타입을 파싱합니다.
// 형식: *<요소개수>\r\n<요소1><요소2>...
// 예시: *2\r\n$4\r\nPING\r\n$4\r\ntest\r\n → ["PING", "test"]
//
// Array 특징:
//   - 다른 RESP 타입들을 요소로 가질 수 있음 (중첩 가능)
//   - Redis 명령어는 주로 배열로 전송됨
//   - null array 표현 가능 (*-1)
//
// 실제 Redis 명령어 예시:
//   - PING: *1\r\n$4\r\nPING\r\n
//   - SET key value: *3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n
func (p *Parser) readArray() ([]interface{}, error) {
	// 첫 줄에서 배열 요소 개수를 읽습니다
	line, err := p.readLine()
	if err != nil {
		return nil, err
	}

	// 문자열을 정수로 변환
	count, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return nil, err
	}

	// -1은 null array를 의미
	if count == -1 {
		return nil, nil
	}

	// 지정된 개수만큼의 슬라이스 생성
	result := make([]interface{}, count)

	// 각 요소를 재귀적으로 파싱
	// 배열의 요소는 어떤 RESP 타입이든 가능 (문자열, 정수, 다른 배열 등)
	for i := int64(0); i < count; i++ {
		value, err := p.Parse() // 재귀 호출로 중첩된 구조 처리
		if err != nil {
			return nil, err
		}
		result[i] = value
	}

	return result, nil
}

// readInteger는 Integer 타입을 파싱합니다.
// 형식: :<정수>\r\n
// 예시:
//   - :42\r\n → 42
//   - :-100\r\n → -100
//   - :0\r\n → 0
//
// Integer 특징:
//   - 부호있는 64비트 정수 (-9223372036854775808 ~ 9223372036854775807)
//   - 주로 개수, 크기, 성공/실패 코드 등을 표현
//   - RPUSH 같은 명령어의 반환값으로 사용 (리스트 길이 등)
func (p *Parser) readInteger() (int64, error) {
	// 한 줄을 읽어서 정수 부분만 추출
	line, err := p.readLine()
	if err != nil {
		return 0, err
	}

	// 문자열을 64비트 정수로 변환 (10진수)
	return strconv.ParseInt(line, 10, 64)
}

// readLine은 \r\n으로 끝나는 한 줄을 읽는 헬퍼 함수입니다.
// RESP 프로토콜에서 모든 데이터는 \r\n(CRLF)로 구분됩니다.
//
// 동작 과정:
//  1. '\n' 문자까지 읽기
//  2. 끝에서 \r\n 제거
//  3. 순수한 데이터만 반환
//
// 예시:
//   - "OK\r\n" → "OK"
//   - "42\r\n" → "42"
//   - "\r\n" → ""
func (p *Parser) readLine() (string, error) {
	// '\n' 문자를 만날 때까지 읽습니다
	// ReadString은 구분자를 포함하여 반환합니다
	line, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Windows 스타일 줄바꿈(\r\n) 처리
	// 끝에서 두 번째 문자가 \r인지 확인
	if len(line) >= 2 && line[len(line)-2] == '\r' {
		// \r\n을 제거하고 반환
		return line[:len(line)-2], nil
	}

	// Unix 스타일 줄바꿈(\n) 처리
	// \n만 제거하고 반환
	return line[:len(line)-1], nil
}
