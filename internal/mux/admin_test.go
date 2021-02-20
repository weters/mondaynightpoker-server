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
		_, err := p1.CreateTable(cbg, fmt.Sprintf("Table %d", i))
		a.NoError(err)
	}

	var tables []*model.TableWithPlayerEmail
	assertGet(t, ts, "/admin/table?rows=3", &tables, http.StatusOK, j1)
	a.Equal(3, len(tables))
	a.Equal(p1.Email, tables[0].Email)
	a.Equal("Table 4", tables[0].Name)
}
