package hgt

import (
	"os"
	"testing"
)

func TestDataDir(t *testing.T) {
	h, err := OpenDataDir("data", nil)
	if err != nil {
		t.Errorf("failed to open data dir: %s", err)
		return
	}
	defer h.Close()
	e, r, err := h.ElevationAt(48.7162, 21.2613)
	if err != nil {
		t.Errorf("failed to read elevation data: %s", err)
		return
	}
	if e != 205 {
		t.Errorf("invalid elevation data: %d", e)
		return
	}
	if r != Resolution1ArcSecond {
		t.Errorf("invalid elevation resolution: %d", r)
		return
	}
	_, _, err = h.ElevationAt(60.7162, 0)
	if err != ErrorOutOfRange {
		t.Errorf("invalid error: %s", err)
		return
	}
	_, _, err = h.ElevationAt(-56.7162, 0)
	if err != ErrorOutOfRange {
		t.Errorf("invalid error: %s", err)
		return
	}
	_, _, err = h.ElevationAt(0, 0)
	if !os.IsNotExist(err) {
		t.Errorf("invalid error: %s", err)
		return
	}
}

func TestFileRegexp(t *testing.T) {
	for _, s := range []string{"48E021", "B48E021", "N48C021"} {
		if fileRegexp.MatchString(s) {
			t.Errorf("invalid file regexp pattern: %v", fileRegexp)
			return
		}
	}
}

func TestSingleFile(t *testing.T) {
	h, err := Open("data/N48E021.hgt", nil)
	if err != nil {
		t.Errorf("failed to open data file: %s", err)
		return
	}
	defer h.Close()
	e, r, err := h.ElevationAt(48.7162, 21.2613)
	if err != nil {
		t.Errorf("failed to read elevation data: %s", err)
		return
	}
	if e != 205 {
		t.Errorf("invalid elevation data: %d", e)
		return
	}
	if r != Resolution1ArcSecond {
		t.Errorf("invalid elevation resolution: %d", r)
		return
	}
	cs := [][]float64{
		[]float64{60.7162, 0},
		[]float64{-56.7162, 0},
		[]float64{47.7162, 21.2613},
		[]float64{47.7162, 22.2613},
		[]float64{48.7162, 22.2613},
		[]float64{49.7162, 21.2613},
		[]float64{49.7162, 22.2613},
		[]float64{48.7162, -21.2613},
		[]float64{-48.7162, 21.2613},
		[]float64{0, 0},
	}
	for _, c := range cs {
		_, _, err = h.ElevationAt(c[0], c[1])
		if err != ErrorOutOfRange {
			t.Errorf("invalid error: %s", err)
			return
		}
	}
}

func TestSingleFileRangeValidator(t *testing.T) {
	h, err := Open("data/N48E021.hgt", &FileOptions{
		RangeValidator: func(lat, lon float64) error {
			if lat < 48.5 || lat >= 48.7 {
				return ErrorOutOfRange
			}
			return nil
		},
	})
	if err != nil {
		t.Errorf("failed to open data file: %s", err)
		return
	}
	defer h.Close()
	_, _, err = h.ElevationAt(48.5, 21.2613)
	if err != nil {
		t.Errorf("failed to read elevation data: %s", err)
		return
	}
	cs := [][]float64{
		[]float64{48.4, 21.2613},
		[]float64{48.7, 21.2613},
	}
	for _, c := range cs {
		_, _, err = h.ElevationAt(c[0], c[1])
		if err != ErrorOutOfRange {
			t.Errorf("invalid error: %s", err)
			return
		}
	}
}
