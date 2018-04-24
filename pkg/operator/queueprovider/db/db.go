package db

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func (w DbQueueProvider) GetQueueEntryKeys() ([]string, error) {
	spec := w.dbQueueResource.Spec

	db, err := sqlx.Open(spec.DbDriver, w.dbDsn)
	if err != nil {
		return []string{}, err
	}

	var result []string
	err = db.Select(&result, spec.EntriesSql)
	if err != nil {
		return []string{}, err
	}

	return result, nil
}
