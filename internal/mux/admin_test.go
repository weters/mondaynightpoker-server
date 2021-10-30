package mux

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMux_getAdminTable(t *testing.T) {
	a := assert.New(t)

	setupJWT()
	p1, j1 := player()

	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	assertGet(t, ts, "/admin/table", nil, http.StatusForbidden, j1)

	p1.IsSiteAdmin = true
	a.NoError(p1.Save(cbg))

	var err errorResponse
	assertGet(t, ts, "/admin/table?rows=0", &err, http.StatusBadRequest, j1)
	a.Equal("rows must be greater than zero", err.Message)

	for i := 0; i < 5; i++ {
		tbl, err := p1.CreateTable(cbg, fmt.Sprintf("Table %d", i))
		a.NoError(err)

		if i == 4 {
			tbl.Deleted = true
			a.NoError(tbl.Save(cbg))
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

	setupJWT()
	player, jwt := player()

	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	player.IsSiteAdmin = true
	a.NoError(player.Save(cbg))

	table, err := player.CreateTable(cbg, "Test Table")
	a.NoError(err)
	a.False(table.Deleted)

	var resp model.Table
	assertPostWithResp(t, ts, fmt.Sprintf("/admin/table/%s", table.UUID), postAdminTableUUIDPayload{true}, &resp, http.StatusOK, jwt)
	a.True(resp.Deleted)

	table2, err := model.GetTableByUUID(cbg, table.UUID)
	a.True(table2.Deleted)

	assertPostWithResp(t, ts, fmt.Sprintf("/admin/table/%s", table.UUID), postAdminTableUUIDPayload{false}, &resp, http.StatusOK, jwt)
	a.False(resp.Deleted)
}
