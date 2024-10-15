package acme

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"log"

	"github.com/go-acme/lego/v4/registration"
)

type User struct {
	Email        string
	Registration *registration.Resource
	PrivateKey   []byte
}

func NewUser(email string) (*User, error) {
	// Using RSA4096
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	return &User{
		Email:      email,
		PrivateKey: x509.MarshalPKCS1PrivateKey(privateKey),
	}, nil
}

func NewUserFull(email string, uri string, privateKey []byte) (*User, error) {
	return &User{
		Email:        email,
		Registration: &registration.Resource{URI: uri},
		PrivateKey:   privateKey,
	}, nil
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *User) GetPrivateKey() crypto.PrivateKey {
	privateKey, err := x509.ParsePKCS1PrivateKey(u.PrivateKey)
	if err != nil {
		log.Println("Failed to parse private key:", err)
		return nil
	}

	return privateKey
}
