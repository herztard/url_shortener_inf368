package main

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"sync"
)

var (
	store = make(map[string]string)
	mu    sync.RWMutex
)

func generateKey(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		b[i] = letters[idx.Int64()]
	}
	return string(b), nil
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	u, err := url.ParseRequestURI(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	var key string
	for {
		key, err = generateKey(6)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		mu.RLock()
		_, exists := store[key]
		mu.RUnlock()
		if !exists {
			break
		}
	}

	mu.Lock()
	store[key] = req.URL
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"key": key,
		"url": req.URL,
	})
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[1:]
	mu.RLock()
	original, ok := store[key]
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, original, http.StatusFound)
}

func main() {
	http.HandleFunc("/shorten", shortenHandler)
	http.HandleFunc("/", redirectHandler)

	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
