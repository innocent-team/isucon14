package main

import (
	"go.nhat.io/otelsql"
	"go.opentelemetry.io/otel"
)

const serverName = "isucon14"

var dbDriverName = "mysql"

var tracer = otel.Tracer(serverName)

func registerOtelsqlDriver() error {
	// https://github.com/nhatthm/otelsql#trace-query
	driverName, err := otelsql.Register("mysql",
		otelsql.TraceQueryWithArgs(),
	)
	if err != nil {
		return err
	}
	dbDriverName = driverName
	return nil
}
