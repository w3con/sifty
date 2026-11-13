package store

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
)

// saveOriginalInTx writes the raw body for jobID to job_originals within tx,
// gzip-compressing it first. An existing row is replaced. A nil or empty body is
// a no-op. The caller owns the transaction and must commit it.
func saveOriginalInTx(tx *sql.Tx, jobID int64, kind, fetchedAt string, body []byte) error {
	if len(body) == 0 {
		return nil
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(body); err != nil {
		return fmt.Errorf("gzip original for job %d: %w", jobID, err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close for job %d: %w", jobID, err)
	}
	if _, err := tx.Exec(
		"INSERT OR REPLACE INTO job_originals(job_id, kind, fetched_at, body) VALUES(?,?,?,?)",
		jobID, kind, fetchedAt, buf.Bytes(),
	); err != nil {
		return fmt.Errorf("insert job_originals for job %d: %w", jobID, err)
	}
	return nil
}

// GetOriginal retrieves and decompresses the stored raw body for jobID.
// Returns ("", "", nil, nil) when no original is stored for that job.
func (s *Store) GetOriginal(jobID int64) (kind, fetchedAt string, body []byte, err error) {
	var compressed []byte
	if err = s.DB.QueryRow(
		"SELECT kind, fetched_at, body FROM job_originals WHERE job_id=?", jobID,
	).Scan(&kind, &fetchedAt, &compressed); err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil, nil
		}
		return "", "", nil, fmt.Errorf("get original for job %d: %w", jobID, err)
	}
	r, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return kind, fetchedAt, nil, fmt.Errorf("gzip reader for job %d: %w", jobID, err)
	}
	defer r.Close()
	body, err = io.ReadAll(r)
	if err != nil {
		return kind, fetchedAt, nil, fmt.Errorf("decompress original for job %d: %w", jobID, err)
	}
	return kind, fetchedAt, body, nil
}
