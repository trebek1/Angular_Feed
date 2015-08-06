package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"qbase/synthos/synthos_core/unixtime"
	"qbase/synthos/synthos_svr/stats"
	"sync/atomic"
)

// Provides access to the user database (email addresses, credentials, etc.).
type UserDb struct {
	objectId int64 // atomically-incremented variable used for assigning new object IDs
	users    []User
}

// Creates a new UserDb instance.
func NewUserDb() *UserDb {
	return &UserDb{
		objectId: 0,
		users:    []User{},
	}
}

// Loads content from the specified data file into a new UserDb instance.
// To save user content to file, use UserDb.Save(filePath).
func LoadUserDb(filePath string) *UserDb {
	logger.Printf("Loading user data from %v", filePath)
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var users []User
	bufReader := bufio.NewReader(f)
	decoder := json.NewDecoder(bufReader)
	err = decoder.Decode(&users)
	if err != nil {
		panic(err)
	}

	largestId := 0
	for _, user := range users {
		largestId = stats.MaxInt(largestId, user.Id)
		for _, watchlist := range user.WatchLists {
			largestId = stats.MaxInt(largestId, watchlist.Id)
		}
	}

	userDb := NewUserDb()
	userDb.users = users
	userDb.objectId = int64(largestId + 1)
	logger.Printf("Setting userDb.objectId to %v", userDb.objectId)
	return userDb
}

// Iterates over each user in the database.
func (me *UserDb) ForEachUser(f func(user User)) {
	for _, user := range me.users {
		f(user)
	}
}

// Adds a new user to the database.
func (me *UserDb) AddUser(email string, pwd string) (User, error) {
	_, userExists := me.GetUserByEmail(email)
	if userExists {
		return User{}, errors.New(fmt.Sprintf("User '%v' already exists.", email))
	}

	newUser := User{
		Id:           me.nextObjectId(),
		Email:        email,
		PasswordHash: hashPassword(pwd),
	}

	me.users = append(me.users, newUser)
	return newUser, nil
}

// Looks up a user by their email address.  Returns nil if user doesn't exist.
func (me *UserDb) GetUserById(id int) (User, bool) {
	user := me.findUserBy(func(u *User) bool {
		return u.Id == id
	})

	return returnUserIfExists(user)
}

// Looks up a user by their email address.  Returns nil if user doesn't exist.
func (me *UserDb) GetUserByEmail(email string) (User, bool) {
	user := me.findUserBy(func(u *User) bool {
		return u.Email == email
	})

	return returnUserIfExists(user)
}

// Looks up a user by their access token.  Returns nil if user doesn't exist.
func (me *UserDb) GetUserByAccessToken(accessToken string) (User, bool) {
	user := me.findUserBy(func(u *User) bool {
		return u.AccessToken == accessToken
	})

	return returnUserIfExists(user)
}

// Assigns the specified access token to the user having the specified email.
func (me *UserDb) SetAccessToken(userId int, accessToken string) {
	for i, _ := range me.users {
		user := &me.users[i]
		if user.Id == userId {
			user.AccessToken = accessToken
		}
	}
}

// Sets the 'LastLogin' timestamp to the current system time.
func (me *UserDb) SetLastLoginToNow(userId int) {
	for i, _ := range me.users {
		user := &me.users[i]
		if user.Id == userId {
			user.LastLogin = unixtime.Now()
		}
	}
}

// Sets the 'TermsAccepted' flag to true.
func (me *UserDb) SetTermsAccepted(userId int) {
	for i, _ := range me.users {
		user := &me.users[i]
		if user.Id == userId {
			user.TermsAccepted = true
		}
	}
}

// Returns the watchlists owned by the specified user.
func (me *UserDb) GetWatchLists(userId int) (watchLists []WatchList, err error) {
	user, wasUserFound := me.GetUserById(userId)
	if !wasUserFound {
		return nil, errors.New(fmt.Sprintf("User:%v doesn't exist", userId))
	}

	if user.WatchLists == nil {
		return []WatchList{}, nil
	} else {
		return user.WatchLists, nil
	}
}

// Adds or updates the specified watchlist to a user's existing watchlists.
// If the watchlist is added, assigns a unique ID to the WatchList object
// passed into this method.
func (me *UserDb) SaveWatchList(userId int, w WatchList) (WatchList, error) {
	// Find the user associated with userId
	var watchlistOwner *User
	for i := 0; i < len(me.users); i++ {
		user := &(me.users[i])
		if user.Id == userId {
			watchlistOwner = user
		}
	}

	if watchlistOwner == nil {
		return WatchList{}, errors.New(fmt.Sprintf("User:%v doesn't exist", userId))
	}

	if w.IsSaved() { // Update an existing WatchList
		wasWatchlistFound := false
		for i := 0; i < len(watchlistOwner.WatchLists); i++ {
			if watchlistOwner.WatchLists[i].Id == w.Id {
				watchlistOwner.WatchLists[i] = w
				wasWatchlistFound = true
				break
			}
		}
		if !wasWatchlistFound {
			return WatchList{}, errors.New(fmt.Sprintf("User:%v: WatchList:%v doesn't exist", userId, w.Id))
		}
	} else { // Insert an new WatchList
		w.Id = me.nextObjectId()
		watchlistOwner.WatchLists = append(watchlistOwner.WatchLists, w)
	}

	return w, nil
}

// Deletes the specified watchlist from the database.
func (me *UserDb) DeleteWatchList(userId int, watchListId int) error {
	removeWatchList := func(watchLists []WatchList, watchListId int) []WatchList {
		filteredWatchLists := make([]WatchList, 0, len(watchLists))
		for _, w := range watchLists {
			if w.Id != watchListId {
				filteredWatchLists = append(filteredWatchLists, w)
			}
		}
		return filteredWatchLists
	}

	for i := 0; i < len(me.users); i++ {
		user := &(me.users[i])
		if user.Id == userId {
			user.WatchLists = removeWatchList(user.WatchLists, watchListId)
			return nil
		}
	}

	return errors.New(fmt.Sprintf("User:%v doesn't exist", userId))
}

// Saves the content of this user db to the specified file.
func (me *UserDb) Save(filePath string) error {
	outputFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	b, err := json.MarshalIndent(me.users, "", "    ")
	if err != nil {
		return err
	}

	bufWriter := bufio.NewWriter(outputFile)
	_, err = bufWriter.Write(b)
	if err != nil {
		return err
	}
	bufWriter.Flush()

	return nil
}

// Specifies a filter that returns true only if the specified User is
// considered a match.
type UserFilter func(u *User) bool

// Generic function for finding a user in a list by one of the user's
// attribute values.  If the filter yields no results, nil is returned.
// Returns a reference to the User object that matches the filter criteria.
func (me *UserDb) findUserBy(matches UserFilter) *User {
	for i := 0; i < len(me.users); i++ {
		user := &me.users[i]
		if matches(user) {
			return user
		}
	}

	return nil
}

func returnUserIfExists(user *User) (User, bool) {
	if user != nil {
		return *user, true
	} else {
		return User{}, false
	}
}

// Returns the next unique object Id.
func (me *UserDb) nextObjectId() int {
	atomic.AddInt64(&me.objectId, 1)
	return int(me.objectId)
}
