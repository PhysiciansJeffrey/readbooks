package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init() {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("获取程序路径失败: %v", err)
	}
	dbPath := filepath.Join(filepath.Dir(exePath), "readbooks.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := createTables(db); err != nil {
		log.Fatalf("建表失败: %v", err)
	}

	DB = db
	log.Printf("数据库已就绪: %s", dbPath)
}

func createTables(db *sql.DB) error {
	tables := map[string]string{
		"books": `CREATE TABLE books (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				title TEXT NOT NULL DEFAULT '',
				author TEXT NULL DEFAULT '',
				description TEXT NOT NULL DEFAULT '',
				parent INTEGER NOT NULL DEFAULT 0,
				sort_order INTEGER NOT NULL DEFAULT 0,
				file_path TEXT NOT NULL DEFAULT '',
				total_pages INTEGER NOT NULL DEFAULT 0,
				current_page INTEGER NOT NULL DEFAULT 0,
				cover_url TEXT NOT NULL DEFAULT '',
				jmid TEXT NULL DEFAULT '',
				status TEXT NOT NULL DEFAULT '未读',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			);`,
		"tags": `CREATE TABLE tags (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL DEFAULT '' UNIQUE,
				color TEXT NOT NULL DEFAULT '',
				sort_order INTEGER NOT NULL DEFAULT 0
			);`,
		"book_tags": `CREATE TABLE book_tags (
				book_id INTEGER NOT NULL,
				tag_id INTEGER NOT NULL,
				PRIMARY KEY (book_id, tag_id),
				FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE,
				FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
			);`}

	for name, ddl := range tables {
		var exists int
		err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("检查表 %s 失败: %w", name, err)
		}
		if exists > 0 {
			log.Printf("表 %s 已存在，跳过", name)
			continue
		}
		if _, err := db.Exec(ddl); err != nil {
			return fmt.Errorf("创建表 %s 失败: %w", name, err)
		}
		log.Printf("表 %s 创建成功", name)
	}

	return nil
}
