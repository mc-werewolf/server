package github

import "testing"

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "bare owner/repo",
			raw:       "github.com/shizuku86/kairo",
			wantOwner: "shizuku86",
			wantRepo:  "kairo",
		},
		{
			name:      "full https url",
			raw:       "https://github.com/shizuku86/kairo",
			wantOwner: "shizuku86",
			wantRepo:  "kairo",
		},
		{
			name:      "www host",
			raw:       "https://www.github.com/shizuku86/kairo",
			wantOwner: "shizuku86",
			wantRepo:  "kairo",
		},
		{
			name:      "trailing releases path",
			raw:       "https://github.com/shizuku86/kairo/releases",
			wantOwner: "shizuku86",
			wantRepo:  "kairo",
		},
		{
			name:      "trailing releases tag path",
			raw:       "https://github.com/shizuku86/kairo/releases/tag/v1.0.0",
			wantOwner: "shizuku86",
			wantRepo:  "kairo",
		},
		{
			name:      "trailing tree path",
			raw:       "https://github.com/shizuku86/kairo/tree/main/src",
			wantOwner: "shizuku86",
			wantRepo:  "kairo",
		},
		{
			name:      "git suffix",
			raw:       "https://github.com/shizuku86/kairo.git",
			wantOwner: "shizuku86",
			wantRepo:  "kairo",
		},
		{
			name:    "wrong host",
			raw:     "https://gitlab.com/shizuku86/kairo",
			wantErr: true,
		},
		{
			name:    "missing repo segment",
			raw:     "https://github.com/shizuku86",
			wantErr: true,
		},
		{
			name:    "empty string",
			raw:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseRepoURL(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseRepoURL(%q) = %q/%q, want error", tt.raw, owner, repo)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRepoURL(%q) unexpected error: %v", tt.raw, err)
			}
			if owner != tt.wantOwner || repo != tt.wantRepo {
				t.Fatalf("ParseRepoURL(%q) = %q/%q, want %q/%q", tt.raw, owner, repo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}
