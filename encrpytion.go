package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"

	"github.com/joho/godotenv"
)

func convertPublicPemKey(Pem []byte) *rsa.PublicKey {
	keyBlock, _ := pem.Decode(Pem)
	key, err := x509.ParsePKIXPublicKey(keyBlock.Bytes)
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert public pemkey")
	}
	keyConverted := key.(*rsa.PublicKey)
	return keyConverted
}

func convertPrivatePemKey(Pem []byte) *rsa.PrivateKey {
	keyBlock, _ := pem.Decode([]byte(Pem))
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)

	if err != nil {
		logger.Fatal().Err(err).Msg("failed to convert private pemkey")
	}

	return key
}

func encryptAES(key []byte, text string) string {
	if len(key) != 32 {
		logger.Error().Err(fmt.Errorf("AES key must be 32 bytes (AES-256), got %d bytes", len(key))).Msg("")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create AES cipher block")
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create AES-GCM")
	}

	iv := make([]byte, 12)
	_, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
		logger.Error().Err(err).Msg("failed to generate IV")
	}

	encrypted := aesgcm.Seal(nil, iv, []byte(text), nil)
	output := base64.StdEncoding.EncodeToString(encrypted)
	ivBase64 := base64.StdEncoding.EncodeToString(iv)

	return output + "iv:" + ivBase64
}

func decryptAES(encryptedText []byte, key []byte, iv []byte) ([]byte, error) {
	if len(key) != 32 {
		return []byte(""), fmt.Errorf("AES key must be 32 bytes (AES-256), got %d bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte(""), fmt.Errorf("failed to create AES cipher block: %w", err)
	}

	// Create AES-GCM
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return []byte(""), fmt.Errorf("failed to create AES-GCM: %w", err)
	}

	// Decrypt
	plaintext, err := aesgcm.Open(nil, iv, encryptedText, nil)
	if err != nil {
		return []byte(""), fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil

}

func encrypt(pKey *rsa.PublicKey, encoded []byte) []byte {
	encrypted, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		pKey,
		encoded,
		nil,
	)
	if err != nil {
		logger.Error().Err(err).Msg("failed to encrypt using rsa")
	}
	return encrypted
}

func decrypt(key *rsa.PrivateKey, encrypted []byte) []byte {
	decrypted, err := rsa.DecryptOAEP(
		sha256.New(),
		rand.Reader,
		key,
		encrypted,
		nil,
	)
	if err != nil {
		logger.Error().Err(err).Msg("failed to decrypt using rsa")
	}
	return decrypted
}

func generateRSAKey() {

	if _, err := os.Stat("./assets"); os.IsNotExist(err) {
		err := os.Mkdir("./assets", 0700)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create assets directory")
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	publicKey = &privateKey.PublicKey

	privKB := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPem = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKB,
	})
	err = os.WriteFile("./assets/private.pem", privateKeyPem, 0644)
	if err != nil {
		logger.Fatal().Msg("Failed to write to /assets/private.pem")
		panic(err)
	}

	pubKB, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		logger.Fatal().Msg("Failed to convert public key to PHIX")
		panic(err)
	}
	publicKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKB,
	})
	err = os.WriteFile("./assets/public.pem", publicKeyPEM, 0644)
	if err != nil {
		logger.Fatal().Msg("Failed to write to /assets/public.pem")
		panic(err)
	}
}

func setupEncryption() {
	err := godotenv.Load(".env")
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load .env file")
	}

	adminPassword = os.Getenv("adminPassword")
	authentication = os.Getenv("authentication")
	publicKeyPEM, err = os.ReadFile("./assets/public.pem")
	if err != nil {
		generateRSAKey()
		logger.Info().Msg("Public key not found generating new RSA pair")
	} else {
		privateKeyPem, err = os.ReadFile("./assets/private.pem")
		if err != nil {
			logger.Fatal().Msg("Failed to read from /assets/private.pem")
			panic(err.Error())
		}
	}

	publicKey = convertPublicPemKey(publicKeyPEM)
	privateKey = convertPrivatePemKey(privateKeyPem)

	plaintext := []byte("hello, world!")
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey.(*rsa.PublicKey), plaintext)
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("RSA Encryption Functioning as Intended")
	fmt.Printf("RSA key size: %d bits\n", privateKey.N.BitLen())
	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
	if err != nil {
		panic(err.Error())
	}
	_ = decrypted
}
