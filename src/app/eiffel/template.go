package eiffel

import (
	"context"
	"encoding/json"
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

// ErrTemplateJsonMissingInfo is returned if the template's JSON does not contain the necessary information (name and version).
var ErrTemplateJsonMissingInfo = errors.New("template json missing necessary information (check name and version)")

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
	TemplateSet uuid.UUID `hvalidate:"required"`
	Type        string    `hvalidate:"required"`
	Json        string    `hvalidate:"required"`
	CreatedBy   uuid.UUID `hvalidate:"required"`
}

// TemplateToUpdate is the template entity that is used to update an existing template.
type TemplateToUpdate struct {
	ID          uuid.UUID `hvalidate:"required"`
	TemplateSet uuid.UUID `hvalidate:"required"`
	Type        string    `hvalidate:"required"`
	Json        string    `hvalidate:"required"`
}

// TemplateNecessaryInfo is the necessary information about a template. It is used to create a new template.
// The template's JSON has to contain this information. It is extracted from the JSON and saved in the database.
type TemplateNecessaryInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
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
	Name        string    `hvalidate:"required"`
	Version     string    `hvalidate:"required"`
	Description string    `hvalidate:"required"`
	CreatedBy   uuid.UUID `hvalidate:"required"`
}

// TemplateSetToUpdate is the template set entity that is used to update an existing template set.
type TemplateSetToUpdate struct {
	ID          uuid.UUID `hvalidate:"required"`
	Name        string    `hvalidate:"required"`
	Version     string    `hvalidate:"required"`
	Description string    `hvalidate:"required"`
}

// PGTemplateRepository is the template repository for PostgreSQL. It holds a reference to the database connection pool.
type PGTemplateRepository struct {
	db *pgxpool.Pool
}

// PGTemplateSetRepository is the template set repository for PostgreSQL. It holds a reference to the database connection pool.
type PGTemplateSetRepository struct {
	db *pgxpool.Pool
}

// TemplateRepository is the template repository it contains the necessary methods to interact with the database.
// TemplateRepository is safe for concurrent use by multiple goroutines.
type TemplateRepository interface {
	persistence.Repository

	// FindByID finds a template by its id. It returns persistence.ErrNotFound if the template could not be found and persistence.ErrReadRow for any other error.
	FindByID(ctx context.Context, id uuid.UUID) (*Template, error)
	// FindByTemplateSetID finds all templates by their template set id. It returns persistence.ErrNotFound if no templates could be found and persistence.ErrReadRow for any other error.
	FindByTemplateSetID(ctx context.Context, templateSetID uuid.UUID) ([]*Template, error)
	// Create creates a new template and returns it. It returns persistence.ErrInsert if the template could not be inserted.
	// It also extracts the necessary information from the template's JSON and saves it in the database.
	// If the JSON does not contain the necessary information, it returns ErrTemplateJsonMissingInfo.
	Create(ctx context.Context, template *TemplateToCreate) (*Template, error)
	// Update updates an existing template and returns it. It returns persistence.ErrUpdate if the template could not be updated.
	// It also extracts the necessary information from the template's JSON and saves it in the database.
	// If the JSON does not contain the necessary information, it returns ErrTemplateJsonMissingInfo.
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

// ToUpdate returns a TemplateToUpdate from a Template.
func (t *Template) ToUpdate() *TemplateToUpdate {
	return &TemplateToUpdate{
		ID:          t.ID,
		TemplateSet: t.TemplateSet,
		Type:        t.Type,
		Json:        t.Json,
	}
}

// NecessaryInfo returns the valid necessary information about a template from a Template.
// It will return ErrTemplateJsonMissingInfo if the template's JSON does not contain the necessary information (name and version).
// This method is used by Created and Update to extract the necessary information from the template's JSON.
func (t *Template) NecessaryInfo() (*TemplateNecessaryInfo, error) {
	info := &TemplateNecessaryInfo{}
	err := json.Unmarshal([]byte(t.Json), info)

	if info.Name == "" || info.Version == "" {
		return nil, ErrTemplateJsonMissingInfo
	}

	return info, err
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
// It also checks if the template's JSON contains the necessary information (name and version).
// If the JSON does not contain the necessary information, it returns ErrTemplateJsonMissingInfo.
func (r *PGTemplateRepository) Create(ctx context.Context, toCreate *TemplateToCreate) (*Template, error) {
	newTemplate := &Template{
		ID:          uuid.New(),
		TemplateSet: toCreate.TemplateSet,
		Type:        toCreate.Type,
		Json:        toCreate.Json,
		CreatedBy:   toCreate.CreatedBy,
		CreatedAt:   time.Now(),
	}

	tmplInfo, err := newTemplate.NecessaryInfo()
	if err != nil {
		return nil, err
	}

	newTemplate.Name = tmplInfo.Name
	newTemplate.Version = tmplInfo.Version

	_, err = r.db.Exec(
		ctx,
		"INSERT INTO templates (id, template_set, name, version, type, json, created_by, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		newTemplate.ID, newTemplate.TemplateSet, newTemplate.Name, newTemplate.Version, newTemplate.Type, newTemplate.Json, newTemplate.CreatedBy, newTemplate.CreatedAt,
	)
	if err != nil {
		return nil, errors.Join(persistence.ErrInsert, err)
	}

	return newTemplate, nil
}

// Update updates an existing template and returns it. It returns persistence.ErrUpdate if the template could not be updated.
// It also checks if the template's JSON contains the necessary information (name and version).
// If the JSON does not contain the necessary information, it returns ErrTemplateJsonMissingInfo.
func (r *PGTemplateRepository) Update(ctx context.Context, toUpdate *TemplateToUpdate) (*Template, error) {
	template := &Template{
		ID:   toUpdate.ID,
		Json: toUpdate.Json,
	}

	tmplInfo, err := template.NecessaryInfo()
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(
		ctx,
		`UPDATE templates
	 	SET template_set = $1, type = $2, name = $3, version = $4, json = $5, updated_at = NOW()
	 	WHERE id = $6
	 	RETURNING template_set, type, name, version, json, created_by, created_at, updated_at`,
		toUpdate.TemplateSet, toUpdate.Type, tmplInfo.Name, tmplInfo.Version, toUpdate.Json, toUpdate.ID,
	).Scan(
		&template.TemplateSet,
		&template.Type,
		&template.Name,
		&template.Version,
		&template.Json,
		&template.CreatedBy,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err != nil {
		return nil, errors.Join(persistence.ErrUpdate, err)
	}

	return template, nil
}

// Delete deletes an existing template by its id. It returns persistence.ErrDelete if the template could not be deleted.
func (r *PGTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM templates WHERE id = $1", id)
	if err != nil {
		return errors.Join(persistence.ErrDelete, err)
	}

	return nil
}

// FindByID finds a template set by its id. It returns persistence.ErrNotFound if the template set could not be found and persistence.ErrReadRow for any other error.
func (r *PGTemplateSetRepository) FindByID(ctx context.Context, id uuid.UUID) (*TemplateSet, error) {
	t := &TemplateSet{}
	err := r.db.QueryRow(ctx, "SELECT id, name, version, description, created_by, created_at, updated_at FROM template_sets WHERE id = $1", id).
		Scan(&t.ID, &t.Name, &t.Version, &t.Description, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	return t, nil
}

// Create creates a new template set and returns it. It returns persistence.ErrInsert if the template set could not be inserted.
func (r *PGTemplateSetRepository) Create(ctx context.Context, toCreate *TemplateSetToCreate) (*TemplateSet, error) {
	newTemplateSet := &TemplateSet{
		ID:          uuid.New(),
		Name:        toCreate.Name,
		Version:     toCreate.Version,
		Description: toCreate.Description,
		CreatedBy:   toCreate.CreatedBy,
		CreatedAt:   time.Now(),
	}

	_, err := r.db.Exec(
		ctx,
		"INSERT INTO template_sets (id, name, version, description, created_by, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		newTemplateSet.ID,
		newTemplateSet.Name,
		newTemplateSet.Version,
		newTemplateSet.Description,
		newTemplateSet.CreatedBy,
		newTemplateSet.CreatedAt,
	)
	if err != nil {
		return nil, errors.Join(persistence.ErrInsert, err)
	}

	return newTemplateSet, nil
}

// Update updates an existing template set and returns it. It returns persistence.ErrUpdate if the template set could not be updated.
func (r *PGTemplateSetRepository) Update(ctx context.Context, toUpdate *TemplateSetToUpdate) (*TemplateSet, error) {
	templateSet := &TemplateSet{
		ID: toUpdate.ID,
	}

	err := r.db.QueryRow(
		ctx,
		`UPDATE template_sets
	 	SET name = $1, version = $2, description = $3, updated_at = NOW()
	 	WHERE id = $4
	 	RETURNING name, version, description, created_by, created_at, updated_at`,
		toUpdate.Name, toUpdate.Version, toUpdate.Description, toUpdate.ID,
	).Scan(
		&templateSet.Name,
		&templateSet.Version,
		&templateSet.Description,
		&templateSet.CreatedBy,
		&templateSet.CreatedAt,
		&templateSet.UpdatedAt,
	)

	if err != nil {
		return nil, errors.Join(persistence.ErrUpdate, err)
	}

	return templateSet, nil
}

// Delete deletes an existing template set by its id. It returns persistence.ErrDelete if the template set could not be deleted.
func (r *PGTemplateSetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM template_sets WHERE id = $1", id)
	if err != nil {
		return errors.Join(persistence.ErrDelete, err)
	}

	return nil
}
