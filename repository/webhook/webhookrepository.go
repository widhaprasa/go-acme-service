package webhook

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type WebhookRepository struct {
	Db *sql.DB
}

func (w *WebhookRepository) CreateTable() (sql.Result, error) {

	return w.Db.Exec(`CREATE TABLE IF NOT EXISTS webhook(
		id INTEGER PRIMARY KEY,
		main TEXT UNIQUE,
		url TEXT,
		headers BLOB
	);`)
}

func (w *WebhookRepository) GetWebhook(main string) (map[string]any, error) {

	stmt, err := w.Db.Prepare("SELECT * FROM webhook WHERE main = ?")
	if err != nil {
		log.Println("Unable to query webhook:", err)
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(main)
	var id int
	var url string
	var headers []byte

	err = row.Scan(&id, &main, &url, &headers)
	if err != nil {
		log.Println("Unable to scan webhook row:", err)
		return nil, err
	}

	var headerMap map[string]any
	err = json.Unmarshal(headers, &headerMap)
	if err != nil {
		headerMap = map[string]any{}
	}

	result := map[string]any{
		"id":      id,
		"main":    main,
		"url":     url,
		"headers": headerMap,
	}

	return result, nil
}

func (w *WebhookRepository) ListWebhook() ([]any, error) {

	rows, err := w.Db.Query("SELECT * FROM webhook")
	if err != nil {
		log.Println("Unable to query webhook:", err)
		return nil, err
	}
	defer rows.Close()

	result := []any{}
	for rows.Next() {
		var id int
		var main, url string
		var headers []byte

		err = rows.Scan(&id, &main, &url, &headers)
		if err != nil {
			log.Println("Unable to scan webhook row:", err)
			return nil, err
		}

		var headerMap map[string]any
		err = json.Unmarshal(headers, &headerMap)
		if err != nil {
			headerMap = map[string]any{}
		}

		item := map[string]any{
			"id":      id,
			"main":    main,
			"url":     url,
			"headers": headerMap,
		}
		result = append(result, item)
	}

	return result, nil
}

func (w *WebhookRepository) MapWebhook() (map[string]any, error) {

	rows, err := w.Db.Query("SELECT * FROM webhook")
	if err != nil {
		log.Println("Unable to query webhook:", err)
		return nil, err
	}
	defer rows.Close()

	result := map[string]any{}
	for rows.Next() {
		var id int
		var main, url string
		var headers []byte

		err = rows.Scan(&id, &main, &url, &headers)
		if err != nil {
			log.Println("Unable to scan webhook row:", err)
			return nil, err
		}

		var headerMap map[string]any
		fmt.Println(headers)
		err = json.Unmarshal(headers, &headerMap)
		fmt.Println(headerMap)
		fmt.Println(err)
		if err != nil {
			headerMap = map[string]any{}
		}

		result[main] = map[string]any{
			"url":     url,
			"headers": headerMap,
		}
	}

	return result, nil
}

func (w *WebhookRepository) UpsertWebhook(main string, url string, headerMap map[string]any) (sql.Result, error) {

	headers, _ := json.Marshal(headerMap)

	return w.Db.Exec(`
		INSERT INTO webhook(main, url, headers)
		VALUES(?, ?, ?)
		ON CONFLICT(main)
		DO UPDATE SET url = excluded.url, headers = excluded.headers;`,
		main, url, headers)
}

func (w *WebhookRepository) DeleteWebhook(main string) (sql.Result, error) {

	return w.Db.Exec(`
		DELETE FROM webhook WHERE main = ?`,
		main)
}
