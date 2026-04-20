package envelope

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestCrossLanguageVectors asserts that Go's signing algorithm produces
// byte-identical canonical JSON and HMAC signatures to the Python reference.
// Vectors are shared at gate-test/vectors/envelope_signing.json.
func TestCrossLanguageVectors(t *testing.T) {
	// Locate vectors file relative to this test (../../../gate-test/vectors/...)
	path := filepath.Join("..", "..", "..", "gate-test", "vectors", "envelope_signing.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("vectors file not found at %s (skip when gate-test not co-located): %v", path, err)
	}

	var data struct {
		SigningKey string `json:"signing_key"`
		Vectors    []struct {
			Name              string                 `json:"name"`
			Input             map[string]interface{} `json:"input"`
			CanonicalJSON     string                 `json:"canonical_json"`
			ExpectedSignature string                 `json:"expected_signature"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parse vectors: %v", err)
	}

	for _, v := range data.Vectors {
		// Go's json.Marshal on map[string]any sorts keys alphabetically and
		// emits no whitespace — matching Python's json.dumps(sort_keys=True, separators=(",",":")).
		canonical, err := json.Marshal(v.Input)
		if err != nil {
			t.Fatalf("%s: marshal: %v", v.Name, err)
		}
		if string(canonical) != v.CanonicalJSON {
			t.Errorf("%s: canonical JSON drift\n  got:  %s\n  want: %s", v.Name, canonical, v.CanonicalJSON)
			continue
		}

		hash := sha256.Sum256(canonical)
		mac := hmac.New(sha256.New, []byte(data.SigningKey))
		mac.Write(hash[:])
		sig := hex.EncodeToString(mac.Sum(nil))

		if sig != v.ExpectedSignature {
			t.Errorf("%s: signature drift\n  got:  %s\n  want: %s", v.Name, sig, v.ExpectedSignature)
		}
	}
}
