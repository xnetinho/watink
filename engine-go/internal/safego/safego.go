// Package safego executa goroutines com contenção de pânico: um pânico numa tarefa em segundo plano é registrado e não
// derruba o processo do engine (que carrega todas as sessões de WhatsApp).
package safego

import (
	"log"
	"runtime/debug"
	"sync/atomic"
)

var panics atomic.Int64

// Panics devolve quantos pânicos foram contidos (para métrica e teste).
func Panics() int64 { return panics.Load() }

// Go roda fn numa goroutine e contém qualquer pânico.
func Go(name string, fn func()) {
	go func() {
		defer Recover(name)
		fn()
	}()
}

// Recover deve ser usado com defer no topo de uma goroutine.
func Recover(name string) {
	if r := recover(); r != nil {
		panics.Add(1)
		log.Printf("[safego] pânico contido em %s: %v\n%s", name, r, debug.Stack())
	}
}

// Run executa fn e converte um pânico em erro, para tratadores que devem seguir para a fila de falhas.
func Run(name string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			panics.Add(1)
			log.Printf("[safego] pânico contido em %s: %v\n%s", name, r, debug.Stack())
			err = &PanicError{Name: name, Value: r}
		}
	}()
	return fn()
}

// PanicError é o erro devolvido por Run quando fn entra em pânico.
type PanicError struct {
	Name  string
	Value any
}

func (e *PanicError) Error() string { return "pânico em " + e.Name }
