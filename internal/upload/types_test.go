package upload

import "testing"

func TestParseS3URI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    S3URI
		wantErr bool
	}{
		{
			name: "bucket and key",
			raw:  "s3://my-bucket/path/to/file.txt",
			want: S3URI{Bucket: "my-bucket", Key: "path/to/file.txt"},
		},
		{
			name: "trims extra key slashes",
			raw:  "s3://my-bucket//path/to/file.txt",
			want: S3URI{Bucket: "my-bucket", Key: "path/to/file.txt"},
		},
		{
			name:    "missing scheme",
			raw:     "my-bucket/path",
			wantErr: true,
		},
		{
			name:    "missing key",
			raw:     "s3://my-bucket",
			wantErr: true,
		},
		{
			name:    "missing bucket",
			raw:     "s3:///path",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseS3URI(tt.raw)
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
