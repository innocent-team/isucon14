package main

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func getChairsByIds(ctx context.Context, q sqlx.QueryerContext, chairIDs []string) (map[string]*Chair, error) {
	if len(chairIDs) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In("SELECT * FROM chairs WHERE id IN (?)", chairIDs)
	if err != nil {
		return nil, err
	}
	var chairRows []*Chair
	if err := sqlx.SelectContext(ctx, q, &chairRows, query, args...);err != nil {
		return nil, err
	}

	chairs := make(map[string]*Chair)
	for _, chair := range chairRows {
		chairs[chair.ID] = chair
	}
	return chairs, nil
}

func getOwnersByIds(ctx context.Context, q sqlx.QueryerContext, ownerIDs []string) (map[string]*Owner, error) {
	if len(ownerIDs) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In("SELECT * FROM owners WHERE id IN (?)", ownerIDs)
	if err != nil {
		return nil, err
	}
	var ownerRows []*Owner
	if err := sqlx.SelectContext(ctx, q, &ownerRows, query, args...);err != nil {
		return nil, err
	}

	owners := make(map[string]*Owner)
	for _, owner := range ownerRows {
		owners[owner.ID] = owner
	}
	return owners, nil
}

func getLatestRideStatusesByRideIds(ctx context.Context, q sqlx.QueryerContext, rideIDs []string) (map[string]*RideStatus, error) {
	if len(rideIDs) == 0 {
		return nil, nil
	}

	// Powered by GitHub Copilot
	rawQuery := `
	SELECT rs.*
	FROM ride_statuses rs
	INNER JOIN (
		SELECT ride_id, MAX(created_at) AS max_created_at
		FROM ride_statuses
		WHERE ride_id IN (?)
		GROUP BY ride_id
	) latest
	ON rs.ride_id = latest.ride_id AND rs.created_at = latest.max_created_at;
	`
	query, args, err := sqlx.In(rawQuery, rideIDs)
	if err != nil {
		return nil, err
	}
	var rideStatusRows []*RideStatus
	if err := sqlx.SelectContext(ctx, q, &rideStatusRows, query, args...);err != nil {
		return nil, err
	}

	rideStatuses := make(map[string]*RideStatus)
	for _, rideStatus := range rideStatusRows {
		if _, ok := rideStatuses[rideStatus.RideID]; !ok {
			rideStatuses[rideStatus.RideID] = rideStatus
		}
	}
	return rideStatuses, nil
}
