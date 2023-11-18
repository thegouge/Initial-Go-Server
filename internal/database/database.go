package database

import (
	"encoding/json"
	"os"
	"sync"
)

type DB struct {
	path   string
	lastID int
	mux    *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	dbContents := DBStructure{
		Chirps: map[int]Chirp{},
	}
	dat, err := json.Marshal(dbContents)
	if err != nil {
		return &DB{}, err
	}

	writeErr := os.WriteFile(path, dat, 0666)
	if writeErr != nil {
		return &DB{}, writeErr
	}

	database := DB{
		path: path,
		mux:  &sync.RWMutex{},
	}

	return &database, err
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	currentStructure, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	nextId := db.lastID + 1

	newChirp := Chirp{
		Id:   nextId,
		Body: body,
	}

	currentStructure.Chirps[nextId] = newChirp
	err = db.writeDB(currentStructure)
	if err != nil {
		return Chirp{}, err
	}

	db.lastID = nextId
	return newChirp, nil
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	currentStructure, err := db.loadDB()
	if err != nil {
		return []Chirp{}, err
	}

	sortedChirps := []Chirp{}

	for _, chirp := range currentStructure.Chirps {
		sortedChirps = append(sortedChirps, chirp)
	}

	return sortedChirps, nil
}

// ensureDB creates a new database file if it doesn't exist
// func (db *DB) ensureDB() error

// loadDB reads the database file into memory
func (db *DB) loadDB() (DBStructure, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()

	rawData, err := os.ReadFile(db.path)
	if err != nil {
		return DBStructure{}, err
	}

	dbData := DBStructure{}
	err = json.Unmarshal(rawData, &dbData)

	if err != nil {
		return DBStructure{}, err
	}

	return dbData, nil
}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStructure DBStructure) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	binData, err := json.Marshal(dbStructure)
	if err != nil {
		return err
	}

	err = os.WriteFile(db.path, binData, 0666)
	if err != nil {
		return err
	}

	return nil
}
