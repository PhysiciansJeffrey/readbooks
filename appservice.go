package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	result["success"] = true
	result["id"] = fmt.Sprintf("%d", id)
	result["title"] = title
	result["total_pages"] = fmt.Sprintf("%d", totalPages)
	return result
}

func (a *AppService) AddsComic(s string) map[string]any {
	result := map[string]any{
		"success": false,
		"added":   0,
		"skipped": 0,
	}

	// Phase 1: 遍历所有文件夹，分类收集
	var metaFolders []string
	var imageFolders []string

	var walk func(path string)
	walk = func(path string) {
		// 有元数据.json → 记录为meta目录，不再递归下层
		if _, err := os.Stat(filepath.Join(path, "元数据.json")); err == nil {
			metaFolders = append(metaFolders, path)
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
			imageFolders = append(imageFolders, path)
		}

		for _, d := range subDirs {
			walk(d)
		}
	}

	walk(s)

	if len(metaFolders) == 0 && len(imageFolders) == 0 {
		result["error"] = "未找到包含图片的文件夹"
		return result
	}

	// Phase 2: 按分类处理
	added := 0
	skipped := 0

	for _, f := range metaFolders {
		r := comicIsJm(f)
		if r["success"] == true {
			added += r["added"].(int)
			skipped += r["skipped"].(int)
		} else {
			skipped++
		}
	}

	for _, f := range imageFolders {
		r := a.AddComic(f)
		if r["success"] == true {
			added++
		} else {
			skipped++
		}
	}

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
	// 传入 jmid 和 parent
	// 如果 parent <= 1：只用 jmid 查同系列漫画
	// 如果 parent > 1：用 parent 查子漫画 + 用 jmid 查同系列，合并返回
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
