package recording

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/alltomatos/watinkdev/business/internal/domain"
	"github.com/google/uuid"
)

// Status da gravação de uma chamada (CallLog.RecordingStatus).
const (
	StatusRecording = "recording"
	StatusReady     = "ready"
	StatusFailed    = "failed"
)

// ErrNoStorage: a instalação não tem armazenamento de objetos (S3), então não há
// como guardar uma gravação.
var ErrNoStorage = errors.New("armazenamento de objetos (S3) não configurado")

// ObjectKey é a chave do objeto no S3: {tenantId}/calls/{callId}.mp3. O banco
// guarda SÓ a chave (nunca uma URL assinada, que expira).
func ObjectKey(tenantID uuid.UUID, callID string) string {
	return fmt.Sprintf("%s/calls/%s.mp3", tenantID, callID)
}

// Upload envia o MP3 já pronto (arquivo temporário) ao armazenamento e REMOVE o
// arquivo temporário em qualquer desfecho. Em falha de envio nada parcial fica
// exposto: o objeto não é criado (o PutObject é atômico) e o chamador registra
// recordingStatus=failed. Devolve a chave em caso de sucesso.
func Upload(ctx context.Context, store domain.ObjectStore, tenantID uuid.UUID, callID string, f *os.File) (string, error) {
	name := f.Name()
	defer func() { _ = f.Close(); _ = os.Remove(name) }()

	if store == nil {
		return "", ErrNoStorage
	}
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() == 0 {
		return "", errors.New("gravação vazia")
	}
	key := ObjectKey(tenantID, callID)
	if err := store.Upload(ctx, key, f, info.Size(), "audio/mpeg"); err != nil {
		return "", fmt.Errorf("enviar gravação: %w", err)
	}
	return key, nil
}
