package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

// fakeQuerier implementa Querier em memória para os testes do Resolver.
type fakeQuerier struct {
	byEmail map[string]User
	calls   int
}

func (f *fakeQuerier) GetUserByEmail(_ context.Context, email string) (User, error) {
	f.calls++
	u, ok := f.byEmail[strings.ToLower(email)]
	if !ok {
		return User{}, sql.ErrNoRows
	}
	return u, nil
}

func (f *fakeQuerier) GetUserByID(_ context.Context, id int64) (User, error) {
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return User{}, sql.ErrNoRows
}

func newFake(users ...User) *fakeQuerier {
	f := &fakeQuerier{byEmail: make(map[string]User)}
	for _, u := range users {
		f.byEmail[strings.ToLower(u.Email)] = u
	}
	return f
}

var (
	admin    = User{ID: 1, Email: "admin@exemplo.pt", Name: "Admin", Role: "admin", Enabled: true}
	utente   = User{ID: 2, Email: "user@exemplo.pt", Name: "Utente", Role: "user", Enabled: true}
	inactivo = User{ID: 3, Email: "off@exemplo.pt", Role: "user", Enabled: false}
)

func TestResolveNormalizaEmail(t *testing.T) {
	r := NewResolver(newFake(utente))
	for _, in := range []string{"user@exemplo.pt", "USER@exemplo.pt", "  User@Exemplo.PT  "} {
		p, err := r.Resolve(context.Background(), in)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", in, err)
		}
		if p.UserID != 2 {
			t.Errorf("Resolve(%q).UserID = %d, esperado 2", in, p.UserID)
		}
	}
}

func TestResolveErros(t *testing.T) {
	r := NewResolver(newFake(utente, inactivo))

	if _, err := r.Resolve(context.Background(), ""); !errors.Is(err, ErrNoEmail) {
		t.Errorf("email vazio: erro = %v, esperado ErrNoEmail", err)
	}
	if _, err := r.Resolve(context.Background(), "ninguem@exemplo.pt"); !errors.Is(err, ErrUnknownUser) {
		t.Errorf("desconhecido: erro = %v, esperado ErrUnknownUser", err)
	}
	if _, err := r.Resolve(context.Background(), "off@exemplo.pt"); !errors.Is(err, ErrDisabledUser) {
		t.Errorf("desativado: erro = %v, esperado ErrDisabledUser", err)
	}
}

func TestResolvePapel(t *testing.T) {
	r := NewResolver(newFake(admin, utente))

	p, err := r.Resolve(context.Background(), admin.Email)
	if err != nil || !p.IsAdmin() {
		t.Errorf("admin: (%+v, %v), esperado IsAdmin", p, err)
	}
	p, err = r.Resolve(context.Background(), utente.Email)
	if err != nil || p.IsAdmin() {
		t.Errorf("utente: (%+v, %v), não devia ser admin", p, err)
	}
}

// TestResolveSemCache confirma o comportamento deliberado: cada chamada a Resolve
// consulta a BD, sem cache própria a servir pedidos repetidos. A expiração de
// identidade fica só a cargo do exp do JWT do Cloudflare Access.
func TestResolveSemCache(t *testing.T) {
	f := newFake(utente)
	r := NewResolver(f)

	for range 5 {
		if _, err := r.Resolve(context.Background(), utente.Email); err != nil {
			t.Fatal(err)
		}
	}
	if f.calls != 5 {
		t.Errorf("%d consultas à BD, esperado 5 (uma por Resolve, sem cache)", f.calls)
	}
}

// TestDesativacaoTemEfeitoImediato: sem cache, a próxima consulta já reflete a
// desativação.
func TestDesativacaoTemEfeitoImediato(t *testing.T) {
	f := newFake(utente)
	r := NewResolver(f)

	if _, err := r.Resolve(context.Background(), utente.Email); err != nil {
		t.Fatal(err)
	}

	desativado := utente
	desativado.Enabled = false
	f.byEmail[strings.ToLower(utente.Email)] = desativado

	if _, err := r.Resolve(context.Background(), utente.Email); !errors.Is(err, ErrDisabledUser) {
		t.Errorf("depois de desativar: erro = %v, esperado ErrDisabledUser (sem atraso de cache)", err)
	}
}
