package s3client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
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

func TestNewClientRequiresOptInForConfiguredHTTPEndpoint(t *testing.T) {
	const endpoint = "http://s3.example.test:9000"
	for _, tc := range []struct {
		name          string
		endpointEnv   string
		profileConfig string
	}{
		{name: "global environment", endpointEnv: "AWS_ENDPOINT_URL"},
		{name: "S3 environment", endpointEnv: "AWS_ENDPOINT_URL_S3"},
		{name: "profile endpoint", profileConfig: "endpoint_url = " + endpoint + "\n"},
		{name: "profile S3 endpoint", profileConfig: "services = test-services\n\n" +
			"[services test-services]\ns3 =\n  endpoint_url = " + endpoint + "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{"AWS_ENDPOINT_URL", "AWS_ENDPOINT_URL_S3"} {
				t.Setenv(key, "")
				if err := os.Unsetenv(key); err != nil {
					t.Fatal(err)
				}
			}
			if tc.endpointEnv != "" {
				t.Setenv(tc.endpointEnv, endpoint)
			}
			dir := t.TempDir()
			configFile := filepath.Join(dir, "config")
			profile := "[profile s3up-test]\nregion = us-east-1\n" +
				"aws_access_key_id = test\naws_secret_access_key = test\n" + tc.profileConfig
			if err := os.WriteFile(configFile, []byte(profile), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("AWS_CONFIG_FILE", configFile)
			t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
			t.Setenv("AWS_IGNORE_CONFIGURED_ENDPOINT_URLS", "false")
			t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

			cfg := config.Config{Profile: "s3up-test"}
			if _, err := newClient(t.Context(), cfg); err == nil ||
				!strings.Contains(err.Error(), "--allow-insecure-endpoint") {
				t.Fatalf("expected HTTP endpoint rejection, got %v", err)
			}

			cfg.AllowInsecureEndpoint = true
			client, err := newClient(t.Context(), cfg)
			if err != nil {
				t.Fatal(err)
			}
			if got := aws.ToString(client.Options().BaseEndpoint); got != endpoint {
				t.Fatalf("got endpoint %q, want %q", got, endpoint)
			}

			cfg.AllowInsecureEndpoint = false
			cfg.EndpointURL = "https://s3.example.test"
			client, err = newClient(t.Context(), cfg)
			if err != nil {
				t.Fatal(err)
			}
			if got := aws.ToString(client.Options().BaseEndpoint); got != cfg.EndpointURL {
				t.Fatalf("got endpoint %q, want explicit override %q", got, cfg.EndpointURL)
			}
		})
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
