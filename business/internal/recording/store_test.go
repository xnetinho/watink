package recording

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStore struct {
	objects  map[string][]byte
	ctype    map[string]string
	uploadFn func() error
	deleted  []string
}

func newFakeStore() *fakeStore {
	return &fakeStore{objects: map[string][]byte{}, ctype: map[string]string{}}
}

func (f *fakeStore) Upload(_ context.Context, key string, r io.Reader, _ int64, ct string) error {
	if f.uploadFn != nil {
		if err := f.uploadFn(); err != nil {
			return err
		}
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.objects[key], f.ctype[key] = b, ct
	return nil
}
func (f *fakeStore) Download(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (f *fakeStore) PresignedGetURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://s3.local/" + key + "?sig=x", nil
}
func (f *fakeStore) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	delete(f.objects, key)
	return nil
}
func (f *fakeStore) Describe() map[string]any { return nil }

func tempMP3(t *testing.T, content string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "rec-*.mp3")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	_, err = f.Seek(0, 0)
	require.NoError(t, err)
	return f
}

var ctx = context.Background()

func TestObjectKey(t *testing.T) {
	tenant := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	assert.Equal(t, "11111111-2222-3333-4444-555555555555/calls/ABC123.mp3", ObjectKey(tenant, "ABC123"))
}

func TestUpload_Success_StoresBytesUnderKeyAndRemovesTempFile(t *testing.T) {
	store := newFakeStore()
	tenant := uuid.New()
	f := tempMP3(t, "ID3-conteudo-mp3")
	name := f.Name()

	key, err := Upload(ctx, store, tenant, "CALL1", f)
	require.NoError(t, err)
	assert.Equal(t, ObjectKey(tenant, "CALL1"), key)
	assert.Equal(t, []byte("ID3-conteudo-mp3"), store.objects[key], "os bytes chegam íntegros")
	assert.Equal(t, "audio/mpeg", store.ctype[key])
	_, statErr := os.Stat(name)
	assert.True(t, os.IsNotExist(statErr), "o arquivo temporário é removido")
	assert.False(t, strings.Contains(key, "?") || strings.Contains(key, "http"), "a chave nunca é uma URL assinada")
}

func TestUpload_Failure_LeavesNoObjectAndRemovesTempFile(t *testing.T) {
	store := newFakeStore()
	store.uploadFn = func() error { return errors.New("s3 fora do ar") }
	f := tempMP3(t, "conteudo")
	name := f.Name()

	key, err := Upload(ctx, store, uuid.New(), "CALL2", f)
	require.Error(t, err)
	assert.Empty(t, key, "sem chave quando o envio falha")
	assert.Empty(t, store.objects, "nenhum áudio parcial fica exposto")
	_, statErr := os.Stat(name)
	assert.True(t, os.IsNotExist(statErr), "o temporário é removido também na falha")
}

func TestUpload_WithoutStoreFailsClearly(t *testing.T) {
	f := tempMP3(t, "conteudo")
	name := f.Name()
	_, err := Upload(ctx, nil, uuid.New(), "CALL3", f)
	assert.Equal(t, ErrNoStorage, err)
	_, statErr := os.Stat(name)
	assert.True(t, os.IsNotExist(statErr), "mesmo sem S3 o temporário não fica órfão no disco")
}

func TestUpload_EmptyRecordingIsRejected(t *testing.T) {
	store := newFakeStore()
	_, err := Upload(ctx, store, uuid.New(), "CALL4", tempMP3(t, ""))
	assert.Error(t, err)
	assert.Empty(t, store.objects, "gravação vazia não vira objeto")
}

// Integração leve: gravação real → MP3 → upload; o objeto guardado decodifica.
func TestUpload_RecordedAudioStoredIsValidMP3(t *testing.T) {
	r, err := New(t.TempDir())
	require.NoError(t, err)
	for i := 0; i < 3*50; i++ {
		r.AddOperator(sineFrame(i, 440, 9000))
		r.AddPeer(sineFrame(i, 880, 5000))
		require.NoError(t, r.Tick())
	}
	f, _, err := r.Finish()
	require.NoError(t, err)

	store := newFakeStore()
	tenant := uuid.New()
	key, err := Upload(ctx, store, tenant, "REAL1", f)
	require.NoError(t, err)

	rate, secs, _, rms := decode(t, store.objects[key])
	assert.Equal(t, 16000, rate)
	assert.InDelta(t, 3, secs, 1.0)
	assert.Greater(t, rms, 0.02, "a gravação contém áudio, não só silêncio")
	assert.True(t, bytes.HasPrefix(store.objects[key], []byte{0xFF}) || bytes.HasPrefix(store.objects[key], []byte("ID3")), "começa como um MP3")
}
