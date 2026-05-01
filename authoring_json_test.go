package apitools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAuthoringJSONCompatibility(t *testing.T) {
	diagnostic, err := json.Marshal(Diagnostic{Severity: "warning", Message: "check"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(diagnostic), `"code":""`) {
		t.Fatalf("diagnostic JSON = %s", diagnostic)
	}

	artifact, err := json.Marshal(Artifact{Path: "intent.hcl"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(artifact), `"media_type":""`) || strings.Contains(string(artifact), `"content"`) {
		t.Fatalf("artifact JSON = %s", artifact)
	}

	emptySet, err := json.Marshal(ArtifactSet{})
	if err != nil {
		t.Fatal(err)
	}
	if string(emptySet) != "{}" {
		t.Fatalf("empty artifact set JSON = %s", emptySet)
	}

	withoutQuestions, err := json.Marshal(ArtifactSet{QuestionPlan: QuestionPlan{}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(withoutQuestions), `"question_plan"`) {
		t.Fatalf("empty question plan should be omitted: %s", withoutQuestions)
	}

	withQuestions, err := json.Marshal(ArtifactSet{QuestionPlan: QuestionPlan{Questions: []Question{{Prompt: "Which input?"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(withQuestions), `"question_plan"`) {
		t.Fatalf("question plan should be present: %s", withQuestions)
	}
}
