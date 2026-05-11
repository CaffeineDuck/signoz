package implsession

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/factory/factorytest"
	"github.com/SigNoz/signoz/pkg/identn"
	"github.com/SigNoz/signoz/pkg/tokenizer/tokenizertest"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// stubIdentN is a hand-rolled identn.IdentN double. The real provider lives in
// trustedheaderidentn — importing it here would create an unnecessary fixture
// dependency. The session module only cares about three things: the IdentN's
// name, what Test() reports, and what GetIdentity() returns.
type stubIdentN struct {
	name     authtypes.IdentNProvider
	matches  bool
	identity *authtypes.Identity
	err      error
}

func (s *stubIdentN) Test(*http.Request) bool { return s.matches }

func (s *stubIdentN) GetIdentity(*http.Request) (*authtypes.Identity, error) {
	return s.identity, s.err
}

func (s *stubIdentN) Name() authtypes.IdentNProvider { return s.name }

// stubResolver returns whatever IdentN it was constructed with — including nil
// to simulate "no IdentN matched."
type stubResolver struct {
	idn identn.IdentN
}

func (r *stubResolver) GetIdentN(*http.Request) identn.IdentN { return r.idn }

func newTestModule(t *testing.T) (*module, *tokenizertest.MockTokenizer) {
	t.Helper()
	tok := tokenizertest.NewMockTokenizer(t)
	m := &module{
		settings:  factory.NewScopedProviderSettings(factorytest.NewSettings(), "github.com/SigNoz/signoz/pkg/modules/session/implsession"),
		tokenizer: tok,
	}
	return m, tok
}

func TestCreateTrustedHeaderAuthNSessionResolverUnset(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/sessions/trustedheader", nil)
	token, err := m.CreateTrustedHeaderAuthNSession(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, token)
	assert.True(t, errors.Ast(err, errors.TypeInternal), "expected internal error when resolver is unset, got %v", err)
}

func TestCreateTrustedHeaderAuthNSessionNoIdentNMatched(t *testing.T) {
	m, _ := newTestModule(t)
	m.SetIdentNResolver(&stubResolver{idn: nil})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/sessions/trustedheader", nil)
	token, err := m.CreateTrustedHeaderAuthNSession(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, token)
	assert.True(t, errors.Ast(err, errors.TypeUnauthenticated))
}

// When some other IdentN matches (e.g. tokenizer because a bearer token was
// supplied), we must refuse rather than mint a fresh JWT — the session bridge
// is specifically for the trusted-header path.
func TestCreateTrustedHeaderAuthNSessionRejectsNonTrustedHeaderIdentN(t *testing.T) {
	m, _ := newTestModule(t)
	m.SetIdentNResolver(&stubResolver{
		idn: &stubIdentN{
			name:    authtypes.IdentNProviderTokenizer,
			matches: true,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/sessions/trustedheader", nil)
	token, err := m.CreateTrustedHeaderAuthNSession(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, token)
	assert.True(t, errors.Ast(err, errors.TypeUnauthenticated))
}

func TestCreateTrustedHeaderAuthNSessionPropagatesIdentityError(t *testing.T) {
	m, _ := newTestModule(t)
	want := errors.New(errors.TypeUnauthenticated, errors.MustNewCode("user_not_found"), "no user found")
	m.SetIdentNResolver(&stubResolver{
		idn: &stubIdentN{
			name:    authtypes.IdentNProviderTrustedHeader,
			matches: true,
			err:     want,
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/sessions/trustedheader", nil)
	token, err := m.CreateTrustedHeaderAuthNSession(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, token)
	assert.Same(t, want, err, "expected IdentN error to propagate untouched")
}

func TestCreateTrustedHeaderAuthNSessionSuccess(t *testing.T) {
	userID := valuer.GenerateUUID()
	orgID := valuer.GenerateUUID()
	email, err := valuer.NewEmail("alice@example.com")
	require.NoError(t, err)

	identity := authtypes.NewPrincipalUserIdentity(userID, orgID, email, authtypes.IdentNProviderTrustedHeader)
	want := &authtypes.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
		UserID:       userID,
	}

	m, tok := newTestModule(t)
	m.SetIdentNResolver(&stubResolver{
		idn: &stubIdentN{
			name:     authtypes.IdentNProviderTrustedHeader,
			matches:  true,
			identity: identity,
		},
	})

	// The token must be minted with IdentNProviderTokenizer — the JWT is a
	// regular session token regardless of how the user was identified.
	tok.EXPECT().
		CreateToken(
			mock.Anything,
			mock.MatchedBy(func(id *authtypes.Identity) bool {
				return id.UserID == userID &&
					id.OrgID == orgID &&
					id.Email == email &&
					id.IdenNProvider == authtypes.IdentNProviderTokenizer &&
					id.Principal == authtypes.PrincipalUser
			}),
			mock.Anything,
		).
		Return(want, nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/api/v2/sessions/trustedheader", nil)
	got, err := m.CreateTrustedHeaderAuthNSession(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, want.AccessToken, got.AccessToken)
	assert.Equal(t, want.RefreshToken, got.RefreshToken)
	assert.Equal(t, want.UserID, got.UserID)
}
