// Code generated by pggen. DO NOT EDIT.

package nested

import (
	"context"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
)

// Querier is a typesafe Go interface backed by SQL queries.
//
// Methods ending with Batch enqueue a query to run later in a pgx.Batch. After
// calling SendBatch on pgx.Conn, pgxpool.Pool, or pgx.Tx, use the Scan methods
// to parse the results.
type Querier interface {
	Nested3(ctx context.Context) ([]Qux, error)
	// Nested3Batch enqueues a Nested3 query into batch to be executed
	// later by the batch.
	Nested3Batch(batch *pgx.Batch)
	// Nested3Scan scans the result of an executed Nested3Batch query.
	Nested3Scan(results pgx.BatchResults) ([]Qux, error)
}

type DBQuerier struct {
	conn genericConn
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
	return &DBQuerier{
		conn: conn,
	}
}

// WithTx creates a new DBQuerier that uses the transaction to run all queries.
func (q *DBQuerier) WithTx(tx pgx.Tx) (*DBQuerier, error) {
	return &DBQuerier{conn: tx}, nil
}

// InventoryItem represents the Postgres composite type "inventory_item".
type InventoryItem struct {
	ItemName pgtype.Text `json:"item_name"`
	Sku      Sku         `json:"sku"`
}

// Qux represents the Postgres composite type "qux".
type Qux struct {
	InvItem InventoryItem `json:"inv_item"`
	Foo     pgtype.Int8   `json:"foo"`
}

// Sku represents the Postgres composite type "sku".
type Sku struct {
	SkuID pgtype.Text `json:"sku_id"`
}

const nested3SQL = `SELECT ROW (ROW ('item_name', ROW ('sku_id')::sku)::inventory_item, 88)::qux AS qux;`

// Nested3 implements Querier.Nested3.
func (q *DBQuerier) Nested3(ctx context.Context) ([]Qux, error) {
	rows, err := q.conn.Query(ctx, nested3SQL)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("query Nested3: %w", err)
	}
	items := []Qux{}
	quxRow := pgtype.CompositeFields{
		pgtype.CompositeFields{
			&pgtype.Text{},
			pgtype.CompositeFields{
				&pgtype.Text{},
			},
		},
		&pgtype.Int8{},
	}
	for rows.Next() {
		var item Qux
		if err := rows.Scan(quxRow); err != nil {
			return nil, fmt.Errorf("scan Nested3 row: %w", err)
		}
		item.InvItem.ItemName = *quxRow[0].(pgtype.CompositeFields)[0].(*pgtype.Text)
		item.InvItem.Sku.SkuID = *quxRow[0].(pgtype.CompositeFields)[1].(pgtype.CompositeFields)[0].(*pgtype.Text)
		item.Foo = *quxRow[1].(*pgtype.Int8)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, err
}

// Nested3Batch implements Querier.Nested3Batch.
func (q *DBQuerier) Nested3Batch(batch *pgx.Batch) {
	batch.Queue(nested3SQL)
}

// Nested3Scan implements Querier.Nested3Scan.
func (q *DBQuerier) Nested3Scan(results pgx.BatchResults) ([]Qux, error) {
	rows, err := results.Query()
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return nil, err
	}
	items := []Qux{}
	quxRow := pgtype.CompositeFields{
		pgtype.CompositeFields{
			&pgtype.Text{},
			pgtype.CompositeFields{
				&pgtype.Text{},
			},
		},
		&pgtype.Int8{},
	}
	for rows.Next() {
		var item Qux
		if err := rows.Scan(quxRow); err != nil {
			return nil, fmt.Errorf("scan Nested3Batch row: %w", err)
		}
		item.InvItem.ItemName = *quxRow[0].(pgtype.CompositeFields)[0].(*pgtype.Text)
		item.InvItem.Sku.SkuID = *quxRow[0].(pgtype.CompositeFields)[1].(pgtype.CompositeFields)[0].(*pgtype.Text)
		item.Foo = *quxRow[1].(*pgtype.Int8)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, err
}
