package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var ErrCancelled = errors.New("cancelled")

// MarkerName is the file in the root of every disc that identifies it.
func MarkerName(disc int) string { return "PHYSICAL_RELEASE_DISC_" + strconv.Itoa(disc) + ".txt" }

// IsDiscRoot reports whether root holds disc number `disc` of this release.
func IsDiscRoot(c *Config, root string, disc int) bool {
	if root == "" {
		return false
	}
	if b, err := os.ReadFile(filepath.Join(root, MarkerName(disc))); err == nil {
		if c.PackageID == "" {
			return true
		}
		for _, ln := range strings.Split(string(b), "\n") {
			if strings.TrimSpace(ln) == "PackageID="+c.PackageID {
				return true
			}
		}
		return false
	}
	// Marker missing (e.g. a hand-made folder): accept it if the parts are there.
	if p, ok := c.FirstPartOnDisc(disc); ok {
		if _, err := os.Stat(filepath.Join(root, p.Dir, p.Name)); err == nil {
			return true
		}
	}
	return false
}

// UI is what the staging engine needs from the front end.
type UI interface {
	// AskDisc blocks until disc `disc` is available and returns its root, or ErrCancelled.
	AskDisc(disc int) (string, error)
	Progress(done, total int64, file string)
	Cancelled() bool
}

const bufSize = 4 << 20

type copier struct {
	ui          UI
	done, total int64
}

// copyFile appends src to dst starting at src offset `from`; returns bytes copied.
func (cp *copier) copyInto(dst *os.File, src string, from int64, label string) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	if from > 0 {
		if _, err := in.Seek(from, io.SeekStart); err != nil {
			return 0, err
		}
	}
	buf := make([]byte, bufSize)
	var n int64
	for {
		if cp.ui.Cancelled() {
			return n, ErrCancelled
		}
		r, rerr := in.Read(buf)
		if r > 0 {
			if _, werr := dst.Write(buf[:r]); werr != nil {
				return n, werr
			}
			n += int64(r)
			cp.done += int64(r)
			cp.ui.Progress(cp.done, cp.total, label)
		}
		if rerr == io.EOF {
			return n, nil
		}
		if rerr != nil {
			return n, rerr
		}
	}
}

// Stage copies the installer from baseDir and rebuilds every split BIN into
// destDir. Discs are requested one after another, so a single optical drive is
// enough; a disc is never needed after the engine has moved on to the next one.
func Stage(c *Config, plan []SplitFile, baseDir, destDir string, ui UI) error {
	cp := &copier{ui: ui, total: c.StagingBytes}
	roots := map[int]string{}
	if IsDiscRoot(c, baseDir, 1) {
		roots[1] = baseDir
	}
	ensure := func(disc int) (string, error) {
		if r := roots[disc]; r != "" && IsDiscRoot(c, r, disc) {
			return r, nil
		}
		delete(roots, disc)
		r, err := ui.AskDisc(disc)
		if err != nil {
			return "", err
		}
		roots[disc] = r
		return r, nil
	}

	// 1. installer EXE (always from the disc/folder the launcher runs from)
	ui.Progress(0, cp.total, c.Installer)
	instDst, err := os.Create(filepath.Join(destDir, filepath.Base(c.Installer)))
	if err != nil {
		return err
	}
	_, err = cp.copyInto(instDst, filepath.Join(baseDir, c.Installer), 0, c.Installer)
	if cerr := instDst.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}

	// 2. BIN files, part by part, in disc order
	for _, sf := range plan {
		out, err := os.Create(filepath.Join(destDir, filepath.Base(sf.Original)))
		if err != nil {
			return err
		}
		for _, p := range sf.Parts {
			var copied int64
			for attempt := 0; ; attempt++ {
				root, aerr := ensure(p.Disc)
				if aerr != nil {
					out.Close()
					return aerr
				}
				n, cerr := cp.copyInto(out, filepath.Join(root, p.Dir, p.Name), copied, p.Name)
				copied += n
				if cerr == nil {
					break
				}
				if errors.Is(cerr, ErrCancelled) || attempt >= 3 {
					out.Close()
					return fmt.Errorf("%s: %w", p.Name, cerr)
				}
				delete(roots, p.Disc) // disc removed or unreadable: ask again, resume at `copied`
			}
			if p.Size > 0 && copied != p.Size {
				out.Close()
				return fmt.Errorf("%s: size mismatch (%d of %d bytes)", p.Name, copied, p.Size)
			}
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}
