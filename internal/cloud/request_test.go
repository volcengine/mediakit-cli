package cloud

import "testing"

func TestResolveRuntime(t *testing.T) {
	t.Setenv(envRuntime, "")
	t.Setenv(envIdentityName, "")
	t.Setenv(envOpenClawServiceMarker, "")

	tests := []struct {
		name                 string
		runtime              string
		identityName         string
		openClawServiceMaker string
		want                 string
	}{
		{
			name:                 "prefer explicit runtime",
			runtime:              "arkclaw",
			identityName:         "IgnoredIdentity",
			openClawServiceMaker: "ignored-marker",
			want:                 "arkclaw",
		},
		{
			name:                 "join identity and marker",
			identityName:         "ArkClaw",
			openClawServiceMaker: "openclaw",
			want:                 "ArkClaw/openclaw",
		},
		{
			name:         "use identity when marker missing",
			identityName: "ArkClaw",
			want:         "ArkClaw",
		},
		{
			name:                 "use marker when identity missing",
			openClawServiceMaker: "openclaw",
			want:                 "openclaw",
		},
		{
			name: "fallback to unknown",
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envRuntime, tt.runtime)
			t.Setenv(envIdentityName, tt.identityName)
			t.Setenv(envOpenClawServiceMarker, tt.openClawServiceMaker)

			if got := resolveRuntime(); got != tt.want {
				t.Fatalf("resolveRuntime() = %q, want %q", got, tt.want)
			}
		})
	}
}
