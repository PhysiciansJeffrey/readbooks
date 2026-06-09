package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"ReadBooks/internal/storage"

	"golang.org/x/sys/windows"
)

var (
	user32     = windows.NewLazySystemDLL("user32.dll")
	findWindow = user32.NewProc("FindWindowW")

	ole32    = windows.NewLazySystemDLL("ole32.dll")
	coInit   = ole32.NewProc("CoInitializeEx")
	coCreate = ole32.NewProc("CoCreateInstance")

	shell32                     = windows.NewLazySystemDLL("shell32.dll")
	sHCreateItemFromParsingName = shell32.NewProc("SHCreateItemFromParsingName")
)

// COM GUIDs for IFileOpenDialog
var (
	clsidFileOpenDialog = windows.GUID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE, Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidIFileDialog      = windows.GUID{Data1: 0x42F85136, Data2: 0xDB7E, Data3: 0x439C, Data4: [8]byte{0x85, 0xF1, 0xE4, 0x07, 0x5D, 0x13, 0x5F, 0xC8}}
	iidIShellItem       = windows.GUID{Data1: 0x43826D1E, Data2: 0xE718, Data3: 0x42EE, Data4: [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}
)

const (
	fosPickFolders   = 0x20
	sigdnFilesysPath = 0x80058000
)

// COM VTable helpers

type comIUnknownVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}

type iFileDialogVtbl struct {
	comIUnknownVtbl
	// IModalWindow
	Show uintptr
	// IFileDialog
	SetFileTypes     uintptr
	SetFileTypeIndex uintptr
	GetFileTypeIndex uintptr
	Advise           uintptr
	Unadvise         uintptr
	SetOptions       uintptr
	GetOptions       uintptr
	SetDefaultFolder uintptr
	SetFolder        uintptr
	GetFolder        uintptr
	GetCurrentSel    uintptr
	SetTitle         uintptr
	SetOkBtnLabel    uintptr
	SetFileName      uintptr
	GetFileName      uintptr
}

type iShellItemVtbl struct {
	comIUnknownVtbl
	BindToHandler  uintptr
	GetParent      uintptr
	GetDisplayName uintptr
	GetAttributes  uintptr
	Compare        uintptr
}

type iUnknownCOM struct {
	vtbl *comIUnknownVtbl
}

func (p *iUnknownCOM) release() {
	syscall.SyscallN(p.vtbl.Release, uintptr(unsafe.Pointer(p)), 0, 0)
}

// selectFolderDialog 使用 Windows IFileOpenDialog（文件对话框样式）选择文件夹
func selectFolderDialog(title string, startDir string) (string, error) {
	// 初始化 COM（允许已经初始化的情况）
	hr, _, _ := coInit.Call(0, windows.COINIT_APARTMENTTHREADED)
	if hr != 0 && hr != 1 /* S_FALSE */ {
		return "", fmt.Errorf("COM 初始化失败: hr=%x", hr)
	}
	defer windows.CoUninitialize()

	// CoCreateInstance(CLSID_FileOpenDialog, nil, CLSCTX_INPROC_SERVER, IID_IFileDialog, &dialog)
	var dialog *iUnknownCOM
	ret, _, _ := coCreate.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)),
		0,
		1, // CLSCTX_INPROC_SERVER
		uintptr(unsafe.Pointer(&iidIFileDialog)),
		uintptr(unsafe.Pointer(&dialog)),
	)
	if ret != 0 {
		return "", fmt.Errorf("创建文件对话框失败: %x", ret)
	}
	defer dialog.release()

	fd := (*iFileDialogVtbl)(unsafe.Pointer(dialog.vtbl))

	// SetOptions(FOS_PICKFOLDERS)
	syscall.SyscallN(fd.SetOptions, uintptr(unsafe.Pointer(dialog)), fosPickFolders, 0)

	// SetTitle
	if titlePtr, err := windows.UTF16PtrFromString(title); err == nil {
		syscall.SyscallN(fd.SetTitle, uintptr(unsafe.Pointer(dialog)), uintptr(unsafe.Pointer(titlePtr)), 0)
	}

	// SetDefaultFolder: 如果提供了 startDir，创建 IShellItem 并设置
	if startDir != "" {
		if startDirPtr, err := windows.UTF16PtrFromString(startDir); err == nil {
			var folderItem *iUnknownCOM
			hr, _, _ := sHCreateItemFromParsingName.Call(
				uintptr(unsafe.Pointer(startDirPtr)),
				0,
				uintptr(unsafe.Pointer(&iidIShellItem)),
				uintptr(unsafe.Pointer(&folderItem)),
			)
			if hr == 0 && folderItem != nil {
				syscall.SyscallN(fd.SetDefaultFolder, uintptr(unsafe.Pointer(dialog)), uintptr(unsafe.Pointer(folderItem)), 0)
				folderItem.release()
			}
		}
	}

	// Show(mainWindowHWND) — 设置父窗口防止被覆盖
	hwnd := findMainWindow()
	hr, _, _ = syscall.SyscallN(fd.Show, uintptr(unsafe.Pointer(dialog)), uintptr(hwnd), 0)
	if hr != 0 {
		return "", fmt.Errorf("取消选择")
	}

	// GetFolder(&shellItem)
	var resultItem *iUnknownCOM
	hr, _, _ = syscall.SyscallN(fd.GetFolder, uintptr(unsafe.Pointer(dialog)), uintptr(unsafe.Pointer(&resultItem)), 0)
	if hr != 0 || resultItem == nil {
		return "", fmt.Errorf("获取选择结果失败: %x", hr)
	}
	defer resultItem.release()

	ri := (*iShellItemVtbl)(unsafe.Pointer(resultItem.vtbl))

	// GetDisplayName(SIGDN_FILESYSPATH, &namePtr)
	var namePtr *uint16
	hr, _, _ = syscall.SyscallN(ri.GetDisplayName, uintptr(unsafe.Pointer(resultItem)), sigdnFilesysPath, uintptr(unsafe.Pointer(&namePtr)))
	if hr != 0 || namePtr == nil {
		return "", fmt.Errorf("获取路径失败: %x", hr)
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(namePtr))

	return windows.UTF16PtrToString(namePtr), nil
}

func findMainWindow() windows.HWND {
	title, _ := windows.UTF16PtrFromString("ReadBooks")
	ret, _, _ := findWindow.Call(uintptr(unsafe.Pointer(title)), 0)
	return windows.HWND(ret)
}

type AppService struct{}

func init() {
	fmt.Println("AppService init")
}

// ========== 书籍 ==========

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

func (a *AppService) GetChapters(jmid int64, parent int) ([]*storage.Book, error) {
	return storage.GetChapters(jmid, parent)
}

func (a *AppService) BookUpdateProgress(bookID int64, page int) error {
	return storage.UpdateBookProgress(bookID, page)
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

func (a *AppService) TagListWithCount() ([]*storage.TagWithCount, error) {
	return storage.ListTagsWithCount()
}

func (a *AppService) BookSetTags(bookID int64, tagIDs []int64) error {
	return storage.SetBookTags(bookID, tagIDs)
}

func (a *AppService) BookGetTags(bookID int64) ([]*storage.Tag, error) {
	return storage.GetBookTags(bookID)
}

// ========== 窗口尺寸 ==========

// loadWindowSize 从 state.json 读取窗口尺寸（包内共享，main 和 AppService 都能用）
func loadWindowSize() (int, int) {
	statePath := getloadStatePath()
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
	statePath := getloadStatePath()
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

// ========== 默认漫画目录 ==========

func (a *AppService) GetDefaultComicDir() string {
	statePath := getloadStatePath()
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
	statePath := getloadStatePath()
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

// getLastFolderPath 从 state.json 读取上次选择的文件夹路径
func getLastFolderPath() string {
	statePath := getloadStatePath()
	data, err := os.ReadFile(statePath)
	if err != nil {
		return ""
	}
	var state struct {
		LastFolder string `json:"last_folder_path"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return ""
	}
	return state.LastFolder
}

// saveLastFolderPath 保存本次选择的文件夹路径到 state.json
func saveLastFolderPath(dir string) {
	if dir == "" {
		return
	}
	statePath := getloadStatePath()
	var state map[string]any
	if data, err := os.ReadFile(statePath); err == nil {
		json.Unmarshal(data, &state)
	}
	if state == nil {
		state = map[string]any{}
	}
	state["last_folder_path"] = dir
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(statePath, data, 0644)
}

func (a *AppService) SelectFolder() (string, error) {
	startDir := getLastFolderPath()
	path, err := selectFolderDialog("选择漫画文件夹", startDir)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("未选择文件夹")
	}
	saveLastFolderPath(path)
	return path, nil
}
