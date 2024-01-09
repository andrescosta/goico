package yamlutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andrescosta/goico/pkg/collection"

	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/yamlutil"
)

type queue struct {
	ID string
}
type data struct {
	Name   string
	ID     string
	Tenant string
	Queues []queue
}

var file = `name: Demo JOB
id: go-demo-job
tenant: demogo
queues:
  - id: queue-default_1
  - id: queue-default_2
`

func TestDecodeFile(t *testing.T) {
	fileName := filepath.Join(t.TempDir(),
		"file.yaml")
	if err := os.WriteFile(fileName, []byte(strings.TrimSpace(file)), os.ModeAppend); err != nil {
		t.Fatalf("os.WriteFile: %s", err)
	}
	d := data{}
	if err := DecodeFile(fileName, &d); err != nil {
		t.Fatalf("DecodeFile: %s", err)
	}
	if d.ID != "go-demo-job" {
		t.Errorf("expected go-demo-job got %s ", d.ID)
	}
	if d.Name != "Demo JOB" {
		t.Errorf("expected Demo JOB got %s ", d.Name)
	}
	if d.Tenant != "demogo" {
		t.Errorf("expected Demo JOB got %s ", d.Tenant)
	}
	if len(d.Queues) != 2 {
		t.Errorf("expected queues len 2 got %d ", len(d.Queues))
	}
	queues := collection.NewSet("queue-default_1", "queue-default_2")
	for _, q := range d.Queues {
		if !queues.Has(q.ID) {
			t.Errorf("not found %s", q)
		}
	}
}

func TestMarchal(t *testing.T) {
	do := data{
		ID:     "id_1",
		Name:   "name_1",
		Tenant: "tenant_1",
		Queues: []queue{{"qid_1"}, {"qid_2"}},
	}
	m, err := Marshal(&do)
	if err != nil {
		t.Fatalf("Marshal %s", err)
	}
	t.Log(m)
	fileName := filepath.Join(t.TempDir(),
		"file2.yaml")
	if err := os.WriteFile(fileName, []byte(*m), os.ModeAppend); err != nil {
		t.Fatalf("os.WriteFile: %s", err)
	}
	d := data{}
	if err := DecodeFile(fileName, &d); err != nil {
		t.Fatalf("DecodeFile: %s", err)
	}
	if d.ID != "id_1" {
		t.Errorf("expected id_1 got %s ", d.ID)
	}
	if d.Name != "name_1" {
		t.Errorf("expected name_1 got %s ", d.Name)
	}
	if d.Tenant != "tenant_1" {
		t.Errorf("expected tenant_1 got %s ", d.Tenant)
	}
	if len(d.Queues) != 2 {
		t.Errorf("expected queues len 2 got %d ", len(d.Queues))
	}
	queues := collection.NewSet("qid_1", "qid_2")
	for _, q := range d.Queues {
		if !queues.Has(q.ID) {
			t.Errorf("not found %s", q)
		}
	}
}

func TestErrors(t *testing.T) {
	d := data{}
	if err := DecodeFile("myfile.yaml", &d); err == nil {
		t.Errorf("DecodeFile: expected error got <nil>")
	}
	fileName := filepath.Join(t.TempDir(),
		"file3.yaml")
	if err := os.WriteFile(fileName, []byte("aaasssdshjk"), os.ModeAppend); err != nil {
		t.Fatalf("os.WriteFile: %s", err)
	}
	d = data{}
	if err := DecodeFile(fileName, &d); err == nil {
		t.Errorf("DecodeFile: expected error got <nil>")
	}
}
