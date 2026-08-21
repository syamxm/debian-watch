package httpx

import (
	"testing"
	"testing/fstest"
)

func TestAssetVersionChangesWithContent(t *testing.T) {
	before := fstest.MapFS{"static/css/app.css": {Data: []byte("body{color:red}")}}
	after := fstest.MapFS{"static/css/app.css": {Data: []byte("body{color:blue}")}}

	first, err := assetVersion(before)
	if err != nil {
		t.Fatalf("assetVersion: %v", err)
	}
	second, err := assetVersion(after)
	if err != nil {
		t.Fatalf("assetVersion: %v", err)
	}

	if first == second {
		t.Fatal("changing an asset must change the version, or browsers keep serving the stale copy")
	}
}

func TestAssetVersionIsStable(t *testing.T) {
	files := fstest.MapFS{
		"static/css/app.css":  {Data: []byte("body{color:red}")},
		"static/js/status.js": {Data: []byte("void 0;")},
	}

	first, err := assetVersion(files)
	if err != nil {
		t.Fatalf("assetVersion: %v", err)
	}
	second, err := assetVersion(files)
	if err != nil {
		t.Fatalf("assetVersion: %v", err)
	}

	if first != second {
		t.Fatalf("version is not stable: %q then %q", first, second)
	}
	if len(first) != 12 {
		t.Errorf("version length = %d, want 12", len(first))
	}
}
