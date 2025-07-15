package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/s-turchinskiy/metrics/internal/server/settings"
)

func ConnectToStore() (*sql.DB, error) {

	dbSettings := settings.Settings.Database
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		dbSettings.Host, dbSettings.Login, dbSettings.Password, dbSettings.DBName)

	db, err := sql.Open("pgx", ps)
	if err != nil {
		return nil, err
	}

	return db, nil

}
