package core

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeUI struct {
	t        *testing.T
	discs    map[int]string
	hidden   map[int]string
	asked    []int
	cancelAt int
}

func (f *fakeUI) AskDisc(d int) (string, error) {
	f.asked = append(f.asked, d)
	// "swap the disc": every other disc disappears, like with one drive
	for n, p := range f.discs {
		if n != d {
			if _, ok := f.hidden[n]; !ok {
				os.Rename(p, p+".out")
				f.hidden[n] = p + ".out"
			}
		}
	}
	if h, ok := f.hidden[d]; ok {
		os.Rename(h, f.discs[d])
		delete(f.hidden, d)
	}
	return f.discs[d], nil
}
func (f *fakeUI) Progress(a, b int64, s string) {}
func (f *fakeUI) Cancelled() bool               { return false }

func TestStageSingleDrive(t *testing.T) {
	tmp := t.TempDir()
	mk := func(p string, b []byte) { os.MkdirAll(filepath.Dir(p), 0o755); os.WriteFile(p, b, 0o644) }
	inst := make([]byte, 3000)
	rand.Read(inst)
	bin1 := make([]byte, 25000)
	rand.Read(bin1)
	bin2 := make([]byte, 11111)
	rand.Read(bin2)
	all := append(append([]byte{}, bin1...), bin2...)
	// split all of bin1+bin2 over 4 discs in 9000 byte parts (bin boundary inside a disc)
	d := map[int]string{}
	for i := 1; i <= 4; i++ {
		d[i] = filepath.Join(tmp, fmt.Sprint("disc", i))
		mk(filepath.Join(d[i], MarkerName(i)), []byte("PHYSICAL RELEASE STUDIO\r\nPackageID=abc123\r\n"))
	}
	mk(filepath.Join(d[1], "setup.exe"), inst)
	ini := "[Game]\nTitle=T\nInstaller=setup.exe\n[Disc]\nPackageID=abc123\nMediaCount=4\n[Staging]\nStagingBytes=" + fmt.Sprint(len(inst)+len(all)) + "\n[SplitFiles]\n"
	off := 0
	var l1, l2 []string
	part := 0
	for off < len(bin1) {
		n := min(9000, len(bin1)-off)
		part++
		name := fmt.Sprintf("setup-1.bin.part%03d", part)
		disc := part
		mk(filepath.Join(d[disc], "SplitFiles", name), bin1[off:off+n])
		l1 = append(l1, fmt.Sprintf("%s@%d:%d", name, disc, n))
		off += n
	}
	mk(filepath.Join(d[3], "SplitFiles", "setup-2.bin.part001"), bin2[:5000])
	mk(filepath.Join(d[4], "SplitFiles", "setup-2.bin.part002"), bin2[5000:])
	l2 = []string{"setup-2.bin.part001@3:5000", "setup-2.bin.part002@4:6111"}
	ini += "File1=setup-1.bin|" + strings.Join(l1, ",") + "\nFile2=setup-2.bin|" + strings.Join(l2, ",") + "\n"
	c := ParseINI(strings.NewReader(ini))
	if len(c.Splits) != 2 || c.PackageID != "abc123" || c.StagingBytes == 0 {
		t.Fatalf("bad parse %+v", c)
	}
	dest := filepath.Join(tmp, "dest")
	os.MkdirAll(dest, 0o755)
	ui := &fakeUI{t: t, discs: d, hidden: map[int]string{}}
	if err := Stage(&c, c.Plan(true), d[1], dest, ui); err != nil {
		t.Fatal(err)
	}
	got1, _ := os.ReadFile(filepath.Join(dest, "setup-1.bin"))
	got2, _ := os.ReadFile(filepath.Join(dest, "setup-2.bin"))
	gi, _ := os.ReadFile(filepath.Join(dest, "setup.exe"))
	if !bytes.Equal(got1, bin1) || !bytes.Equal(got2, bin2) || !bytes.Equal(gi, inst) {
		t.Fatal("content mismatch")
	}
	t.Log("asked for discs:", ui.asked)
	for i, v := range []int{2, 3, 4} {
		if ui.asked[i] != v {
			t.Fatalf("order %v", ui.asked)
		}
	}
}

func TestWrongPackageRejected(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, MarkerName(2)), []byte("PackageID=other"), 0o644)
	c := Config{PackageID: "abc"}
	if IsDiscRoot(&c, tmp, 2) {
		t.Fatal("should reject")
	}
}

func TestQuickModeRootBins(t *testing.T) {
	tmp := t.TempDir()
	d1, d2 := filepath.Join(tmp, "d1"), filepath.Join(tmp, "d2")
	os.MkdirAll(d1, 0o755)
	os.MkdirAll(d2, 0o755)
	os.WriteFile(filepath.Join(d1, MarkerName(1)), []byte("PackageID=z"), 0o644)
	os.WriteFile(filepath.Join(d2, MarkerName(2)), []byte("PackageID=z"), 0o644)
	os.WriteFile(filepath.Join(d1, "s.exe"), []byte("EXE"), 0o644)
	os.WriteFile(filepath.Join(d1, "s-1.bin"), []byte("AAAA"), 0o644)
	os.WriteFile(filepath.Join(d2, "s-2.bin"), []byte("BB"), 0o644)
	c := ParseINI(strings.NewReader("[Game]\nInstaller=s.exe\n[Disc]\nPackageID=z\n[Bins]\nFile1=s-1.bin|s-1.bin@1:4\nFile2=s-2.bin|s-2.bin@2:2\n"))
	if c.Forced() || len(c.Bins) != 2 {
		t.Fatal("parse")
	}
	dest := filepath.Join(tmp, "dest")
	os.MkdirAll(dest, 0o755)
	ui := &fakeUI{t: t, discs: map[int]string{1: d1, 2: d2}, hidden: map[int]string{}}
	if err := Stage(&c, c.Plan(false), d1, dest, ui); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dest, "s-2.bin"))
	if string(b) != "BB" {
		t.Fatal("content")
	}
}
