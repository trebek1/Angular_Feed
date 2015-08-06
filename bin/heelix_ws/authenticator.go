package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"qbase/synthos/synthos_core/webapp"
	"regexp"
	"strings"
)

// Handles logic related to authenticating a user, issuing access tokens to users, and
// securing REST endpoints using the generated tokens.
type Authenticator struct {
	userDb *UserDb
}

// Creates a new Authenticator bound to the specified user database.
func NewAuthenticator(userDb *UserDb) *Authenticator {
	return &Authenticator{
		userDb: userDb,
	}
}

// Http wrapper that authorizes a request based on a valid access token.  The access token
// is presumed to have been provided from a successful AuthenticateUser() call.
func (me *Authenticator) AuthorizeUser(h webapp.UserHttpHandler) webapp.HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		// queryParams := r.URL.Query()
		// paramValues := queryParams["access_token"]
		// if paramValues == nil {
		// 	http.Error(w, "access_token missing from request", http.StatusBadRequest)
		// 	return
		// }
		//
		// accessToken := paramValues[0]
		// user, wasUserFound := me.userDb.GetUserByAccessToken(accessToken)
		// if !wasUserFound {
		// 	http.Error(w, fmt.Sprintf("Invalid access token: %v", accessToken), http.StatusUnauthorized)
		// 	return
		// }
		
		logger.Printf("WARN: >>>>>>> AUTHORIZATION DISABLED!!! <<<<<<<<<")
		fakeUserId := 1
		h(w, r, fakeUserId)
	}
}

// Authenticates a user using HTTP Basic Authentication.
func (me *Authenticator) AuthenticateUser() webapp.HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// HTTP Basic Auth requires an 'Authorization' header that contains the
		// base64-encoded username and password.
		authHeader := r.Header["Authorization"]
		if authHeader == nil {
			http.Error(w, "'Authorization' header missing", http.StatusBadRequest)
			return
		}

		// The header should be of the form "Basic <base64-encoded credentials>",
		authHeaderParts := regexp.MustCompile(" +").Split(authHeader[0], -1)
		if len(authHeaderParts) != 2 || authHeaderParts[0] != "Basic" {
			http.Error(w, "Malformed Authorization header", http.StatusBadRequest)
			return
		}

		// Decode the base64-encoded credentials into the constituent username and
		// cleartext password. The credentials string should have the form
		// "<username>:<password>".
		payload, _ := base64.StdEncoding.DecodeString(authHeaderParts[1])
		credentials := strings.SplitN(string(payload), ":", 2)
		if len(credentials) != 2 {
			http.Error(w, "Malformed base64 payload", http.StatusBadRequest)
			return
		}

		// Validate user credentials
		email, password := credentials[0], credentials[1]
		logger.Printf("Authenticating '%v'", email)
		user, userExists := me.userDb.GetUserByEmail(email)
		if !userExists || !isValidPassword(password, user) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		accessToken := generateAccessToken()
		me.userDb.SetAccessToken(user.Id, accessToken)
		me.userDb.SetLastLoginToNow(user.Id)

		logger.Printf("'%v' successfully authenticated.", email)
		response := map[string]interface{}{
			"access_token":   accessToken,
			"terms_accepted": user.TermsAccepted,
		}
		sendJsonResponse(response, w)
	}
}

// Generates a cryptographically secure 32-byte random hex value that is returned
// to the client upon successful authentiation.
func generateAccessToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Returns true if the provided cleartext password is valid for the specified user.
func isValidPassword(cleartextPassword string, user User) bool {
	return hashPassword(cleartextPassword) == user.PasswordHash
}

// Returns a hash of the cleartext password, which can be stored in the user db.
func hashPassword(cleartextPassword string) string {
	hash := sha256.New()
	hash.Write([]byte(cleartextPassword))
	return hex.EncodeToString(hash.Sum(nil))
}
