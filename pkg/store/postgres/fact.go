// ENTIRE FILE IS CUSTOM
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pgvector/pgvector-go"

	"github.com/getzep/zep/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func NewFactDAO(db *bun.DB, appState *models.AppState, sessionID string) (*FactDAO, error) {
	if sessionID == "" {
		return nil, errors.New("sessionID cannot be empty")
	}
	return &FactDAO{
		db:        db,
		appState:  appState,
		sessionID: sessionID,
	}, nil
}

type FactDAO struct {
	db        *bun.DB
	appState  *models.AppState
	sessionID string
}

// Create create a new fact for a session. Does not create a session if it does not exist.
func (dao *FactDAO) Create(
	ctx context.Context,
	fact *models.Fact,
) (*models.Fact, error) {
	pgFact := FactStoreSchema{
		UUID:       fact.UUID,
		SessionID:  dao.sessionID,
		Content:    fact.Content,
		TokenCount: fact.TokenCount,
		Metadata:   fact.Metadata,
	}

	// Insert
	_, err := dao.db.NewInsert().
		Model(&pgFact).
		Returning("*").
		Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return &models.Fact{
		UUID:       pgFact.UUID,
		CreatedAt:  pgFact.CreatedAt,
		UpdatedAt:  pgFact.UpdatedAt,
		Content:    pgFact.Content,
		TokenCount: pgFact.TokenCount,
		Metadata:   pgFact.Metadata,
	}, nil
}

func (dao *FactDAO) CreateMany(
	ctx context.Context,
	facts []models.Fact,
) ([]models.Fact, error) {
	if len(facts) == 0 {
		return nil, nil
	}

	pgFacts := make([]FactStoreSchema, len(facts))
	for i, fact := range facts {
		pgFacts[i] = FactStoreSchema{
			UUID:       fact.UUID,
			SessionID:  dao.sessionID,
			Content:    fact.Content,
			TokenCount: fact.TokenCount,
			Metadata:   fact.Metadata,
		}
	}

	_, err := dao.db.NewInsert().
		Model(&pgFacts).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create facts: %w", err)
	}

	facts = factsFromStoreSchema(pgFacts)

	return facts, nil
}

func (dao *FactDAO) Update(
	ctx context.Context,
	fact *models.Fact,
) (*models.Fact, error) {
	if fact.UUID == uuid.Nil {
		return nil, errors.New("fact UUID cannot be empty")
	}

	tx, err := dao.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer rollbackOnError(tx)

	metadata, err := mergeMetadata(
		ctx,
		tx,
		"uuid",
		fact.UUID.String(),
		"fact",
		fact.Metadata,
		true,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update fact metadata: %w", err)
	}

	pgFact := FactStoreSchema{
		UUID:       fact.UUID,
		Content:    fact.Content,
		TokenCount: fact.TokenCount,
		Metadata:   metadata,
	}

	columns := []string{"content", "metadata", "token_count"}
	_, err = tx.NewUpdate().
		Model(&pgFact).
		Column(columns...).
		Where("uuid = ?", fact.UUID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update fact: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return fact, nil

}

// Get by UUID
func (dao *FactDAO) Get(
	ctx context.Context,
	factUUID uuid.UUID,
) (*models.Fact, error) {
	var facts FactStoreSchema
	err := dao.db.NewSelect().
		Model(&facts).
		Where("session_id = ?", dao.sessionID).
		Where("uuid = ?", factUUID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.NewNotFoundError(fmt.Sprintf("fact with UUID %s not found", factUUID))
		}
		return nil, fmt.Errorf("failed to get fact: %w", err)
	}

	return &models.Fact{
		UUID:       facts.UUID,
		CreatedAt:  facts.CreatedAt,
		UpdatedAt:  facts.UpdatedAt,
		Content:    facts.Content,
		TokenCount: facts.TokenCount,
		Metadata:   facts.Metadata,
	}, nil
}

// Get all for this session
func (dao *FactDAO) GetAll(
	ctx context.Context,
) ([]models.Fact, error) {
	var facts []FactStoreSchema
	err := dao.db.NewSelect().
		Model(&facts).
		Where("session_id = ?", dao.sessionID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get facts: %w", err)
	}

	return factsFromStoreSchema(facts), nil
}

// Delete All
func (dao *FactDAO) DeleteAll(
	ctx context.Context,
) error {

	tx, err := dao.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	_, err = tx.NewDelete().
		Model((*FactStoreSchema)(nil)).
		Where("session_id = ?", dao.sessionID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete facts: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func factsFromStoreSchema(facts []FactStoreSchema) []models.Fact {
	factList := make([]models.Fact, len(facts))
	for i, fact := range facts {
		factList[i] = models.Fact{
			UUID:       fact.UUID,
			CreatedAt:  fact.CreatedAt,
			UpdatedAt:  fact.UpdatedAt,
			Content:    fact.Content,
			TokenCount: fact.TokenCount,
			Metadata:   fact.Metadata,
		}
	}
	return factList
}
