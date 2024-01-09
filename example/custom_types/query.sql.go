// Code generated by pggen. DO NOT EDIT.

package custom_types

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/atomicleads/pggen/example/custom_types/mytype"
)

// Querier is a typesafe Go interface backed by SQL queries.
type Querier interface {
	CustomTypes(ctx context.Context) (CustomTypesRow, error)

	CustomMyInt(ctx context.Context) (int, error)

	IntArray(ctx context.Context) ([][]int32, error)
}

type DBQuerier struct {
	conn  genericConn   // underlying Postgres transport to use
	types *typeResolver // resolve types by name
}

var _ Querier = &DBQuerier{}

// genericConn is a connection to a Postgres database. This is usually backed by
// *pgx.Conn, pgx.Tx, or *pgxpool.Pool.
type genericConn interface {
	// Query executes sql with args. If there is an error the returned Rows will
	// be returned in an error state. So it is allowed to ignore the error
	// returned from Query and handle it in Rows.
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)

	// QueryRow is a convenience wrapper over Query. Any error that occurs while
	// querying is deferred until calling Scan on the returned Row. That Row will
	// error with pgx.ErrNoRows if no rows are returned.
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row

	// Exec executes sql. sql can be either a prepared statement name or an SQL
	// string. arguments should be referenced positionally from the sql string
	// as $1, $2, etc.
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

// NewQuerier creates a DBQuerier that implements Querier. conn is typically
// *pgx.Conn, pgx.Tx, or *pgxpool.Pool.
func NewQuerier(conn genericConn) *DBQuerier {
	return &DBQuerier{conn: conn, types: newTypeResolver()}
}

// WithTx creates a new DBQuerier that uses the transaction to run all queries.
func (q *DBQuerier) WithTx(tx pgx.Tx) (*DBQuerier, error) {
	return &DBQuerier{conn: tx}, nil
}

// typeResolver looks up the pgtype.ValueTranscoder by Postgres type name.
type typeResolver struct {
	connInfo *pgtype.ConnInfo // types by Postgres type name
}

func newTypeResolver() *typeResolver {
	ci := pgtype.NewConnInfo()
	return &typeResolver{connInfo: ci}
}

// findValue find the OID, and pgtype.ValueTranscoder for a Postgres type name.
func (tr *typeResolver) findValue(name string) (uint32, pgtype.ValueTranscoder, bool) {
	typ, ok := tr.connInfo.DataTypeForName(name)
	if !ok {
		return 0, nil, false
	}
	v := pgtype.NewValue(typ.Value)
	return typ.OID, v.(pgtype.ValueTranscoder), true
}

// setValue sets the value of a ValueTranscoder to a value that should always
// work and panics if it fails.
func (tr *typeResolver) setValue(vt pgtype.ValueTranscoder, val interface{}) pgtype.ValueTranscoder {
	if err := vt.Set(val); err != nil {
		panic(fmt.Sprintf("set ValueTranscoder %T to %+v: %s", vt, val, err))
	}
	return vt
}

const customTypesSQL = `SELECT 'some_text', 1::bigint;`

type CustomTypesRow struct {
	Column mytype.String `json:"?column?"`
	Int8   CustomInt     `json:"int8"`
}

// CustomTypes implements Querier.CustomTypes.
func (q *DBQuerier) CustomTypes(ctx context.Context) (CustomTypesRow, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "CustomTypes")
	row := q.conn.QueryRow(ctx, customTypesSQL)
	var item CustomTypesRow
	if err := row.Scan(&item.Column, &item.Int8); err != nil {
		return item, fmt.Errorf("query CustomTypes: %w", err)
	}
	return item, nil
}

const customMyIntSQL = `SELECT '5'::my_int as int5;`

// CustomMyInt implements Querier.CustomMyInt.
func (q *DBQuerier) CustomMyInt(ctx context.Context) (int, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "CustomMyInt")
	row := q.conn.QueryRow(ctx, customMyIntSQL)
	var item int
	if err := row.Scan(&item); err != nil {
		return item, fmt.Errorf("query CustomMyInt: %w", err)
	}
	return item, nil
}

const intArraySQL = `SELECT ARRAY ['5', '6', '7']::int[] as ints;`

// IntArray implements Querier.IntArray.
func (q *DBQuerier) IntArray(ctx context.Context) ([][]int32, error) {
	ctx = context.WithValue(ctx, "pggen_query_name", "IntArray")
	rows, err := q.conn.Query(ctx, intArraySQL)
	if err != nil {
		return nil, fmt.Errorf("query IntArray: %w", err)
	}
	defer rows.Close()
	items := [][]int32{}
	for rows.Next() {
		var item []int32
		if err := rows.Scan(&item); err != nil {
			return nil, fmt.Errorf("scan IntArray row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close IntArray rows: %w", err)
	}
	return items, err
}

// textPreferrer wraps a pgtype.ValueTranscoder and sets the preferred encoding
// format to text instead binary (the default). pggen uses the text format
// when the OID is unknownOID because the binary format requires the OID.
// Typically occurs if the results from QueryAllDataTypes aren't passed to
// NewQuerierConfig.
type textPreferrer struct {
	pgtype.ValueTranscoder
	typeName string
}

// PreferredParamFormat implements pgtype.ParamFormatPreferrer.
func (t textPreferrer) PreferredParamFormat() int16 { return pgtype.TextFormatCode }

func (t textPreferrer) NewTypeValue() pgtype.Value {
	return textPreferrer{ValueTranscoder: pgtype.NewValue(t.ValueTranscoder).(pgtype.ValueTranscoder), typeName: t.typeName}
}

func (t textPreferrer) TypeName() string {
	return t.typeName
}

// unknownOID means we don't know the OID for a type. This is okay for decoding
// because pgx call DecodeText or DecodeBinary without requiring the OID. For
// encoding parameters, pggen uses textPreferrer if the OID is unknown.
const unknownOID = 0
