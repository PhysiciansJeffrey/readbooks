package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type ApiService struct {
	hsr       *http.Server
	asr       *AppService
	app       *application.App
	stopChan  chan struct{}
	isRunning bool
	mu        sync.Mutex
	apiMux    http.Handler // 共享的 API 路由，Wails 和 HTTP 服务器共用
}

// ========== 服务启停 ===========

func (a *ApiService) SwitchHttpModel(isOpen bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	fmt.Println(isOpen)
	if isOpen {
		if a.isRunning {
			return
		}
		a.stopChan = make(chan struct{})
		a.isRunning = true
		go a.startupHttp()
	} else {
		if a.isRunning && a.stopChan != nil {
			close(a.stopChan)
			a.stopChan = nil
		}
	}
	setStateHttp(isOpen)
}
func (a *ApiService) GetHttpLink() string {
	port := "11789"
	if s := a.LoadState(); s != nil && s.Port != "" {
		port = s.Port
	}
	return "http://localhost:" + port
}
func getLocalIP() string {
	// 建立 UDP 连接（不会实际发送数据）
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func (a *ApiService) APIHandler() http.Handler {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.apiMux != nil {
		return a.apiMux
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/api/books", a.handleBooks)
	mux.HandleFunc("/api/books/search", a.handleBookSearch)
	mux.HandleFunc("/api/books/images", a.handleBookImages)
	mux.HandleFunc("/api/books/progress", a.handleBookProgress)
	mux.HandleFunc("/api/books/chapters", a.handleBookChapters)
	mux.HandleFunc("/api/books/tags", a.handleBookTags)
	mux.HandleFunc("/api/books/import", a.handleBookImport)
	mux.HandleFunc("/api/tags", a.handleTags)
	mux.HandleFunc("/api/image", a.handleImage)
	mux.HandleFunc("/api/settings/comic-dir", a.handleComicDir)
	mux.HandleFunc("/api/settings", a.handleSettings)
	mux.HandleFunc("/api/hello", a.hello)

	a.apiMux = mux
	return mux
}

// 创建httpserver
func (a *ApiService) createMux() http.Handler {
	mux := http.NewServeMux()
	// API 路由复用 APIHandler，挂在 /api/ 前缀下
	mux.Handle("/api/", a.APIHandler())

	subFS, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		fmt.Println("embed sub fs error:", err)
	} else {
		// 静态资源兜底
		mux.Handle("/", http.FileServer(http.FS(subFS)))
	}

	return mux
}

// 启动server
func (a *ApiService) startupHttp() {
	state := a.LoadState()
	port := "11789"
	if state != nil && state.Port != "" {
		port = state.Port
	}
	hsr := &http.Server{
		Addr:    ":" + port,
		Handler: a.createMux(),
	}
	a.mu.Lock()
	a.hsr = hsr
	stopchan := a.stopChan
	a.mu.Unlock()

	fmt.Printf("API 服务器已启动 → http://%s:%s\n", getLocalIP(), port)
	setStateHttpIP(fmt.Sprintf("http://%s:%s", getLocalIP(), port))

	go func() {
		<-stopchan

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := hsr.Shutdown(ctx); err != nil {
			fmt.Println("http server shutdown err:", err)
		}
		fmt.Println("---- api server close ----")
		a.mu.Lock()
		a.isRunning = false
		a.hsr = nil
		a.mu.Unlock()
	}()

	if err := hsr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		println("API 服务错误:", err.Error())
		a.mu.Lock()
		a.isRunning = false
		a.hsr = nil
		a.mu.Unlock()
	}
}

func (a *ApiService) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	// 默认启动 HTTP 服务器
	if a.LoadState().OpenServer {

		a.SwitchHttpModel(true)
	}
	return nil
}

func (a *ApiService) ServiceShutdown(ctx context.Context) error {
	a.SwitchHttpModel(false)
	return nil
}

// =================== REST API Handlers ===================

func (a *ApiService) hello(w http.ResponseWriter, r *http.Request) {
	respone, _, _ := a.asr.BookList(1, 10)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(respone)
}

// GET /api/books?page=1&pageSize=10
// GET /api/books?id=123 | GET /api/books?title=xxx
// DELETE /api/books?id=123[&withFiles=true]
func (a *ApiService) handleBooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		idStr := r.URL.Query().Get("id")
		if idStr != "" {
			// 获取单本书
			book, err := a.asr.BookGet(idStr)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(book)
			return
		}
		// 列表
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 {
			pageSize = 10
		}
		books, total, err := a.asr.BookList(page, pageSize)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"books": books,
			"total": total,
		})

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "missing or invalid id", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("withFiles") == "true" {
			err = a.asr.BookDeleteWithFiles(id)
		} else {
			err = a.asr.BookDelete(id)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// GET /api/books/search?keyword=xxx&page=1&pageSize=10
func (a *ApiService) handleBookSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	keyword := r.URL.Query().Get("keyword")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	books, total, err := a.asr.BookSearch(keyword, page, pageSize)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"books": books,
		"total": total,
	})
}

// GET /api/books/images?id=123&page=1
func (a *ApiService) handleBookImages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	images, err := a.asr.BookGetImage(id, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"images": images})
}

// PUT /api/books/progress?id=123&page=5
func (a *ApiService) handleBookProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := a.asr.BookUpdateProgress(id, page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// GET /api/books/chapters?jmid=123&parent=0
func (a *ApiService) handleBookChapters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	jmid, _ := strconv.ParseInt(r.URL.Query().Get("jmid"), 10, 64)
	parent, _ := strconv.Atoi(r.URL.Query().Get("parent"))
	chapters, err := a.asr.GetChapters(jmid, parent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"chapters": chapters})
}

// GET /api/books/tags?id=123
// PUT /api/books/tags?id=123  body: {"tag_ids": [1,2,3]}
func (a *ApiService) handleBookTags(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		tags, err := a.asr.BookGetTags(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"tags": tags})
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body failed", http.StatusBadRequest)
			return
		}
		var req struct {
			TagIDs []int64 `json:"tag_ids"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := a.asr.BookSetTags(id, req.TagIDs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// POST /api/books/import  body: {"path": "..."}
func (a *ApiService) handleBookImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.Path == "" {
		http.Error(w, "invalid json, need 'path'", http.StatusBadRequest)
		return
	}
	result := a.asr.AddsComic(req.Path)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/tags
// POST /api/tags  body: {"name": "...", "color": ""}
// PUT /api/tags?id=123  body: {"name": "...", "color": ""}
// DELETE /api/tags?id=123
func (a *ApiService) handleTags(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		tags, err := a.asr.TagListWithCount()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"tags": tags})

	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body failed", http.StatusBadRequest)
			return
		}
		var req struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.Name == "" {
			http.Error(w, "invalid json, need 'name'", http.StatusBadRequest)
			return
		}
		id, err := a.asr.TagCreate(req.Name, req.Color)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"id": id, "ok": true})

	case http.MethodPut:
		idStr := r.URL.Query().Get("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body failed", http.StatusBadRequest)
			return
		}
		var req struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := a.asr.TagUpdate(id, req.Name, req.Color); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := a.asr.TagDelete(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// GET /api/image?p=...&cover=1
func (a *ApiService) handleImage(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("p")
	if path == "" {
		http.Error(w, "missing path", 400)
		return
	}
	cover := r.URL.Query().Get("cover") == "1"
	serveThumbnail(w, r, path, cover)
}

// GET /api/settings
func (a *ApiService) handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	state := a.LoadState()
	json.NewEncoder(w).Encode(state)
}

// GET /api/settings/comic-dir
// PUT /api/settings/comic-dir  body: {"dir": "..."}
func (a *ApiService) handleComicDir(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		dir := a.asr.GetDefaultComicDir()
		json.NewEncoder(w).Encode(map[string]string{"dir": dir})
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body failed", http.StatusBadRequest)
			return
		}
		var req struct {
			Dir string `json:"dir"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.Dir == "" {
			http.Error(w, "invalid json, need 'dir'", http.StatusBadRequest)
			return
		}
		if err := a.asr.SetDefaultComicDir(req.Dir); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// WailsMiddleware 返回中间件函数，挂载 REST API 到 Wails 的 Asset Server
func (a *ApiService) WailsMiddleware() func(http.Handler) http.Handler {
	api := a.APIHandler()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				api.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
