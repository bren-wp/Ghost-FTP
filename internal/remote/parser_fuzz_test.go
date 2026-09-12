package remote

import "testing"

func FuzzRemoteListingParsers(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("type=file;size=42;modify=20260912103045; report.txt\r\n"),
		[]byte("type=dir;modify=20260912103045; public_html\r\n"),
		[]byte("type=OS.unix=slink;size=0; link -> target\r\n"),
		[]byte("-rw-r--r-- 1 user group 42 Sep 12 10:30 report.txt"),
		[]byte("lrwxrwxrwx 1 user group 6 Sep 12 10:30 link -> target"),
		[]byte("09-12-26  10:30AM  <DIR>          public_html"),
		[]byte("09-12-26  10:30AM  123456 report.zip"),
		[]byte("type=file;size=999999999999999999999999999999; huge.bin"),
		[]byte("\x00\xff\xfe malformed"),
		{},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		line := string(data)

		if item, ok := parseMLSDLine(line); ok {
			if item.Name == "" || item.Name == "." || item.Name == ".." {
				t.Fatalf("MLSD parser accepted unsafe empty/dot name %q", item.Name)
			}
			if item.Size < 0 {
				t.Fatalf("MLSD parser produced negative size %d", item.Size)
			}
		}

		items, _, err := parseMLSD(data)
		if err == nil {
			if len(items) > maxDirectoryItems {
				t.Fatalf("MLSD parser exceeded item limit: %d", len(items))
			}
			for _, item := range items {
				if item.Name == "" || item.Name == "." || item.Name == ".." {
					t.Fatalf("MLSD parser returned unsafe name %q", item.Name)
				}
				if item.Size < 0 {
					t.Fatalf("MLSD parser returned negative size %d", item.Size)
				}
			}
		}

		if item, ok := parseListLine(line); ok && item.Size < 0 {
			t.Fatalf("LIST parser produced negative size %d", item.Size)
		}
	})
}
