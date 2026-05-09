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

func TestDefaultRegionUsesUSEast1WhenEmpty(t *testing.T) {
	if got := defaultRegion(""); got != "us-east-1" {
		t.Fatalf("got region %q", got)
	}
}

func TestDefaultRegionKeepsExplicitRegion(t *testing.T) {
	t.Parallel()

	if got := defaultRegion("eu-central-1"); got != "eu-central-1" {
		t.Fatalf("got region %q", got)
	}
}
