package mealplanner

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/mail"
	"net/smtp"
	"regexp"

	"github.com/eldelto/core/internal/web"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

const (
	recipeBucket = "recipes"
)

var (
	todoItemRegex = regexp.MustCompile(`-?\s*(\[([xX ])?\])?\s*([^\[]+)`)

	//go:embed login.tmpl
	rawLoginTemplate string
	loginTemplate    = template.New("login")
)

func init() {
	_, err := loginTemplate.Parse(rawLoginTemplate)
	if err != nil {
		panic(fmt.Errorf("failed to parse login template: %w", err))
	}
}

type Service struct {
	db       *bbolt.DB
	host     string
	smtpHost string
	smtpAuth smtp.Auth
	auth     *web.Authenticator
}

func NewService(db *bbolt.DB,
	host string,
	smtpHost string,
	smtpAuth smtp.Auth,
	auth *web.Authenticator) (*Service, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(recipeBucket))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &Service{
		db:       db,
		host:     host,
		smtpHost: smtpHost,
		smtpAuth: smtpAuth,
		auth:     auth,
	}, nil
}

func getUserAuth(ctx context.Context) (*web.UserAuth, error) {
	auth, err := web.GetAuth(ctx)
	if err != nil {
		return nil, err
	}

	userAuth, ok := auth.(*web.UserAuth)
	if !ok {
		return nil, fmt.Errorf("only allowed for logged in users: %w", web.ErrUnauthenticated)
	}

	return userAuth, nil
}

type loginData struct {
	Host  string
	Token web.TokenID
}

func (s Service) SendLoginEmail(email mail.Address, token web.TokenID) error {
	data := loginData{Host: s.host, Token: token}

	content := bytes.Buffer{}
	if err := loginTemplate.Execute(&content, data); err != nil {
		return fmt.Errorf("failed to execute login template: %w", err)
	}

	if s.smtpAuth == nil {
		log.Println(content.String())
		return nil
	}

	return smtp.SendMail(s.smtpHost, s.smtpAuth, "no-reply@eldelto.net",
		[]string{email.Address}, content.Bytes())
}

func (s *Service) ListMyRecipes(ctx context.Context) ([]Recipe, error) {
		recipe1, _ := ParseRecipe(`
Carbonara

$Portions: 2
$Time: 20

Cut {100 g | guanciale} into small pieces and start searing them in
a pan with butter.

Meanwhile cook {300 g | spaghetti} in a pot of salted water.
`)
		recipe2, _ := ParseRecipe(`
Burritos

$Portions: 3
$Time: 35

Cut {1 | red onion}, {1 | zucchini} and {2 cloves | garlic}.

Preheat a pan with {oil} and add {2 tsp | paprika powder} and let simmer until fragrant.

Add the {garlic}, {onion} and the {zucchini}.
`)

	recipes := []Recipe{recipe1, recipe2}
	return recipes, nil
}

func (s *Service) GetRecipe(ctx context.Context, id uuid.UUID) (Recipe, error) {
			return ParseRecipe(`
Carbonara

$Portions: 2
$Time: 20

Cut {100 g | guanciale} into small pieces and start searing them in
a pan with butter.

Meanwhile cook {300 g | spaghetti} in a pot of salted water.
`)
}

func (s *Service) NewRecipe(ctx context.Context, rawRecipe string) (Recipe, error) {
	recipe, err := ParseRecipe(rawRecipe)
	if err != nil {
		return recipe, err
	}

	// TODO: Store
	return recipe, nil
}
