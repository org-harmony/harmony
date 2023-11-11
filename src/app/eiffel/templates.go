package eiffel

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/src/core/persistence"
	"time"
)

const (
	// TemplateRepositoryName is the name of the template repository. It can be used to retrieve the repository from the persistence.RepositoryProvider.
	TemplateRepositoryName = "TemplateRepository"
	// TemplateSetRepositoryName is the name of the template set repository. It can be used to retrieve the repository from the persistence.RepositoryProvider.
	TemplateSetRepositoryName = "TemplateSetRepository"
)

type TemplateRepository interface {
	persistence.Repository

	// FindByID finds a template by its id. It returns persistence.ErrNotFound if the template could not be found and persistence.ErrReadRow for any other error.
	FindByID(ctx context.Context, id uuid.UUID) (*Template, error)
	// FindByTemplateSetID finds all templates by their template set id. It returns persistence.ErrNotFound if no templates could be found and persistence.ErrReadRow for any other error.
	FindByTemplateSetID(ctx context.Context, templateSetID uuid.UUID) ([]*Template, error)
	// Create creates a new template and returns it. It returns persistence.ErrInsert if the template could not be inserted.
	Create(ctx context.Context, template *TemplateToCreate) (*Template, error)
	// Update updates an existing template and returns it. It returns persistence.ErrUpdate if the template could not be updated.
	Update(ctx context.Context, template *TemplateToUpdate) (*Template, error)
	// Delete deletes an existing template by its id. It returns persistence.ErrDelete if the template could not be deleted.
	Delete(ctx context.Context, id uuid.UUID) error
}

// TemplateSetRepository is the template set repository it contains the necessary methods to interact with the database.
// TemplateSetRepository is safe for concurrent use by multiple goroutines.
type TemplateSetRepository interface {
	persistence.Repository

	// FindByID finds a template set by its id. It returns persistence.ErrNotFound if the template set could not be found and persistence.ErrReadRow for any other error.
	FindByID(ctx context.Context, id uuid.UUID) (*TemplateSet, error)
	// Create creates a new template set and returns it. It returns persistence.ErrInsert if the template set could not be inserted.
	Create(ctx context.Context, templateSet *TemplateSetToCreate) (*TemplateSet, error)
	// Update updates an existing template set and returns it. It returns persistence.ErrUpdate if the template set could not be updated.
	Update(ctx context.Context, templateSet *TemplateSetToUpdate) (*TemplateSet, error)
	// Delete deletes an existing template set by its id. It returns persistence.ErrDelete if the template set could not be deleted.
	Delete(ctx context.Context, id uuid.UUID) error
}

// Template is the template entity that is saved in the database. It contains the template's metadata.
// Each template belongs to a template set. Templates are versioned and the information about the template should always match the template's JSON.
// Actually, Type, Name and Version are redundant, but they are used for easier querying.
type Template struct {
	ID          uuid.UUID
	TemplateSet uuid.UUID
	Type        string
	Name        string
	Version     string
	Json        string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

// TemplateToCreate is the template entity that is used to create a new template.
type TemplateToCreate struct {
	TemplateSet uuid.UUID
	Type        string
	Json        string
	CreatedBy   uuid.UUID
}

// TemplateToUpdate is the template entity that is used to update an existing template.
type TemplateToUpdate struct {
	ID          uuid.UUID
	TemplateSet uuid.UUID
	Type        string
	Json        string
}

// TemplateSet is the template set entity. Each template belongs to a template set. Each template set can have multiple templates.
// It also contains the necessary information about the template.
type TemplateSet struct {
	ID          uuid.UUID
	Name        string
	Version     string
	Description string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

// TemplateSetToCreate is the template set entity that is used to create a new template set.
type TemplateSetToCreate struct {
	Name        string
	Version     string
	Description string
	CreatedBy   uuid.UUID
}

// TemplateSetToUpdate is the template set entity that is used to update an existing template set.
type TemplateSetToUpdate struct {
	ID          uuid.UUID
	Name        string
	Version     string
	Description string
}

// PGTemplateRepository is the template repository for PostgreSQL. It holds a reference to the database connection pool.
type PGTemplateRepository struct {
	db *pgxpool.Pool
}

// PGTemplateSetRepository is the template set repository for PostgreSQL. It holds a reference to the database connection pool.
type PGTemplateSetRepository struct {
	db *pgxpool.Pool
}

// ToUpdate returns a TemplateToUpdate from a Template.
func (t *Template) ToUpdate() *TemplateToUpdate {
	return &TemplateToUpdate{
		ID:          t.ID,
		TemplateSet: t.TemplateSet,
		Type:        t.Type,
		Json:        t.Json,
	}
}

// ToUpdate returns a TemplateSetToUpdate from a TemplateSet.
func (t *TemplateSet) ToUpdate() *TemplateSetToUpdate {
	return &TemplateSetToUpdate{
		ID:          t.ID,
		Name:        t.Name,
		Version:     t.Version,
		Description: t.Description,
	}
}

// NewTemplateRepository constructs a new PGTemplateRepository with the passed in database connection pool.
func NewTemplateRepository(db *pgxpool.Pool) TemplateRepository {
	return &PGTemplateRepository{db: db}
}

// NewTemplateSetRepository constructs a new PGTemplateSetRepository with the passed in database connection pool.
func NewTemplateSetRepository(db *pgxpool.Pool) TemplateSetRepository {
	return &PGTemplateSetRepository{db: db}
}

// RepositoryName returns the name of the repository. This name is used to identify the repository in the persistence.RepositoryProvider.
func (r *PGTemplateRepository) RepositoryName() string {
	return TemplateRepositoryName
}

// RepositoryName returns the name of the repository. This name is used to identify the repository in the persistence.RepositoryProvider.
func (r *PGTemplateSetRepository) RepositoryName() string {
	return TemplateSetRepositoryName
}

// FindByID finds a template by its id. It returns persistence.ErrNotFound if the template could not be found and persistence.ErrReadRow for any other error.
func (r *PGTemplateRepository) FindByID(ctx context.Context, id uuid.UUID) (*Template, error) {
	t := &Template{}
	err := r.db.QueryRow(ctx, "SELECT id, template_set, type, name, version, json, created_by, created_at, updated_at FROM templates WHERE id = $1", id).
		Scan(&t.ID, &t.TemplateSet, &t.Type, &t.Name, &t.Version, &t.Json, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	return t, nil
}

// FindByTemplateSetID finds all templates by their template set id. It returns persistence.ErrNotFound if no templates could be found and persistence.ErrReadRow for any other error.
func (r *PGTemplateRepository) FindByTemplateSetID(ctx context.Context, templateSetID uuid.UUID) ([]*Template, error) {
	rows, err := r.db.Query(ctx, "SELECT id, template_set, type, name, version, json, created_by, created_at, updated_at FROM templates WHERE template_set = $1", templateSetID)
	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	var templates []*Template
	for rows.Next() {
		t := &Template{}
		err := rows.Scan(&t.ID, &t.TemplateSet, &t.Type, &t.Name, &t.Version, &t.Json, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, persistence.PGReadErr(err)
		}

		templates = append(templates, t)
	}

	return templates, nil
}

// Create creates a new template and returns it. It returns persistence.ErrInsert if the template could not be inserted.
func (r *PGTemplateRepository) Create(ctx context.Context, template *TemplateToCreate) (*Template, error) {
	newTemplate := &Template{
		ID:          uuid.New(),
		TemplateSet: template.TemplateSet,
		Type:        template.Type,
		Json:        template.Json,
		CreatedBy:   template.CreatedBy,
	}

	_, err := r.db.Exec(
		ctx,
		"INSERT INTO templates (id, template_set, type, json, created_by) VALUES ($1, $2, $3, $4, $5)",
		newTemplate.ID, newTemplate.TemplateSet, newTemplate.Type, newTemplate.Json, newTemplate.CreatedBy,
	)
	if err != nil {
		return nil, errors.Join(persistence.ErrInsert, err)
	}

	return newTemplate, nil
}

// Update updates an existing template and returns it. It returns persistence.ErrUpdate if the template could not be updated.
func (r *PGTemplateRepository) Update(ctx context.Context, template *TemplateToUpdate) (*Template, error) {
	updateTemplate := &Template{
		ID: template.ID,
	}

	err := r.db.QueryRow(
		ctx,
		`UPDATE templates
	 	SET template_set = $1, type = $2, json = $3, updated_at = NOW()
	 	WHERE id = $4
	 	RETURNING template_set, type, name, version, json, created_by, created_at, updated_at`,
		template.TemplateSet, template.Type, template.Json, template.ID,
	).Scan(
		&updateTemplate.TemplateSet,
		&updateTemplate.Type,
		&updateTemplate.Name,
		&updateTemplate.Version,
		&updateTemplate.Json,
		&updateTemplate.CreatedBy,
		&updateTemplate.CreatedAt,
		&updateTemplate.UpdatedAt,
	)

	if err != nil {
		return nil, errors.Join(persistence.ErrUpdate, err)
	}

	return updateTemplate, nil
}

// Delete deletes an existing template by its id. It returns persistence.ErrDelete if the template could not be deleted.
func (r *PGTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM templates WHERE id = $1", id)
	if err != nil {
		return errors.Join(persistence.ErrDelete, err)
	}

	return nil
}
