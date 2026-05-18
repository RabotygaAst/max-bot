package main

import "testing"

func TestMaskDatabaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "valid PostgreSQL URL with password",
			url:  "postgres://dbuser:secret@localhost:5432/maxbot?sslmode=disable",
			want: "postgres://***@localhost:5432/maxbot?sslmode=disable",
		},
		{
			name: "URL without at sign",
			url:  "postgres://localhost:5432/maxbot?sslmode=disable",
			want: "***",
		},
		{
			name: "URL without scheme separator",
			url:  "postgres:user:secret@localhost:5432/maxbot",
			want: "***",
		},
		{
			name: "short string",
			url:  "short",
			want: "***",
		},
		{
			name: "empty string",
			url:  "",
			want: "***",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := maskDatabaseURL(tt.url); got != tt.want {
				t.Fatalf("maskDatabaseURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
