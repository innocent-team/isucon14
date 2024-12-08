package main

import (
	"context"
	"database/sql"

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

	// ride_statuses.chair_id を埋める
	var rides []*Ride
	if err := goquDialect.DB(db).
		From("rides").
		ScanStructsContext(ctx, &rides); err != nil {
		return err
	}
	chairIdByRideId := make(map[string]sql.NullString)
	for _, ride := range rides {
		chairIdByRideId[ride.ID] = ride.ChairID
	}
	for rideID, chairID := range chairIdByRideId {
		if !chairID.Valid {
			continue
		}
		_, err := goquDialect.DB(db).
			Update("ride_statuses").
			Set(goqu.Record{"chair_id": chairID.String}).
			Where(goqu.Ex{"ride_id": rideID}).
			Executor().ExecContext(ctx)
		if err != nil {
			return err
		}
	}

	initialLatestRideStatuses := make([]*LatestRideStatus, 0, len(rideStatusRows))
	for _, rideStatus := range rideStatusRows {
		initialLatestRideStatuses = append(initialLatestRideStatuses, rideStatus.ToLatestRideStatus(chairIdByRideId[rideStatus.RideID]))
	}

	_, err := goquDialect.DB(db).
		Insert("latest_ride_statuses").
		Rows(initialLatestRideStatuses).As("new").
		OnConflict(goqu.DoUpdate("", goqu.Record{
			"chair_id":   goqu.L("new.chair_id"),
			"status":     goqu.L("new.status"),
			"created_at": goqu.L("new.created_at"),
		})).
		Executor().ExecContext(ctx)
	if err != nil {
		return err
	}

	initialLatestChairStatuses := make([]*LatestChairStatus, 0, len(rideStatusRows))
	for _, rideStatus := range initialLatestRideStatuses {
		if rideStatus.ChairID.Valid {
			initialLatestChairStatuses = append(initialLatestChairStatuses, &LatestChairStatus{
				ChairID:   rideStatus.ChairID.String,
				Status:    rideStatus.Status,
				CreatedAt: rideStatus.CreatedAt,
			})
		}
	}

	if len(initialLatestChairStatuses) > 0 {
		_, err = goquDialect.DB(db).
			Insert("latest_chair_statuses").
			Rows(initialLatestChairStatuses).As("new").
			OnConflict(goqu.DoUpdate("", goqu.Record{
				"status":     goqu.L("new.status"),
				"created_at": goqu.L("new.created_at"),
			})).
			Executor().ExecContext(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateLatestRideStatus(ctx context.Context, e sqlx.ExtContext, rideStatus *LatestRideStatus) error {
	query, args, err := goquDialect.
		Insert("latest_ride_statuses").
		Rows(rideStatus).As("new").
		OnConflict(goqu.DoUpdate("", goqu.Record{
			"chair_id":   goqu.L("new.chair_id"),
			"status":     goqu.L("new.status"),
			"created_at": goqu.L("new.created_at"),
		})).
		ToSQL()
	if err != nil {
		return err
	}

	if _, err := e.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	if !rideStatus.ChairID.Valid {
		return nil
	}
	chairStatus := &LatestChairStatus{
		Status:    rideStatus.Status,
		CreatedAt: rideStatus.CreatedAt,
		ChairID:   rideStatus.ChairID.String,
	}
	query, args, err = goquDialect.
		Insert("latest_chair_statuses").
		Rows(chairStatus).As("new").
		OnConflict(goqu.DoUpdate("", goqu.Record{
			"status":     goqu.L("new.status"),
			"created_at": goqu.L("new.created_at"),
		})).
		ToSQL()
	if err != nil {
		return err
	}

	if _, err := e.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return nil
}
