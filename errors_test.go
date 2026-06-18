package aihubsdk

import "testing"

func TestAPIError_Error(t *testing.T) {
	err := &APIError{Status: 401, Message: "无效的 API Key"}
	want := "aihub: status=401 message=无效的 API Key"
	if got := err.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}
