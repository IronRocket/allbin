package main

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"

	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/phuslu/log"
)

const PORT = "5000"
const DEV = true

var kill = false

type system struct {
	Name string
	Func int
}

type clientRSAKey struct {
	RSAKey string `json:"publicKey"`
}

type EncryptedPacket struct {
	RSAProAESKEY string `json:"aesKey"`
	AESProData   string `json:"data"`
}

type Dimension struct {
	Id                string `json:"id"`
	Title             string `json:"title"`
	Encrypted         bool   `json:"encrypted"`
	Visibility        bool   `json:"visibility"`
	Text              string `json:"text"`
	FileName          string `json:"fileName"`
	Reads             int    `json:"reads"`
	DownloadLimit     int    `json:"downloadLimit"`
	ExpirationDate    int    `json:"expirationDate"`
	ExpirationDateISO string `json:"expirationDateISO"`
}

type insert struct {
	Id    string `json:"id"`
	Token string `json:"token"`
}

type Page struct {
	html File
	css  File
	js   File
}

type File struct {
	content []byte
	name    string
}

var importantSystems string

var UPLOAD_TOKEN string
var adminPassword string
var authentication string
var publicKeyPEM []byte
var privateKeyPem []byte
var publicKey any
var privateKey *rsa.PrivateKey

var enterPage Page
var grabPage Page
var tmpl *template.Template

var logger log.Logger

func prettyDate(t int) string {
	return time.UnixMilli(int64(t)).UTC().Format("Jan 2, 2006 at 3:04pm UTC")
}

func GetIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

func getPublicKey(w http.ResponseWriter, r *http.Request) {
	logger.Info().Str("ip", GetIP(r)).Msg("client retrieving public key")
	w.Header().Add("Content-Type", "text/plain")
	w.Write(publicKeyPEM)
}

func sanatizeDimension(dim Dimension) error {
	threeMonthsLater := time.Now().AddDate(0, 3, 0)
	if time.UnixMilli(int64(dim.ExpirationDate)).After(threeMonthsLater) {
		return fmt.Errorf("expiration date is too long, expires: %s", prettyDate(dim.ExpirationDate))
	}
	return nil
}

// Post
func enterApi(w http.ResponseWriter, r *http.Request) {
	logger.Info().Str("ip", GetIP(r)).Msg("client creating a dimension")

	encryptedData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var lockAndPick EncryptedPacket

	json.Unmarshal(encryptedData, &lockAndPick)

	decodedPayload, err := base64.StdEncoding.DecodeString(lockAndPick.RSAProAESKEY)
	if err != nil {
		http.Error(w, "Base64 decode failed", http.StatusBadRequest)
		return
	}
	AESKEY := decrypt(privateKey, decodedPayload)

	decodedAESIV, err := base64.StdEncoding.DecodeString(lockAndPick.AESProData)
	if err != nil {
		http.Error(w, "Base64 decode failed", http.StatusBadRequest)
		return
	}

	decryptedJSON, err := decryptAES(decodedAESIV[12:], AESKEY, decodedAESIV[:12])
	if err != nil {
		logger.Error().Err(err).Msg("failed to decrypt dimension")
	}

	var dim Dimension
	if err := json.Unmarshal(decryptedJSON, &dim); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	fmt.Println("dim", dim)
	// Generate ID and token
	id := uuid.New().String()
	var token string
	if dim.FileName != "" {
		token = generateToken(id, time.Now().Add(10*time.Minute))
	}

	if err := sanatizeDimension(dim); err != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("nuh uh"))
		return
	}
	err = spaceWrite(id, dim)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// responed with id and/not token. Token is to auth
	// any file and the id points to sql row
	resp := insert{
		Id:    id,
		Token: token,
	}
	newDimension = true
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Get
func grabApi(w http.ResponseWriter, r *http.Request) {
	var path string = r.URL.Path
	var prefix string = "/api/grab/"
	if !strings.HasPrefix(path, prefix) {
		http.NotFound(w, r)
		return
	}

	id := strings.TrimPrefix(path, prefix)
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	dim, visibility, exists, err := spaceRead(id, false)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}

	if exists {
		fmt.Println(dim)

		dim.Visibility = visibility
		dim.ExpirationDateISO = prettyDate(dim.ExpirationDate)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dim)
	} else {
		http.Error(w, "nuh uh", http.StatusNotFound)
	}

	logger.Info().Str("dimension-id", id).Msg(prefix + " request")
}

func buildWebsite() {
	dist := exec.Command("rm", "-rf", "./dist")
	if err := dist.Run(); err != nil {
		logger.
			Fatal().Err(err).
			Msg("failed to remove old website production folder, command: " + strings.Join(dist.Args, " "))
	}

	build := exec.Command("npm", "run", "build")

	if err := build.Run(); err != nil {
		logger.
			Fatal().Err(err).
			Msg("failed to build production website, command: " + strings.Join(build.Args, " "))
	}
}

func filefinder(path string) Page {
	files, err := os.ReadDir(path)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to read website assets from " + path)
	}

	var page Page
	for _, file := range files {
		if !file.IsDir() && !strings.Contains(file.Name(), "map") {
			if strings.Contains(file.Name(), "html") {
				html, _ := os.ReadFile(path + file.Name())
				page.html = File{
					content: html,
					name:    "/" + file.Name(),
				}

			} else if strings.Contains(file.Name(), "css") {
				css, _ := os.ReadFile(path + file.Name())
				page.css = File{
					content: css,
					name:    "/" + file.Name(),
				}

			} else if strings.Contains(file.Name(), "js") {
				js, _ := os.ReadFile(path + file.Name())
				page.js = File{
					content: js,
					name:    "/" + file.Name(),
				}
			}
		}
	}
	return page
}

func grab(w http.ResponseWriter, r *http.Request) {
	var found bool = true

	var path string = r.URL.Path
	var prefix string = "/grab"
	if !strings.HasPrefix(path, prefix) {
		http.NotFound(w, r)
		return
	}

	truePath := strings.TrimPrefix(path, prefix)

	switch truePath {
	case "/", "/index.html":
		w.Header().Set("Content-Type", "text/html")
		w.Write(grabPage.html.content)
	case grabPage.css.name:
		w.Header().Set("Content-Type", "text/css")
		w.Write(grabPage.css.content)
	case grabPage.js.name:
		w.Header().Add("Content-Type", "text/javascript")
		w.Write(grabPage.js.content)
	default:
		found = false
		http.NotFound(w, r)
	}
	if found {
		logger.Info().Str("ip", GetIP(r)).Msg("grab" + truePath + " request")
	}
}

func enter(w http.ResponseWriter, r *http.Request) {
	var found bool = true

	switch r.URL.Path {
	case "/", "/index.html":
		w.Header().Set("Content-Type", "text/html")
		w.Write(enterPage.html.content)
	case enterPage.css.name:
		w.Header().Set("Content-Type", "text/css")
		w.Write(enterPage.css.content)
	case enterPage.js.name:
		w.Header().Add("Content-Type", "text/javascript")
		w.Write(enterPage.js.content)
	default:
		found = false
		http.NotFound(w, r)
	}
	if found {
		logger.Info().Str("ip", GetIP(r)).Msg("enter" + r.URL.Path)
	}
}

func main() {
	var err error

	tmpl, err = template.ParseFiles("./dist/grab/index.html")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to parse template")
	}

	runtime.GOMAXPROCS(2)
	UPLOAD_TOKEN = uuid.New().String()
	logger = log.Logger{
		Level:      log.InfoLevel,
		Caller:     1,
		TimeField:  "date",
		TimeFormat: "2006/01/02 03:04:05.000000",
		Writer: &log.MultiEntryWriter{
			&log.ConsoleWriter{ColorOutput: true},
			&log.FileWriter{
				Filename: filepath.Join("./logs", "log.json"),
				MaxSize:  500 * 1024 * 1024,
				Cleaner: func(filename string, maxBackups int, matches []os.FileInfo) {
					var dir = filepath.Dir(filename)
					for i, fi := range matches {
						filename := filepath.Join(dir, fi.Name())
						switch {
						case i > maxBackups:
							os.Remove(filename)
						case !strings.HasSuffix(filename, ".gz"):
							go exec.Command("nice", "gzip", filename).Run()
						}
					}
				},
			},
		},
	}

	buildWebsite() // rm -rf ./dist && npm run build

	// find all files in ./dist and load them into memory
	enterPage = filefinder("./dist/enter/")
	grabPage = filefinder("./dist/grab/")

	setupEncryption()
	initDatebase()     // sqllite database
	initFileUploader() // tus protocol
	go publicHandler()

	if DEV {
		enterDev := http.FileServer(http.Dir("./dist/enter"))
		http.Handle("/", enterDev)

		grabDev := http.FileServer(http.Dir("./dist/grab"))
		http.Handle("/grab/", http.StripPrefix("/grab/", grabDev))
	} else {
		http.HandleFunc("/", enter)
		http.HandleFunc("/grab/", grab)
	}

	http.HandleFunc("/public/", getPublicKey)

	http.HandleFunc("/api/enter", enterApi)
	http.HandleFunc("/api/grab/", grabApi)
	http.HandleFunc("/api/public/", publicDimensionsReq)

	http.Handle("/files", http.StripPrefix("/files", tusdHandler))
	http.Handle("/files/", http.StripPrefix("/files/", tusdHandler))

	fmt.Println("http://localhost:" + PORT)

	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			panic(err)
		}
	}()

	server := &http.Server{
		Addr:    ":" + PORT,
		Handler: nil, // or use your own mux/router
	}

	// Handle OS interrupt (Ctrl+C)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			logger.Fatal().Err(err).Msgf("failed to open http on port=%s", PORT)
		}
	}()

	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Err(err).Msg("Graceful shutdown failed")
	} else {
		logger.Info().Msg("Server gracefully stopped")
	}
}
