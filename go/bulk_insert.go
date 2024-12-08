package main

import (
	"context"

	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
	_ "github.com/utgwkk/go-ext/goquext/mysql/dialect" // for goqu.Dialect("safemysql8")
)

var goquDialect = goqu.Dialect("safemysql8")

func initializeLatestRideStatuses(ctx context.Context, db *sqlx.DB) error {
	// Powered by GitHub Copilot
	query := `
	SELECT rs.*
	FROM ride_statuses rs
	INNER JOIN (
		SELECT ride_id, MAX(created_at) AS max_created_at
		FROM ride_statuses
		GROUP BY ride_id
	) latest
	ON rs.ride_id = latest.ride_id AND rs.created_at = latest.max_created_at;
	`
	var rideStatusRows []*RideStatus
	if err := db.SelectContext(ctx, &rideStatusRows, query); err != nil {
		return err
	}

	initialLatestRideStatuses := make([]*LatestRideStatus, 0, len(rideStatusRows))
	for _, rideStatus := range rideStatusRows {
		initialLatestRideStatuses = append(initialLatestRideStatuses, rideStatus.ToLatestRideStatus())
	}

	_, err := goquDialect.DB(db).
		Insert("latest_ride_statuses").
		Rows(initialLatestRideStatuses).As("new").
		OnConflict(goqu.DoUpdate("", goqu.Record{
			"status":     goqu.L("new.status"),
			"created_at": goqu.L("new.created_at"),
		})).
		Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	return nil
}
