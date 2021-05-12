package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/gorilla/mux"
)

// Record represents key-value pair
type Record struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
	w.Write(data)
}

// StoreHandler stores given key value pair in DB
func StoreHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var rec Record
	err := json.NewDecoder(r.Body).Decode(&rec)
	if err != nil {
		msg := "unable to marshal server settings"
		handleError(w, r, msg, err)
		return
	}

	// create hash value for given key
	h := sha1.New()
	h.Write([]byte(rec.Key))
	rec.Value = hex.EncodeToString(h.Sum(nil))

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