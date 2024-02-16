package database_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"testing"

	//revive:disable-next-line:dot-imports
	. "github.com/andrescosta/goico/pkg/database"
	"github.com/andrescosta/goico/pkg/test"
)

type addresses []address

var tenant = "100"

type (
	data struct {
		Idd       string
		Name      string
		Age       uint64
		Addresses addresses
	}
	address struct {
		City    string
		Country string
	}
	step interface {
		execute(*scenario) error
	}
	scenario struct {
		memory []data
		table  *Table[data]
		name   string
		steps  []step
	}
)

type (
	addStep            struct{}
	deleteStep         struct{}
	deleteDBOnlyStep   struct{}
	updateStep         struct{}
	allStep            struct{}
	getStep            struct{}
	notfoundStep       struct{}
	fillRandomDataStep struct{}
)

var (
	ErrEmptyMemory = errors.New("no data")
	ErrMashal      = errors.New("marshal")
	ErrUnmashal    = errors.New("unmarshal")
)

type faultyMarshaller[T Identifiable] struct{}

type dataIsDifferentError struct {
	message string
}

var (
	add            = addStep{}
	deleteone      = deleteStep{}
	onlydelete     = deleteDBOnlyStep{}
	update         = updateStep{}
	all            = allStep{}
	get            = getStep{}
	notfound       = notfoundStep{}
	fillrandomdata = fillRandomDataStep{}
)

func TestPathError(t *testing.T) {
	t.Parallel()
	_, err := Open(context.Background(), "", Option{})
	test.NotNil(t, err)
}

func TestOperations(t *testing.T) {
	t.Parallel()
	scenarios := []*scenario{
		newscenario("add", fillrandomdata, add),
		newscenario("get", fillrandomdata, add, get),
		newscenario("delete", fillrandomdata, add, onlydelete, notfound),
		newscenario("update", fillrandomdata, add, update, get),
		newscenario("getall", fillrandomdata, fillrandomdata, fillrandomdata, add, all),
		newscenario("getall_and_delete", fillrandomdata, fillrandomdata, add, deleteone, all),
		newscenario("getall_and_update", fillrandomdata, fillrandomdata, add, update, all),
	}
	ops := []Option{{}, {InMemory: true}}
	for _, o := range ops {
		db, err := Open(context.Background(), filepath.Join(t.TempDir(), "database"), o)
		test.Nil(t, err)
		defer func() {
			err := db.Close(context.Background())
			test.Nil(t, err)
		}()

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				s.execute(t, db)
			})
		}
	}
}

func TestMarshallerError(t *testing.T) {
	t.Parallel()
	scenariosErrors := []*scenario{
		newscenario("add", add),
		newscenario("get", get),
		newscenario("update", update),
		newscenario("all", all),
	}
	scenariosData := newscenario("data", fillrandomdata, add)

	dbName := filepath.Join(t.TempDir(), "database-m-error.md")
	db, err := Open(context.Background(), dbName, Option{})
	test.Nil(t, err)
	defer func() {
		err := db.Close(context.Background())
		test.Nil(t, err)
	}()

	scenariosData.execute(t, db)

	for _, s := range scenariosErrors {
		t.Run(s.name, func(t *testing.T) {
			s.memory = scenariosData.memory
			s.executeMarshallerError(t, db, scenariosData.table.Name)
		})
	}
}

func (s *scenario) executeMarshallerError(t *testing.T, db *Database, tableName string) {
	marshaller := faultyMarshaller[data]{}
	table := NewTable(db, tableName, tenant, marshaller)
	s.table = table
	for _, step := range s.steps {
		err := step.execute(s)
		if !errors.Is(err, ErrMashal) &&
			!errors.Is(err, ErrUnmashal) {
			t.Errorf("expected marshaller error got %s", err)
		}
	}
}

func (s *scenario) execute(t *testing.T, db *Database) {
	marshaller := BinaryMarshaller[data]{}
	tableName := fmt.Sprintf("%s%s", t.Name(), "table")
	table := NewTable(db, tableName, tenant, marshaller)
	s.table = table
	for _, step := range s.steps {
		err := step.execute(s)
		test.Nil(t, err)
	}
}

func (a fillRandomDataStep) execute(s *scenario) error {
	d := randomData()
	s.memory = append(s.memory, d)
	return nil
}

func (a addStep) execute(s *scenario) error {
	if len(s.memory) == 0 {
		return ErrEmptyMemory
	}
	for _, d := range s.memory {
		if err := s.table.Add(d); err != nil {
			return err
		}
	}
	return nil
}

func (u updateStep) execute(s *scenario) error {
	if len(s.memory) == 0 {
		return ErrEmptyMemory
	}
	s.memory[0].Name = randomString(8)
	s.memory[0].Age = randomInt(90)
	s.memory[0].Addresses[0].City = randomString(10)
	if err := s.table.Update(s.memory[0]); err != nil {
		return err
	}
	return nil
}

func (g getStep) execute(s *scenario) error {
	if len(s.memory) == 0 {
		return ErrEmptyMemory
	}
	d, err := s.table.Get(s.memory[0].Idd)
	if err != nil {
		return err
	}
	if d == nil {
		return dataIsDifferentError{fmt.Sprintf("expected %s got <nil>", s.memory[0])}
	}
	if !reflect.DeepEqual(s.memory[0], *d) {
		return dataIsDifferentError{fmt.Sprintf("expected %s got %s", s.memory[0], d)}
	}
	return nil
}

func (n notfoundStep) execute(s *scenario) error {
	if len(s.memory) == 0 {
		return ErrEmptyMemory
	}
	d, err := s.table.Get(s.memory[0].Idd)
	if err != nil {
		return err
	}
	if d != nil {
		return fmt.Errorf("expected <nil> got %s", d)
	}
	return nil
}

func (d deleteStep) execute(s *scenario) error {
	err := deleteDBOnlyStep{}.execute(s)
	if err != nil {
		return err
	}
	s.memory = s.memory[1:]
	return nil
}

func (d deleteDBOnlyStep) execute(s *scenario) error {
	if len(s.memory) == 0 {
		return ErrEmptyMemory
	}

	if err := s.table.Delete(s.memory[0].Idd); err != nil {
		return err
	}
	return nil
}

func (a allStep) execute(s *scenario) error {
	data1, err := s.table.All()
	if err != nil {
		return err
	}

	sort.Slice(data1, func(i, j int) bool { return data1[i].Idd < data1[j].Idd })
	sort.Slice(s.memory, func(i, j int) bool { return s.memory[i].Idd < s.memory[j].Idd })
	e := slices.EqualFunc(data1, s.memory,
		func(d1, d2 data) bool { return reflect.DeepEqual(d1, d2) })
	if !e {
		return dataIsDifferentError{"slices are different"}
	}
	return nil
}

func (d data) ID() string { return d.Idd }

func newscenario(name string, steps ...step) *scenario {
	return &scenario{name: name, steps: steps}
}

func randomData() data {
	return data{
		Idd:  randomString(10),
		Name: randomString(12),
		Age:  randomInt(100),
		Addresses: addresses{
			{randomString(15), randomString(10)},
		},
	}
}

func randomString(size int) string {
	rb := make([]byte, size)
	_, _ = rand.Read(rb)
	rs := base64.URLEncoding.EncodeToString(rb)
	return rs
}

func randomInt(max uint64) uint64 {
	i, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return i.Uint64()
}

func (d data) String() string {
	return fmt.Sprintf(`{id:%s, name:%s, age: %d, addresses: %s}`,
		d.Idd, d.Name, d.Age, d.Addresses)
}

func (a addresses) String() string {
	b := bytes.Buffer{}
	b.WriteString("{")
	for idx, aa := range a {
		if idx > 0 {
			b.WriteString(",")
		}
		d := fmt.Sprintf("[%d]{%s,%s}", idx, aa.City, aa.Country)
		b.WriteString(d)
	}
	b.WriteString("}")
	return b.String()
}

func (d dataIsDifferentError) Error() string {
	return d.message
}

func (d faultyMarshaller[T]) Marshal(_ T) (string, []byte, error) {
	return "", nil, ErrMashal
}

func (d faultyMarshaller[T]) Unmarshal(_ []byte) (T, error) {
	var t T
	return t, ErrUnmashal
}
