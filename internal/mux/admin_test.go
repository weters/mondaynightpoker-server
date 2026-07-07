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
		tbl, err := testRepos.Tables.CreateTable(cbg, p1, fmt.Sprintf("Table %d", i))
		a.NoError(err)

		if i == 4 {
			tbl.Deleted = true
			a.NoError(testRepos.Tables.Save(cbg, tbl))
		}
	}

	var tables []*model.TableWithPlayerEmail
	assertGet(t, ts, "/admin/table?rows=3", &tables, http.StatusOK, j1)
	a.Equal(3, len(tables))
	a.Equal(p1.Email, tables[0].Email)
	a.Equal("Table 4", tables[0].Name)
	a.True(tables[0].Deleted)
	a.False(tables[1].Deleted)
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
