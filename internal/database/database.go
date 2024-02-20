package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	path      string
	lastChirp int
	lastUser  int
	mux       *sync.RWMutex
}

type DBStructure struct {
	Chirps        map[int]Chirp             `json:"chirps"`
	Users         map[int]AuthenticatedUser `json:"users"`
	RevokedTokens map[string]RevokedToken   `json:"revoked_tokens"`
}

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type User struct {
	Email string `json:"email"`
	Id    int    `json:"id"`
}

type AuthenticatedUser struct {
	Id       int    `json:"id"`
	Email    string `json:"email"`
	Password []byte `json:"password"`
}

type RevokedToken struct {
	Value string `json:"value"`
	Time  string `json:"time"`
}

type EditingUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

const ACCESS_ISSUER = "chirpy-access"
const REFRESH_ISSUER = "chirpy-refresh"

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	database := DB{
		path: path,
		mux:  &sync.RWMutex{},
	}

	_, err := os.ReadFile(path)
	if err != nil {
		err = database.ensureDB(path)
	}

	return &database, err
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	currentStructure, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	nextId := db.lastChirp + 1

	newChirp := Chirp{
		Id:   nextId,
		Body: body,
	}

	currentStructure.Chirps[nextId] = newChirp
	err = db.writeDB(currentStructure)
	if err != nil {
		return Chirp{}, err
	}

	db.lastChirp = nextId
	return newChirp, nil
}

// CreateUser creates a new chirp User and saves it to disk
func (db *DB) CreateUser(email string, password string) (User, error) {
	currentStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	nextId := db.lastUser + 1
	hashword, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	if err != nil {
		return User{}, err
	}

	newUser := AuthenticatedUser{
		Id:       nextId,
		Email:    email,
		Password: hashword,
	}

	currentStructure.Users[nextId] = newUser
	err = db.writeDB(currentStructure)
	if err != nil {
		return User{}, err
	}

	db.lastUser = nextId

	userResponse := User{
		Id:    newUser.Id,
		Email: newUser.Email,
	}

	return userResponse, nil
}

func (db *DB) EditUser(id int, newUserData EditingUser) (AuthenticatedUser, error) {
	currentDB, err := db.loadDB()

	if err != nil {
		return AuthenticatedUser{}, err
	}

	databaseUser := currentDB.Users[id]
	if newUserData.Password != "" {
		hashword, err := bcrypt.GenerateFromPassword([]byte(newUserData.Password), 0)

		if err != nil {
			return AuthenticatedUser{}, err
		}

		databaseUser.Password = hashword
	}
	if newUserData.Email != "" {
		databaseUser.Email = newUserData.Email
	}

	currentDB.Users[id] = databaseUser

	err = db.writeDB(currentDB)

	if err != nil {
		return AuthenticatedUser{}, err
	}

	return databaseUser, nil
}

func (db *DB) GetUserByEmail(email string) (AuthenticatedUser, bool, error) {
	currentDB, err := db.loadDB()
	if err != nil {
		return AuthenticatedUser{}, false, err
	}

	matchingUser := AuthenticatedUser{}

	for _, user := range currentDB.Users {
		if user.Email == email {
			matchingUser = user
			break
		}
	}

	if matchingUser.Id == 0 {
		return AuthenticatedUser{}, false, nil
	}

	return matchingUser, true, nil
}

type AuthUserResponse struct {
	Id           int
	Token        string
	RefreshToken string
}

func createJWT(expiration string, secret string, id string, keyType string) (string, error) {
	expirationDuration, _ := time.ParseDuration(expiration)

	claims := &jwt.RegisteredClaims{
		Issuer:    keyType,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expirationDuration)),
		Subject:   id,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedString, err := token.SignedString([]byte(secret))

	if err != nil {
		return "", err
	}

	return signedString, nil
}

// AuthenticateUser checks to see if the email and password match the one on disk
func (db *DB) AuthenticateUser(email string, password string, secret string) (bool, AuthUserResponse, error) {
	userResponse := AuthUserResponse{
		Id:    0,
		Token: "",
	}
	matchingUser, exists, err := db.GetUserByEmail(email)
	if err != nil || !exists {
		return false, userResponse, errors.New("User does not exist")
	}

	validationError := bcrypt.CompareHashAndPassword(matchingUser.Password, []byte(password))
	if validationError != nil {
		return false, userResponse, nil
	}

	stringifiedId := fmt.Sprint(matchingUser.Id)

	accessToken, err := createJWT("1h", secret, stringifiedId, ACCESS_ISSUER)
	if err != nil {
		return false, userResponse, err
	}

	refreshToken, err := createJWT("1440h", secret, stringifiedId, REFRESH_ISSUER)
	if err != nil {
		return false, userResponse, err
	}

	return true, AuthUserResponse{Id: matchingUser.Id, Token: accessToken, RefreshToken: refreshToken}, nil
}

func (db *DB) VerifyAccessToken(jwtToken string, secret string) (int, error) {
	token, err := jwt.ParseWithClaims(
		jwtToken,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

	if err != nil {
		return -1, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return -1, errors.New("Couldn't parse claims")
	}

	if claims.Issuer != ACCESS_ISSUER {
		return -1, errors.New("Token is not an access token")
	}

	if claims.ExpiresAt.UTC().Unix() < time.Now().UTC().Unix() {
		return -1, errors.New("JWT has expired")
	}

	subject, err := claims.GetSubject()
	if err != nil {
		return -1, err
	}

	userId, err := strconv.Atoi(subject)
	if err != nil {
		return -1, err
	}

	return userId, nil
}

func (db *DB) VerifyRefreshToken(jwtToken string, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(jwtToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return "", errors.New("Couldn't parse claims")
	}

	if claims.Issuer != REFRESH_ISSUER {
		return "", errors.New("Token is not a refresh token")
	}

	if claims.ExpiresAt.UTC().Unix() < time.Now().UTC().Unix() {
		return "", errors.New("JWT has expired")
	}

	currentDB, err := db.loadDB()
	if err != nil {
		return "", errors.New("Something went wrong")
	}

	_, isRevoked := currentDB.RevokedTokens[jwtToken]
	if isRevoked {
		return "", errors.New("Token is revoked")
	}

	subject, err := claims.GetSubject()
	if err != nil {
		return "", err
	}

	newAccessToken, err := createJWT("1h", secret, subject, ACCESS_ISSUER)

	if err != nil {
		return "", err
	}

	return newAccessToken, nil
}

func (db *DB) RevokeRefreshToken(jwtToken string) error {
	currentDB, err := db.loadDB()
	if err != nil {
		return err
	}

	currentDB.RevokedTokens[jwtToken] = RevokedToken{
		Value: jwtToken,
		Time:  time.Now().UTC().Format("StampMilli"),
	}

	err = db.writeDB(currentDB)
	if err != nil {
		return err
	}

	return nil
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	currentStructure, err := db.loadDB()
	if err != nil {
		return []Chirp{}, err
	}

	chirpSlice := []Chirp{}

	for _, chirp := range currentStructure.Chirps {
		chirpSlice = append(chirpSlice, chirp)
	}

	return chirpSlice, nil
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB(path string) error {
	dbContents := DBStructure{
		Chirps:        map[int]Chirp{},
		Users:         map[int]AuthenticatedUser{},
		RevokedTokens: map[string]RevokedToken{},
	}
	dat, err := json.Marshal(dbContents)
	if err != nil {
		return err
	}

	writeErr := os.WriteFile(path, dat, 0666)
	if writeErr != nil {
		return writeErr
	}

	return nil
}

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
