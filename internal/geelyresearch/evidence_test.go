package geelyresearch

import "testing"

func TestPublicEvidenceHasProvenanceAndBoundedScope(t *testing.T) {
	evidence := PublicEvidence()
	if len(evidence) == 0 {
		t.Fatal("PublicEvidence() is empty")
	}
	for _, item := range evidence {
		if item.ID == "" || item.Title == "" || item.Publisher == "" || item.URL == "" || item.RetrievedAt == "" || item.Scope == "" || item.Confidence == "" {
			t.Errorf("evidence item lacks provenance or scope: %+v", item)
		}
	}
}

func TestPublicEvidenceReturnsDefensiveCopy(t *testing.T) {
	first := PublicEvidence()
	first[0].Title = "mutated test value"
	second := PublicEvidence()
	if second[0].Title == "mutated test value" {
		t.Fatal("PublicEvidence() exposes mutable shared catalog state")
	}
}
