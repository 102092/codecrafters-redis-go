package handler

// equalStringSlices는 두 문자열 슬라이스가 같은지 비교하는 헬퍼 함수입니다.
// Go 1.21 이전 버전에서는 slices.Equal을 사용할 수 없으므로 직접 구현합니다.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}