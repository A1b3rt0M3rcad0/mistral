package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/application"
	identity "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/modules/identity/domain"
	sharedpostgres "github.com/A1b3rt0M3rcad0/mistral/packages/mistral-core/shared/infra/postgres"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, errors.New("postgres database is required")
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Bind(ctx context.Context, ownership identity.Ownership) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	validated, err := identity.NewOwnership(ownership.SubjectID, ownership.CharacterID)
	if err != nil {
		return err
	}
	result, err := sharedpostgres.Runner(ctx, r.db).ExecContext(ctx, `INSERT INTO character_ownerships (character_id, subject_id) VALUES ($1, $2) ON CONFLICT (character_id) DO NOTHING`, validated.CharacterID, validated.SubjectID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return application.ErrCharacterAlreadyOwned
	}
	return nil
}

func (r *Repository) ByCharacter(ctx context.Context, characterID string) (identity.Ownership, error) {
	if err := ctx.Err(); err != nil {
		return identity.Ownership{}, err
	}
	if characterID == "" {
		return identity.Ownership{}, application.ErrOwnershipNotFound
	}
	var subjectID string
	if err := sharedpostgres.Runner(ctx, r.db).QueryRowContext(ctx, `SELECT subject_id FROM character_ownerships WHERE character_id = $1`, characterID).Scan(&subjectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return identity.Ownership{}, application.ErrOwnershipNotFound
		}
		return identity.Ownership{}, err
	}
	return identity.Ownership{SubjectID: subjectID, CharacterID: characterID}, nil
}

var _ application.Repository = (*Repository)(nil)
