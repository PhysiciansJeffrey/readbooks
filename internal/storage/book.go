package storage

import (
	"database/sql"
	"fmt"
	"time"
)

type Book struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Description string    `json:"description"`
	Parent      int       `json:"parent"`
	SortOrder   int       `json:"sort_order"`
	FilePath    string    `json:"file_path"`
	TotalPages  int       `json:"total_pages"`
	CurrentPage int       `json:"current_page"`
	CoverURL    string    `json:"cover_url"`
	JMID        int64     `json:"jmid"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Tags        []*Tag    `json:"tags"`
}

func CreateBook(book *Book) (int64, error) {
	res, err := DB.Exec(
		`INSERT INTO books (title, author, description, parent, sort_order, file_path, total_pages, current_page, cover_url, jmid, status, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		book.Title, book.Author, book.Description, book.Parent, book.SortOrder, book.FilePath, book.TotalPages, book.CurrentPage, book.CoverURL, book.JMID, book.Status, time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("插入书籍失败: %w", err)
	}
	return res.LastInsertId()
}

func GetBook(id int64) (*Book, error) {
	var b Book
	err := DB.QueryRow(
		`SELECT id, title, author, description, parent, sort_order, file_path, total_pages, current_page, cover_url, jmid, status, created_at, updated_at
		 FROM books WHERE id = ?`, id,
	).Scan(&b.ID, &b.Title, &b.Author, &b.Description, &b.Parent, &b.SortOrder, &b.FilePath, &b.TotalPages, &b.CurrentPage, &b.CoverURL, &b.JMID, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("查询书籍失败: %w", err)
	}
	return &b, nil
}

func ExistsByJMID(jmid int64, title string) bool {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM books WHERE jmid = ?`, jmid).Scan(&count)
	if err == nil && count > 0 {
		return true
	}
	if title == "" {
		return false
	}
	err = DB.QueryRow(`SELECT COUNT(*) FROM books WHERE title = ?`, title).Scan(&count)
	return err == nil && count > 0
}

func GetBookByTitle(title string) (*Book, error) {
	var b Book
	err := DB.QueryRow(
		`SELECT id, title, author, description, parent, sort_order, file_path, total_pages, current_page, cover_url, jmid, status, created_at, updated_at
		 FROM books WHERE title = ?`, title,
	).Scan(&b.ID, &b.Title, &b.Author, &b.Description, &b.Parent, &b.SortOrder, &b.FilePath, &b.TotalPages, &b.CurrentPage, &b.CoverURL, &b.JMID, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("查询书籍失败: %w", err)
	}
	return &b, nil
}

func ListBooks(page, pageSize int) ([]*Book, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	var total int
	err := DB.QueryRow(`SELECT COUNT(*) FROM books where parent <= 1`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("统计书籍数量失败: %w", err)
	}

	rows, err := DB.Query(
		`SELECT id, title, author, description, parent, sort_order, file_path, total_pages, current_page, cover_url, jmid, status, created_at, updated_at
		 FROM books
		 where parent <= 1
		 ORDER BY sort_order,updated_at asc LIMIT ? OFFSET ?`,
		pageSize, (page-1)*pageSize,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("查询书籍列表失败: %w", err)
	}
	defer rows.Close()

	var books []*Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Description, &b.Parent, &b.SortOrder, &b.FilePath, &b.TotalPages, &b.CurrentPage, &b.CoverURL, &b.JMID, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描书籍数据失败: %w", err)
		}
		books = append(books, &b)
	}

	// 批量补充 tags（避免 N+1）
	if len(books) > 0 {
		ids := make([]int64, len(books))
		for i, b := range books {
			ids[i] = b.ID
		}
		tagMap, err := BatchGetBookTags(ids)
		if err == nil {
			for _, b := range books {
				b.Tags = tagMap[b.ID]
			}
		}
	}

	return books, total, nil
}

func SearchBooks(keyword string, page, pageSize int) ([]*Book, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	like := "%" + keyword + "%"

	// 融合搜索：标题、作者、JM号、标签名，去重
	query := `
		SELECT DISTINCT b.id, b.title, b.author, b.description, b.parent, b.sort_order, b.file_path, b.total_pages, b.current_page, b.cover_url, b.jmid, b.status, b.created_at, b.updated_at
		FROM books b
		LEFT JOIN book_tags bt ON bt.book_id = b.id
		LEFT JOIN tags t ON t.id = bt.tag_id
		WHERE (b.title LIKE ?
		   OR b.author LIKE ?
		   OR b.jmid = ?
		   OR t.name LIKE ?) and parent <= 1
		ORDER BY b.updated_at asc
		LIMIT ? OFFSET ?`

	rows, err := DB.Query(query, like, like, keyword, like, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("搜索书籍失败: %w", err)
	}
	defer rows.Close()

	var books []*Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Description, &b.Parent, &b.SortOrder, &b.FilePath, &b.TotalPages, &b.CurrentPage, &b.CoverURL, &b.JMID, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("扫描书籍数据失败: %w", err)
		}
		books = append(books, &b)
	}

	// 批量补充 tags（避免 N+1）
	if len(books) > 0 {
		ids := make([]int64, len(books))
		for i, b := range books {
			ids[i] = b.ID
		}
		tagMap, err := BatchGetBookTags(ids)
		if err == nil {
			for _, b := range books {
				b.Tags = tagMap[b.ID]
			}
		}
	}

	// 统计总数（去重）
	countQuery := `
		SELECT COUNT(DISTINCT b.id)
		FROM books b
		LEFT JOIN book_tags bt ON bt.book_id = b.id
		LEFT JOIN tags t ON t.id = bt.tag_id
		WHERE (b.title LIKE ?
		   OR b.author LIKE ?
		   OR b.jmid = ?
		   OR t.name LIKE ?) and parent <= 1`
	var total int
	err = DB.QueryRow(countQuery, like, like, keyword, like).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("统计搜索结果失败: %w", err)
	}

	return books, total, nil
}

func UpdateBookProgress(id int64, page int) error {
	_, err := DB.Exec(
		`UPDATE books SET current_page=?, updated_at=? WHERE id=?`,
		page, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("更新阅读进度失败: %w", err)
	}
	return nil
}

func GetChapters(jmid int64, parent int) ([]*Book, error) {
	var rows *sql.Rows
	var err error

	if parent <= 1 {
		// 顶层：查同系列（jmid 相同）
		rows, err = DB.Query(
			`SELECT id, title, author, description, parent, sort_order, file_path, total_pages, current_page, cover_url, jmid, status, created_at, updated_at
			 FROM books WHERE jmid = ? or parent = ?
			 ORDER BY sort_order ASC`, jmid, jmid,
		)
	} else {
		// 子级：查直接子漫画 + 同系列
		rows, err = DB.Query(
			`SELECT id, title, author, description, parent, sort_order, file_path, total_pages, current_page, cover_url, jmid, status, created_at, updated_at
			 FROM books WHERE parent = ? OR jmid = ?
			 ORDER BY sort_order ASC`, parent, parent,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("查询章节失败: %w", err)
	}
	defer rows.Close()

	var books []*Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Description, &b.Parent, &b.SortOrder, &b.FilePath, &b.TotalPages, &b.CurrentPage, &b.CoverURL, &b.JMID, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描章节数据失败: %w", err)
		}
		books = append(books, &b)
	}
	return books, nil
}

func DeleteBook(id int64) error {
	_, err := DB.Exec(`DELETE FROM books WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("删除书籍失败: %w", err)
	}
	return nil
}
