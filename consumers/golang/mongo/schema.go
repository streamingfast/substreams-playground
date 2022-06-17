package main

type tables map[string]fields
type fields map[string]databaseType
type databaseType string

const (
	INTEGER   databaseType = "integer"
	DOUBLE    databaseType = "double"
	BOOLEAN   databaseType = "boolean"
	TIMESTAMP databaseType = "timestamp"
	NULL      databaseType = "null"
	DATE      databaseType = "date"
)
