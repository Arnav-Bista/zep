package mysql

import (
	"context"
	"database/sql"

	// "errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/uptrace/bun/extra/bunotel"

	"github.com/getzep/zep/pkg/store/mysql/migrations"

	"github.com/go-sql-driver/mysql"

	"github.com/getzep/zep/pkg/llms"
	"github.com/uptrace/bun/dialect/mysqldialect"

	"github.com/getzep/zep/pkg/models"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

const defaultEmbeddingDims = 1536

var maxOpenConns = 4 * runtime.GOMAXPROCS(0)

type SessionSchema struct {
	bun.BaseModel `bun:"table:session,alias:s" yaml:"-"`

	UUID      string                 `bun:",pk,type:char(36)"                          yaml:"uuid,omitempty"`
	ID        int64                  `bun:","                                        yaml:"id,omitempty"` // used as a cursor for pagination
	SessionID string                 `bun:",unique,notnull"                                       yaml:"session_id,omitempty"`
	CreatedAt time.Time              `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP()"    yaml:"created_at,omitempty"`
	UpdatedAt time.Time              `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP() ON UPDATE CURRENT_TIMESTAMP()" yaml:"updated_at,omitempty"`
	DeletedAt sql.NullTime           `bun:"type:timestamp,nullzero" yaml:"deleted_at,omitempty"`
	Metadata  map[string]interface{} `bun:"type:json,nullzero"                                    yaml:"metadata,omitempty"`
	UserID    *string                `bun:","                                                           yaml:"user_id,omitempty"`
	User      *UserSchema            `bun:"rel:belongs-to,join:user_id=user_id,on_delete:cascade"       yaml:"-"`
}

var _ bun.BeforeAppendModelHook = (*SessionSchema)(nil)

func (s *SessionSchema) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.UpdateQuery); ok {
		s.UpdatedAt = time.Now()
	}
	return nil
}

type MessageStoreSchema struct {
	bun.BaseModel `bun:"table:message,alias:m" yaml:"-"`

	UUID uuid.UUID `bun:",pk,type:char(36)"                     yaml:"uuid"`
	// ID is used only for sorting / slicing purposes as we can't sort by CreatedAt for messages created simultaneously
	ID         int64                  `bun:","                                              yaml:"id,omitempty"`
	CreatedAt  time.Time              `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP()"          yaml:"created_at,omitempty"`
	UpdatedAt  time.Time              `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP() ON UPDATE CURRENT_TIMESTAMP()" yaml:"updated_at,omitempty"`
	DeletedAt  sql.NullTime           `bun:"type:timestamp,nullzero" yaml:"deleted_at,omitempty"`
	SessionID  string                 `bun:",notnull"                                                    yaml:"session_id,omitempty"`
	Role       string                 `bun:",notnull"                                                    yaml:"role,omitempty"`
	Content    string                 `bun:",notnull"                                                    yaml:"content,omitempty"`
	TokenCount int                    `bun:",notnull"                                                    yaml:"token_count,omitempty"`
	Metadata   map[string]interface{} `bun:"type:json,nullzero"                                          yaml:"metadata,omitempty"`
	Session    *SessionSchema         `bun:"rel:belongs-to,join:session_id=session_id,on_delete:cascade" yaml:"-"`
}

var _ bun.BeforeAppendModelHook = (*MessageStoreSchema)(nil)

func (s *MessageStoreSchema) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.UpdateQuery); ok {
		s.UpdatedAt = time.Now()
	}
	return nil
}

// MessageVectorStoreSchema stores the embeddings for a message.
type MessageVectorStoreSchema struct {
	bun.BaseModel `bun:"table:message_embedding,alias:me"`

	UUID        uuid.UUID           `bun:",pk,type:char(36)"` // Removed default:uuid()
	CreatedAt   time.Time           `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP()"`
	UpdatedAt   time.Time           `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP() ON UPDATE CURRENT_TIMESTAMP()"`
	DeletedAt   sql.NullTime        `bun:"type:timestamp,nullzero" yaml:"deleted_at,omitempty"`
	SessionID   string              `bun:",notnull"`
	MessageUUID uuid.UUID           `bun:"type:char(36),notnull,unique"`
	Embedding   []float32           `bun:"type:json"`
	IsEmbedded  bool                `bun:"type:bool,notnull,default:false"`
	Session     *SessionSchema      `bun:"rel:belongs-to,join:session_id=session_id,on_delete:cascade"`
	Message     *MessageStoreSchema `bun:"rel:belongs-to,join:message_uuid=uuid,on_delete:cascade"`
}

var _ bun.BeforeAppendModelHook = (*MessageVectorStoreSchema)(nil)

func (s *MessageVectorStoreSchema) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.UpdateQuery); ok {
		s.UpdatedAt = time.Now()
	}
	return nil
}

type SummaryStoreSchema struct {
	bun.BaseModel `bun:"table:summary,alias:su" ,yaml:"-"`

	UUID             uuid.UUID              `bun:",pk,type:char(36)"` // Removed default:uuid()
	CreatedAt        time.Time              `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP()"`
	UpdatedAt        time.Time              `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP() ON UPDATE CURRENT_TIMESTAMP()"`
	DeletedAt        sql.NullTime           `bun:"type:timestamp,nullzero" yaml:"deleted_at,omitempty"`
	SessionID        string                 `bun:",notnull"`
	Content          string                 `bun:",nullzero"` // allow null as we might want to use Metadata without a summary
	Metadata         map[string]interface{} `bun:"type:json,nullzero"`
	TokenCount       int                    `bun:",notnull"`
	SummaryPointUUID uuid.UUID              `bun:"type:char(36),notnull,unique"` // the UUID of the most recent message that was used to create the summary
	Session          *SessionSchema         `bun:"rel:belongs-to,join:session_id=session_id,on_delete:cascade"`
	Message          *MessageStoreSchema    `bun:"rel:belongs-to,join:summary_point_uuid=uuid,on_delete:cascade"`
	Facts            []string               `bun:"type:json,nullzero"`
}

var _ bun.BeforeAppendModelHook = (*SummaryStoreSchema)(nil)

func (s *SummaryStoreSchema) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.UpdateQuery); ok {
		s.UpdatedAt = time.Now()
	}
	return nil
}

type SummaryVectorStoreSchema struct {
	bun.BaseModel `bun:"table:summary_embedding,alias:se" yaml:"-"`

	UUID        uuid.UUID           `bun:",pk,type:char(36)"` // Removed default:uuid()
	CreatedAt   time.Time           `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP()"`
	UpdatedAt   time.Time           `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP() ON UPDATE CURRENT_TIMESTAMP()"`
	DeletedAt   sql.NullTime        `bun:"type:timestamp,nullzero" yaml:"deleted_at,omitempty"`
	SessionID   string              `bun:",notnull"`
	SummaryUUID uuid.UUID           `bun:"type:char(36),notnull,unique"`
	Embedding   []float32           `bun:"type:json"`
	IsEmbedded  bool                `bun:"type:bool,notnull,default:false"`
	Summary     *SummaryStoreSchema `bun:"rel:belongs-to,join:summary_uuid=uuid,on_delete:cascade"`
	Session     *SessionSchema      `bun:"rel:belongs-to,join:session_id=session_id,on_delete:cascade"`
}

var _ bun.BeforeAppendModelHook = (*SummaryVectorStoreSchema)(nil)

func (s *SummaryVectorStoreSchema) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.UpdateQuery); ok {
		s.UpdatedAt = time.Now()
	}
	return nil
}

/*
	END CUSTOM ADDITIONS
*/
// DocumentCollectionSchema represents the schema for the DocumentCollectionDAO table.
type DocumentCollectionSchema struct {
	bun.BaseModel             `bun:"table:document_collection,alias:dc" yaml:"-"`
	models.DocumentCollection `                                         yaml:",inline"`
}

var _ bun.BeforeAppendModelHook = (*DocumentCollectionSchema)(nil)

func (s *DocumentCollectionSchema) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.UpdateQuery); ok {
		s.UpdatedAt = time.Now()
	}
	return nil
}

// DocumentSchemaTemplate represents the schema template for Document tables.
// TextData is manually added when createDocumentTable is run in order to set the correct dimensions.
// This means the embedding is not returned when querying using bun.
type DocumentSchemaTemplate struct {
	bun.BaseModel `bun:"table:document,alias:d"`
	models.DocumentBase
}

type UserSchema struct {
	bun.BaseModel `bun:"table:users,alias:u" yaml:"-"`

	UUID      string                 `bun:",pk,type:char(36),notnull"                  yaml:"uuid,omitempty"`
	ID        int64                  `bun:""                                    yaml:"id,omitempty"` // used as a cursor for pagination
	CreatedAt time.Time              `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP()" yaml:"created_at,omitempty"`
	UpdatedAt time.Time              `bun:"type:timestamp,notnull,default:CURRENT_TIMESTAMP() ON UPDATE CURRENT_TIMESTAMP()" yaml:"updated_at,omitempty"`
	DeletedAt sql.NullTime           `bun:"type:timestamp,nullzero" yaml:"deleted_at,omitempty"`
	UserID    string                 `bun:",unique,notnull"                                   yaml:"user_id,omitempty"`
	Email     string                 `bun:","                                                 yaml:"email,omitempty"`
	FirstName string                 `bun:","                                                 yaml:"first_name,omitempty"`
	LastName  string                 `bun:","                                                 yaml:"last_name,omitempty"`
	Metadata  map[string]interface{} `bun:"type:json,nullzero"                                yaml:"metadata,omitempty"`
}

var _ bun.BeforeAppendModelHook = (*UserSchema)(nil)

func (u *UserSchema) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.UpdateQuery); ok {
		u.UpdatedAt = time.Now()
	}
	return nil
}

// Create session_id indexes after table creation
var _ bun.AfterCreateTableHook = (*SessionSchema)(nil)
var _ bun.AfterCreateTableHook = (*MessageStoreSchema)(nil)
var _ bun.AfterCreateTableHook = (*MessageVectorStoreSchema)(nil)
var _ bun.AfterCreateTableHook = (*SummaryStoreSchema)(nil)
var _ bun.AfterCreateTableHook = (*SummaryVectorStoreSchema)(nil)
var _ bun.AfterCreateTableHook = (*UserSchema)(nil)

// Custom
// var _ bun.AfterCreateTableHook = (*FactStoreSchema)(nil)
// var _ bun.AfterCreateTableHook = (*FactVectorStoreSchema)(nil)

// Create Collection Name index after table creation
var _ bun.AfterCreateTableHook = (*DocumentCollectionSchema)(nil)

func (*SessionSchema) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	indexes := []struct {
		name   string
		column string
	}{
		{"session_session_id_idx", "session_id"},
		{"session_user_id_idx", "user_id"},
	}

	for _, index := range indexes {
		var exists bool
		err := query.DB().NewSelect().
			Table("information_schema.statistics").
			ColumnExpr("COUNT(*) > 0").
			Where("table_schema = DATABASE()").
			Where("table_name = 'session'").
			Where("index_name = ?", index.name).
			Scan(ctx, &exists)
		if err != nil {
			return fmt.Errorf("error checking if index %s exists: %w", index.name, err)
		}

		if !exists {
			_, err := query.DB().NewCreateIndex().
				Model((*SessionSchema)(nil)).
				Index(index.name).
				Column(index.column).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("error creating index %s: %w", index.name, err)
			}
		}
	}

	return nil
}

func (*MessageStoreSchema) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	colsToIndex := []string{"session_id", "id"}
	for _, col := range colsToIndex {
		indexName := fmt.Sprintf("memstore_%s_idx", col)
		var exists bool
		err := query.DB().NewSelect().
			Table("information_schema.statistics").
			ColumnExpr("COUNT(*) > 0").
			Where("table_schema = DATABASE()").
			Where("table_name = 'message'").
			Where("index_name = ?", indexName).
			Scan(ctx, &exists)
		if err != nil {
			return fmt.Errorf("error checking if index %s exists: %w", indexName, err)
		}

		if !exists {
			_, err := query.DB().NewCreateIndex().
				Model((*MessageStoreSchema)(nil)).
				Index(indexName).
				Column(col).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("error creating index %s: %w", indexName, err)
			}
		}
	}
	return nil
}

func (*MessageVectorStoreSchema) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	var exists bool
	err := query.DB().NewSelect().
		Table("information_schema.statistics").
		ColumnExpr("COUNT(*) > 0").
		Where("table_schema = DATABASE()").
		Where("table_name = 'message_embedding'").
		Where("index_name = 'mem_vec_store_session_id_idx'").
		Scan(ctx, &exists)
	if err != nil {
		return fmt.Errorf("error checking if index mem_vec_store_session_id_idx exists: %w", err)
	}

	if !exists {
		_, err := query.DB().NewCreateIndex().
			Model((*MessageVectorStoreSchema)(nil)).
			Index("mem_vec_store_session_id_idx").
			Column("session_id").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("error creating index mem_vec_store_session_id_idx: %w", err)
		}
	}
	return nil
}

func (*SummaryStoreSchema) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	var exists bool
	err := query.DB().NewSelect().
		Table("information_schema.statistics").
		ColumnExpr("COUNT(*) > 0").
		Where("table_schema = DATABASE()").
		Where("table_name = 'summary'").
		Where("index_name = 'sumstore_session_id_idx'").
		Scan(ctx, &exists)
	if err != nil {
		return fmt.Errorf("error checking if index sumstore_session_id_idx exists: %w", err)
	}

	if !exists {
		_, err := query.DB().NewCreateIndex().
			Model((*SummaryStoreSchema)(nil)).
			Index("sumstore_session_id_idx").
			Column("session_id").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("error creating index sumstore_session_id_idx: %w", err)
		}
	}
	return nil
}

func (*SummaryVectorStoreSchema) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	var exists bool
	err := query.DB().NewSelect().
		Table("information_schema.statistics").
		ColumnExpr("COUNT(*) > 0").
		Where("table_schema = DATABASE()").
		Where("table_name = 'summary_embedding'").
		Where("index_name = 'sumvecstore_session_id_idx'").
		Scan(ctx, &exists)
	if err != nil {
		return fmt.Errorf("error checking if index sumvecstore_session_id_idx exists: %w", err)
	}

	if !exists {
		_, err := query.DB().NewCreateIndex().
			Model((*SummaryVectorStoreSchema)(nil)).
			Index("sumvecstore_session_id_idx").
			Column("session_id").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("error creating index sumvecstore_session_id_idx: %w", err)
		}
	}
	return nil
}

// CUSTOM

// func (*FactStoreSchema) AfterCreateTable(
// 	ctx context.Context,
// 	query *bun.CreateTableQuery,
// ) error {
// 	_, err := query.DB().NewCreateIndex().
// 		Model((*FactStoreSchema)(nil)).
// 		Index("factstore_session_id_idx").
// 		Column("session_id").
// 		Exec(ctx)
// 	return err
// }

// func (*FactVectorStoreSchema) AfterCreateTable(
// 	ctx context.Context,
// 	query *bun.CreateTableQuery,
// ) error {
// 	_, err := query.DB().NewCreateIndex().
// 		Model((*SummaryVectorStoreSchema)(nil)).
// 		Index("factvecstore_session_id_idx").
// 		Column("session_id").
// 		Exec(ctx)
// 	return err
// }

// END CUSTOM

func (*DocumentCollectionSchema) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	// Check if the index already exists
	var exists bool
	err := query.DB().NewSelect().
		Table("information_schema.statistics").
		ColumnExpr("COUNT(*) > 0").
		Where("table_schema = DATABASE()").
		Where("table_name = 'document_collection'").
		Where("index_name = 'document_collection_name_idx'").
		Scan(ctx, &exists)
	if err != nil {
		return fmt.Errorf("error checking if index exists: %w", err)
	}

	// If the index does not exist, create it
	if !exists {
		_, err := query.DB().NewCreateIndex().
			Model((*DocumentCollectionSchema)(nil)).
			Index("document_collection_name_idx").
			Column("name").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("error creating index: %w", err)
		}
	}

	return nil
}

func (*UserSchema) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	// Check if the user_user_id_idx index already exists
	var exists bool
	err := query.DB().NewSelect().
		Table("information_schema.statistics").
		ColumnExpr("COUNT(*) > 0").
		Where("table_schema = DATABASE()").
		Where("table_name = 'users'").
		Where("index_name = 'user_user_id_idx'").
		Scan(ctx, &exists)
	if err != nil {
		return fmt.Errorf("error checking if user_user_id_idx index exists: %w", err)
	}

	// If the index does not exist, create it
	if !exists {
		_, err = query.DB().NewCreateIndex().
			Model((*UserSchema)(nil)).
			Index("user_user_id_idx").
			Column("user_id").
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	// Check if the user_email_idx index already exists
	err = query.DB().NewSelect().
		Table("information_schema.statistics").
		ColumnExpr("COUNT(*) > 0").
		Where("table_schema = DATABASE()").
		Where("table_name = 'users'").
		Where("index_name = 'user_email_idx'").
		Scan(ctx, &exists)
	if err != nil {
		return fmt.Errorf("error checking if user_email_idx index exists: %w", err)
	}

	// If the index does not exist, create it
	if !exists {
		_, err = query.DB().NewCreateIndex().
			Model((*UserSchema)(nil)).
			Index("user_email_idx").
			Column("email").
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

var messageTableList = []bun.AfterCreateTableHook{
	&MessageVectorStoreSchema{},
	&SummaryVectorStoreSchema{},
	&SummaryStoreSchema{},
	&MessageStoreSchema{},
	&SessionSchema{},
}

// generateDocumentTableName generates a table name for a collection.
// If the table already exists, the table is not recreated.
func createDocumentTable(
	ctx context.Context,
	appState *models.AppState,
	db *bun.DB,
	tableName string,
	embeddingDimensions int,
) error {
	schema := &DocumentSchemaTemplate{}
	_, err := db.NewCreateTable().
		Model(schema).
		// override default table name
		ModelTableExpr("?", bun.Ident(tableName)).
		// create the embedding column using the provided dimensions
		ColumnExpr("embedding JSON").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("error creating document table: %w", err)
	}

	// Create document_id index
	_, err = db.NewCreateIndex().
		Model(schema).
		// override default table name
		ModelTableExpr("?", bun.Ident(tableName)).
		Index(tableName + "document_id_idx").
		Column("document_id").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("error creating session_session_id_idx: %w", err)
	}

	return nil
}

// CreateSchema creates the db schema if it does not exist.
func CreateSchema(
	ctx context.Context,
	appState *models.AppState,
	db *bun.DB,
) error {
	// Create new tableList slice and append DocumentCollectionSchema to it
	tableList := append( //nolint:gocritic
		messageTableList,
		&UserSchema{},
		&DocumentCollectionSchema{},
	)
	// iterate through messageTableList in reverse order to create tables with foreign keys first
	for i := len(tableList) - 1; i >= 0; i-- {
		schema := tableList[i]
		_, err := db.NewCreateTable().
			Model(schema).
			IfNotExists().
			WithForeignKeys().
			Exec(ctx)
		if err != nil {
			// bun still trying to create indexes despite IfNotExists flag
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("error creating table for schema %T: %w", schema, err)
		}
	}

	// apply migrations
	if err := migrations.Migrate(ctx, db); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// check that the message and summary embedding dimensions match the configured model
	if err := checkEmbeddingDims(ctx, appState, db, "message", "message_embedding"); err != nil {
		return fmt.Errorf("error checking message embedding dimensions: %w", err)
	}
	if err := checkEmbeddingDims(ctx, appState, db, "summary", "summary_embedding"); err != nil {
		return fmt.Errorf("error checking summary embedding dimensions: %w", err)
	}

	// Custom
	// if err := checkEmbeddingDims(ctx, appState, db, "fact", "fact_embedding"); err != nil {
	// 	return fmt.Errorf("error checking fact embedding dimensions: %w", err)
	// }
	// ---

	return nil
}

// checkEmbeddingDims checks the dimensions of the embedding column against the
// dimensions of the configured embedding model. If they do not match, the column is dropped and
// recreated with the correct dimensions.
func checkEmbeddingDims(
	ctx context.Context,
	appState *models.AppState,
	db *bun.DB,
	documentType string,
	tableName string,
) error {
	model, err := llms.GetEmbeddingModel(appState, documentType)
	if err != nil {
		return fmt.Errorf("error getting %s embedding model: %w", documentType, err)
	}
	width, err := getEmbeddingColumnWidth(ctx, tableName, db)
	if err != nil {
		return fmt.Errorf("error getting embedding column width: %w", err)
	}

	if width != model.Dimensions {
		log.Warnf(
			"%s embedding dimensions are %d, expected %d.\n migrating %s embedding column width to %d. this may result in loss of existing embedding vectors",
			documentType,
			width,
			model.Dimensions,
			documentType,
			model.Dimensions,
		)
		err := MigrateEmbeddingDims(ctx, db, tableName, model.Dimensions)
		if err != nil {
			return fmt.Errorf("error migrating %s embedding dimensions: %w", documentType, err)
		}
	}
	return nil
}

// getEmbeddingColumnWidth returns the width of the embedding column in the provided table.
func getEmbeddingColumnWidth(ctx context.Context, tableName string, db *bun.DB) (int, error) {
	var width int
	err := db.NewSelect().
		Table("information_schema.columns").
		ColumnExpr("CHARACTER_MAXIMUM_LENGTH").
		Where("table_name = ?", tableName).
		Where("column_name = 'embedding'").
		Scan(ctx, &width)
	if err != nil {
		// Something strange has happened. Debug the schema.
		schema, dumpErr := dumpTableSchema(ctx, db, tableName)
		if dumpErr != nil {
			return 0, fmt.Errorf(
				"error getting embedding column width for %s: %w. Original error: %w",
				tableName,
				dumpErr,
				err,
			)
		}
		return 0, fmt.Errorf(
			"error getting embedding column width for %s. Schema: %s: %w",
			tableName,
			schema,
			err,
		)
	}
	return width, nil
}

// dumpTableSchema enables debugging of schema issues
func dumpTableSchema(ctx context.Context, db *bun.DB, tableName string) (string, error) {
	type ColumnInfo struct {
		bun.BaseModel `bun:"table:information_schema.columns" yaml:"-"`
		ColumnName    string         `bun:"column_name"`
		DataType      string         `bun:"data_type"`
		CharMaxLength sql.NullInt32  `bun:"character_maximum_length"`
		ColumnDefault sql.NullString `bun:"column_default"`
		IsNullable    string         `bun:"is_nullable"`
	}

	var columns []ColumnInfo
	err := db.NewSelect().
		Model(&columns).
		Where("table_name = ?", tableName).
		Order("ordinal_position").
		Scan(ctx)
	if err != nil {
		return "", fmt.Errorf("error getting table schema for %s: %w", tableName, err)
	}

	tableSchema := fmt.Sprintf("%+v", columns)

	return tableSchema, nil
}

// MigrateEmbeddingDims drops the old embedding column and creates a new one with the
// correct dimensions.
func MigrateEmbeddingDims(
	ctx context.Context,
	db *bun.DB,
	tableName string,
	dimensions int,
) error {
	// we may be missing a config key, so use the default dimensions if none are provided
	if dimensions == 0 {
		dimensions = defaultEmbeddingDims
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("MigrateEmbeddingDims error starting transaction: %w", err)
	}
	defer rollbackOnError(tx)

	// MySQL doesn't directly support vector types, so we're using JSON arrays
	columnQuery := `ALTER TABLE ? DROP COLUMN IF EXISTS embedding;
	ALTER TABLE ? ADD COLUMN embedding JSON;
`
	_, err = tx.ExecContext(
		ctx,
		columnQuery,
		bun.Ident(tableName),
		bun.Ident(tableName),
	)
	if err != nil {
		return fmt.Errorf("MigrateEmbeddingDims error dropping column embedding: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("MigrateEmbeddingDims error committing transaction: %w", err)
	}

	return nil
}

// NewMySQLConn creates a new bun.DB connection to a MySQL database using the provided DSN.
// The connection is configured to pool connections based on the number of PROCs available.
func NewMySQLConn(appState *models.AppState) (*bun.DB, error) {
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Configure the MySQL DSN and settings
	cfg := mysql.NewConfig()
	cfg.Net = "tcp"
	cfg.Addr = appState.Config.Store.MySQL.Address
	cfg.User = appState.Config.Store.MySQL.Username
	cfg.Passwd = appState.Config.Store.MySQL.Password
	cfg.DBName = appState.Config.Store.MySQL.DatabaseName
	cfg.ParseTime = true
	cfg.ReadTimeout = 10 * time.Minute

	// Open the database connection
	sqldb, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxOpenConns)

	// Create a new bun.DB instance
	db := bun.NewDB(sqldb, mysqldialect.New())
	db.AddQueryHook(bunotel.NewQueryHook(bunotel.WithDBName("zep")))

	return db, nil
}

// NewMySQLConnForQueue creates a new MySQL connection to a MySQL database using the provided DSN.
// This connection is intended to be used for queueing tasks.
func NewMySQLConnForQueue(appState *models.AppState) (*sql.DB, error) {
	// Configure the MySQL DSN and settings
	cfg := mysql.NewConfig()
	cfg.Net = "tcp"
	cfg.Addr = appState.Config.Store.MySQL.Address
	cfg.User = appState.Config.Store.MySQL.Username
	cfg.Passwd = appState.Config.Store.MySQL.Password
	cfg.DBName = appState.Config.Store.MySQL.DatabaseName
	cfg.ParseTime = true
	cfg.ReadTimeout = 10 * time.Minute

	// Open the database connection
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	return db, nil
}
