package record

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSave_And_Get(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	rec := Record{
		Script:     "query-rocketmq-msg",
		Params:     map[string]string{"topic": "t1"},
		Status:     "success",
		ExitCode:   0,
		Stdout:     "result",
		ExecutedAt: time.Now().UTC().Truncate(time.Second),
	}

	id, err := store.Save(rec)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "rec-")

	got, err := store.Get(id)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, rec.Script, got.Script)
	assert.Equal(t, rec.Status, got.Status)
	assert.Equal(t, rec.ExitCode, got.ExitCode)
	assert.Equal(t, rec.Stdout, got.Stdout)
	assert.Equal(t, "t1", got.Params["topic"])
}

func TestGet_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	_, err := store.Get("rec-nonexistent")
	assert.Error(t, err)
}

func TestList_Pagination(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	// Insert 5 records
	for i := 0; i < 5; i++ {
		rec := Record{
			Script:     "query-rocketmq-msg",
			Status:     "success",
			ExecutedAt: time.Now().UTC(),
		}
		_, err := store.Save(rec)
		require.NoError(t, err)
	}

	// Page 1: 2 items
	result, err := store.List(1, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.PageSize)
	assert.Len(t, result.Records, 2)

	// Page 2: 2 items
	result2, err := store.List(2, 2)
	require.NoError(t, err)
	assert.Len(t, result2.Records, 2)

	// Page 3: 1 item
	result3, err := store.List(3, 2)
	require.NoError(t, err)
	assert.Len(t, result3.Records, 1)
}

func TestList_Empty(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	result, err := store.List(1, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Records)
}

func TestSave_IDUniqueness(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id, err := store.Save(Record{Script: "test", ExecutedAt: time.Now()})
		require.NoError(t, err)
		assert.False(t, ids[id], "duplicate ID: %s", id)
		ids[id] = true
	}
}

func TestSave_RecordIDPopulated(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	id, err := store.Save(Record{Script: "test", ExecutedAt: time.Now()})
	require.NoError(t, err)

	got, err := store.Get(id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
}
