package database_test

import (
	"bytes"
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

	. "github.com/andrescosta/goico/pkg/database"
)

type addresses []address

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
		execute(*testing.T, *scenario) error
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
	deleteNotStep      struct{}
	updateStep         struct{}
	allStep            struct{}
	getStep            struct{}
	notgetStep         struct{}
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
	delete         = deleteStep{}
	deletenot      = deleteNotStep{}
	update         = updateStep{}
	all            = allStep{}
	get            = getStep{}
	notget         = notgetStep{}
	fillrandomdata = fillRandomDataStep{}
)

func TestDatabaseError(t *testing.T) {
	if _, err := Open("{}\\/empty"); err == nil {
		t.Errorf("expecting path not found got <nil>")
	}
}
func TestOperations(t *testing.T) {
	t.Parallel()
	scenarios := []scenario{
		new("add", fillrandomdata, add),
		new("get", fillrandomdata, add, get),
		new("delete", fillrandomdata, add, deletenot, notget),
		new("update", fillrandomdata, add, update, get),
		new("all", fillrandomdata, fillrandomdata, fillrandomdata, add, all),
		new("all_del", fillrandomdata, fillrandomdata, add, delete, all),
		new("all_update", fillrandomdata, fillrandomdata, add, update, all),
	}
	dbName := filepath.Join(t.TempDir(), "database.md")
	db, err := Open(dbName)
	if err != nil {
		t.Fatalf("Database.Open: %s", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Database.Close: %s", err)
		}
	}()

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			s.execute(t, db)
		})
	}
}

func TestBucketErrors(t *testing.T) {
	t.Parallel()
	scenarios := []scenario{
		new("add", add),
		new("get", get),
		new("delete", delete),
		new("update", update),
		new("all", all),
	}
	dbName := filepath.Join(t.TempDir(), "database-b-error.md")
	db, err := Open(dbName)
	if err != nil {
		t.Fatalf("Database.Open: %s", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Database.Close: %s", err)
		}
	}()

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			fillrandomdata.execute(t, &s)
			s.executeBucketError(t, db)
		})
	}
}
func TestMarshallerError(t *testing.T) {
	t.Parallel()
	scenariosErrors := []scenario{
		new("add", add),
		new("get", get),
		new("update", update),
		new("all", all),
	}
	scenariosData := new("data", fillrandomdata, add)

	dbName := filepath.Join(t.TempDir(), "database-m-error.md")
	db, err := Open(dbName)
	if err != nil {
		t.Fatalf("Database.Open: %s", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Database.Close: %s", err)
		}
	}()

	scenariosData.execute(t, db)

	for _, s := range scenariosErrors {
		t.Run(s.name, func(t *testing.T) {
			s.memory = scenariosData.memory
			s.executeMarshallerError(t, db, scenariosData.table.Name)
		})
	}
}

func (s *scenario) executeBucketError(t *testing.T, db *Database) {
	marshaller := BinaryMarshaller[data]{}
	s.table = NewTable(db, "any", marshaller)
	for _, step := range s.steps {
		err := step.execute(t, s)
		if !errors.As(err, &NoTableError{}) {
			t.Errorf("expected database.NoTableError got %s", err)
			return
		}
		if err.Error() == "" {
			t.Error("expected error message")
		}
	}
}
func (s *scenario) executeMarshallerError(t *testing.T, db *Database, tableName string) {
	marshaller := faultyMarshaller[data]{}
	table, err := CreateTableIfNotExist(db, tableName, marshaller)
	if err != nil {
		t.Fatalf("Table.NewTable: %s", err)
	}
	s.table = table
	for _, step := range s.steps {
		err := step.execute(t, s)
		if !errors.Is(err, ErrMashal) &&
			!errors.Is(err, ErrUnmashal) {
			t.Errorf("expected marshaller error got %s", err)
		}
	}
}

func (s *scenario) execute(t *testing.T, db *Database) {
	marshaller := BinaryMarshaller[data]{}
	tableName := fmt.Sprintf("%s/%s", t.Name(), "table")
	table, err := CreateTableIfNotExist(db, tableName, marshaller)
	if err != nil {
		t.Fatalf("Table.NewTable: %s", err)
	}
	s.table = table
	for _, step := range s.steps {
		if err := step.execute(t, s); err != nil {
			t.Error(err)
			return
		}
	}
}

func (a fillRandomDataStep) execute(t *testing.T, s *scenario) error {
	d := randomData(t)
	s.memory = append(s.memory, d)
	return nil
}

func (a addStep) execute(t *testing.T, s *scenario) error {
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

func (u updateStep) execute(t *testing.T, s *scenario) error {
	if len(s.memory) == 0 {
		return ErrEmptyMemory
	}
	s.memory[0].Name = randomString(t, 8)
	s.memory[0].Age = randomInt(t, 90)
	s.memory[0].Addresses[0].City = randomString(t, 10)
	if err := s.table.Update(s.memory[0]); err != nil {
		return err
	}
	return nil
}

func (g getStep) execute(t *testing.T, s *scenario) error {
	if len(s.memory) == 0 {
		return ErrEmptyMemory
	}
	d, err := s.table.Get(s.memory[0].Idd)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(s.memory[0], *d) {
		return dataIsDifferentError{fmt.Sprintf("expected %s got %s", s.memory[0], d)}
	}
	return nil
}

func (n notgetStep) execute(t *testing.T, s *scenario) error {
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

func (d deleteStep) execute(t *testing.T, s *scenario) error {
	err := deleteNotStep{}.execute(t, s)
	if err != nil {
		return err
	}
	s.memory = s.memory[1:]
	return nil
}
func (d deleteNotStep) execute(t *testing.T, s *scenario) error {
	if len(s.memory) == 0 {
		return ErrEmptyMemory
	}

	if err := s.table.Delete(s.memory[0].Idd); err != nil {
		return err
	}
	return nil
}

func (a allStep) execute(t *testing.T, s *scenario) error {
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

func (d data) Id() string { return d.Idd }

func new(name string, steps ...step) scenario {
	return scenario{name: name, steps: steps}
}

func randomData(t *testing.T) data {
	return data{
		Idd:  randomString(t, 10),
		Name: randomString(t, 12),
		Age:  randomInt(t, 100),
		Addresses: addresses{
			{randomString(t, 15), randomString(t, 10)},
		},
	}
}

func randomString(t *testing.T, size int) string {
	rb := make([]byte, size)
	_, err := rand.Read(rb)
	if err != nil {
		t.Errorf("rand.Read:%s", err)
	}
	rs := base64.URLEncoding.EncodeToString(rb)
	return rs
}

func randomInt(t *testing.T, max uint64) uint64 {
	i, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		t.Errorf("rand.Int:%s", err)
	}
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

func (d faultyMarshaller[T]) Marshal(v T) (string, []byte, error) {
	return "", nil, ErrMashal
}

func (d faultyMarshaller[T]) Unmarshal(v []byte) (T, error) {
	var t T
	return t, ErrUnmashal
}
