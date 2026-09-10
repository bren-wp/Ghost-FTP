package directorycompare

import (
	"reflect"
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func item(name string, size int64, modified time.Time) model.Item {
	return model.Item{Name: name, Size: size, Modified: modified}
}

func TestCompareClassifiesPresenceAndStableOrdering(t *testing.T) {
	local := []model.Item{
		{Name: "z-local", Size: 1},
		{Name: "shared", IsDirectory: true},
	}
	remote := []model.Item{
		{Name: "a-remote", Size: 2},
		{Name: "shared", IsDirectory: true},
	}

	got, err := Compare(local, remote, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		name   string
		status Status
	}{
		{"a-remote", StatusRemoteOnly},
		{"shared", StatusSame},
		{"z-local", StatusLocalOnly},
	}
	if len(got) != len(want) {
		t.Fatalf("entries=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i].name || got[i].Status != want[i].status {
			t.Fatalf("entry[%d]=%q/%q want %q/%q", i, got[i].Name, got[i].Status, want[i].name, want[i].status)
		}
	}
}

func TestCompareUsesExactNamesWithoutCaseFolding(t *testing.T) {
	got, err := Compare(
		[]model.Item{{Name: "File.txt", Size: 1}},
		[]model.Item{{Name: "file.txt", Size: 1}},
		Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("entries=%d want=2", len(got))
	}
	if got[0].Name != "File.txt" || got[0].Status != StatusLocalOnly {
		t.Fatalf("first=%+v", got[0])
	}
	if got[1].Name != "file.txt" || got[1].Status != StatusRemoteOnly {
		t.Fatalf("second=%+v", got[1])
	}
}

func TestCompareTreatsUnknownTimestampsConservatively(t *testing.T) {
	known := time.Date(2026, 9, 10, 1, 2, 3, 0, time.UTC)
	tests := []struct {
		name       string
		localSize  int64
		remoteSize int64
		localTime  time.Time
		remoteTime time.Time
		want       Status
	}{
		{"same size remote time unknown", 10, 10, known, time.Time{}, StatusUnknown},
		{"same size both times unknown", 10, 10, time.Time{}, time.Time{}, StatusUnknown},
		{"different size remote time unknown", 10, 20, known, time.Time{}, StatusConflict},
		{"different size both times unknown", 10, 20, time.Time{}, time.Time{}, StatusConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Compare(
				[]model.Item{item("a.txt", tc.localSize, tc.localTime)},
				[]model.Item{item("a.txt", tc.remoteSize, tc.remoteTime)},
				Options{},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].Status != tc.want {
				t.Fatalf("got=%+v want=%q", got, tc.want)
			}
		})
	}
}

func TestCompareTimestampToleranceAndNewerDirection(t *testing.T) {
	base := time.Date(2026, 9, 10, 1, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		localSize  int64
		remoteSize int64
		localTime  time.Time
		remoteTime time.Time
		want       Status
	}{
		{"same metadata within tolerance", 10, 10, base.Add(time.Second), base, StatusSame},
		{"size conflict within tolerance", 10, 20, base.Add(time.Second), base, StatusConflict},
		{"local newer beyond tolerance", 10, 20, base.Add(3 * time.Second), base, StatusNewerLocal},
		{"remote newer beyond tolerance", 10, 10, base, base.Add(3 * time.Second), StatusNewerRemote},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Compare(
				[]model.Item{item("a.txt", tc.localSize, tc.localTime)},
				[]model.Item{item("a.txt", tc.remoteSize, tc.remoteTime)},
				Options{},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].Status != tc.want {
				t.Fatalf("got=%+v want=%q", got, tc.want)
			}
		})
	}
}

func TestCompareSymlinkAndTypeMismatchAreNeverSame(t *testing.T) {
	got, err := Compare(
		[]model.Item{
			{Name: "link", IsSymlink: true},
			{Name: "type", IsDirectory: true},
		},
		[]model.Item{
			{Name: "link", IsSymlink: true},
			{Name: "type", IsDirectory: false},
		},
		Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Status != StatusUnknown {
		t.Fatalf("symlink status=%q", got[0].Status)
	}
	if got[1].Status != StatusConflict {
		t.Fatalf("type mismatch status=%q", got[1].Status)
	}
}

func TestCompareDuplicateExactNamesFailClosedToConflict(t *testing.T) {
	got, err := Compare(
		[]model.Item{{Name: "dup"}, {Name: "dup"}},
		[]model.Item{{Name: "dup"}},
		Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Status != StatusConflict {
		t.Fatalf("got=%+v", got)
	}
}

func TestCompareDoesNotMutateInputs(t *testing.T) {
	local := []model.Item{{Name: "b"}, {Name: "a"}}
	remote := []model.Item{{Name: "c"}}
	localBefore := append([]model.Item(nil), local...)
	remoteBefore := append([]model.Item(nil), remote...)
	if _, err := Compare(local, remote, Options{}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(local, localBefore) || !reflect.DeepEqual(remote, remoteBefore) {
		t.Fatalf("inputs mutated: local=%+v remote=%+v", local, remote)
	}
}

func TestNormalizeOptionsRejectsUnsafeTolerance(t *testing.T) {
	if _, err := NormalizeOptions(Options{TimestampTolerance: -time.Second}); err == nil {
		t.Fatal("negative tolerance accepted")
	}
	if _, err := NormalizeOptions(Options{TimestampTolerance: MaxTimestampTolerance + time.Second}); err == nil {
		t.Fatal("oversized tolerance accepted")
	}
	got, err := NormalizeOptions(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got.TimestampTolerance != DefaultTimestampTolerance {
		t.Fatalf("default tolerance=%s", got.TimestampTolerance)
	}
}
