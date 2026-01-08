package common

import (
	crand "crypto/rand"
	"fmt"
	"math/big"
	"math/rand"
)

var letterRunes = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

const idLetterCount = 12

func MakeRandomID() string {
	b := make([]rune, idLetterCount)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func MakeSigninRequestToken() string {
	b := make([]rune, 16)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func MakeSigninRequestPin() (pin string, err error) {
	pinNum, err := crand.Int(crand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return
	}
	pin = fmt.Sprintf("%06d", pinNum)
	return
}
