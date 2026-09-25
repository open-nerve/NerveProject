package httpadapter_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11).
func TestMain(m *testing.M) { apitest.Main(m, "instance") }
