package mux

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"mondaynightpoker-server/pkg/table"
	"strings"
	"testing"
)

func Test_getTable(t *testing.T) {
	setupJWT()
	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	p, j := player()

	tbl1, _ := p.CreateTable(cbg, "Table 1")
	tbl2, _ := p.CreateTable(cbg, "Table 2")
	tbl3, _ := p.CreateTable(cbg, "Table 3")

	p2, j2 := player()
	tbl4, _ := p2.CreateTable(cbg, "Table 4")
	_, _ = p2.Join(cbg, tbl2)

	var tables []*table.Table
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
	setupJWT()
	p, j := player()

	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	// verify it requires admin access
	assertPost(t, ts, "/table", postTablePayload{Name: "Test"}, nil, 403, j)

	// actually test it
	_ = p.SetIsSiteAdmin(context.Background(), true)
	var tbl *table.Table
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
	setupJWT()
	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	p, j := player()
	tbl, _ := p.CreateTable(context.Background(), "My Table")

	path := fmt.Sprintf("/table/%s/seat", tbl.UUID)
	var errObj errorResponse
	assertPost(t, ts, path, nil, &errObj, 400, j)
	assert.Equal(t, "player is already at the table", errObj.Message)

	_, j2 := player()
	var respObj *table.PlayerTable
	assertPost(t, ts, path, nil, &respObj, 201, j2)
	assert.Equal(t, 0, respObj.Balance)
	assert.True(t, respObj.Active)
}

func Test_getTableUUID(t *testing.T) {
	setupJWT()
	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	p1, j := player()
	p2, _ := player()

	tbl, _ := p1.CreateTable(context.Background(), "My Table")
	_, _ = p2.Join(context.Background(), tbl)


	path := fmt.Sprintf("/table/%s", tbl.UUID)
	var respObj getTableUUIDResponse
	assertGet(t, ts, path, &respObj, 200, j)

	assert.Equal(t, tbl.UUID, respObj.Table.UUID)
	assert.Equal(t, 2, len(respObj.Players))
}