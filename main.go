package main

import (
	"embed"
	_ "embed"
	"log"
	"net/http"
	"strings"

	"ReadBooks/internal/storage"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	storage.Init()
}

func main() {
	app := application.New(application.Options{
		Name:        "ReadBooks",
		Description: " ",
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
						w.Header().Set("Cache-Control", "max-age=86400")
						http.ServeFile(w, r, path)
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

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "ReadBooks",
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
