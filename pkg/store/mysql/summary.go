package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/getzep/zep/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// NewSummaryDAO creates a new SummaryDAO.
func NewSummaryDAO(db *bun.DB, appState *models.AppState, sessionID string) (*SummaryDAO, error) {
	if sessionID == "" {
		return nil, errors.New("sessionID cannot be empty")
	}
	return &SummaryDAO{
		db:        db,
		appState:  appState,
		sessionID: sessionID,
	}, nil
}

type SummaryDAO struct {
	db        *bun.DB
	appState  *models.AppState
	sessionID string
}

// Create stores a new summary for a session. The SummaryPointUUID is the UUID of the most recent
// message in the session when the summary was created.
func (s *SummaryDAO) Create(
	ctx context.Context,
	summary *models.Summary,
) (*models.Summary, error) {
	mysqlSummary := &SummaryStoreSchema{
		SessionID:        s.sessionID,
		Content:          summary.Content,
		Metadata:         summary.Metadata,
		SummaryPointUUID: summary.SummaryPointUUID,
		TokenCount:       summary.TokenCount,
		Facts:            summary.Facts,
	}

	_, err := s.db.NewInsert().Model(mysqlSummary).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create summary %w", err)
	}

	return &models.Summary{
		UUID:             mysqlSummary.UUID,
		CreatedAt:        mysqlSummary.CreatedAt,
		Content:          mysqlSummary.Content,
		SummaryPointUUID: mysqlSummary.SummaryPointUUID,
		Metadata:         mysqlSummary.Metadata,
		TokenCount:       mysqlSummary.TokenCount,
		Facts:            mysqlSummary.Facts,
	}, nil
}

func (s *SummaryDAO) Update(
	ctx context.Context,
	summary *models.Summary,
	includeContent bool,
) (*models.Summary, error) {
	if summary.UUID == uuid.Nil {
		return nil, errors.New("summary UUID cannot be empty")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer rollbackOnError(tx)

	metadata, err := mergeMetadata(
		ctx,
		tx,
		"uuid",
		summary.UUID.String(),
		"summary",
		summary.Metadata,
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update summary metadata: %w", err)
	}

	mysqlSummary := &SummaryStoreSchema{
		UUID:       summary.UUID,
		Content:    summary.Content,
		Metadata:   metadata,
		TokenCount: summary.TokenCount,
	}

	columns := []string{"metadata", "token_count"}
	if includeContent {
		columns = append(columns, "content")
	}
	_, err = tx.NewUpdate().
		Model(mysqlSummary).
		Column(columns...).
		Where("uuid = ?", summary.UUID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update summary metadata: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return summary, nil
}

// Get returns the most recent summary for a session
func (s *SummaryDAO) Get(ctx context.Context) (*models.Summary, error) {
	summary := SummaryStoreSchema{}
	err := s.db.NewSelect().
		Model(&summary).
		Where("session_id = ?", s.sessionID).
		Where("deleted_at IS NULL").
		// Get the most recent summary
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.Summary{}, nil
		}
		return &models.Summary{}, fmt.Errorf("failed to get session %w", err)
	}

	return &models.Summary{
		UUID:             summary.UUID,
		CreatedAt:        summary.CreatedAt,
		Content:          summary.Content,
		SummaryPointUUID: summary.SummaryPointUUID,
		Metadata:         summary.Metadata,
		TokenCount:       summary.TokenCount,
		Facts:            summary.Facts,
	}, nil
}

// GetByUUID returns a summary by UUID
func (s *SummaryDAO) GetByUUID(
	ctx context.Context,
	uuid uuid.UUID) (*models.Summary, error) {
	summary := SummaryStoreSchema{}
	err := s.db.NewSelect().
		Model(&summary).
		Where("session_id = ?", s.sessionID).
		Where("uuid = ?", uuid).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.NewNotFoundError("summary " + uuid.String())
		}
		return &models.Summary{}, fmt.Errorf("failed to get session %w", err)
	}

	return &models.Summary{
		UUID:             summary.UUID,
		CreatedAt:        summary.CreatedAt,
		Content:          summary.Content,
		SummaryPointUUID: summary.SummaryPointUUID,
		Metadata:         summary.Metadata,
		TokenCount:       summary.TokenCount,
		Facts:            summary.Facts,
	}, nil
}

// PutEmbedding stores a summary embedding
func (s *SummaryDAO) PutEmbedding(
	ctx context.Context,
	embedding *models.TextData,
) error {
	record := SummaryVectorStoreSchema{
		SessionID:   s.sessionID,
		Embedding:   embedding.Embedding,
		SummaryUUID: embedding.TextUUID,
		IsEmbedded:  true,
	}
	_, err := s.db.NewInsert().Model(&record).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert summary embedding %w", err)
	}

	return nil
}

// GetEmbeddings retrieves all summary embeddings for a session. Note: Does not return the summary content.
func (s *SummaryDAO) GetEmbeddings(
	ctx context.Context,
) ([]models.TextData, error) {
	var embeddings []SummaryVectorStoreSchema
	err := s.db.NewSelect().
		Model(&embeddings).
		Where("session_id = ?", s.sessionID).
		Where("is_embedded = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary embeddings %w", err)
	}

	retEmbeddings := make([]models.TextData, len(embeddings))
	for i, embedding := range embeddings {
		retEmbeddings[i] = models.TextData{
			TextUUID:  embedding.SummaryUUID,
			Embedding: embedding.Embedding,
		}
	}

	return retEmbeddings, nil
}

// GetList returns a list of summaries for a session
func (s *SummaryDAO) GetList(ctx context.Context,
	currentPage int,
	pageSize int,
) (*models.SummaryListResponse, error) {
	var summariesDB []SummaryStoreSchema
	err := s.db.NewSelect().
		Model(&summariesDB).
		Where("session_id = ?", s.sessionID).
		Where("deleted_at IS NULL").
		Order("created_at ASC").
		Offset((currentPage - 1) * pageSize).
		Limit(pageSize).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get sessions %w", err)
	}

	summaries := make([]models.Summary, len(summariesDB))
	for i, summary := range summariesDB {
		summaries[i] = models.Summary{
			UUID:             summary.UUID,
			CreatedAt:        summary.CreatedAt,
			Content:          summary.Content,
			SummaryPointUUID: summary.SummaryPointUUID,
			Metadata:         summary.Metadata,
			TokenCount:       summary.TokenCount,
			Facts:            summary.Facts,
		}
	}

	respSummary := models.SummaryListResponse{
		Summaries: summaries,
		RowCount:  len(summaries),
	}

	return &respSummary, nil
}