package handler

import (
	"strings"
	"testing"
)

func TestBuildTeardownPromptIncludesContentAndTone(t *testing.T) {
	system, user := buildTeardownPrompt(teardownRequest{
		Content: "少年得到一座会生长城市的书塔",
		Title:   "北境书塔",
		Genre:   "玄幻",
		Tone:    "savage",
	})
	if !strings.Contains(system, "JSON") {
		t.Fatalf("system prompt should require JSON output: %q", system)
	}
	for _, kw := range []string{"北境书塔", "玄幻", "少年得到一座会生长城市的书塔", "毒舌"} {
		if !strings.Contains(user, kw) {
			t.Fatalf("user prompt missing %q: %s", kw, user)
		}
	}
}

func TestTeardownToneHint(t *testing.T) {
	if !strings.Contains(teardownToneHint("gentle"), "鼓励") {
		t.Fatal("gentle tone should be encouraging")
	}
	if !strings.Contains(teardownToneHint("savage"), "毒舌") {
		t.Fatal("savage tone should be 毒舌")
	}
}

func TestBuildScriptPromptByForm(t *testing.T) {
	_, user := buildScriptPrompt(scriptRequest{Content: "主角推开书塔大门", Form: "storyboard", Episodes: 3})
	for _, kw := range []string{"分镜", "主角推开书塔大门", "约 3 集"} {
		if !strings.Contains(user, kw) {
			t.Fatalf("script prompt missing %q: %s", kw, user)
		}
	}
	_, short := buildScriptPrompt(scriptRequest{Content: "x", Form: "short"})
	if !strings.Contains(short, "短剧") {
		t.Fatalf("short form should mention 短剧: %s", short)
	}
}

func TestExtractJSONObject(t *testing.T) {
	cases := map[string]string{
		`{"a":1}`:                              `{"a":1}`,
		"```json\n{\"a\":1}\n```":              `{"a":1}`,
		"前缀 {\"a\":1} 后缀":                       `{"a":1}`,
		"```\n{\"x\": {\"y\":2}}\n```":          `{"x": {"y":2}}`,
	}
	for in, want := range cases {
		if got := extractJSONObject(in); got != want {
			t.Fatalf("extractJSONObject(%q) = %q, want %q", in, got, want)
		}
	}
}
