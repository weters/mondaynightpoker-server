package mux

import (
	"context"
	"fmt"
	"mondaynightpoker-server/pkg/model"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getTable(t *testing.T) {
	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	p, j := player()

	p.IsSiteAdmin = true // so we can rapidly create tables
	tbl1, _ := testRepos.Tables.CreateTable(cbg, p, "Table 1")
	tbl2, _ := testRepos.Tables.CreateTable(cbg, p, "Table 2")
	tbl3, _ := testRepos.Tables.CreateTable(cbg, p, "Table 3")

	p2, j2 := player()
	tbl4, _ := testRepos.Tables.CreateTable(cbg, p2, "Table 4")
	_, _ = testRepos.Tables.Join(cbg, p2, tbl2)

	var tables []*model.Table
	assertGet(t, ts, "/table", &tables, 200, j)
	assert.Equal(t, 3, len(tables))
	assert.Equal(t, tbl3.UUID, tables[0].UUID)
	assert.Equal(t, tbl2.UUID, tables[1].UUID)
	assert.Equal(t, tbl1.UUID, tables[2].UUID)

	assertGet(t, ts, "/table?start=1&rows=1", &tables, 200, j)
	assert.Equal(t, 1, len(tables))
	assert.Equal(t, tbl2.UUID, tables[0].UUID)

	assertGet(t, ts, "/table", &tables, 200, j2)
	assert.Equal(t, 2, len(tables))
	assert.Equal(t, tbl2.UUID, tables[0].UUID)
	assert.Equal(t, tbl4.UUID, tables[1].UUID)

	// bad pagination
	var err errorResponse
	assertGet(t, ts, "/table?start=-1", &err, 400, j2)
	assert.Equal(t, "start cannot be less than zero", err.Message)
}

func Test_postTable(t *testing.T) {
	p, j := player()

	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	// verify it requires admin access
	assertPost(t, ts, "/table", postTablePayload{Name: "Test"}, nil, 401)

	// actually test it
	_ = testRepos.Players.SetIsSiteAdmin(context.Background(), p, true)
	var tbl *model.Table
	assertPost(t, ts, "/table", postTablePayload{Name: "Test"}, &tbl, 201, j)
	assert.Equal(t, "Test", tbl.Name)
	assert.NotEmpty(t, tbl.UUID)

	// require valid name
	var err errorResponse
	assertPost(t, ts, "/table", postTablePayload{Name: "Te"}, &err, 400, j)
	assert.Equal(t, "name must be 3-40 characters", err.Message)

	// require valid name
	err = errorResponse{}
	assertPost(t, ts, "/table", postTablePayload{Name: strings.Repeat("A", 41)}, &err, 400, j)
	assert.Equal(t, "name must be 3-40 characters", err.Message)
}

func Test_postTableUUIDJoin(t *testing.T) {
	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	p, j := player()
	tbl, _ := testRepos.Tables.CreateTable(context.Background(), p, "My Table")

	path := fmt.Sprintf("/table/%s/seat", tbl.UUID)
	var errObj errorResponse
	assertPost(t, ts, path, nil, &errObj, 400, j)
	assert.Equal(t, "player is already at the table", errObj.Message)

	_, j2 := player()
	var respObj *model.PlayerTable
	assertPost(t, ts, path, nil, &respObj, 201, j2)
	assert.Equal(t, 0, respObj.Balance)
	assert.True(t, respObj.Active)
}

func Test_postTableUUIDClone(t *testing.T) {
	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	admin, jAdmin := player()
	_ = testRepos.Players.SetIsSiteAdmin(context.Background(), admin, true) // bypass cooldown
	tbl, _ := testRepos.Tables.CreateTable(context.Background(), admin, "Original")

	p2, j2 := player()
	_, _ = testRepos.Tables.Join(context.Background(), p2, tbl)

	path := fmt.Sprintf("/table/%s/clone", tbl.UUID)

	// non-admin at table is rejected
	var errObj errorResponse
	assertPost(t, ts, path, postTablePayload{Name: "Clone"}, &errObj, 400, j2)
	assert.Equal(t, "only a table admin can clone a table", errObj.Message)

	// invalid name
	errObj = errorResponse{}
	assertPost(t, ts, path, postTablePayload{Name: "no"}, &errObj, 400, jAdmin)
	assert.Equal(t, "name must be 3-40 characters", errObj.Message)

	// admin can clone
	var newTbl *model.Table
	assertPost(t, ts, path, postTablePayload{Name: "Cloned"}, &newTbl, 201, jAdmin)
	assert.Equal(t, "Cloned", newTbl.Name)
	assert.NotEqual(t, tbl.UUID, newTbl.UUID)
	assert.Equal(t, admin.ID, newTbl.PlayerID)

	clonedTbl, err := testRepos.Tables.GetTableByUUID(context.Background(), newTbl.UUID)
	assert.NoError(t, err)
	clonedPlayers, err := testRepos.Tables.GetPlayers(context.Background(), clonedTbl)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(clonedPlayers))
	for _, cp := range clonedPlayers {
		assert.Equal(t, 0, cp.Balance)
		assert.False(t, cp.Active)
	}
}

func Test_getTableUUID(t *testing.T) {
	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	p1, j := player()
	p2, _ := player()

	tbl, _ := testRepos.Tables.CreateTable(context.Background(), p1, "My Table")
	_, _ = testRepos.Tables.Join(context.Background(), p2, tbl)

	path := fmt.Sprintf("/table/%s", tbl.UUID)
	var respObj getTableUUIDResponse
	assertGet(t, ts, path, &respObj, 200, j)

	assert.Equal(t, tbl.UUID, respObj.Table.UUID)
	assert.Equal(t, 2, len(respObj.Players))
}
