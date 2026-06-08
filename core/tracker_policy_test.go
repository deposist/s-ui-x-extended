package core

import (
	"os"
	"strings"
	"testing"
)

func TestTrackerPolicyMatchesSingBoxDependency(t *testing.T) {
	version := requiredModuleVersion(t, "../go.mod", TrackerValidatedSingBoxModule)
	if TrackerValidatedSingBoxVersion != version {
		t.Fatalf("%s bumped from %s to %s; revalidate trackers and update %s",
			TrackerValidatedSingBoxModule,
			TrackerValidatedSingBoxVersion,
			version,
			TrackerRevalidationPolicyDoc,
		)
	}

	status := SingBoxTrackerRevalidationStatus(version)
	if status.Required {
		t.Fatalf("current sing-box version should be covered by tracker policy: %#v", status)
	}
	if len(status.Checks) == 0 {
		t.Fatal("tracker revalidation policy must include explicit checks")
	}
}

func TestTrackerPolicyRequiresRevalidationOnVersionChange(t *testing.T) {
	status := SingBoxTrackerRevalidationStatus("v99.0.0")
	if !status.Required {
		t.Fatal("unexpectedly accepted unvalidated sing-box version")
	}
}

func TestTrackerPolicyDocCoversCurrentChecklist(t *testing.T) {
	doc, err := os.ReadFile("../" + TrackerRevalidationPolicyDoc)
	if err != nil {
		t.Fatal(err)
	}
	text := string(doc)
	if !strings.Contains(text, TrackerValidatedSingBoxVersion) {
		t.Fatalf("policy doc does not mention validated sing-box version %s", TrackerValidatedSingBoxVersion)
	}
	for _, check := range TrackerRevalidationChecks {
		if !strings.Contains(text, check) {
			t.Fatalf("policy doc missing tracker check %q", check)
		}
	}
}

func requiredModuleVersion(t *testing.T, path string, module string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[0] == module {
			return fields[1]
		}
	}
	t.Fatalf("module %s not found in %s", module, path)
	return ""
}
