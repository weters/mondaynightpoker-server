package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"os"
	"testing"

	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/db"
	"mondaynightpoker-server/pkg/model"
)

var cbg = context.Background()

var (
	testDB    *sql.DB
	testRepos *model.Repositories
	testPriv  *rsa.PrivateKey
	testPub   *rsa.PublicKey
)

const (
	testIssuer   = "https://oauth.test"
	testResource = "https://oauth.test/mcp"
	testPassword = "sup3r-s3cret-pw"
)

func TestMain(m *testing.M) {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	testDB, err = db.Connect(cfg.Database.DSN)
	if err != nil {
		panic(err)
	}

	testRepos = model.NewRepositories(testDB)

	testPriv, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	testPub = &testPriv.PublicKey

	os.Exit(m.Run())
}

// newTestServer returns a Server backed by the test repositories and RSA keys, applying
// any config overrides.
func newTestServer(t *testing.T, override func(*Config)) *Server {
	t.Helper()
	cfg := Config{Issuer: testIssuer, Resource: testResource}
	if override != nil {
		override(&cfg)
	}
	return New(testRepos, testPriv, testPub, cfg)
}

// seedAdmin creates a verified site-admin player with testPassword and returns it.
func seedAdmin(t *testing.T) *model.Player {
	t.Helper()
	return seedPlayer(t, true)
}

// seedPlayer creates a verified player (optionally a site admin) with testPassword.
func seedPlayer(t *testing.T, admin bool) *model.Player {
	t.Helper()
	email := util.RandomEmail()
	player, err := testRepos.Players.CreatePlayer(cbg, email, "Test Player", testPassword, "127.0.0.1")
	if err != nil {
		t.Fatalf("could not create player: %v", err)
	}

	player.Status = model.PlayerStatusVerified
	if err := testRepos.Players.Save(cbg, player); err != nil {
		t.Fatalf("could not verify player: %v", err)
	}

	if admin {
		if err := testRepos.Players.SetIsSiteAdmin(cbg, player, true); err != nil {
			t.Fatalf("could not set site admin: %v", err)
		}
	}

	return player
}

// seedClient creates a public OAuth client with the given redirect URIs.
func seedClient(t *testing.T, redirectURIs ...string) *model.OAuthClient {
	t.Helper()
	clientID, err := randomToken(16)
	if err != nil {
		t.Fatalf("could not generate client id: %v", err)
	}

	client := &model.OAuthClient{
		ClientID:                clientID,
		ClientName:              "Test Client",
		RedirectURIs:            redirectURIs,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethod: "none",
	}
	if err := testRepos.OAuth.CreateClient(cbg, client); err != nil {
		t.Fatalf("could not create client: %v", err)
	}

	return client
}
