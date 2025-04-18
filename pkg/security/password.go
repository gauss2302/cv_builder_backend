package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strings"
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

func VerifyPassword(password, encodedHash string) (bool, error) {
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, nil
	}

	computedHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterators,
		params.Memory,
		params.Parallelism,
		params.KeyLength)

	if subtle.ConstantTimeCompare(hash, computedHash) == 1 {
		return true, nil
	}
	return false, nil
}

func decodeHash(encodedHash string) (*Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int

	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, err
	}

	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	var memory, iterations uint32
	var parallelism uint8

	if _, err := fmt.Sscanf(parts[3], "m=%d, t=%d, p=%d", &memory, &iterations, &parallelism); err != nil {
		return nil, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, err
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])

	if err != nil {
		return nil, nil, nil, err
	}

	params := &Argon2Params{
		Memory:      memory,
		Iterators:   iterations,
		Parallelism: parallelism,
		SaltLength:  uint32(len(salt)),
		KeyLength:   uint32(len(hash)),
	}

	return params, salt, hash, nil
}
