package services

import "testing"

func TestSSEHub_HasSubscribers(t *testing.T) {
	hub := NewSSEHub()
	if hub.HasSubscribers("user:x:1") {
		t.Fatal("sala sem conexão não pode ter inscritos")
	}
	_, clean := hub.Register("c1", []string{"tenant:t", "user:x:1"})
	if !hub.HasSubscribers("user:x:1") {
		t.Fatal("sala com conexão aberta deve ter inscritos")
	}
	if hub.HasSubscribers("user:x:2") {
		t.Fatal("outra sala não pode herdar o inscrito")
	}
	clean()
	if hub.HasSubscribers("user:x:1") {
		t.Fatal("depois de fechar a conexão a sala volta a ficar sem inscritos")
	}
}
