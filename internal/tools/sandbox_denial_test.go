package tools

import (
	"testing"

	holtSandbox "github.com/askspecter/holt/internal/sandbox"
)

func TestLikelySandboxDeniedDetectsReferenceKeywords(t *testing.T) {
	plan := holtSandbox.CommandPlan{
		Wrapped:       true,
		TargetBackend: holtSandbox.BackendLinuxBwrap,
	}
	output := "touch: cannot touch '/home/user/.npm/cache': Read-only file system"
	if !likelySandboxDenied(plan, 1, output) {
		t.Fatalf("expected reference sandbox denial keyword to be classified as sandbox denied")
	}
}

func TestLikelySandboxDeniedDetectsNetworkDenialEvenWithZeroExit(t *testing.T) {
	plan := holtSandbox.CommandPlan{
		Wrapped:       true,
		TargetBackend: holtSandbox.BackendLinuxBwrap,
		Policy:        holtSandbox.Policy{Network: holtSandbox.NetworkDeny},
		PermissionProfile: holtSandbox.PermissionProfile{
			Network: holtSandbox.NetworkPolicy{Mode: holtSandbox.NetworkDeny},
		},
	}
	if !likelySandboxDenied(plan, 0, "Cannot open a network socket.") {
		t.Fatal("network-denied socket output with exit 0 must be classified as sandbox denied")
	}
	meta := map[string]string{}
	markLikelySandboxDenial(meta, plan, 0, "Cannot open a network socket.")
	if meta[SandboxLikelyDeniedMeta] != "true" || meta[SandboxDenialKindMeta] != SandboxDenialKindNetwork {
		t.Fatalf("network denial meta = %#v", meta)
	}
}

func TestLikelySandboxDeniedIgnoresUnsandboxedFailure(t *testing.T) {
	plan := holtSandbox.CommandPlan{Wrapped: false}
	if likelySandboxDenied(plan, 1, "permission denied") {
		t.Fatal("unsandboxed command output must not be classified as a sandbox denial")
	}
}
