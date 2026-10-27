package wdc

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/wowsims/forever/tools/db2tool/dbd"
)

// The two table hashes ApplyHotfixes special-cases are known constants, so they
// pin the SStrHash port without needing any game files data. A DBCache record is
// matched to a table by comparing this hash against the WDC5 header's
// TableHash, so a wrong hash silently drops every hotfix for a table.
func TestSStrHashKnownTableHashes(t *testing.T) {
	cases := map[string]uint32{
		"TACTKEY":       0xDF2F53CF,
		"BROADCASTTEXT": 0x021826BB,
	}
	for name, want := range cases {
		if got := SStrHash(name); got != want {
			t.Errorf("SStrHash(%q) = 0x%08X, want 0x%08X", name, got, want)
		}
	}
	// Callers must uppercase; the hash is case-sensitive and the lowercase form
	// is a different value.
	if SStrHash("tactkey") == SStrHash("TACTKEY") {
		t.Error("SStrHash is unexpectedly case-insensitive")
	}
	if SStrHash("") != 0x7fed7fed {
		t.Errorf("SStrHash(\"\") = 0x%08X, want the 0x7fed7fed seed", SStrHash(""))
	}
}

// xfthEntry builds one DBCache record: per-entry magic, the 28-byte header, then
// the payload.
func xfthEntry(pushID int32, tableHash uint32, recordID int32, status byte, data []byte) []byte {
	b := make([]byte, 0, 32+len(data))
	b = append(b, []byte(hotfixMagic)...)
	var hdr [28]byte
	binary.LittleEndian.PutUint32(hdr[0:], 1) // RegionID
	binary.LittleEndian.PutUint32(hdr[4:], uint32(pushID))
	binary.LittleEndian.PutUint32(hdr[8:], 42) // UniqueID
	binary.LittleEndian.PutUint32(hdr[12:], tableHash)
	binary.LittleEndian.PutUint32(hdr[16:], uint32(recordID))
	binary.LittleEndian.PutUint32(hdr[20:], uint32(len(data)))
	hdr[24] = status
	b = append(b, hdr[:]...)
	return append(b, data...)
}

func xfthFile(version, build int32, entries ...[]byte) []byte {
	buf := make([]byte, 44)
	copy(buf, hotfixMagic)
	binary.LittleEndian.PutUint32(buf[4:], uint32(version))
	binary.LittleEndian.PutUint32(buf[8:], uint32(build))
	// buf[12:44] is the v>=5 32-byte hash, skipped by the parser.
	for _, e := range entries {
		buf = append(buf, e...)
	}
	return buf
}

func TestParseHotfixV9(t *testing.T) {
	const build = 68571
	raw := xfthFile(9, build,
		xfthEntry(100, 0xAABBCCDD, 7, 1, []byte{1, 2, 3, 4}),
		xfthEntry(101, 0xAABBCCDD, 8, 0, nil), // delete: status 0, no payload
	)

	h, err := parseHotfix(raw)
	if err != nil {
		t.Fatal(err)
	}
	if h.Version != 9 || h.BuildID != build {
		t.Fatalf("header = version %d build %d, want 9 / %d", h.Version, h.BuildID, build)
	}
	if len(h.records) != 2 {
		t.Fatalf("parsed %d records, want 2", len(h.records))
	}

	first := h.records[0]
	if first.PushID != 100 || first.TableHash != 0xAABBCCDD || first.RecordID != 7 {
		t.Errorf("first record = %+v", first)
	}
	if !first.IsValid {
		t.Error("status byte 1 must parse as valid")
	}
	if first.DataSize != 4 || string(first.Data) != "\x01\x02\x03\x04" {
		t.Errorf("first payload = %v (size %d), want 4 bytes 1..4", first.Data, first.DataSize)
	}
	if h.records[1].IsValid {
		t.Error("status byte 0 must parse as invalid")
	}
	if h.records[1].DataSize != 0 {
		t.Errorf("second DataSize = %d, want 0", h.records[1].DataSize)
	}
}

// The payload must be capped so a later append cannot reach into the following
// record's bytes.
func TestParseHotfixPayloadIsCapped(t *testing.T) {
	raw := xfthFile(9, 1,
		xfthEntry(1, 0x1, 1, 1, []byte{9, 9}),
		xfthEntry(2, 0x1, 2, 1, []byte{7, 7}),
	)
	h, err := parseHotfix(raw)
	if err != nil {
		t.Fatal(err)
	}
	_ = append(h.records[0].Data, 0xff)
	if h.records[1].Data[0] != 7 {
		t.Errorf("appending to record 0's payload corrupted record 1: %v", h.records[1].Data)
	}
}

func TestParseHotfixRejectsBadInput(t *testing.T) {
	badMagic := xfthFile(9, 1)
	copy(badMagic, "ZZZZ")

	tests := map[string][]byte{
		"short header":     []byte("XFT"),
		"bad magic":        badMagic,
		"truncated ext":    append([]byte(hotfixMagic), 0x09, 0, 0, 0, 0x01, 0, 0, 0),
		"bad entry magic":  append(xfthFile(9, 1), []byte("NOPE????")...),
		"truncated data":   append(xfthFile(9, 1), xfthEntry(1, 0x1, 1, 1, []byte{1, 2, 3, 4})[:34]...),
		"unsupported ver8": xfthFile(8, 1),
	}

	for name, raw := range tests {
		if _, err := parseHotfix(raw); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

// Records with an unsupported DataSize must not be silently accepted: a
// negative size would otherwise index backwards.
func TestParseHotfixRejectsNegativeDataSize(t *testing.T) {
	e := xfthEntry(1, 0x1, 1, 1, nil)
	binary.LittleEndian.PutUint32(e[4+20:], 0xFFFFFFFF) // DataSize = -1
	if _, err := parseHotfix(xfthFile(9, 1, e)); err == nil {
		t.Error("expected an error for a negative DataSize")
	}
}

func TestCombineDedupsOnFullRecordIdentity(t *testing.T) {
	rec := func(push int32, data []byte) []byte { return xfthEntry(push, 0x1, 5, 1, data) }

	base, err := parseHotfix(xfthFile(9, 1, rec(1, []byte{1})))
	if err != nil {
		t.Fatal(err)
	}
	// Identical record: dropped.
	same, _ := parseHotfix(xfthFile(9, 1, rec(1, []byte{1})))
	base.Combine(same)
	if len(base.records) != 1 {
		t.Fatalf("identical record was not deduped: %d records", len(base.records))
	}
	// Same 5-tuple but different payload bytes: kept, because identity includes
	// the data.
	other, _ := parseHotfix(xfthFile(9, 1, rec(1, []byte{2})))
	base.Combine(other)
	if len(base.records) != 2 {
		t.Fatalf("record differing only in payload was dropped: %d records", len(base.records))
	}
	// A different build is ignored entirely.
	otherBuild, _ := parseHotfix(xfthFile(9, 2, rec(9, []byte{3})))
	base.Combine(otherBuild)
	if len(base.records) != 2 {
		t.Fatalf("record from another build was merged in: %d records", len(base.records))
	}
}

func TestHighestPushCountsOnlyServerPushes(t *testing.T) {
	h, err := parseHotfix(xfthFile(9, 1,
		xfthEntry(-1, 0x1, 1, 1, []byte{1}),
		xfthEntry(112405, 0x1, 2, 1, []byte{1}),
		xfthEntry(112426, 0x1, 3, 1, []byte{1}),
		xfthEntry(0x01000005, 0x1, 4, 1, []byte{1}),
	))
	if err != nil {
		t.Fatal(err)
	}
	if got := h.HighestPush(); got != 112426 {
		t.Errorf("HighestPush = %d, want 112426", got)
	}

	onDemand, err := parseHotfix(xfthFile(9, 1, xfthEntry(-1, 0x1, 1, 1, []byte{1}), xfthEntry(0x01000005, 0x1, 2, 1, []byte{1})))
	if err != nil {
		t.Fatal(err)
	}
	if got := onDemand.HighestPush(); got != 0 {
		t.Errorf("HighestPush of on-demand rows alone = %d, want 0", got)
	}
}

func TestOrderHotfixReaders(t *testing.T) {
	const own = 70630
	readers := map[uint32]*HotfixReader{}
	for _, build := range []int32{70532, own, 70450, 70700} {
		r, err := parseHotfix(xfthFile(9, build, xfthEntry(1, 0x1, 1, 1, []byte{1})))
		if err != nil {
			t.Fatal(err)
		}
		readers[uint32(build)] = r
	}
	builds := func(ordered []*HotfixReader) []int32 {
		out := make([]int32, len(ordered))
		for i, r := range ordered {
			out[i] = r.BuildID
		}
		return out
	}

	if got, want := builds(OrderHotfixReaders(readers, own, true)), []int32{70450, 70532, 70700, own}; !slices.Equal(got, want) {
		t.Errorf("carried order = %v, want %v (other builds ascending, own build last)", got, want)
	}
	if got, want := builds(OrderHotfixReaders(readers, own, false)), []int32{own}; !slices.Equal(got, want) {
		t.Errorf("order without carry = %v, want %v", got, want)
	}

	delete(readers, own)
	if got := OrderHotfixReaders(readers, own, false); len(got) != 0 {
		t.Errorf("without carry and no cache for the build, got builds %v, want none", builds(got))
	}
	if got, want := builds(OrderHotfixReaders(readers, own, true)), []int32{70450, 70532, 70700}; !slices.Equal(got, want) {
		t.Errorf("carried order with no own cache = %v, want %v", got, want)
	}
}

func TestHotfixLayoutMatches(t *testing.T) {
	builds := func(ns ...uint32) []dbd.Build {
		out := make([]dbd.Build, len(ns))
		for i, n := range ns {
			out[i] = dbd.Build{Expansion: 1, Major: 60, Minor: 1, Build: n}
		}
		return out
	}
	oldFields := []dbd.Definition{{Name: "ID", IsID: true, Size: 32}, {Name: "Field", Size: 16}}
	newFields := []dbd.Definition{{Name: "ID", IsID: true, Size: 32}, {Name: "Field", Size: 32}}

	hashed := dbd.DBDefinition{VersionDefinitions: []dbd.VersionDefinitions{
		{LayoutHashes: []string{"0A1B2C3D"}, Builds: builds(70100), Definitions: oldFields},
		{LayoutHashes: []string{"11223344"}, Builds: builds(70532, 70630), Definitions: newFields},
	}}
	unhashed := dbd.DBDefinition{VersionDefinitions: []dbd.VersionDefinitions{
		{Builds: builds(70100), Definitions: oldFields},
		{Builds: builds(70532, 70630), Definitions: newFields},
	}}

	for name, def := range map[string]dbd.DBDefinition{"layout hashes": hashed, "no layout hashes": unhashed} {
		version, err := dbd.SelectVersion(def, 70630)
		if err != nil {
			t.Fatal(err)
		}
		if !HotfixLayoutMatches(def, version, 70532) {
			t.Errorf("%s: a build in the same version block must carry", name)
		}
		if HotfixLayoutMatches(def, version, 70100) {
			t.Errorf("%s: a build in another version block must not carry", name)
		}
		if HotfixLayoutMatches(def, version, 69000) {
			t.Errorf("%s: a build the definition does not list must not carry", name)
		}
	}

	// A .dbd may give two blocks the same columns and different layout hashes;
	// the guard still takes them as different layouts.
	sameColumns := dbd.DBDefinition{VersionDefinitions: []dbd.VersionDefinitions{
		{LayoutHashes: []string{"0A1B2C3D"}, Builds: builds(70532), Definitions: newFields},
		{LayoutHashes: []string{"11223344"}, Builds: builds(70630), Definitions: newFields},
	}}
	version, err := dbd.SelectVersion(sameColumns, 70630)
	if err != nil {
		t.Fatal(err)
	}
	if HotfixLayoutMatches(sameColumns, version, 70532) {
		t.Error("a build in another block with the same columns must not carry")
	}
}

// The WoW install is often reached through a symlink (the checked-in BaseDir
// pointing into a wine prefix). WalkDir does not descend into a symlinked
// root, so without resolving it first the scan finds nothing and the run is
// silently hotfix-free.
func TestLoadHotfixCachesFollowsSymlinkedRoots(t *testing.T) {
	dir := t.TempDir()
	install := filepath.Join(dir, "install")
	cacheDir := filepath.Join(install, "_classic_beta_", "Cache", "ADB", "enUS")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := xfthFile(9, 69913, xfthEntry(1, 0x919BE54E, 276631, 1, []byte{1, 2, 3}))
	if err := os.WriteFile(filepath.Join(cacheDir, "DBCache.bin"), file, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(install, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	for _, root := range []string{install, link} {
		readers, err := LoadHotfixCaches(filepath.Join(dir, "no-caches-dir"), root)
		if err != nil {
			t.Fatalf("LoadHotfixCaches(%s): %v", root, err)
		}
		r := readers[69913]
		if r == nil || len(r.records) != 1 {
			t.Errorf("LoadHotfixCaches(%s): want 1 record for build 69913, got %+v", root, readers)
		}
	}
}
