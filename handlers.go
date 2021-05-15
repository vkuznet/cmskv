package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"log"
	"net/http"
	"strings"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
)

// Record represents key-value pair
type Record struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// HTTPRecord represents key-value pair
type HTTPRecord struct {
	Sha string `json:"sha"`
	Record
}

// helper function to handle http server errors
func handleError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	log.Println(msg, err)
	rec := make(map[string]string)
	rec["message"] = msg
	rec["error"] = fmt.Sprintf("%v", err)
	data, e := json.Marshal(rec)
	if e != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	w.Write(data)
}

// StoreHandler stores given key value pair in DB
func StoreHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var rec HTTPRecord
	err := json.NewDecoder(r.Body).Decode(&rec)
	if err != nil {
		msg := "unable to marshal server settings"
		handleError(w, r, msg, err)
		return
	}

	// create hash value for given key
	var h hash.Hash
	sha := strings.ToLower(Config.SHA)
	if sha == "sha256" || rec.Sha == "sha256" {
		h = sha256.New()
	} else if sha == "sha512" || rec.Sha == "sha512" {
		h = sha512.New()
	} else {
		h = sha1.New()
	}
	h.Write([]byte(rec.Key))
	// if record value is not provided we'll create a hash for it
	// this will allow to anonimise the data
	if rec.Value == "" {
		rec.Value = hex.EncodeToString(h.Sum(nil))
	}

	// commit new key-value records into our store
	txn := DB.NewTransaction(true)
	defer txn.Discard()
	err = txn.Set([]byte(rec.Key), []byte(rec.Value))
	if err != nil {
		msg := "unable to set new key-value pair"
		handleError(w, r, msg, err)
	}
	err = txn.Commit()
	if err != nil {
		msg := "unable to commit new key-value pair"
		handleError(w, r, msg, err)
	}
	if Config.Verbose > 0 {
		log.Printf("record key=%s value=%s", rec.Key, rec.Value)
	}

	txn = DB.NewTransaction(true)
	defer txn.Discard()
	err = txn.Set([]byte(rec.Value), []byte(rec.Key))
	if err != nil {
		msg := "unable to set new key-value pair"
		handleError(w, r, msg, err)
	}
	err = txn.Commit()
	if err != nil {
		msg := "unable to commit new key-value pair"
		handleError(w, r, msg, err)
	}
	if Config.Verbose > 0 {
		log.Printf("record key=%s value=%s", rec.Value, rec.Key)
	}
}

// InfoHandler fetches key-value pair from DB
func InfoHandler(w http.ResponseWriter, r *http.Request) {
	rec := make(map[string]string)
	rec["server"] = Info()
	data, err := json.Marshal(rec)
	if err != nil {
		log.Fatalf("Fail to marshal records, %v", err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// FetchHandler fetches key-value pair from DB
func FetchHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	err := DB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(vars["key"]))
		if err != nil {
			return err
		}
		val, _ := item.ValueCopy(nil)
		rec := Record{Key: vars["key"], Value: string(val)}
		data, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		w.Write(data)
		return nil
	})
	if err != nil {
		msg := "unable to fetch key value"
		handleError(w, r, msg, err)
		return
	}
}

var (
	//go:embed static/index.html
	index string
)

// IndexHandler fetches key-value pair from DB
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(index))
}
