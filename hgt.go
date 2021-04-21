package hgt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	// Resolution1ArcSecond ...
	Resolution1ArcSecond = 1
	// Resolution3ArcSecond ...
	Resolution3ArcSecond = 3
)

var (
	// ErrorUnsupportedResolution ...
	ErrorUnsupportedResolution = errors.New("unsupported resolution")
	// ErrorInvalidFileName ...
	ErrorInvalidFileName = errors.New("invalid file name")
	// ErrorOutOfRange ...
	ErrorOutOfRange = errors.New("out of range")
)

var fileRegexp = regexp.MustCompile(`[NS][\d]{2}[EW][\d]{3}`)

type (
	// FileOptions ...
	FileOptions struct {
		RangeValidator       func(lat, lon float64) error
		IgnoreTileValidation bool
	}
	// DataDirOptions ...
	DataDirOptions struct {
		Cache          *Cache
		RangeValidator func(lat, lon float64) error
	}
	// File ...
	File struct {
		file       *os.File
		info       os.FileInfo
		resolution int
		size       int
		options    *FileOptions
	}
	// DataDir ...
	DataDir struct {
		dir     string
		options *DataDirOptions
	}
	// Cache ...
	Cache struct {
		OnGet   func(key string) (*File, bool)
		OnAdd   func(key string, value *File)
		OnClear func() error
	}
)

func pad(str string, length int, pad string) string {
	if length <= 0 || len(str) >= length || len(pad) == 0 {
		return str
	}
	return strings.Repeat(pad, length-len(str)) + str
}

func cpad(coord float64, length int) string {
	return pad(strconv.Itoa(int(math.Abs(math.Floor(coord)))), length, "0")
}

func fileName(lat, lon float64) string {
	ns := "N"
	if lat < 0 {
		ns = "S"
	}
	ew := "E"
	if lon < 0 {
		ew = "W"
	}
	return ns + cpad(lat, 2) + ew + cpad(lon, 3) + ".hgt"
}

func byteOffset(lat, lon float64, samples int64) int64 {
	x := int64(math.Floor((lon - math.Floor(lon)) * float64(samples)))
	y := int64(math.Floor((lat - math.Floor(lat)) * float64(samples)))
	return (x + (samples-y-1)*samples) * 2
}

// Open ...
func Open(name string, options *FileOptions) (*File, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	i, err := f.Stat()
	if err != nil {
		return nil, err
	}
	var r int
	var s int
	switch i.Size() {
	case 1442401 * 2:
		r = Resolution3ArcSecond
		s = 1201
	case 12967201 * 2:
		r = Resolution1ArcSecond
		s = 3601
	default:
		return nil, ErrorUnsupportedResolution
	}
	if options == nil {
		options = &FileOptions{
			RangeValidator: DefaultRangeValidator(),
		}
	}
	return &File{
		file:       f,
		info:       i,
		resolution: r,
		size:       s,
		options:    options,
	}, nil
}

// Close ...
func (f *File) Close() error {
	return f.file.Close()
}

// File ...
func (f *File) File() *os.File {
	return f.file
}

// ElevationAt ...
func (f *File) ElevationAt(lat, lon float64) (int16, int, error) {
	if !f.options.IgnoreTileValidation {
		if f.options.RangeValidator != nil {
			err := f.options.RangeValidator(lat, lon)
			if err != nil {
				return 0, 0, err
			}
		}
		name := strings.TrimSuffix(f.info.Name(), filepath.Ext(f.info.Name()))
		if len(name) != 7 {
			return 0, 0, ErrorInvalidFileName
		}
		if !fileRegexp.MatchString(name) {
			return 0, 0, ErrorInvalidFileName
		}
		minLat, _ := strconv.ParseFloat(name[1:3], 64)
		minLon, _ := strconv.ParseFloat(name[4:7], 64)
		if name[:1] == "N" && (minLat > lat || minLat+1 <= lat) {
			return 0, 0, ErrorOutOfRange
		} else if name[:1] == "S" && (minLat > lat*-1 || minLat+1 <= lat*-1) {
			return 0, 0, ErrorOutOfRange
		} else if name[3:4] == "E" && (minLon > lon || minLon+1 <= lon) {
			return 0, 0, ErrorOutOfRange
		} else if name[3:4] == "W" && (minLon > lon*-1 || minLon+1 <= lon*-1) {
			return 0, 0, ErrorOutOfRange
		}
	}
	b := make([]byte, 2)
	_, err := f.file.ReadAt(b, byteOffset(lat, lon, int64(f.size)))
	if err != nil {
		return 0, 0, err
	}
	var e int16
	err = binary.Read(bytes.NewBuffer(b), binary.BigEndian, &e)
	if err != nil {
		return 0, 0, err
	}
	if e == -32768 {
		return 0, 0, errors.New("void")
	}
	return e, f.resolution, nil
}

// DefaultRangeValidator ...
func DefaultRangeValidator() func(lat, lon float64) error {
	return func(lat, lon float64) error {
		if lat < -56.0 || lat >= 60.0 {
			return ErrorOutOfRange
		}
		return nil
	}
}

// DefaultCache ...
func DefaultCache() *Cache {
	cacheMap := &sync.Map{}
	cache := &Cache{
		OnGet: func(key string) (*File, bool) {
			if v, ok := cacheMap.Load(key); ok {
				return v.(*File), true
			}
			return nil, false
		},
		OnAdd: func(key string, value *File) {
			cacheMap.Store(key, value)
		},
		OnClear: func() error {
			cacheMap.Range(func(key, value interface{}) bool {
				if v, ok := cacheMap.Load(key); ok {
					v.(*File).Close()
					cacheMap.Delete(key)
				}
				return true
			})
			return nil
		},
	}
	return cache
}

// OpenDataDir ...
func OpenDataDir(dir string, options *DataDirOptions) (*DataDir, error) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, err
	}
	if options == nil {
		options = &DataDirOptions{
			Cache:          DefaultCache(),
			RangeValidator: DefaultRangeValidator(),
		}
	}
	return &DataDir{
		dir:     dir,
		options: options,
	}, nil
}

// Close ...
func (h *DataDir) Close() error {
	if h.options.Cache != nil {
		return h.options.Cache.OnClear()
	}
	return nil
}

// ElevationAt ...
func (h *DataDir) ElevationAt(lat, lon float64) (int16, int, error) {
	if h.options.RangeValidator != nil {
		err := h.options.RangeValidator(lat, lon)
		if err != nil {
			return 0, 0, err
		}
	}
	name := fileName(lat, lon)
	if h.options.Cache != nil {
		var f *File
		f, ok := h.options.Cache.OnGet(name)
		if !ok {
			var err error
			f, err = Open(h.dir+string(os.PathSeparator)+name, &FileOptions{
				IgnoreTileValidation: true,
			})
			if err != nil {
				return 0, 0, err
			}
			h.options.Cache.OnAdd(name, f)
		}
		return f.ElevationAt(lat, lon)
	}
	f, err := Open(h.dir+string(os.PathSeparator)+name, &FileOptions{
		IgnoreTileValidation: true,
	})
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	return f.ElevationAt(lat, lon)
}
