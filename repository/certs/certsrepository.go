package certs

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type CertsRepository struct {
	Db *sql.DB
}

func (c *CertsRepository) CreateTable() (sql.Result, error) {

	return c.Db.Exec(`CREATE TABLE IF NOT EXISTS certs(
		id INTEGER PRIMARY KEY,
		main TEXT UNIQUE,
		sans TEXT,
		email TEXT,
		private_key BLOB,
		certificate BLOB,
		not_before_ts INTEGER,
		not_after_ts INTEGER,
		upserted_ts INTEGER
	);`)
}

func (c *CertsRepository) GetCerts(main string) (map[string]any, error) {

	stmt, err := c.Db.Prepare("SELECT * FROM certs WHERE sans LIKE ?")
	if err != nil {
		log.Println("Unable to query certs:", err)
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow("%" + main + "%")
	var id, notBeforeTs, notAfterTs, upsertedTs int
	var sans, email string
	var privateKey, certificate []byte

	err = row.Scan(&id, &main, &sans, &email, &privateKey, &certificate, &notBeforeTs, &notAfterTs, &upsertedTs)
	if err != nil {
		log.Println("Unable to scan certs row:", err)
		return nil, err
	}

	result := map[string]any{
		"id":            id,
		"main":          main,
		"sans":          sans,
		"email":         email,
		"private_key":   privateKey,
		"certificate":   certificate,
		"not_before_ts": notBeforeTs,
		"not_after_ts":  notAfterTs,
		"upserted_ts":   upsertedTs,
	}

	return result, nil
}

func (c *CertsRepository) GetCertsByMain(domains []string) (map[string]any, error) {

	// Create prepared statements
	count := len(domains)

	anys := make([]any, count)
	preparedStatements := make([]string, count)
	for i := 0; i < count; i++ {
		anys[i] = domains[i]
		preparedStatements[i] = "?"
	}

	stmt, err := c.Db.Prepare("SELECT * FROM certs WHERE main IN " + fmt.Sprintf("%s", preparedStatements))
	if err != nil {
		log.Println("Unable to query certs:", err)
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(anys...)
	var id, notBeforeTs, notAfterTs, upsertedTs int
	var main, sans, email string
	var privateKey, certificate []byte

	err = row.Scan(&id, &main, &sans, &email, &privateKey, &certificate, &notBeforeTs, &notAfterTs, &upsertedTs)
	if err != nil {
		log.Println("Unable to scan certs row:", err)
		return nil, err
	}

	result := map[string]any{
		"id":            id,
		"main":          main,
		"sans":          sans,
		"email":         email,
		"private_key":   privateKey,
		"certificate":   certificate,
		"not_before_ts": notBeforeTs,
		"not_after_ts":  notAfterTs,
		"upserted_ts":   upsertedTs,
	}

	return result, nil
}

func (c *CertsRepository) ListCerts() ([]any, error) {

	rows, err := c.Db.Query("SELECT * FROM certs")
	if err != nil {
		log.Println("Unable to query certs:", err)
		return nil, err
	}
	defer rows.Close()

	result := []any{}
	for rows.Next() {
		var id, notBeforeTs, notAfterTs, upsertedTs int
		var main, sans, email string
		var privateKey, certificate []byte

		err = rows.Scan(&id, &main, &sans, &email, &privateKey, &certificate, &notBeforeTs, &notAfterTs, &upsertedTs)
		if err != nil {
			log.Println("Unable to scan certs row:", err)
			return nil, err
		}

		item := map[string]any{
			"id":            id,
			"main":          main,
			"sans":          sans,
			"email":         email,
			"private_key":   privateKey,
			"certificate":   certificate,
			"not_before_ts": notBeforeTs,
			"not_after_ts":  notAfterTs,
			"upserted_ts":   upsertedTs,
		}
		result = append(result, item)
	}

	return result, nil
}

func (c *CertsRepository) UpsertCerts(main string, sans string, email string, privateKey, certificate []byte, notBeforeTs int64, notAfterTs int64, upsertedTs int64) (sql.Result, error) {
	return c.Db.Exec(`
		INSERT INTO certs(main, sans, email, private_key, certificate, not_before_ts, not_after_ts, upserted_ts)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(main)
		DO UPDATE SET sans = excluded.sans, email = excluded.email, private_key = excluded.private_key, certificate = excluded.certificate, not_before_ts = excluded.not_before_ts,
			not_after_ts = excluded.not_after_ts, upserted_ts = excluded.upserted_ts;`,
		main, sans, email, privateKey, certificate, notBeforeTs, notAfterTs, upsertedTs)
}

func (c *CertsRepository) DeleteCerts(main string) (sql.Result, error) {

	return c.Db.Exec(`
		DELETE FROM certs WHERE main = ?`,
		main)
}
