package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"ReadBooks/internal/storage"

	"github.com/sqweek/dialog"
	"golang.org/x/sys/windows"
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	findWindow       = user32.NewProc("FindWindowW")
	setForegroundWnd = user32.NewProc("SetForegroundWindow")
)

func findMainWindow() windows.HWND {
	title, _ := windows.UTF16PtrFromString("ReadBooks")
	ret, _, _ := findWindow.Call(uintptr(unsafe.Pointer(title)), 0)
	return windows.HWND(ret)
}

func bringToFront() {
	hwnd := findMainWindow()
	if hwnd != 0 {
		setForegroundWnd.Call(uintptr(hwnd))
	}
}

type AppService struct{}

func init() {
	fmt.Println("AppService init")
}

// ========== 书籍 ==========

func (a *AppService) AddComic(s string) map[string]any {
	result := map[string]any{
		"success": false,
	}

	// 重复检查：file_path
	books, _, _ := storage.ListBooks(1, 10000)
	for _, b := range books {
		if b.FilePath == s {
			result["error"] = fmt.Sprintf("路径已存在 (ID:%d)", b.ID)
			return result
		}
	}

	// 重复检查：标题（文件夹名）
	title := filepath.Base(s)
	for _, b := range books {
		if b.Title == title {
			result["error"] = fmt.Sprintf("标题已存在 (ID:%d)", b.ID)
			return result
		}
	}

	// 扫描图片
	images, err := scanImages(s)
	if err != nil {
		result["error"] = err.Error()
		return result
	}

	totalPages := len(images)
	if totalPages == 0 {
		result["error"] = "该文件夹下没有图片文件"
		return result
	}
	coverURL := images[0]

	// 入库
	book := &storage.Book{
		Title:      title,
		FilePath:   s,
		TotalPages: totalPages,
		CoverURL:   coverURL,
		Status:     "未读",
	}
	id, err := storage.CreateBook(book)
	if err != nil {
		result["error"] = err.Error()
		return result
	}
	images = nil
	result["success"] = true
	result["id"] = fmt.Sprintf("%d", id)
	result["title"] = title
	result["total_pages"] = fmt.Sprintf("%d", totalPages)
	return result
}

// createBookFromFolder 扫描图片并创建漫画记录（不做重复检查，由调用方保证）
func (a *AppService) createBookFromFolder(s string) (int64, string, int, error) {
	images, err := scanImages(s)
	if err != nil {
		return 0, "", 0, err
	}
	totalPages := len(images)
	if totalPages == 0 {
		return 0, "", 0, fmt.Errorf("该文件夹下没有图片文件")
	}
	title := filepath.Base(s)
	id, err := storage.CreateBook(&storage.Book{
		Title:      title,
		FilePath:   s,
		TotalPages: totalPages,
		CoverURL:   images[0],
		Status:     "未读",
	})
	if err != nil {
		return 0, "", 0, err
	}
	return id, title, totalPages, nil
}

func (a *AppService) AddsComic(s string) map[string]any {
	result := map[string]any{
		"success": false,
		"added":   0,
		"skipped": 0,
	}

	// 预加载已有书籍用于去重（只查一次）
	books, _, _ := storage.ListBooks(1, 10000)
	pathSet := make(map[string]struct{}, len(books))
	titleSet := make(map[string]struct{}, len(books))
	for _, b := range books {
		pathSet[b.FilePath] = struct{}{}
		titleSet[b.Title] = struct{}{}
	}
	books = nil // 释放引用，让 GC 可以回收

	var (
		mu             sync.Mutex
		wg             sync.WaitGroup
		sem            = make(chan struct{}, 4) // 最多 4 个并发，防止内存暴增
		added, skipped int
	)

	var walk func(path string)
	walk = func(path string) {
		// 有元数据.json → 按 meta 处理，不再递归下层
		if _, err := os.Stat(filepath.Join(path, "元数据.json")); err == nil {
			mu.Lock()
			_, dup := pathSet[path]
			mu.Unlock()
			if dup {
				mu.Lock()
				skipped++
				mu.Unlock()
				return
			}
			wg.Add(1)
			go func(folder string) {
				sem <- struct{}{}
				defer func() { <-sem }()
				defer wg.Done()
				r := comicIsJm(folder)
				mu.Lock()
				if r["success"] == true {
					added += r["added"].(int)
					skipped += r["skipped"].(int)
				} else {
					skipped++
				}
				mu.Unlock()
			}(path)
			return
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}

		hasImage := false
		var subDirs []string
		for _, e := range entries {
			if e.IsDir() {
				subDirs = append(subDirs, filepath.Join(path, e.Name()))
			} else if isImage(e.Name()) {
				hasImage = true
			}
		}

		if hasImage {
			title := filepath.Base(path)
			mu.Lock()
			_, pathDup := pathSet[path]
			_, titleDup := titleSet[title]
			mu.Unlock()

			if pathDup || titleDup {
				mu.Lock()
				skipped++
				mu.Unlock()
			} else {
				wg.Add(1)
				go func(folder string) {
					sem <- struct{}{}
					defer func() { <-sem }()
					defer wg.Done()
					_, title, _, err := a.createBookFromFolder(folder)
					mu.Lock()
					if err == nil {
						pathSet[folder] = struct{}{}
						titleSet[title] = struct{}{}
						added++
					} else {
						skipped++
					}
					mu.Unlock()
				}(path)
			}
		}

		for _, d := range subDirs {
			walk(d)
		}
	}

	walk(s)
	wg.Wait()

	result["success"] = true
	result["added"] = added
	result["skipped"] = skipped
	return result
}

func (a *AppService) BookCreate(book *storage.Book) (int64, error) {
	return storage.CreateBook(book)
}

func (a *AppService) BookGet(idStr string) (map[string]any, error) {
	// 自动判断是标题名还是id进行获取
	var book *storage.Book
	var err error

	// 尝试按 ID 查询
	var id int64
	if _, err = fmt.Sscanf(idStr, "%d", &id); err == nil && id > 0 {
		book, err = storage.GetBook(id)
		if err != nil {
			return nil, err
		}
	} else {
		// 按标题查询
		book, err = storage.GetBookByTitle(idStr)
		if err != nil {
			return nil, err
		}
	}

	// 获取关联的 tag
	tags, err := storage.GetBookTags(book.ID)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"id":           book.ID,
		"title":        book.Title,
		"author":       book.Author,
		"file_path":    book.FilePath,
		"description":  book.Description,
		"parent":       book.Parent,
		"sort_order":   book.SortOrder,
		"total_pages":  book.TotalPages,
		"current_page": book.CurrentPage,
		"cover_url":    book.CoverURL,
		"jmid":         book.JMID,
		"status":       book.Status,
		"created_at":   book.CreatedAt,
		"updated_at":   book.UpdatedAt,
		"tags":         tags,
	}
	return result, nil
}
func (a *AppService) BookGetImage(id int, page int) ([]string, error) {
	// 根据传入的id获取该漫画的实际路径
	book, err := storage.GetBook(int64(id))
	if err != nil {
		return nil, err
	}
	// 扫描目录下所有图片，按名字升序
	images, err := scanImages(book.FilePath)
	if err != nil {
		return nil, err
	}
	return images, nil
}

func (a *AppService) BookList(page, pageSize int) ([]*storage.Book, int, error) {
	return storage.ListBooks(page, pageSize)
}

func (a *AppService) BookSearch(keyword string, page, pageSize int) ([]*storage.Book, int, error) {
	return storage.SearchBooks(keyword, page, pageSize)
}

func (a *AppService) BookUpdate(book *storage.Book) error {
	return storage.UpdateBook(book)
}

func (a *AppService) BookUpdateProgress(bookID int64, page int) error {
	return storage.UpdateBookProgress(bookID, page)
}

func (a *AppService) GetChapters(jmid int64, parent int) ([]*storage.Book, error) {
	return storage.GetChapters(jmid, parent)
}

func (a *AppService) BookGetChapters(jmid int64, parent int) ([]map[string]any, error) {
	books, err := storage.GetChapters(jmid, parent)
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	for _, b := range books {
		result = append(result, map[string]any{
			"id":           b.ID,
			"title":        b.Title,
			"total_pages":  b.TotalPages,
			"current_page": b.CurrentPage,
			"jmid":         b.JMID,
			"status":       b.Status,
		})
	}
	return result, nil
}

func (a *AppService) BookDelete(id int64) error {
	return storage.DeleteBook(id)
}

func (a *AppService) BookDeleteWithFiles(id int64) error {
	book, err := storage.GetBook(id)
	if err != nil {
		return fmt.Errorf("获取书籍信息失败: %w", err)
	}

	if err := storage.DeleteBook(id); err != nil {
		return err
	}

	if book.FilePath != "" {
		if err := os.RemoveAll(book.FilePath); err != nil {
			return fmt.Errorf("删除文件夹失败: %w", err)
		}
	}

	return nil
}

// ========== 标签 ==========

func (a *AppService) TagCreate(name, color string) (int64, error) {
	return storage.CreateTag(name, color)
}

func (a *AppService) TagUpdate(id int64, name, color string) error {
	return storage.UpdateTag(id, name, color)
}

func (a *AppService) TagDelete(id int64) error {
	return storage.DeleteTag(id)
}

func (a *AppService) TagList() ([]*storage.Tag, error) {
	return storage.ListTags()
}

func (a *AppService) TagListWithCount() ([]*storage.TagWithCount, error) {
	return storage.ListTagsWithCount()
}

func (a *AppService) BookSetTags(bookID int64, tagIDs []int64) error {
	return storage.SetBookTags(bookID, tagIDs)
}

func (a *AppService) BookGetTags(bookID int64) ([]*storage.Tag, error) {
	return storage.GetBookTags(bookID)
}

func (a *AppService) BookGetByTag(tagID int64, page, pageSize int) ([]*storage.Book, int, error) {
	return storage.GetBooksByTag(tagID, page, pageSize)
}

// ========== 图片 ==========

func (a *AppService) GetImage(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取图片失败: %w", err)
	}
	return data, nil
}

// ========== 窗口尺寸 ==========

// loadWindowSize 从 state.json 读取窗口尺寸（包内共享，main 和 AppService 都能用）
func loadWindowSize() (int, int) {
	statePath := getDownloadStatePath()
	data, err := os.ReadFile(statePath)
	if err != nil {
		return 0, 0
	}
	var state struct {
		Width  int `json:"window_width"`
		Height int `json:"window_height"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return 0, 0
	}
	if state.Width <= 0 || state.Height <= 0 {
		return 0, 0
	}
	return state.Width, state.Height
}

func (a *AppService) SaveWindowSize(width, height int) error {
	statePath := getDownloadStatePath()
	var state map[string]any
	if data, err := os.ReadFile(statePath); err == nil {
		json.Unmarshal(data, &state)
	}
	if state == nil {
		state = map[string]any{}
	}
	state["window_width"] = width
	state["window_height"] = height
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化状态失败: %w", err)
	}
	return os.WriteFile(statePath, data, 0644)
}

func (a *AppService) LoadWindowSize() (int, int, error) {
	w, h := loadWindowSize()
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("无效的窗口尺寸")
	}
	return w, h, nil
}

// ========== 下载目录 ==========

func getDownloadStatePath() string {
	exePath, _ := os.Executable()
	return filepath.Join(filepath.Dir(exePath), "state.json")
}

func (a *AppService) GetDefaultDownloadDir() string {
	exePath, _ := os.Executable()
	return filepath.Join(filepath.Dir(exePath), "file-download")
}

func (a *AppService) GetDownloadDir() string {
	statePath := getDownloadStatePath()
	data, err := os.ReadFile(statePath)
	if err != nil {
		// state.json 不存在，返回默认值
		exePath, _ := os.Executable()
		return filepath.Join(filepath.Dir(exePath), "file-download")
	}
	var state struct {
		DownloadDir string `json:"download_dir"`
	}
	if err := json.Unmarshal(data, &state); err != nil || state.DownloadDir == "" {
		exePath, _ := os.Executable()
		return filepath.Join(filepath.Dir(exePath), "file-download")
	}
	return state.DownloadDir
}

func (a *AppService) SetDownloadDir(dir string) error {
	statePath := getDownloadStatePath()
	var state map[string]any
	if data, err := os.ReadFile(statePath); err == nil {
		json.Unmarshal(data, &state)
	}
	if state == nil {
		state = map[string]any{}
	}
	state["download_dir"] = dir
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	return nil
}

// ========== 默认漫画目录 ==========

func (a *AppService) GetDefaultComicDir() string {
	statePath := getDownloadStatePath()
	data, err := os.ReadFile(statePath)
	if err != nil {
		return ""
	}
	var state struct {
		ComicDir string `json:"comic_dir"`
	}
	if err := json.Unmarshal(data, &state); err != nil || state.ComicDir == "" {
		return ""
	}
	return state.ComicDir
}

func (a *AppService) SetDefaultComicDir(dir string) error {
	statePath := getDownloadStatePath()
	var state map[string]any
	if data, err := os.ReadFile(statePath); err == nil {
		json.Unmarshal(data, &state)
	}
	if state == nil {
		state = map[string]any{}
	}
	state["comic_dir"] = dir
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}
	return nil
}

func (a *AppService) SelectFolder() (string, error) {
	bringToFront()
	time.Sleep(100 * time.Millisecond)
	path, err := dialog.Directory().Title("选择漫画文件夹").Browse()
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("未选择文件夹")
	}
	return path, nil
}
