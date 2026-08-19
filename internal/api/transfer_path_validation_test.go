package api

import "testing"

func TestEngineRejectsDirectoryLikeSingleFileRemotePathBeforeQueue(t *testing.T) {
	e := &Engine{}
	for _, remotePath := range []string{"", ".", "/", "/public_html/", `public_html\`} {
		if _, err := e.AddTransfer("upload", "local-file", remotePath, "local-root"); err == nil {
			t.Fatalf("AddTransfer accepted directory-like remote path %q", remotePath)
		}
	}
}

func TestEngineRejectsInvalidBatchRemotePathBeforeQueue(t *testing.T) {
	e := &Engine{}
	_, err := e.AddTransfers([]TransferRequest{
		{Direction: "upload", LocalPath: "one", RemotePath: "/public_html/one.txt", LocalRoot: "."},
		{Direction: "download", LocalPath: "two", RemotePath: "/", LocalRoot: "."},
	})
	if err == nil {
		t.Fatal("AddTransfers accepted a directory-like remote path")
	}
}
