package template

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/src/core/persistence"
	"strings"
	"time"
)

const (
	// RepositoryName is the name of the template repository. It can be used to retrieve the repository from the persistence.RepositoryProvider.
	RepositoryName = "Repository"
	// SetRepositoryName is the name of the template set repository. It can be used to retrieve the repository from the persistence.RepositoryProvider.
	SetRepositoryName = "SetRepository"
	// Pkg is the package name for logging.
	Pkg = "template"
)

var (
	// ErrInvalidTemplate is returned when a template is invalid. More errors are expected to further describe the problem.
	ErrInvalidTemplate = errors.New("eiffel.parser.error.invalid-template")
	// ErrTemplateConfigMissingInfo is returned if the template's config JSON does not contain the necessary information (name, version and type).
	ErrTemplateConfigMissingInfo = errors.New("template's config json missing necessary information (check name, version and type)")
)

// Template is the template entity that is saved in the database. It contains the template's metadata.
// Each template belongs to a template set. Templates are versioned and the information about the template should always match the template's config JSON.
// Actually, Type, Name and Version are redundant, but they are used for easier querying.
type Template struct {
	ID          uuid.UUID
	TemplateSet uuid.UUID
	Type        string
	Name        string
	Version     string
	Config      string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

// ToCreate is the template entity that is used to create a new template.
// TODO Evaluate if ToCreate and ToUpdate should be merged into one struct. It is convenient to have them separated, but also complicates the code.
type ToCreate struct {
	TemplateSet uuid.UUID `hvalidate:"required"`
	Type        string    `hvalidate:"required"`
	Config      string    `hvalidate:"required"`
	CreatedBy   uuid.UUID `hvalidate:"required"`
}

// ToUpdate is the template entity that is used to update an existing template.
type ToUpdate struct {
	ID          uuid.UUID `hvalidate:"required"`
	TemplateSet uuid.UUID `hvalidate:"required"`
	Type        string    `hvalidate:"required"`
	Config      string    `hvalidate:"required"`
}

// NecessaryInfo is the necessary information about a template. It is used to create a new template.
// The template's config JSON has to contain this information. It is extracted from the config JSON and saved in the database.
type NecessaryInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Set is the template set entity. Each template belongs to a template set. Each template set can have multiple templates.
// It also contains the necessary information about the template.
type Set struct {
	ID          uuid.UUID
	Name        string
	Version     string
	Description string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

// SetToCreate is the template set entity that is used to create a new template set.
type SetToCreate struct {
	Name        string    `hvalidate:"required"`
	Version     string    `hvalidate:"required,semVer"`
	CreatedBy   uuid.UUID `hvalidate:"required"`
	Description string
}

// SetToUpdate is the template set entity that is used to update an existing template set.
type SetToUpdate struct {
	ID          uuid.UUID `hvalidate:"required"`
	Name        string    `hvalidate:"required"`
	Version     string    `hvalidate:"required,semVer"`
	Description string
}

// PGRepository is the template repository for PostgreSQL. It holds a reference to the database connection pool.
type PGRepository struct {
	db *pgxpool.Pool
}

// PGSetRepository is the template set repository for PostgreSQL. It holds a reference to the database connection pool.
type PGSetRepository struct {
	db *pgxpool.Pool
}

// Repository is the template repository it contains the necessary methods to interact with the database.
// Repository is safe for concurrent use by multiple goroutines.
type Repository interface {
	persistence.Repository

	// FindByID finds a template by its id. It returns persistence.ErrNotFound if the template could not be found and persistence.ErrReadRow for any other error.
	FindByID(ctx context.Context, id uuid.UUID) (*Template, error)
	// FindByTemplateSetID finds all templates by their template set id. It returns persistence.ErrNotFound if no templates could be found and persistence.ErrReadRow for any other error.
	FindByTemplateSetID(ctx context.Context, templateSetID uuid.UUID) ([]*Template, error)
	// Create creates a new template and returns it. It returns persistence.ErrInsert if the template could not be inserted.
	// It also extracts the necessary information from the template's config JSON and saves it in the database.
	// If the config JSON does not contain the necessary information, it returns ErrTemplateConfigMissingInfo.
	Create(ctx context.Context, template *ToCreate) (*Template, error)
	// Update updates an existing template and returns it. It returns persistence.ErrUpdate if the template could not be updated.
	// It also extracts the necessary information from the template's config JSON and saves it in the database.
	// If the config JSON does not contain the necessary information, it returns ErrTemplateConfigMissingInfo.
	Update(ctx context.Context, template *ToUpdate) (*Template, error)
	// Delete deletes an existing template by its id. It returns persistence.ErrDelete if the template could not be deleted.
	Delete(ctx context.Context, id uuid.UUID) error
}

// SetRepository is the template set repository it contains the necessary methods to interact with the database.
// SetRepository is safe for concurrent use by multiple goroutines.
// TODO move SetRepository and Repository together to handle template concerns all in one repo.
type SetRepository interface {
	persistence.Repository

	// FindByID finds a template set by its id. It returns persistence.ErrNotFound if the template set could not be found and persistence.ErrReadRow for any other error.
	FindByID(ctx context.Context, id uuid.UUID) (*Set, error)
	// FindByCreatedBy finds all template sets for a user. It returns persistence.ErrNotFound if no template sets could be found and persistence.ErrReadRow for any other error.
	FindByCreatedBy(ctx context.Context, userID uuid.UUID) ([]*Set, error)
	// Create creates a new template set and returns it. It returns persistence.ErrInsert if the template set could not be inserted.
	Create(ctx context.Context, templateSet *SetToCreate) (*Set, error)
	// Update updates an existing template set and returns it. It returns persistence.ErrUpdate if the template set could not be updated.
	Update(ctx context.Context, templateSet *SetToUpdate) (*Set, error)
	// Delete deletes an existing template set by its id. It returns persistence.ErrDelete if the template set could not be deleted.
	Delete(ctx context.Context, id uuid.UUID) error
}

// ToUpdate returns a ToUpdate from a Template.
func (t *Template) ToUpdate() *ToUpdate {
	return &ToUpdate{
		ID:          t.ID,
		TemplateSet: t.TemplateSet,
		Type:        t.Type,
		Config:      t.Config,
	}
}

// ToCreateFromConfig returns a ToCreate after extracting the information from the config JSON supplied.
// The type will be converted to lowercase. It will return ErrTemplateConfigMissingInfo if the config JSON does not contain a type field.
func ToCreateFromConfig(config string) (*ToCreate, error) {
	t := struct {
		Type string `json:"type"`
	}{}
	err := json.Unmarshal([]byte(config), &t)

	if t.Type == "" || err != nil {
		return nil, ErrTemplateConfigMissingInfo
	}

	return &ToCreate{
		Type:   strings.ToLower(t.Type),
		Config: config,
	}, nil
}

// NecessaryInfo returns the valid necessary information about a template from a Template.
// It will return ErrTemplateConfigMissingInfo if the template's config JSON does not contain the necessary information (name and version).
// This method is used by Created and Update to extract the necessary information from the template's config JSON.
func (t *Template) NecessaryInfo() (*NecessaryInfo, error) {
	info := &NecessaryInfo{}
	err := json.Unmarshal([]byte(t.Config), info)

	if info.Name == "" || info.Version == "" {
		return nil, ErrTemplateConfigMissingInfo
	}

	return info, err
}

// ToUpdate returns a SetToUpdate from a Set.
func (t *Set) ToUpdate() *SetToUpdate {
	return &SetToUpdate{
		ID:          t.ID,
		Name:        t.Name,
		Version:     t.Version,
		Description: t.Description,
	}
}

// NewRepository constructs a new PGRepository with the passed in database connection pool.
func NewRepository(db *pgxpool.Pool) Repository {
	return &PGRepository{db: db}
}

// NewSetRepository constructs a new PGSetRepository with the passed in database connection pool.
func NewSetRepository(db *pgxpool.Pool) SetRepository {
	return &PGSetRepository{db: db}
}

// RepositoryName returns the name of the repository. This name is used to identify the repository in the persistence.RepositoryProvider.
func (r *PGRepository) RepositoryName() string {
	return RepositoryName
}

// RepositoryName returns the name of the repository. This name is used to identify the repository in the persistence.RepositoryProvider.
func (r *PGSetRepository) RepositoryName() string {
	return SetRepositoryName
}

// FindByID finds a template by its id. It returns persistence.ErrNotFound if the template could not be found and persistence.ErrReadRow for any other error.
func (r *PGRepository) FindByID(ctx context.Context, id uuid.UUID) (*Template, error) {
	t := &Template{}
	err := r.db.QueryRow(ctx, "SELECT id, template_set, type, name, version, config, created_by, created_at, updated_at FROM templates WHERE id = $1", id).
		Scan(&t.ID, &t.TemplateSet, &t.Type, &t.Name, &t.Version, &t.Config, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	return t, nil
}

// FindByTemplateSetID finds all templates by their template set id. It returns persistence.ErrNotFound if no templates could be found and persistence.ErrReadRow for any other error.
func (r *PGRepository) FindByTemplateSetID(ctx context.Context, templateSetID uuid.UUID) ([]*Template, error) {
	rows, err := r.db.Query(ctx, "SELECT id, template_set, type, name, version, config, created_by, created_at, updated_at FROM templates WHERE template_set = $1", templateSetID)
	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	var templates []*Template
	for rows.Next() {
		t := &Template{}
		err := rows.Scan(&t.ID, &t.TemplateSet, &t.Type, &t.Name, &t.Version, &t.Config, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, persistence.PGReadErr(err)
		}

		templates = append(templates, t)
	}

	return templates, nil
}

// Create creates a new template and returns it. It returns persistence.ErrInsert if the template could not be inserted.
// It also checks if the template's config JSON contains the necessary information (name and version).
// If the config JSON does not contain the necessary information, it returns ErrTemplateConfigMissingInfo.
func (r *PGRepository) Create(ctx context.Context, toCreate *ToCreate) (*Template, error) {
	newTemplate := &Template{
		ID:          uuid.New(),
		TemplateSet: toCreate.TemplateSet,
		Type:        toCreate.Type,
		Config:      toCreate.Config,
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
		"INSERT INTO templates (id, template_set, name, version, type, config, created_by, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		newTemplate.ID, newTemplate.TemplateSet, newTemplate.Name, newTemplate.Version, newTemplate.Type, newTemplate.Config, newTemplate.CreatedBy, newTemplate.CreatedAt,
	)
	if err != nil {
		return nil, errors.Join(persistence.ErrInsert, err)
	}

	return newTemplate, nil
}

// Update updates an existing template and returns it. It returns persistence.ErrUpdate if the template could not be updated.
// It also checks if the template's config JSON contains the necessary information (name and version).
// If the config JSON does not contain the necessary information, it returns ErrTemplateConfigMissingInfo.
func (r *PGRepository) Update(ctx context.Context, toUpdate *ToUpdate) (*Template, error) {
	template := &Template{
		ID:     toUpdate.ID,
		Config: toUpdate.Config,
	}

	tmplInfo, err := template.NecessaryInfo()
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(
		ctx,
		`UPDATE templates
	 	SET template_set = $1, type = $2, name = $3, version = $4, config = $5, updated_at = NOW()
	 	WHERE id = $6
	 	RETURNING template_set, type, name, version, config, created_by, created_at, updated_at`,
		toUpdate.TemplateSet, toUpdate.Type, tmplInfo.Name, tmplInfo.Version, toUpdate.Config, toUpdate.ID,
	).Scan(
		&template.TemplateSet,
		&template.Type,
		&template.Name,
		&template.Version,
		&template.Config,
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
func (r *PGRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM templates WHERE id = $1", id)
	if err != nil {
		return errors.Join(persistence.ErrDelete, err)
	}

	return nil
}

// FindByID finds a template set by its id. It returns persistence.ErrNotFound if the template set could not be found and persistence.ErrReadRow for any other error.
func (r *PGSetRepository) FindByID(ctx context.Context, id uuid.UUID) (*Set, error) {
	t := &Set{}
	err := r.db.QueryRow(ctx, "SELECT id, name, version, description, created_by, created_at, updated_at FROM template_sets WHERE id = $1", id).
		Scan(&t.ID, &t.Name, &t.Version, &t.Description, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	return t, nil
}

// FindByCreatedBy finds all template sets for a user. It returns persistence.ErrNotFound if no template sets could be found and persistence.ErrReadRow for any other error.
func (r *PGSetRepository) FindByCreatedBy(ctx context.Context, userID uuid.UUID) ([]*Set, error) {
	rows, err := r.db.Query(ctx, "SELECT id, name, version, description, created_by, created_at, updated_at FROM template_sets WHERE created_by = $1", userID)
	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	var templates []*Set
	for rows.Next() {
		t := &Set{}
		err := rows.Scan(&t.ID, &t.Name, &t.Version, &t.Description, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, persistence.PGReadErr(err)
		}

		templates = append(templates, t)
	}

	return templates, nil
}

// Create creates a new template set and returns it. It returns persistence.ErrInsert if the template set could not be inserted.
func (r *PGSetRepository) Create(ctx context.Context, toCreate *SetToCreate) (*Set, error) {
	newTemplateSet := &Set{
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
func (r *PGSetRepository) Update(ctx context.Context, toUpdate *SetToUpdate) (*Set, error) {
	templateSet := &Set{
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
func (r *PGSetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM template_sets WHERE id = $1", id)
	if err != nil {
		return errors.Join(persistence.ErrDelete, err)
	}

	return nil
}
