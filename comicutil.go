package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"ReadBooks/internal/storage"
)

var imageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
	".bmp":  true,
	".svg":  true,
	".ico":  true,
	".tiff": true,
	".tif":  true,
}

func isImage(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return imageExts[ext]
}

// scanImages 扫描目录下所有图片文件（不递归），按文件名排序返回完整路径
func scanImages(folder string) ([]string, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	var images []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if isImage(e.Name()) {
			images = append(images, filepath.Join(folder, e.Name()))
		}
	}
	sort.Strings(images)
	return images, nil
}

// collectImageFolders 递归遍历目录，收集所有包含图片的文件夹路径
// 如果当前文件夹包含图片，记录该路径并继续下探子文件夹
func collectImageFolders(root string, result *[]string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}

	hasImage := false
	var subDirs []string
	for _, e := range entries {
		if e.IsDir() {
			subDirs = append(subDirs, filepath.Join(root, e.Name()))
		} else if isImage(e.Name()) {
			hasImage = true
		}
	}

	if hasImage {
		*result = append(*result, root)
	}

	for _, d := range subDirs {
		collectImageFolders(d, result)
	}
}

type flexString string

func (s *flexString) UnmarshalJSON(data []byte) error {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch v := raw.(type) {
	case string:
		*s = flexString(v)
	case float64:
		*s = flexString(strconv.FormatFloat(v, 'f', -1, 64))
	}
	return nil
}

func (s flexString) Int64() int64 {
	v, _ := strconv.ParseInt(string(s), 10, 64)
	return v
}

type chapterInfo struct {
	ChapterID    flexString `json:"chapterid"`
	ChapterTitle string     `json:"chapterTitle"`
	Order        int        `json:"order"`
}

type comicMeta struct {
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Author       flexString    `json:"author"`
	Tags         []string      `json:"tags"`
	ChapterInfos []chapterInfo `json:"chapterinfos"`
}

// comicIsJm 处理包含元数据.json的漫画目录
// 返回与 AddsComic 相同的 { success, added, skipped, error? } 结构
func comicIsJm(s string) map[string]any {
	result := map[string]any{
		"success": false,
		"added":   0,
		"skipped": 0,
	}

	// 解析元数据
	metaPath := filepath.Join(s, "元数据.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		result["error"] = fmt.Sprintf("读取元数据失败: %v", err)
		return result
	}

	var meta comicMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		result["error"] = fmt.Sprintf("解析元数据失败: %v", err)
		return result
	}

	if len(meta.ChapterInfos) == 0 {
		result["error"] = "元数据中没有章节信息"
		return result
	}

	// 按 order 排序
	sort.Slice(meta.ChapterInfos, func(i, j int) bool {
		return meta.ChapterInfos[i].Order < meta.ChapterInfos[j].Order
	})

	// 加载已有书籍用于重复检查
	existing, _, _ := storage.ListBooks(1, 10000)

	coverURL := filepath.Join(s, "cover.jpg")
	added := 0
	skipped := 0

	if len(meta.ChapterInfos) == 1 {
		// 单章节：parent=0，正常创建
		ci := meta.ChapterInfos[0]
		chapterPath := filepath.Join(s, ci.ChapterTitle)

		// 重复检查
		for _, b := range existing {
			if b.FilePath == chapterPath {
				result["error"] = fmt.Sprintf("路径已存在 (ID:%d)", b.ID)
				return result
			}
		}

		images, err := scanImages(chapterPath)
		if err != nil {
			result["error"] = err.Error()
			return result
		}
		if len(images) == 0 {
			result["error"] = fmt.Sprintf("该章节下没有图片: %s", chapterPath)
			return result
		}

		id, err := storage.CreateBook(&storage.Book{
			Title:       meta.Name,
			Author:      string(meta.Author),
			Description: meta.Description,
			Parent:      0,
			SortOrder:   ci.Order,
			FilePath:    chapterPath,
			TotalPages:  len(images),
			CoverURL:    coverURL,
			JMID:        ci.ChapterID.Int64(),
			Status:      "未读",
		})
		if err != nil {
			result["error"] = err.Error()
			return result
		}

		// 绑定标签
		bindTags(id, meta.Tags)
		added = 1
	} else {
	// 多章节
	parentID := meta.ChapterInfos[0].ChapterID.Int64()

	for _, ci := range meta.ChapterInfos {
		chapterPath := filepath.Join(s, ci.ChapterTitle)

		// 重复检查
		dup := false
		for _, b := range existing {
			if b.FilePath == chapterPath {
				skipped++
				dup = true
				break
			}
		}
		if dup {
			continue
		}

		images, err := scanImages(chapterPath)
		if err != nil || len(images) == 0 {
			skipped++
			continue
		}

		title := meta.Name
		parentVal := 0
		if ci.Order == 1 {
			// 第一本作为父级
			parentVal = 1
		} else {
			title = meta.Name + ci.ChapterTitle
			parentVal = int(parentID)
		}

		id, err := storage.CreateBook(&storage.Book{
			Title:       title,
			Author:      string(meta.Author),
			Description: meta.Description,
			Parent:      parentVal,
			SortOrder:   ci.Order,
			FilePath:    chapterPath,
			TotalPages:  len(images),
			CoverURL:    coverURL,
			JMID:        ci.ChapterID.Int64(),
			Status:      "未读",
		})
			if err != nil {
				skipped++
				continue
			}

			bindTags(id, meta.Tags)
			added++
		}
	}

	result["success"] = true
	result["added"] = added
	result["skipped"] = skipped
	return result
}

// bindTags 根据标签名列表查找或创建标签，并绑定到书籍
func bindTags(bookID int64, tagNames []string) {
	if len(tagNames) == 0 {
		return
	}
	var tagIDs []int64
	for _, name := range tagNames {
		id, err := storage.CreateTag(name, "")
		if err != nil {
			continue
		}
		tagIDs = append(tagIDs, id)
	}
	if len(tagIDs) > 0 {
		storage.SetBookTags(bookID, tagIDs)
	}
}
