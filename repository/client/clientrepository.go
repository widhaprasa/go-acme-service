package client

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type ClientRepository struct {
	Db *sql.DB
}

func (c *ClientRepository) CreateTable() (sql.Result, error) {

	return c.Db.Exec(`CREATE TABLE IF NOT EXISTS client(
		id INTEGER PRIMARY KEY,
		email TEXT UNIQUE,
		uri TEXT,
		private_key BLOB,
		upserted_ts INTEGER
	);`)
}

func (c *ClientRepository) GetClient(email string) (map[string]any, error) {

	stmt, err := c.Db.Prepare("SELECT * FROM client WHERE email = ?")
	if err != nil {
		log.Println("Unable to query client:", err)
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(email)
	var id, upsertedTs int
	var uri string
	var privateKey []byte

	err = row.Scan(&id, &email, &uri, &privateKey, &upsertedTs)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"id":          id,
		"email":       email,
		"uri":         uri,
		"private_key": privateKey,
		"upserted_ts": upsertedTs,
	}

	return result, nil
}

func (c *ClientRepository) UpsertClient(email string, uri string, privateKey []byte, upsertedTs int64) (sql.Result, error) {
	return c.Db.Exec(`
		INSERT INTO client(email, uri, private_key, upserted_ts)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(email)
		DO UPDATE SET uri = excluded.uri, private_key = excluded.private_key, upserted_ts = excluded.upserted_ts;`,
		email, uri, privateKey, upsertedTs)
}

func (c *ClientRepository) DeleteCerts(email string) (sql.Result, error) {

	return c.Db.Exec(`
		DELETE FROM client WHERE email = ?`,
		email)
}
