package mail

import "testing"

func TestSafeFilename(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"invoice.pdf":               "invoice.pdf",
		"../secret.txt":             "secret.txt",
		"/tmp/absolute.txt":         "absolute.txt",
		"nested\\windows\\file.txt": "nested-windows-file.txt",
		"":                          "attachment",
	}
	for input, want := range tests {
		if got := safeFilename(input); got != want {
			t.Fatalf("safeFilename(%q) = %q, want %q", input, got, want)
		}
	}
}
