package controllers

import (
	"encoding/json"

	"github.com/alltomatos/watinkdev/business/internal/models"
)

// quotedMediaOf descreve a mídia citada para o engine desenhar a citação como o cliente oficial (áudio com microfone e
// duração, imagem com miniatura). nil para texto e tipos sem desenho próprio: ali vale o texto de quotedBody.
func quotedMediaOf(q models.Message) map[string]interface{} {
	switch q.MediaType {
	case "audio", "image", "video", "document", "sticker":
	default:
		return nil
	}
	var d struct {
		Mimetype  string `json:"mimetype"`
		Voice     bool   `json:"voice"`
		FileName  string `json:"fileName"`
		Thumbnail string `json:"thumbnail"`
		Rich      struct {
			Voice   bool `json:"voice"`
			Seconds int  `json:"seconds"`
		} `json:"rich"`
	}
	_ = json.Unmarshal([]byte(q.DataJson), &d)
	out := map[string]interface{}{"type": q.MediaType}
	if d.Mimetype != "" {
		out["mimeType"] = d.Mimetype
	}
	if d.Voice || d.Rich.Voice {
		out["voice"] = true
	}
	if d.Rich.Seconds > 0 {
		out["seconds"] = d.Rich.Seconds
	}
	if d.FileName != "" {
		out["fileName"] = d.FileName
	}
	if d.Thumbnail != "" {
		out["thumbnail"] = d.Thumbnail
	}
	if q.Body != "" {
		out["caption"] = q.Body
	}
	return out
}
