package persistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	db = InitTestDB("./../../../")
	ctx = context.Background()
	result := m.Run()
	db.Close()
	os.Exit(result)
}

var (
	db  *pgxpool.Pool
	ctx context.Context
)

type MockPayload struct {
	Foo string
	Bar int
}

type MockMeta struct {
	Baz string
	Qux bool
}

type MockSession struct {
	Session[MockPayload, MockMeta]
}

func TestPGSessionFunctions(t *testing.T) {
	sessionWrite := newMockSession()

	err := PGWriteSession(ctx, db, &sessionWrite.Session)
	assert.NoError(t, err)
	assert.Nil(t, sessionWrite.UpdatedAt)

	sessionWrite.Payload = MockPayload{Foo: "bar", Bar: 69}
	sessionWrite.Meta = MockMeta{Baz: "qux", Qux: false}

	err = PGWriteSession(ctx, db, &sessionWrite.Session)
	assert.NoError(t, err)

	var sessionRead MockSession
	err = PGReadSession(ctx, db, sessionWrite.ID, &sessionRead.Session)
	assert.NoError(t, err)

	TruncateSessionDates(&sessionWrite.Session)
	TruncateSessionDates(&sessionRead.Session)

	assert.Equal(t, sessionWrite, sessionRead)

	var sessionReadNotFound MockSession
	err = PGReadSession(ctx, db, uuid.New(), &sessionReadNotFound.Session)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	err = PGDeleteSession(ctx, db, sessionWrite.ID)
	assert.NoError(t, err)

	err = PGReadSession(ctx, db, sessionWrite.ID, &sessionRead.Session)
	assert.ErrorIs(t, err, pgx.ErrNoRows)

	err = PGDeleteSession(ctx, db, sessionWrite.ID)
	assert.NoError(t, err)

	err = PGDeleteSession(ctx, db, uuid.New())
	assert.NoError(t, err)

	expiredSessionWrite := newMockSession()
	expiredSessionWrite.ExpiresAt = time.Now().Add(-time.Hour)
	err = PGWriteSession(ctx, db, &expiredSessionWrite.Session)

	var sessionReadExpired MockSession
	err = PGReadSession(ctx, db, expiredSessionWrite.ID, &sessionReadExpired.Session)
	assert.NoError(t, err)
	assert.Equal(t, expiredSessionWrite.ID, sessionReadExpired.ID)

	sessionReadExpired = MockSession{}
	err = PGReadValidSession(ctx, db, expiredSessionWrite.ID, &sessionReadExpired.Session)
	assert.ErrorIs(t, err, ErrSessionExpired)

	err = PGReadSession(ctx, db, expiredSessionWrite.ID, &sessionReadExpired.Session)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func newMockSession() MockSession {
	return MockSession{
		Session: Session[MockPayload, MockMeta]{
			ID:        uuid.New(),
			Type:      "mock",
			Payload:   MockPayload{Foo: "foo", Bar: 42},
			Meta:      MockMeta{Baz: "baz", Qux: true},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}
}
