package mux

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mondaynightpoker-server/pkg/model"

	"github.com/stretchr/testify/assert"
)

func TestMux_getAdminTable(t *testing.T) {
	a := assert.New(t)

	p1, j1 := player()

	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	assertGet(t, ts, "/admin/table", nil, http.StatusForbidden, j1)

	p1.IsSiteAdmin = true
	a.NoError(testRepos.Players.Save(cbg, p1))

	var err errorResponse
	assertGet(t, ts, "/admin/table?rows=0", &err, http.StatusBadRequest, j1)
	a.Equal("rows must be greater than zero", err.Message)

	for i := 0; i < 5; i++ {
		tbl, err := testRepos.Tables.CreateTable(cbg, p1, fmt.Sprintf("TestMux_getAdminTable %d", i))
		a.NoError(err)

		if i == 4 {
			tbl.Deleted = true
			a.NoError(testRepos.Tables.Save(cbg, tbl))
		}
	}

	// other tests running concurrently against the same database may create tables
	// of their own, so a small page (rows=3) cannot be asserted positionally. Just
	// confirm pagination is honored by page size...
	var page []*model.TableWithPlayerEmail
	assertGet(t, ts, "/admin/table?rows=3", &page, http.StatusOK, j1)
	a.Equal(3, len(page))

	// ...then fetch a generously large page (the max allowed) and filter down to this
	// test's own tables (identified by the fresh, unique player email) to verify
	// pagination correctness, deleted-table inclusion, the email join, and ordering.
	var tables []*model.TableWithPlayerEmail
	assertGet(t, ts, "/admin/table?rows=100", &tables, http.StatusOK, j1)

	var mine []*model.TableWithPlayerEmail
	for _, tbl := range tables {
		if tbl.Email == p1.Email {
			mine = append(mine, tbl)
		}
	}

	if a.Equal(5, len(mine)) {
		for i, tbl := range mine {
			a.Equal(fmt.Sprintf("TestMux_getAdminTable %d", 4-i), tbl.Name)
			a.Equal(p1.Email, tbl.Email)

			if i == 0 {
				a.True(tbl.Deleted)
			} else {
				a.False(tbl.Deleted)
			}
		}
	}
}

func TestMux_adminPostTableUUID(t *testing.T) {
	a := assert.New(t)

	player, jwt := player()

	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	player.IsSiteAdmin = true
	a.NoError(testRepos.Players.Save(cbg, player))

	table, err := testRepos.Tables.CreateTable(cbg, player, "Test Table")
	a.NoError(err)
	a.False(table.Deleted)

	var resp model.Table
	assertPost(t, ts, fmt.Sprintf("/admin/table/%s", table.UUID), postAdminTableUUIDPayload{true}, &resp, http.StatusOK, jwt)
	a.True(resp.Deleted)

	table2, err := testRepos.Tables.GetTableByUUID(cbg, table.UUID)
	a.True(table2.Deleted)
	a.NoError(err)

	assertPost(t, ts, fmt.Sprintf("/admin/table/%s", table.UUID), postAdminTableUUIDPayload{false}, &resp, http.StatusOK, jwt)
	a.False(resp.Deleted)

	// the route must match uppercase hex UUIDs like the non-admin table routes do
	// (postgres' uuid type compares case-insensitively, so the request succeeds end-to-end)
	assertPost(t, ts, fmt.Sprintf("/admin/table/%s", strings.ToUpper(table.UUID)), postAdminTableUUIDPayload{true}, &resp, http.StatusOK, jwt)
	a.True(resp.Deleted)
}
