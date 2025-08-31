package handler

import (
	"testing"
)

// TestErrorTypes는 다양한 에러 타입들을 테스트합니다.
func TestErrorTypes(t *testing.T) {
	// 테스트 케이스 1: UnknownCommandError
	unknownErr := &UnknownCommandError{Command: "BADCMD"}
	expectedMsg := "-ERR unknown command 'BADCMD'"
	if unknownErr.Error() != expectedMsg {
		t.Errorf("Expected %q, got %q", expectedMsg, unknownErr.Error())
	}

	// 테스트 케이스 2: WrongNumberOfArgumentsError
	wrongArgsErr := &WrongNumberOfArgumentsError{Command: "set"}
	expectedMsg = "-ERR wrong number of arguments for 'set' command"
	if wrongArgsErr.Error() != expectedMsg {
		t.Errorf("Expected %q, got %q", expectedMsg, wrongArgsErr.Error())
	}

	// 테스트 케이스 3: InvalidArgumentError
	invalidArgErr := &InvalidArgumentError{Message: "syntax error"}
	expectedMsg = "-ERR syntax error"
	if invalidArgErr.Error() != expectedMsg {
		t.Errorf("Expected %q, got %q", expectedMsg, invalidArgErr.Error())
	}
}