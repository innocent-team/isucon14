package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/jmoiron/sqlx"
)

type RideType struct {
	ID              string `db:"id"`
	PickupLatitude  int    `db:"pickup_latitude"`
	PickupLongitude int    `db:"pickup_longitude"`
}

type ChairType struct {
	ID string `db:"id"`
}

func searchNearestbyAvaiableChair(ctx context.Context, db *sqlx.DB, latitude int, longitude int) (*ChairType, bool, error) {
	// PickupLatitude, PickupLongitude を使って、使える近くの椅子を探す
	chair := &ChairType{}
	if err := db.GetContext(ctx, chair, `
	SELECT id
	FROM ( 
	
	SELECT id, latitude, longitude,
		(SELECT COUNT(*) = 0 FROM
				(SELECT COUNT(chair_sent_at) = 6 AS completed
					FROM ride_statuses
					WHERE ride_id IN (SELECT id FROM rides WHERE chair_id = chairs.id)
					GROUP BY ride_id) is_completed
					WHERE completed = FALSE) AS avaiable
		FROM chairs
		LEFT JOIN latest_chair_locations cl ON cl.chair_id = chairs.id
		WHERE chairs.is_active = TRUE
		
	) AS av
		WHERE avaiable = TRUE
		ORDER BY (ABS(av.latitude - ?) + ABS(av.longitude - ?))
		LIMIT 1`,
		latitude, longitude); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, true, nil
		}
		return nil, false, err
	}
	return chair, false, nil
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

	for i := 0; i < 2; i++ {
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
	}
	w.WriteHeader(http.StatusNoContent)
}
