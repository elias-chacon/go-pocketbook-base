package securefile

import (
	"crypto/hmac"
	"crypto/sha256"
)

// pbkdf2 derives a key of keyLen bytes from a password (RFC 8018, PBKDF2 with
// HMAC-SHA-256). It is written here because the standard library only has it in
// recent Go versions, and the code of an app is Go 1.17.
func pbkdf2(password, salt []byte, iterations, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	blocks := (keyLen + hashLen - 1) / hashLen

	var out []byte
	buf := make([]byte, 4)
	for block := 1; block <= blocks; block++ {
		prf.Reset()
		prf.Write(salt)
		buf[0], buf[1], buf[2], buf[3] = byte(block>>24), byte(block>>16), byte(block>>8), byte(block)
		prf.Write(buf)
		u := prf.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}
