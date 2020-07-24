# HGT

Go module to read elevation data from SRTM HGT files.

## Supported data

- 30m (1 arc second)
- 90m (3 arc seconds)

## Usage

### Read from data directory

```go
h, err := hgt.OpenDataDir("data", nil)
if err != nil {
    log.Fatal(err)
}
e, r, err := h.ElevationAt(48.7162, 21.2613)
if err != nil {
    log.Println(err)
}
log.Printf("elevation=%d resolution=%d", e, r)
```

Example using custom cache with module [golang-lru](https://github.com/hashicorp/golang-lru).

```go
lruCache, err := lru.NewWithEvict(1000, func(key, value interface{}) {
    if file, ok := value.(*hgt.File); ok {
        file.Close()
    }
})
if err != nil {
    log.Fatal(err)
}
cache := &hgt.Cache{
    OnGet: func(key string) (*hgt.File, bool) {
        if v, ok := lruCache.Get(key); ok {
            return v.(*hgt.File), true
        }
        return nil, false
    },
    OnAdd: func(key string, value *hgt.File) {
        lruCache.Add(key, value)
    },
    OnClear: func() error {
        lruCache.Purge()
        return nil
    },
}
h, err := hgt.OpenDataDir("data", &hgt.DataDirOptions{
    Cache:          cache,
    RangeValidator: hgt.DefaultRangeValidator(),
})
if err != nil {
    log.Fatal(err)
}
defer h.Close()
e, r, err := h.ElevationAt(48.7162, 21.2613)
if err != nil {
    log.Println(err)
}
log.Printf("elevation=%d resolution=%d", e, r)
```

### Read from single file

```go
h, err := hgt.Open("data/N48E021.hgt", nil)
if err != nil {
    log.Fatal(err)
}
defer h.Close()
e, r, err := h.ElevationAt(48.7162, 21.2613)
if err != nil {
    log.Println(err)
}
log.Printf("elevation=%d resolution=%d", e, r)
```

### Range validator

Default range validator respects SRTM latitude limits. For other datasets it is possible to configure custom validator.

```go
h, err := hgt.OpenDataDir("data", &hgt.DataDirOptions{
    Cache: hgt.DefaultCache(),
    RangeValidator: func(lat, lon float64) error {
        if lat < 48.5 || lat >= 48.7 {
            return hgt.ErrorOutOfRange
        }
        return nil
    },
})
if err != nil {
    log.Fatal(err)
}
```

## License

Licensed under MIT license.
