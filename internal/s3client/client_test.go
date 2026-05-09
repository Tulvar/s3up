package s3client

import (
	"testing"

	"github.com/tulvar/s3up/internal/config"
)

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

func TestUsePathStyleDefaultsToPathForCustomEndpoint(t *testing.T) {
	t.Parallel()

	got, err := usePathStyle(config.Config{EndpointURL: "https://s3.example.test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatalf("expected path-style for custom endpoint")
	}
}

func TestUsePathStyleDefaultsToVirtualForAWS(t *testing.T) {
	t.Parallel()

	got, err := usePathStyle(config.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatalf("did not expect path-style without custom endpoint")
	}
}

func TestUsePathStyleAcceptsExplicitVirtual(t *testing.T) {
	t.Parallel()

	got, err := usePathStyle(config.Config{
		EndpointURL:     "https://s3.example.test",
		AddressingStyle: "virtual",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatalf("did not expect path-style with virtual addressing")
	}
}

func TestUsePathStyleRejectsInvalidAddressingStyle(t *testing.T) {
	t.Parallel()

	_, err := usePathStyle(config.Config{AddressingStyle: "wat"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUsePathStyleRejectsConflictingPathAndVirtual(t *testing.T) {
	t.Parallel()

	_, err := usePathStyle(config.Config{
		AddressingStyle: "virtual",
		PathStyle:       true,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
