package vault

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "errors"
    "io"
)

func Encrypt(plaintext, pass []byte) ([]byte, error) {
    if len(pass) == 0 { return nil, errors.New("missing passphrase") }
    key := derive(pass)
    block, err := aes.NewCipher(key)
    if err != nil { return nil, err }
    gcm, err := cipher.NewGCM(block)
    if err != nil { return nil, err }
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil { return nil, err }
    out := gcm.Seal(nonce, nonce, plaintext, nil)
    return out, nil
}

func Decrypt(ciphertext, pass []byte) ([]byte, error) {
    if len(pass) == 0 { return nil, errors.New("missing passphrase") }
    key := derive(pass)
    block, err := aes.NewCipher(key)
    if err != nil { return nil, err }
    gcm, err := cipher.NewGCM(block)
    if err != nil { return nil, err }
    ns := gcm.NonceSize()
    if len(ciphertext) < ns { return nil, errors.New("bad data") }
    nonce := ciphertext[:ns]
    data := ciphertext[ns:]
    return gcm.Open(nil, nonce, data, nil)
}

func derive(pass []byte) []byte {
    key := make([]byte, 32)
    for i := range key { key[i] = pass[i%len(pass)] }
    return key
}

