package list

import "testing"

func TestParseS3Prefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    S3Prefix
		wantErr bool
	}{
		{
			name: "bucket only",
			raw:  "s3://bucket",
			want: S3Prefix{Bucket: "bucket"},
		},
		{
			name: "bucket and prefix",
			raw:  "s3://bucket/site/assets/",
			want: S3Prefix{Bucket: "bucket", Prefix: "site/assets/"},
		},
		{
			name: "trims leading prefix slashes",
			raw:  "s3://bucket//site/",
			want: S3Prefix{Bucket: "bucket", Prefix: "site/"},
		},
		{
			name:    "missing scheme",
			raw:     "bucket/site/",
			wantErr: true,
		},
		{
			name:    "missing bucket",
			raw:     "s3://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseS3Prefix(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
