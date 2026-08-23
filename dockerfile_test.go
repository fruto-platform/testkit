package main

import (
	"os"
	"strings"
	"testing"
)

func TestDockerfileUsesBuildKitTargetPlatform(t *testing.T) {
	dockerfile, err := os.ReadFile("Dockerfile")
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}
	contents := string(dockerfile)
	for _, argument := range []string{"TARGETOS", "TARGETARCH"} {
		if strings.Contains(contents, "ARG "+argument+"=") {
			t.Fatalf("Dockerfile overrides BuildKit automatic argument %s", argument)
		}
		if !strings.Contains(contents, "ARG "+argument) {
			t.Fatalf("Dockerfile does not declare BuildKit automatic argument %s", argument)
		}
	}
}
