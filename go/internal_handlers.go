package main

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"

	"github.com/jmoiron/sqlx"
)

type RideType struct {
	ID              string `db:"id"`
	PickupLatitude  int    `db:"pickup_latitude"`
	PickupLongitude int    `db:"pickup_longitude"`
}

type ChairType struct {
	ID        string        `db:"id"`
	Latitude  sql.Null[int] `db:"latitude"`
	Longitude sql.Null[int] `db:"longitude"`
	Speed     int           `db:"speed"`
}

func searchNearestbyAvaiableChair(ctx context.Context, db *sqlx.DB, latitude int, longitude int) (*ChairType, bool, error) {
	// PickupLatitude, PickupLongitude を使って、使える近くの椅子を探す
	chairCandidates := []*ChairType{}
	query := `
		SELECT id, latitude, longitude, speed
		FROM ( 
		SELECT id, latitude, longitude, cm.speed,
			(SELECT COUNT(*) = 0 FROM
					(SELECT COUNT(chair_sent_at) = 6 AS completed
						FROM ride_statuses
						WHERE ride_id IN (SELECT id FROM rides WHERE chair_id = chairs.id)
						GROUP BY ride_id) is_completed
						WHERE completed = FALSE) AS avaiable
			FROM chairs
			LEFT JOIN latest_chair_locations cl ON cl.chair_id = chairs.id
			INNER JOIN chair_models cm ON cm.name = chairs.model
			WHERE chairs.is_active = TRUE
			
		) AS av
			WHERE avaiable = TRUE AND latitude IS NOT NULL AND longitude IS NOT NULL`
	if err := db.SelectContext(ctx, &chairCandidates, query); err != nil {
		return nil, false, err
	}
	if len(chairCandidates) == 0 {
		return nil, true, nil
	}

	// 一番スコアが高い椅子を探す
	slices.SortFunc(chairCandidates, func(a, b *ChairType) int {
		aScore := calculateMatchingScore(a, latitude, longitude)
		bScore := calculateMatchingScore(b, latitude, longitude)
		return cmp.Compare(bScore, aScore)
	})

	return chairCandidates[0], false, nil
}

func calculateMatchingScore(chair *ChairType, pickupLatitute, pickupLongitude int) float64 {
	distance := calculateDistance(chair.Latitude.V, chair.Longitude.V, pickupLatitute, pickupLongitude)
	// 速度で割ることで、速い椅子を優先する
	return -float64(distance)/float64(chair.Speed)
}

func matcher(ctx context.Context, db *sqlx.DB, ride *RideType) (bool, error) {
	matched, missing, err := searchNearestbyAvaiableChair(ctx, db, ride.PickupLatitude, ride.PickupLongitude)
	if err != nil {
		return false, err
	}
	if missing {
		return true, nil
	}

	if _, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", matched.ID, ride.ID); err != nil {
		return false, err
	}

	return false, nil
}

// GET /api/internal/matching
//
// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// MEMO: 一旦最も待たせているリクエストに適当な空いている椅子マッチさせる実装とする。おそらくもっといい方法があるはず…
	// PickupLatitude, PickupLongitude を使って、使える近くの椅子を探す
	// キューの最初と最後からライドを得る

	ride := &RideType{}
	if err := db.GetContext(ctx, ride, `SELECT id FROM rides WHERE chair_id IS NULL ORDER BY created_at LIMIT 1`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	missing, err := matcher(ctx, db, ride)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if missing {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ride = &RideType{}
	if err := db.GetContext(ctx, ride, `SELECT id FROM rides WHERE chair_id IS NULL ORDER BY created_at LIMIT 1`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	missing, err = matcher(ctx, db, ride)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if missing {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
