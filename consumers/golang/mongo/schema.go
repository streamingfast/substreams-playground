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

//todo: have the schema come from a file and let the user
// pass the file as a flag
// Only need to defined non-string fields as string fields
// are the default value for mongodb
var schema = `
{
	"pair": {
		"timestamp" : "timestamp",
		"block": "integer"
	},
	"token": {
		
	}
}
`
