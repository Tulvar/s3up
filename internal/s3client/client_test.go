package s3client

import "testing"

func TestValidateEndpointRejectsHTTPWithoutOptIn(t *testing.T) {
	t.Parallel()

	err := validateEndpoint("http://localhost:9000", false)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidateEndpointAllowsHTTPWithOptIn(t *testing.T) {
	t.Parallel()

	err := validateEndpoint("http://localhost:9000", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
