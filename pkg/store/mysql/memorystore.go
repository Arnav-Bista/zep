package mysql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/getzep/zep/pkg/store"
	"github.com/google/uuid"

	"github.com/getzep/zep/internal"

	"github.com/getzep/zep/pkg/models"
	"github.com/uptrace/bun"
	// _ "github.com/go-sql-driver/mysql"
)

var log = internal.GetLogger()

// NewMySQLMemoryStore returns a new MySQLMemoryStore. Use this to correctly initialize the store.
func NewMySQLMemoryStore(
	appState *models.AppState,
	client *bun.DB,
) (*MySQLMemoryStore, error) {
	if appState == nil {
		return nil, store.NewStorageError("nil appState received", nil)
	}

	mms := &MySQLMemoryStore{
		BaseMemoryStore: store.BaseMemoryStore[*bun.DB]{Client: client},
		SessionStore:    NewSessionDAO(client),
		appState:        appState,
	}

	err := mms.OnStart(context.Background())
	if err != nil {
		return nil, store.NewStorageError("failed to run OnInit", err)
	}
	return mms, nil
}

// Force compiler to validate that MySQLMemoryStore implements the MemoryStore interface.
var _ models.MemoryStore[*bun.DB] = &MySQLMemoryStore{}

type MySQLMemoryStore struct {
	store.BaseMemoryStore[*bun.DB]
	SessionStore *SessionDAO
	appState     *models.AppState
}

func (mms *MySQLMemoryStore) OnStart(
	ctx context.Context,
) error {
	err := CreateSchema(ctx, mms.appState, mms.Client)
	if err != nil {
		return store.NewStorageError("failed to ensure mysql schema setup", err)
	}

	return nil
}

func (mms *MySQLMemoryStore) GetClient() *bun.DB {
	return mms.Client
}

// GetSession retrieves a Session for a given sessionID.
func (mms *MySQLMemoryStore) GetSession(
	ctx context.Context,
	sessionID string,
) (*models.Session, error) {
	return mms.SessionStore.Get(ctx, sessionID)
}

// CreateSession creates or updates a Session for a given sessionID.
func (mms *MySQLMemoryStore) CreateSession(
	ctx context.Context,
	session *models.CreateSessionRequest,
) (*models.Session, error) {
	return mms.SessionStore.Create(ctx, session)
}

// UpdateSession creates or updates a Session for a given sessionID.
func (mms *MySQLMemoryStore) UpdateSession(
	ctx context.Context,
	session *models.UpdateSessionRequest,
) (*models.Session, error) {
	return mms.SessionStore.Update(ctx, session, false)
}

// DeleteSession deletes a session from the memory store. This is a soft Delete.
func (mms *MySQLMemoryStore) DeleteSession(ctx context.Context, sessionID string) error {
	return mms.SessionStore.Delete(ctx, sessionID)
}

// ListSessions returns a list of all Sessions.
func (mms *MySQLMemoryStore) ListSessions(
	ctx context.Context,
	cursor int64,
	limit int,
) ([]*models.Session, error) {
	return mms.SessionStore.ListAll(ctx, cursor, limit)
}

// ListSessionsOrdered returns an ordered list of all Sessions, paginated by pageNumber and pageSize.
// orderedBy is the column to order by. asc is a boolean indicating whether to order ascending or descending.
func (mms *MySQLMemoryStore) ListSessionsOrdered(
	ctx context.Context,
	pageNumber int,
	pageSize int,
	orderedBy string,
	asc bool,
) (*models.SessionListResponse, error) {
	return mms.SessionStore.ListAllOrdered(ctx, pageNumber, pageSize, orderedBy, asc)
}

// GetMemory returns the most recent Summary and a list of messages for a given sessionID.
// GetMemory returns:
//   - the most recent Summary, if one exists
//   - the lastNMessages messages, if lastNMessages > 0
//   - all messages since the last SummaryPoint, if lastNMessages == 0
//   - if no Summary (and no SummaryPoint) exists and lastNMessages == 0, returns
//     all undeleted messages up to the configured message window
//   CUSTOM ADDITION
//   - all facts for that session
func (mms *MySQLMemoryStore) GetMemory(
	ctx context.Context,
	sessionID string,
	lastNMessages int,
) (*models.Memory, error) {
	if lastNMessages < 0 {
		return nil, errors.New("cannot specify negative lastNMessages")
	}

	memoryDAO, err := NewMemoryDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create memoryDAO: %w", err)
	}

	return memoryDAO.Get(ctx, lastNMessages)
}

func (mms *MySQLMemoryStore) PutMemory(
	ctx context.Context,
	sessionID string,
	memoryMessages *models.Memory,
	skipNotify bool,
) error {
	log.Debugf("PutMemory called for session %s =============================================================", sessionID)
	memoryDAO, err := NewMemoryDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return fmt.Errorf("failed to create memoryDAO: %w", err)
	}

	return memoryDAO.Create(ctx, memoryMessages, skipNotify)
}

// GetMessageList retrieves a list of messages for a given sessionID. Paginated by cursor and limit.
func (mms *MySQLMemoryStore) GetMessageList(
	ctx context.Context,
	sessionID string,
	pageNumber int,
	pageSize int,
) (*models.MessageListResponse, error) {
	messageDAO, err := NewMessageDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create messageDAO: %w", err)
	}

	return messageDAO.GetListBySession(ctx, pageNumber, pageSize)
}

func (mms *MySQLMemoryStore) GetMessagesByUUID(
	ctx context.Context,
	sessionID string,
	uuids []uuid.UUID,
) ([]models.Message, error) {
	messageDAO, err := NewMessageDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create messageDAO: %w", err)
	}

	return messageDAO.GetListByUUID(ctx, uuids)
}

func (mms *MySQLMemoryStore) GetSummary(
	ctx context.Context,
	sessionID string,
) (*models.Summary, error) {
	summaryDAO, err := NewSummaryDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create summaryDAO: %w", err)
	}

	return summaryDAO.Get(ctx)
}

func (mms *MySQLMemoryStore) GetSummaryByUUID(
	ctx context.Context,
	sessionID string,
	uuid uuid.UUID) (*models.Summary, error) {
	summaryDAO, err := NewSummaryDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create summaryDAO: %w", err)
	}

	return summaryDAO.GetByUUID(ctx, uuid)
}

func (mms *MySQLMemoryStore) GetSummaryList(
	ctx context.Context,
	sessionID string,
	pageNumber int,
	pageSize int,
) (*models.SummaryListResponse, error) {
	summaryDAO, err := NewSummaryDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create summaryDAO: %w", err)
	}

	return summaryDAO.GetList(ctx, pageNumber, pageSize)
}

func (mms *MySQLMemoryStore) CreateSummary(
	ctx context.Context,
	sessionID string,
	summary *models.Summary,
) error {
	summaryDAO, err := NewSummaryDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return fmt.Errorf("failed to create summaryDAO: %w", err)
	}

	retSummary, err := summaryDAO.Create(ctx, summary)
	if err != nil {
		return store.NewStorageError("failed to create summary", err)
	}

	// Publish a message to the message summary embeddings topic
	task := models.MessageSummaryTask{
		UUID: retSummary.UUID,
	}
	err = mms.appState.TaskPublisher.Publish(
		models.MessageSummaryEmbedderTopic,
		map[string]string{
			"session_id": sessionID,
		},
		task,
	)
	if err != nil {
		return fmt.Errorf("MessageSummaryTask publish failed: %w", err)
	}

	err = mms.appState.TaskPublisher.Publish(
		models.MessageSummaryNERTopic,
		map[string]string{
			"session_id": sessionID,
		},
		task,
	)
	if err != nil {
		return fmt.Errorf("MessageSummaryTask publish failed: %w", err)
	}

	return nil
}

func (mms *MySQLMemoryStore) UpdateSummary(ctx context.Context,
	sessionID string,
	summary *models.Summary,
	metadataOnly bool,
) error {
	summaryDAO, err := NewSummaryDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return fmt.Errorf("failed to create summaryDAO: %w", err)
	}

	_, err = summaryDAO.Update(ctx, summary, metadataOnly)
	if err != nil {
		return fmt.Errorf("failed to update summary metadata %w", err)
	}

	return nil
}

func (mms *MySQLMemoryStore) PutSummaryEmbedding(
	ctx context.Context,
	sessionID string,
	embedding *models.TextData,
) error {
	summaryDAO, err := NewSummaryDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return fmt.Errorf("failed to create summaryDAO: %w", err)
	}

	return summaryDAO.PutEmbedding(ctx, embedding)
}

// CUSTOM

// func (mms *MySQLMemoryStore) CreateFacts(
// 	ctx context.Context,
// 	sessionID string,
// 	facts []models.Fact,
// ) error {
// 	factDAO, err := NewFactDAO(mms.Client, mms.appState, sessionID)
// 	if err != nil {
// 		return fmt.Errorf("failed to create factDAO: %w", err)
// 	}
// 	_, err = factDAO.CreateMany(ctx, facts)
// 	if err != nil {
// 		return fmt.Errorf("failed to create facts: %w", err)
// 	}
// 	// PUBLISHING!
//
// 	// task := models.FactExtractorTask{
// 	// 	UUID: retFacts[0].UUID,
// 	// }
//
//
// 	
// 	return nil
// }
//

// CUSTOM END

func (mms *MySQLMemoryStore) UpdateMessages(
	ctx context.Context,
	sessionID string,
	messages []models.Message,
	isPrivileged bool,
	includeContent bool,
) error {
	messageDAO, err := NewMessageDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return fmt.Errorf("failed to create messageDAO: %w", err)
	}

	return messageDAO.UpdateMany(ctx, messages, includeContent, isPrivileged)
}

func (mms *MySQLMemoryStore) SearchMemory(
	ctx context.Context,
	sessionID string,
	query *models.MemorySearchPayload,
	limit int,
) ([]models.MemorySearchResult, error) {
	memoryDAO, err := NewMemoryDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create memoryDAO: %w", err)
	}
	return memoryDAO.Search(ctx, query, limit)
}

func (mms *MySQLMemoryStore) Close() error {
	if mms.Client != nil {
		return mms.Client.Close()
	}
	return nil
}

func (mms *MySQLMemoryStore) CreateMessageEmbeddings(ctx context.Context,
	sessionID string,
	embeddings []models.TextData,
) error {
	messageDAO, err := NewMessageDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return fmt.Errorf("failed to create messageDAO: %w", err)
	}

	return messageDAO.CreateEmbeddings(ctx, embeddings)
}

func (mms *MySQLMemoryStore) GetMessageEmbeddings(ctx context.Context,
	sessionID string,
) ([]models.TextData, error) {
	messageDAO, err := NewMessageDAO(mms.Client, mms.appState, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create messageDAO: %w", err)
	}

	return messageDAO.GetEmbeddingListBySession(ctx)
}

func (mms *MySQLMemoryStore) PurgeDeleted(ctx context.Context) error {
	err := purgeDeleted(ctx, mms.Client)
	if err != nil {
		return store.NewStorageError("failed to purge deleted", err)
	}

	return nil
}

func generateLockID(key string) uint64 {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hash := hasher.Sum(nil)
	return binary.BigEndian.Uint64(hash[:8])
}

// tryAcquireAdvisoryLock attempts to acquire a MySQL advisory lock using GET_LOCK.
// This function will fail if it's unable to immediately acquire a lock.
// Accepts a bun.IDB, which can be either a *bun.DB or *bun.Tx.
// Returns the lock ID and a boolean indicating if the lock was successfully acquired.
func tryAcquireAdvisoryLock(ctx context.Context, db bun.IDB, key string) (uint64, error) {
	lockID := generateLockID(key)

	var acquired bool
	if err := db.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", lockID).Scan(&acquired); err != nil {
		return 0, fmt.Errorf("tryAcquireAdvisoryLock: %w", err)
	}
	if !acquired {
		return 0, models.NewAdvisoryLockError(fmt.Errorf("failed to acquire advisory lock for %s", key))
	}
	return lockID, nil
}

// acquireAdvisoryLock acquires a MySQL advisory lock for the given key.
func acquireAdvisoryLock(ctx context.Context, db bun.IDB, key string) (uint64, error) {
	lockID := generateLockID(key)

	if _, err := db.ExecContext(ctx, "SELECT GET_LOCK(?, -1)", lockID); err != nil {
		return 0, store.NewStorageError("failed to acquire advisory lock", err)
	}

	return lockID, nil
}

// releaseAdvisoryLock releases a MySQL advisory lock for the given key.
// Accepts a bun.IDB, which can be either a *bun.DB or *bun.Tx.
func releaseAdvisoryLock(ctx context.Context, db bun.IDB, lockID uint64) error {
	if _, err := db.ExecContext(ctx, "SELECT RELEASE_LOCK(?)", lockID); err != nil {
		return store.NewStorageError("failed to release advisory lock", err)
	}

	return nil
}

// rollbackOnError rolls back the transaction if an error is encountered.
// If the error is sql.ErrTxDone, the transaction has already been committed or rolled back
// and we ignore the error.
func rollbackOnError(tx bun.Tx) {
	if rollBackErr := tx.Rollback(); rollBackErr != nil && !errors.Is(rollBackErr, sql.ErrTxDone) {
		log.Error("failed to rollback transaction", rollBackErr)
	}
}
