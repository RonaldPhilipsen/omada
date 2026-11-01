package cloud

import "testing"

func TestAPIErrorMessage(t *testing.T) {
	err := NewAPIError(-44116, "Open API Authorized failed...")
	want := "omada: API error -44116: Open API Authorized failed..."
	if err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
}
