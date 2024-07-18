package mysql

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// purgeDeleted hard deletes all soft deleted records from the memory store.
func purgeDeleted(ctx context.Context, db *bun.DB) error {
	log.Debugf("purging memory store")

	for _, schema := range messageTableList {
		log.Debugf("purging schema %T", schema)
		_, err := db.NewDelete().
			Model(schema).
			WhereDeleted().
			ForceDelete().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("error purging rows from %T: %w", schema, err)
		}
	}

	// Optimize and analyze the database post-purge. This is to ensure that the indexes are updated.
	_, err := db.ExecContext(ctx, "OPTIMIZE TABLE message")
	if err != nil {
		return fmt.Errorf("error optimizing table: %w", err)
	}

	log.Info("completed purging store")

	return nil
}
