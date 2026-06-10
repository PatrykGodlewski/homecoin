package postgres

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedMigrationsMatchSource(t *testing.T) {
	root := filepath.Join("..", "..", "..", "migrations")
	embedded := filepath.Join("migrations")

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read source migrations: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatalf("read source %s: %v", entry.Name(), err)
		}
		emb, err := os.ReadFile(filepath.Join(embedded, entry.Name()))
		if err != nil {
			t.Fatalf("embedded migration missing %s (run: make migrations): %v", entry.Name(), err)
		}
		if string(src) != string(emb) {
			t.Fatalf("migration drift: %s differs from embedded copy; run: make migrations", entry.Name())
		}
	}
}
