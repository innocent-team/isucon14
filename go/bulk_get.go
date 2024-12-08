package main

import (
	"context"

	"github.com/doug-martin/goqu/v9"
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
	if err := sqlx.SelectContext(ctx, q, &chairRows, query, args...); err != nil {
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
	if err := sqlx.SelectContext(ctx, q, &ownerRows, query, args...); err != nil {
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

	query, args, err := goquDialect.From("latest_ride_statuses").
		Where(goqu.Ex{"ride_id": rideIDs}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	var rideStatusRows []*LatestRideStatus
	if err := sqlx.SelectContext(ctx, q, &rideStatusRows, query, args...); err != nil {
		return nil, err
	}

	rideStatuses := make(map[string]*RideStatus)
	for _, rideStatus := range rideStatusRows {
		if _, ok := rideStatuses[rideStatus.RideID]; !ok {
			rideStatuses[rideStatus.RideID] = &RideStatus{
				RideID:    rideStatus.RideID,
				Status:    rideStatus.Status,
				CreatedAt: rideStatus.CreatedAt,
			}
		}
	}
	return rideStatuses, nil
}

func calculateDiscountedFaresByRides(ctx context.Context, q sqlx.QueryerContext, rides []Ride) (map[string]int, error) {
	if len(rides) == 0 {
		return nil, nil
	}

	rideIDs := make([]string, 0, len(rides))
	for _, ride := range rides {
		rideIDs = append(rideIDs, ride.ID)
	}

	query, args, err := sqlx.In("SELECT * FROM coupons WHERE used_by IN (?)", rideIDs)
	if err != nil {
		return nil, err
	}

	var couponRows []*Coupon
	if err := sqlx.SelectContext(ctx, q, &couponRows, query, args...); err != nil {
		return nil, err
	}

	couponByRideID := make(map[string]*Coupon)
	for _, coupon := range couponRows {
		if coupon.UsedBy == nil {
			continue
		}
		couponByRideID[*coupon.UsedBy] = coupon
	}

	discountedFareByRideId := make(map[string]int)
	for _, ride := range rides {
		discount := 0
		coupon, ok := couponByRideID[ride.ID]
		if ok {
			discount = coupon.Discount
		}

		destLatitude := ride.DestinationLatitude
		destLongitude := ride.DestinationLongitude
		pickupLatitude := ride.PickupLatitude
		pickupLongitude := ride.PickupLongitude

		meteredFare := farePerDistance * calculateDistance(pickupLatitude, pickupLongitude, destLatitude, destLongitude)
		discountedMeteredFare := max(meteredFare-discount, 0)

		discountedFareByRideId[ride.ID] = initialFare + discountedMeteredFare
	}

	return discountedFareByRideId, nil
}

func getLatestChairStatusesByChairIDs(ctx context.Context, q sqlx.QueryerContext, chairIDs []string) (map[string]*LatestChairStatus, error) {
	if len(chairIDs) == 0 {
		return nil, nil
	}

	query, args, err := goquDialect.From("latest_chair_statuses").
		Where(goqu.Ex{"chair_id": chairIDs}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	var chairStatusRows []*LatestChairStatus
	if err := sqlx.SelectContext(ctx, q, &chairStatusRows, query, args...); err != nil {
		return nil, err
	}

	chairStatuses := make(map[string]*LatestChairStatus)
	for _, chairStatus := range chairStatusRows {
		if _, ok := chairStatuses[chairStatus.ChairID]; !ok {
			chairStatuses[chairStatus.ChairID] = &LatestChairStatus{
				ChairID:   chairStatus.ChairID,
				Status:    chairStatus.Status,
				CreatedAt: chairStatus.CreatedAt,
			}
		}
	}
	return chairStatuses, nil
}

func getLatestChairLocationsByChairIds(ctx context.Context, q sqlx.QueryerContext, chairIDs []string) (map[string]*LatestChairLocation, error) {
	if len(chairIDs) == 0 {
		return nil, nil
	}

	query, args, err := goquDialect.From("latest_chair_locations").
		Where(goqu.Ex{"chair_id": chairIDs}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	var chairLocationRows []*LatestChairLocation
	if err := sqlx.SelectContext(ctx, q, &chairLocationRows, query, args...); err != nil {
		return nil, err
	}

	chairLocations := make(map[string]*LatestChairLocation)
	for _, chairLocation := range chairLocationRows {
		chairLocations[chairLocation.ChairID] = chairLocation
	}
	return chairLocations, nil
}
