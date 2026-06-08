package service

import "testing"

func TestClearWebDomainAndAddress(t *testing.T) {
	s := initSettingTestDB(t)

	// Seed the panel addressing fields plus an unrelated field that must survive.
	seed := map[string]string{
		"webDomain": "panel.example.com",
		"webListen": "10.0.0.5",
		"webURI":    "https://panel.example.com:2096/app/",
		"webPath":   "/keep/",
	}
	for key, value := range seed {
		if err := s.setString(key, value); err != nil {
			t.Fatalf("seed %s: %v", key, err)
		}
	}

	if err := s.ClearWebDomainAndAddress(); err != nil {
		t.Fatalf("ClearWebDomainAndAddress: %v", err)
	}

	for _, key := range []string{"webDomain", "webListen", "webURI"} {
		got, err := s.getString(key)
		if err != nil {
			t.Fatalf("get %s: %v", key, err)
		}
		if got != "" {
			t.Errorf("%s = %q after clear, want empty", key, got)
		}
	}

	// Unrelated settings must be left untouched.
	webPath, err := s.GetWebPath()
	if err != nil {
		t.Fatalf("GetWebPath: %v", err)
	}
	if webPath != "/keep/" {
		t.Errorf("webPath = %q after clear, want %q", webPath, "/keep/")
	}
}
