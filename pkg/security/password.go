package security

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/argon2"
)

var (
	// ErrInvalidHash indicates that the hash format is invalid
	ErrInvalidHash = errors.New("invalid hash format")
	// ErrIncompatibleVersion indicates that the argon2 version is not compatible
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)

type Argon2Params struct {
	Memory      uint32
	Iterators   uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultArgonParams() *Argon2Params {
	return &Argon2Params{
		Memory:      64 * 1024, // 64Mb
		Iterators:   3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}
}

func HashPassword(password string, params *Argon2Params) (string, error) {
	if params == nil {
		params = DefaultArgonParams()
	}

	salt := make([]byte, params.SaltLength)

	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterators,
		params.Memory,
		params.Parallelism,
		params.KeyLength)

	b64salt := base64.RawStdEncoding.EncodeToString(salt)
	b64hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterators,
		params.Parallelism,
		b64salt,
		b64hash,
	)
	return encodedHash, nil
}
