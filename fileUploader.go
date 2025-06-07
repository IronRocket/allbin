package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
	slog "golang.org/x/exp/slog"
)

var tusdHandler *tusd.Handler

func generateToken(userID string, expiry time.Time) string {
	payload := fmt.Sprintf("%s:%d", userID, expiry.Unix())
	mac := hmac.New(sha256.New, []byte(UPLOAD_TOKEN))
	mac.Write([]byte(payload))
	sig := base64.URLEncoding.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s.%s", payload, sig)
}

func verifyToken(token string) (bool, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false, errors.New("invalid token format")
	}

	payload := parts[0] // "user123:1714160000"
	signature := parts[1]

	// Verify signature
	mac := hmac.New(sha256.New, []byte(UPLOAD_TOKEN))
	mac.Write([]byte(payload))
	expectedSig := base64.URLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return false, errors.New("invalid token signature")
	}

	// Check expiration
	payloadParts := strings.Split(payload, ":")
	if len(payloadParts) != 2 {
		return false, errors.New("invalid payload format")
	}

	expiryUnix, err := strconv.ParseInt(payloadParts[1], 10, 64)
	if err != nil {
		return false, errors.New("invalid expiry time")
	}

	if time.Now().Unix() > expiryUnix {
		return false, errors.New("token expired")
	}

	return true, nil
}

func initFileUploader() {
	composer := tusd.NewStoreComposer()
	store := filestore.New("./fileDatabase")
	store.UseIn(composer)
	silentLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := tusd.Config{
		BasePath:      "/files/",
		StoreComposer: composer,
		Logger:        silentLogger,
		PreUploadCreateCallback: func(hook tusd.HookEvent) (tusd.HTTPResponse, tusd.FileInfoChanges, error) {

			token := hook.HTTPRequest.Header.Get("token")

			ok, err := verifyToken(token)
			if !ok || err != nil {
				return tusd.ErrFileLocked.HTTPResponse,
					tusd.FileInfoChanges{},
					fmt.Errorf("invalid token: %v", err)
			}

			id := hook.HTTPRequest.Header.Get("id")
			dim, _, exists, err := spaceRead(id, true)
			if err != nil {
				return tusd.ErrFileLocked.HTTPResponse,
					tusd.FileInfoChanges{},
					fmt.Errorf("invalid id: %v", err)
			}
			if exists == true {
				if dim.FileName == "" {
					logger.
						Warn().
						Str(id, "dimension id requested").
						Msg(`request demands to add file to a
							already established data entry,
							possible malicious attack on data entry
							`)
					return tusd.ErrFileLocked.HTTPResponse,
						tusd.FileInfoChanges{},
						fmt.Errorf("invalid id")
				}
			} else {
				logger.Warn().Str(id, "dimension id requested").Msg("file couldn't find owner")
				return tusd.ErrFileLocked.HTTPResponse,
					tusd.FileInfoChanges{},
					fmt.Errorf("invalid id")
			}

			_, err = composer.Core.GetUpload(hook.Context, id)
			if err != nil {
				if errors.Is(err, tusd.ErrNotFound) {
					logger.Info().Msg("uuid didn't collide within file database")
				} else {
					logger.Error().Err(err).Str("id", id).Msg("file name conflict check failed")
					return tusd.ErrFileLocked.HTTPResponse,
						tusd.FileInfoChanges{},
						fmt.Errorf("invalid token: %v", err)
				}
			}

			return tusd.HTTPResponse{}, tusd.FileInfoChanges{ID: id}, nil
		},
	}
	tusdHandler, _ = tusd.NewHandler(config)
}
