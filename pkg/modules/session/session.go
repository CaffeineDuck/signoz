package session

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/SigNoz/signoz/pkg/identn"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type Module interface {
	// Gets the session context for the user. The context contains information on what the user has to do in order to create a session.
	GetSessionContext(ctx context.Context, email valuer.Email, siteURL *url.URL) (*authtypes.SessionContext, error)

	// Create a session for a user using password authn provider.
	CreatePasswordAuthNSession(ctx context.Context, authNProvider authtypes.AuthNProvider, email valuer.Email, password string, orgID valuer.UUID) (*authtypes.Token, error)

	// Create a session for a user using callback authn providers.
	CreateCallbackAuthNSession(ctx context.Context, authNProvider authtypes.AuthNProvider, values url.Values) (string, error)

	// Create a session for a user identified by the trusted-header IdentN. The
	// request must carry the headers the IdentN expects (e.g. X-Authentik-Email);
	// the IdentN is what authenticates — this method only translates the resolved
	// identity into a tokenizer-issued JWT.
	CreateTrustedHeaderAuthNSession(ctx context.Context, req *http.Request) (*authtypes.Token, error)

	// Rotate a token.
	RotateSession(ctx context.Context, accessToken string, refreshToken string) (*authtypes.Token, error)

	// Delete a session.
	DeleteSession(ctx context.Context, accessToken string) error

	// Get the rotation interval for the session.
	GetRotationInterval(ctx context.Context) time.Duration

	// SetIdentNResolver wires the IdentN resolver into the session module. It is
	// a bootstrap-time hook: the resolver depends on the user setter which is in
	// turn owned by the modules container, so the resolver cannot be passed at
	// construction time. Call this exactly once after both the modules and the
	// resolver have been built.
	SetIdentNResolver(resolver identn.IdentNResolver)
}

type Handler interface {
	// Get the session context for the user.
	GetSessionContext(http.ResponseWriter, *http.Request)

	// Create a session for a user using email and password.
	CreateSessionByEmailPassword(http.ResponseWriter, *http.Request)

	// Create a session for a user using google callback.
	CreateSessionByGoogleCallback(http.ResponseWriter, *http.Request)

	// Create a session for a user using saml callback.
	CreateSessionBySAMLCallback(http.ResponseWriter, *http.Request)

	// Create a session for a user using oidc callback.
	CreateSessionByOIDCCallback(http.ResponseWriter, *http.Request)

	// Create a session for a user identified by the trusted-header IdentN.
	CreateSessionByTrustedHeader(http.ResponseWriter, *http.Request)

	// Rotate a token.
	RotateSession(http.ResponseWriter, *http.Request)

	// Delete a session.
	DeleteSession(http.ResponseWriter, *http.Request)
}
