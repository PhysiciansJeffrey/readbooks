package main

import (
	"ReadBooks/internal/storage"
	"bytes"
	"embed"
	_ "embed"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// simple thumbnail cache
var (
	thumbCache sync.Map // path → []byte
	thumbMu    sync.Mutex
	thumbMax   = 20 // cache at most 20 thumbnails
)

func serveThumbnail(w http.ResponseWriter, r *http.Request, path string, cover bool) {
	// non-cover images: serve original directly
	if !cover {
		http.ServeFile(w, r, path)
		return
	}

	// check cache
	if cached, ok := thumbCache.Load(path); ok {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Write(cached.([]byte))
		return
	}

	// decode original
	f, err := os.Open(path)
	if err != nil {
		http.ServeFile(w, r, path)
		return
	}
	defer f.Close()

	// detect format and decode
	config, _, err := image.DecodeConfig(f)
	if err != nil || config.Width <= 400 {
		f.Close()
		http.ServeFile(w, r, path)
		return
	}
	f.Seek(0, 0)

	img, _, err := image.Decode(f)
	if err != nil {
		f.Close()
		http.ServeFile(w, r, path)
		return
	}

	// resize to max 400px width (nearest-neighbor, good enough for thumbnails)
	ratio := 400.0 / float64(config.Width)
	newW := 400
	newH := int(float64(config.Height) * ratio)

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := range newH {
		srcY := int(float64(y) / ratio)
		for x := range newW {
			srcX := int(float64(x) / ratio)
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}

	// encode as JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80}); err != nil {
		http.ServeFile(w, r, path)
		return
	}
	data := buf.Bytes()

	// store in cache with eviction
	thumbMu.Lock()
	var cachedCount int
	thumbCache.Range(func(key, value any) bool {
		cachedCount++
		if cachedCount >= thumbMax {
			thumbCache.Delete(key)
		}
		return true
	})
	thumbCache.Store(path, data)
	thumbMu.Unlock()

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "max-age=86400")
	w.Write(data)
}

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	storage.Init()
}

func main() {

	app := application.New(application.Options{
		Name:        "ReadBooks",
		Description: "",
		Services: []application.Service{
			application.NewService(&AppService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/api/image") {
					path := r.URL.Query().Get("p")
					if path == "" {
						http.Error(w, "missing path", 400)
						return
					}
					cover := r.URL.Query().Get("cover") == "1"
					serveThumbnail(w, r, path, cover)
					return
				}
					next.ServeHTTP(w, r)
				})
			},
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	w, h := loadWindowSize()
	if w <= 0 || h <= 0 {
		w, h = 1200, 800
	}

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "ReadBooks",
		Width:  w,
		Height: h,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
