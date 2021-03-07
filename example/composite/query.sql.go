// Code generated by pggen. DO NOT EDIT.

package composite

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
	SearchScreenshots(ctx context.Context, params SearchScreenshotsParams) ([]SearchScreenshotsRow, error)
	// SearchScreenshotsBatch enqueues a SearchScreenshots query into batch to be executed
	// later by the batch.
	SearchScreenshotsBatch(batch *pgx.Batch, params SearchScreenshotsParams)
	// SearchScreenshotsScan scans the result of an executed SearchScreenshotsBatch query.
	SearchScreenshotsScan(results pgx.BatchResults) ([]SearchScreenshotsRow, error)

	InsertScreenshotBlocks(ctx context.Context, screenshotID int, body string) (InsertScreenshotBlocksRow, error)
	// InsertScreenshotBlocksBatch enqueues a InsertScreenshotBlocks query into batch to be executed
	// later by the batch.
	InsertScreenshotBlocksBatch(batch *pgx.Batch, screenshotID int, body string)
	// InsertScreenshotBlocksScan scans the result of an executed InsertScreenshotBlocksBatch query.
	InsertScreenshotBlocksScan(results pgx.BatchResults) (InsertScreenshotBlocksRow, error)
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

const searchScreenshotsSQL = `SELECT
  screenshots.id,
  array_agg(blocks) AS blocks
FROM screenshots
  JOIN blocks ON blocks.screenshot_id = screenshots.id
WHERE  blocks.body LIKE $1 || '%'
GROUP BY screenshots.id
ORDER BY id
LIMIT $2 OFFSET $3;`

type SearchScreenshotsParams struct {
	Body   string
	Limit  int
	Offset int
}

type SearchScreenshotsRow struct {
	ID     int      `json:"id"`
	Blocks []Blocks `json:"blocks"`
}

// Blocks represents the Postgres composite type "blocks".
type Blocks struct {
	ID           int    `json:"id"`
	ScreenshotID int    `json:"screenshot_id"`
	Body         string `json:"body"`
}

// ignoredOID means we don't know or care about the OID for a type. This is
// typically okay because we only need the OID when encoding values or when
// relying on pgx to figure out how decode a Postgres query response.
const ignoredOID = 0

// SearchScreenshots implements Querier.SearchScreenshots.
func (q *DBQuerier) SearchScreenshots(ctx context.Context, params SearchScreenshotsParams) ([]SearchScreenshotsRow, error) {
	rows, err := q.conn.Query(ctx, searchScreenshotsSQL, params.Body, params.Limit, params.Offset)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("query SearchScreenshots: %w", err)
	}
	items := []SearchScreenshotsRow{}
	blockRow, _ := pgtype.NewCompositeTypeValues("blocks", []pgtype.CompositeTypeField{
		{Name: "id", OID: pgtype.Int4OID},
		{Name: "screenshot_id", OID: pgtype.Int8OID},
		{Name: "body", OID: pgtype.TextOID},
	}, []pgtype.ValueTranscoder{
		&pgtype.Int4{},
		&pgtype.Int8{},
		&pgtype.Text{},
	})
	blockArray := pgtype.NewArrayType("_block", ignoredOID, func() pgtype.ValueTranscoder {
		return blockRow.NewTypeValue().(*pgtype.CompositeType)
	})
	for rows.Next() {
		var item SearchScreenshotsRow
		if err := rows.Scan(&item.ID, blockArray); err != nil {
			return nil, fmt.Errorf("scan SearchScreenshots row: %w", err)
		}
		blockArray.AssignTo(&item.Blocks)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close SearchScreenshots rows: %w", err)
	}
	return items, err
}

// SearchScreenshotsBatch implements Querier.SearchScreenshotsBatch.
func (q *DBQuerier) SearchScreenshotsBatch(batch *pgx.Batch, params SearchScreenshotsParams) {
	batch.Queue(searchScreenshotsSQL, params.Body, params.Limit, params.Offset)
}

// SearchScreenshotsScan implements Querier.SearchScreenshotsScan.
func (q *DBQuerier) SearchScreenshotsScan(results pgx.BatchResults) ([]SearchScreenshotsRow, error) {
	rows, err := results.Query()
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		return nil, err
	}
	items := []SearchScreenshotsRow{}
	for rows.Next() {
		var item SearchScreenshotsRow
		if err := rows.Scan(&item.ID, &item.Blocks); err != nil {
			return nil, fmt.Errorf("scan SearchScreenshotsBatch row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("close SearchScreenshotsBatch rows: %w", err)
	}
	return items, err
}

const insertScreenshotBlocksSQL = `WITH screens AS (
  INSERT INTO screenshots (id) VALUES ($1)
    ON CONFLICT DO NOTHING
)
INSERT
INTO blocks (screenshot_id, body)
VALUES ($1, $2)
RETURNING id, screenshot_id, body;`

type InsertScreenshotBlocksRow struct {
	ID           int    `json:"id"`
	ScreenshotID int    `json:"screenshot_id"`
	Body         string `json:"body"`
}

// InsertScreenshotBlocks implements Querier.InsertScreenshotBlocks.
func (q *DBQuerier) InsertScreenshotBlocks(ctx context.Context, screenshotID int, body string) (InsertScreenshotBlocksRow, error) {
	row := q.conn.QueryRow(ctx, insertScreenshotBlocksSQL, screenshotID, body)
	var item InsertScreenshotBlocksRow
	if err := row.Scan(&item.ID, &item.ScreenshotID, &item.Body); err != nil {
		return item, fmt.Errorf("query InsertScreenshotBlocks: %w", err)
	}
	return item, nil
}

// InsertScreenshotBlocksBatch implements Querier.InsertScreenshotBlocksBatch.
func (q *DBQuerier) InsertScreenshotBlocksBatch(batch *pgx.Batch, screenshotID int, body string) {
	batch.Queue(insertScreenshotBlocksSQL, screenshotID, body)
}

// InsertScreenshotBlocksScan implements Querier.InsertScreenshotBlocksScan.
func (q *DBQuerier) InsertScreenshotBlocksScan(results pgx.BatchResults) (InsertScreenshotBlocksRow, error) {
	row := results.QueryRow()
	var item InsertScreenshotBlocksRow
	if err := row.Scan(&item.ID, &item.ScreenshotID, &item.Body); err != nil {
		return item, fmt.Errorf("scan InsertScreenshotBlocksBatch row: %w", err)
	}
	return item, nil
}