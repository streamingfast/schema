package schema

import (
	"database/sql"
)

// Snowflake dialect implementation based on Snowflake Information Schema
// Reference: https://docs.snowflake.com/en/sql-reference/info-schema

const snowflakeAllColumns = `SELECT * FROM %s LIMIT 0`

const snowflakeTableNamesWithSchema = `
	SELECT
		table_schema,
		table_name
	FROM
		information_schema.tables
	WHERE
		table_type = 'BASE TABLE' AND
		table_schema NOT IN ('INFORMATION_SCHEMA')
	ORDER BY
		table_schema,
		table_name
`

const snowflakeViewNamesWithSchema = `
	SELECT
		table_schema,
		table_name
	FROM
		information_schema.tables
	WHERE
		table_type = 'VIEW' AND
		table_schema NOT IN ('INFORMATION_SCHEMA')
	ORDER BY
		table_schema,
		table_name
`

// Snowflake doesn't have KEY_COLUMN_USAGE in Information Schema
// Instead, use SHOW PRIMARY KEYS command which returns the needed information
// Reference: https://docs.snowflake.com/en/sql-reference/sql/show-primary-keys
const snowflakePrimaryKey = `
	SELECT
		"column_name"
	FROM
		TABLE(RESULT_SCAN(LAST_QUERY_ID()))
	WHERE
		"schema_name" = CURRENT_SCHEMA() AND
		"table_name" = ?
	ORDER BY
		"key_sequence"
`

const snowflakePrimaryKeyWithSchema = `
	SELECT
		"column_name"
	FROM
		TABLE(RESULT_SCAN(LAST_QUERY_ID()))
	WHERE
		"schema_name" = ? AND
		"table_name" = ?
	ORDER BY
		"key_sequence"
`

type snowflakeDialect struct{}

func (snowflakeDialect) escapeIdent(ident string) string {
	// Snowflake uses double quotes for identifiers
	// This preserves case sensitivity
	return escapeWithDoubleQuotes(ident)
}

func (d snowflakeDialect) ColumnTypes(db *sql.DB, schema, name string) ([]*sql.ColumnType, error) {
	return fetchColumnTypes(db, snowflakeAllColumns, schema, name, d.escapeIdent)
}

func (d snowflakeDialect) PrimaryKey(db *sql.DB, schema, name string) ([]string, error) {
	// First execute SHOW PRIMARY KEYS command to populate RESULT_SCAN
	var showCmd string
	if schema == "" {
		showCmd = "SHOW PRIMARY KEYS IN TABLE " + d.escapeIdent(name)
	} else {
		showCmd = "SHOW PRIMARY KEYS IN TABLE " + d.escapeIdent(schema) + "." + d.escapeIdent(name)
	}

	_, err := db.Exec(showCmd)
	if err != nil {
		return nil, err
	}

	// Now query the results using TABLE(RESULT_SCAN(LAST_QUERY_ID()))
	if schema == "" {
		return fetchNames(db, snowflakePrimaryKey, "", name)
	}
	return fetchNames(db, snowflakePrimaryKeyWithSchema, schema, name)
}

func (snowflakeDialect) TableNames(db *sql.DB) ([][2]string, error) {
	return fetchObjectNames(db, snowflakeTableNamesWithSchema)
}

func (snowflakeDialect) ViewNames(db *sql.DB) ([][2]string, error) {
	return fetchObjectNames(db, snowflakeViewNamesWithSchema)
}
