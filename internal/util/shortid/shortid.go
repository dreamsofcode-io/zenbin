package shortid

import (
	"errors"
	"math/big"

	"github.com/google/uuid"
)

// Base57 character set
const base57Charset = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz123456789"

var base57Map = func() map[byte]int {
	m := make(map[byte]int)
	for i, b := range base57Charset {
		m[byte(b)] = i
	}
	return m
}()

// Base57Encode encodes a byte slice into a Base57 string.
func Base57Encode(data []byte) string {
	var result []byte
	value := new(big.Int).SetBytes(data)

	base := big.NewInt(57)
	zero := big.NewInt(0)
	mod := new(big.Int)

	for value.Cmp(zero) > 0 {
		value.DivMod(value, base, mod)
		result = append([]byte{base57Charset[mod.Int64()]}, result...)
	}

	return string(result)
}

// Base57Decode decodes a Base57 string back into a byte slice.
func Base57Decode(encoded string) ([]byte, error) {
	value := big.NewInt(0)
	base := big.NewInt(57)

	for _, char := range []byte(encoded) {
		digit, ok := base57Map[char]
		if !ok {
			return nil, errors.New("invalid Base57 character")
		}
		value.Mul(value, base)
		value.Add(value, big.NewInt(int64(digit)))
	}

	return value.Bytes(), nil
}

func GetShortID(u uuid.UUID) string {
	return Base57Encode(u[:])
}

func GetLongID(sid string) (uuid.UUID, error) {
	data, err := Base57Decode(sid)
	if err != nil {
		return uuid.UUID{}, err
	}

	// UUIDs are 16 bytes
	if len(data) != 16 {
		return uuid.UUID{}, errors.New("invalid UUID length")
	}

	var u uuid.UUID
	copy(u[:], data)
	return u, nil

}
