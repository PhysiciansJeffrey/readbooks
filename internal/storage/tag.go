package storage

import (
	"fmt"
	"strings"
)

type Tag struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
}

// ========== 标签 CRUD ==========

func CreateTag(name, color string) (int64, error) {
	// 先查是否已存在
	var id int64
	err := DB.QueryRow(`SELECT id FROM tags WHERE name = ?`, name).Scan(&id)
	if err == nil {
		return id, nil
	}

	res, err := DB.Exec(
		`INSERT INTO tags (name, color) VALUES (?, ?)`,
		name, color,
	)
	if err != nil {
		return 0, fmt.Errorf("插入标签失败: %w", err)
	}
	return res.LastInsertId()
}

func UpdateTag(id int64, name, color string) error {
	_, err := DB.Exec(
		`UPDATE tags SET name=?, color=? WHERE id=?`,
		name, color, id,
	)
	if err != nil {
		return fmt.Errorf("更新标签失败: %w", err)
	}
	return nil
}

func DeleteTag(id int64) error {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM book_tags WHERE tag_id=?`, id); err != nil {
		return fmt.Errorf("删除标签关联失败: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM tags WHERE id=?`, id); err != nil {
		return fmt.Errorf("删除标签失败: %w", err)
	}
	return tx.Commit()
}

// TagWithCount 标签及其关联漫画数量
type TagWithCount struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
	BookCount int    `json:"bookCount"`
}

// ListTagsWithCount 返回所有标签及每个标签关联的漫画数量
func ListTagsWithCount() ([]*TagWithCount, error) {
	rows, err := DB.Query(
		`SELECT t.id, t.name, t.color, t.sort_order,
		        COUNT(bt.book_id) AS book_count
		 FROM tags t
		 LEFT JOIN book_tags bt ON bt.tag_id = t.id
		 GROUP BY t.id
		 ORDER BY t.sort_order, t.id`,
	)
	if err != nil {
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}
	defer rows.Close()

	var tags []*TagWithCount
	for rows.Next() {
		var t TagWithCount
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.SortOrder, &t.BookCount); err != nil {
			return nil, fmt.Errorf("扫描标签数据失败: %w", err)
		}
		tags = append(tags, &t)
	}
	return tags, nil
}

// ========== 书籍-标签关联 ==========

func SetBookTags(bookID int64, tagIDs []int64) error {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM book_tags WHERE book_id=?`, bookID); err != nil {
		return fmt.Errorf("清除旧标签关联失败: %w", err)
	}

	for _, tagID := range tagIDs {
		if _, err := tx.Exec(
			`INSERT INTO book_tags (book_id, tag_id) VALUES (?, ?)`,
			bookID, tagID,
		); err != nil {
			return fmt.Errorf("插入标签关联失败: %w", err)
		}
	}
	return tx.Commit()
}

func GetBookTags(bookID int64) ([]*Tag, error) {
	rows, err := DB.Query(
		`SELECT t.id, t.name, t.color, t.sort_order
		 FROM tags t
		 INNER JOIN book_tags bt ON bt.tag_id = t.id
		 WHERE bt.book_id = ?
		 ORDER BY t.sort_order, t.id`,
		bookID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询书籍标签失败: %w", err)
	}
	defer rows.Close()

	var tags []*Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.SortOrder); err != nil {
			return nil, fmt.Errorf("扫描标签数据失败: %w", err)
		}
		tags = append(tags, &t)
	}
	return tags, nil
}

// BatchGetBookTags 批量查询多本书的标签（替代 N+1 循环）
func BatchGetBookTags(bookIDs []int64) (map[int64][]*Tag, error) {
	if len(bookIDs) == 0 {
		return map[int64][]*Tag{}, nil
	}

	placeholders := make([]string, len(bookIDs))
	args := make([]any, len(bookIDs))
	for i, id := range bookIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT bt.book_id, t.id, t.name, t.color, t.sort_order
		 FROM tags t
		 INNER JOIN book_tags bt ON bt.tag_id = t.id
		 WHERE bt.book_id IN (%s)
		 ORDER BY t.sort_order, t.id`,
		strings.Join(placeholders, ","),
	)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("批量查询书籍标签失败: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]*Tag, len(bookIDs))
	for rows.Next() {
		var bookID int64
		var t Tag
		if err := rows.Scan(&bookID, &t.ID, &t.Name, &t.Color, &t.SortOrder); err != nil {
			return nil, fmt.Errorf("扫描标签数据失败: %w", err)
		}
		result[bookID] = append(result[bookID], &t)
	}

	for _, id := range bookIDs {
		if _, ok := result[id]; !ok {
			result[id] = []*Tag{}
		}
	}

	return result, nil
}

