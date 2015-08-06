package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"qbase/synthos/synthos_core/unixtime"
	"testing"
)

func TestForEachUser(t *testing.T) {
	userDb := UserDb{
		users: []User{
			User{Id: 100, Email: "u100@example.com"},
			User{Id: 101, Email: "u101@example.com"},
		},
	}

	users := []User{}
	userDb.ForEachUser(func(u User) {
		users = append(users, u)
	})

	assert.Equal(t, 2, len(users))
}

func TestAddUser(t *testing.T) {
	userDb := NewUserDb()

	// Add a new user
	username := "joe@example.com"
	newUser, err := userDb.AddUser(username, "password-123456")
	assert.Nil(t, err)
	assert.Equal(t, newUser.Email, username)

	// Add another user with a duplicate email address -- should cause an error.
	_, err = userDb.AddUser(username, "some-other-password-%^&*")
	assert.NotNil(t, err)
}

func TestGetUserByEmail(t *testing.T) {
	userDb := UserDb{
		users: []User{
			User{Id: 100, Email: "u100@example.com"},
			User{Id: 101, Email: "u101@example.com"},
		},
	}

	// Look up a user who is known to be in the database
	user, userExists := userDb.GetUserByEmail("u101@example.com")
	assert.True(t, userExists)
	assert.Equal(t, user.Email, "u101@example.com")

	// Look up a user who we know is *not* in the database
	user, userExists = userDb.GetUserByEmail("bogus_user@example.com")
	assert.False(t, userExists)

	// empty string should never resolve to a user
	user, userExists = userDb.GetUserByEmail("")
	assert.False(t, userExists)
}

func TestGetUserByAccessToken(t *testing.T) {
	userDb := UserDb{
		users: []User{
			User{Id: 100, Email: "u100@example.com", AccessToken: "ABC"},
			User{Id: 101, Email: "u101@example.com", AccessToken: "DEF"},
		},
	}

	user, wasUserFound := userDb.GetUserByAccessToken("DEF")
	assert.True(t, wasUserFound)
	assert.Equal(t, user.Id, 101)

	bogusToken := "!!!NOT_A_VALID_TOKEN!!!"
	_, wasUserFound = userDb.GetUserByAccessToken(bogusToken)
	assert.False(t, wasUserFound)

	// empty string should never resolve to a user
	_, wasUserFound = userDb.GetUserByAccessToken(bogusToken)
	assert.False(t, wasUserFound)
}

func TestGetUserById(t *testing.T) {
	userDb := UserDb{
		users: []User{
			User{Id: 100, Email: "u100@example.com"},
			User{Id: 101, Email: "u101@example.com"},
		},
	}

	user, wasFound := userDb.GetUserById(100)
	assert.True(t, wasFound)
	assert.Equal(t, user.Email, "u100@example.com")

	nonexistentUserId := -999999999
	user, wasFound = userDb.GetUserById(nonexistentUserId)
	assert.False(t, wasFound)
}

func TestSetLastLoginToNow(t *testing.T) {
	userId := 100
	userEmail := "joe@example.com"
	userDb := UserDb{
		users: []User{
			User{Id: userId, Email: userEmail},
		},
	}

	userDb.SetLastLoginToNow(userId)
	user, userExists := userDb.GetUserByEmail(userEmail)
	assert.True(t, userExists)
	assert.Equal(t, unixtime.Now(), user.LastLogin)
}

func TestSetAccessToken(t *testing.T) {
	userId := 100
	userEmail := "joe@example.com"
	userDb := UserDb{
		users: []User{
			User{Id: userId, Email: userEmail, AccessToken: "T100"},
		},
	}

	// Assign the token and verify that it got set.
	userDb.SetAccessToken(userId, "T100")
	user, userExists := userDb.GetUserByEmail(userEmail)
	assert.True(t, userExists)
	assert.Equal(t, "T100", user.AccessToken)
}

func TestSetTermsAccepted(t *testing.T) {
	userId := 100
	userEmail := "joe@example.com"
	userDb := UserDb{
		users: []User{
			User{Id: userId, Email: userEmail},
		},
	}

	user, _ := userDb.GetUserByEmail(userEmail)
	assert.False(t, user.TermsAccepted)
	userDb.SetTermsAccepted(userId)
	user, _ = userDb.GetUserByEmail(userEmail)
	assert.True(t, user.TermsAccepted)
}

func TestAddAndGetWatchLists(t *testing.T) {
	userDb := createUserDbForTest()
	user, _ := userDb.GetUserByEmail("etakahashi@synthostech.com")

	// First, get the watchlists for a user that we know has none.
	watchLists, err := userDb.GetWatchLists(user.Id)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(watchLists))

	// Now add a watchlist to this user's watchlist.
	watchListTitle := "My WatchList"
	w := makeWatchList(watchListTitle)
	assert.True(t, w.Id == 0)
	w, err = userDb.SaveWatchList(user.Id, w)
	assert.Nil(t, err)
	assert.True(t, w.Id != 0)

	watchLists, err = userDb.GetWatchLists(user.Id)
	assert.Nil(t, err)

	// Make sure we received our watchlist that we just saved.
	assert.Equal(t, 1, len(watchLists))

	// Make sure an ID was assigned to the WatchList object that as re-loaded.
	assert.True(t, watchLists[0].Id > 0)

	// Make sure we got the right WatchList back
	assert.Equal(t, watchListTitle, watchLists[0].Title)
}

func TestGetWatchLists_nonexistentUser(t *testing.T) {
	userDb := createUserDbForTest()
	nonexistentUserId := -9999999
	_, err := userDb.GetWatchLists(nonexistentUserId)
	assert.NotNil(t, err)
}

func TestUpdateWatchList(t *testing.T) {
	userDb := UserDb{
		users: []User{
			User{
				Id:    100,
				Email: "u100@example.com",
				WatchLists: []WatchList{
					WatchList{Id: 1, Title: "WatchList 1"},
					WatchList{Id: 2, Title: "WatchList 2"},
				},
			},
		},
	}

	// Update WatchList:1
	watchListToSave := WatchList{Id: 1, Title: "WatchList 1-A"}
	_, err := userDb.SaveWatchList(100, watchListToSave)

	// Verify update took place
	assert.Nil(t, err)
	watchLists := userDb.users[0].WatchLists
	assert.Equal(t, WatchList{Id: 1, Title: "WatchList 1-A"}, watchLists[0])
	assert.Equal(t, WatchList{Id: 2, Title: "WatchList 2"}, watchLists[1])

	// Update a WatchList that doesn't exist
	_, err = userDb.SaveWatchList(100, WatchList{Id: 999999, Title: "Nonexistent Watchlist!"})
	assert.NotNil(t, err)
}

func TestGetWatchLists_noWatchListsSaved(t *testing.T) {
	userDb := createUserDbForTest()
	user, _ := userDb.GetUserByEmail("etakahashi@synthostech.com")
	watchLists, err := userDb.GetWatchLists(user.Id)
	assert.Nil(t, err)
	assert.NotNil(t, watchLists)
	assert.Equal(t, 0, len(watchLists))
}

func TestSaveWatchLists_nonexistentUser(t *testing.T) {
	userDb := createUserDbForTest()
	nonexistentUserId := -9999999
	_, err := userDb.SaveWatchList(nonexistentUserId, makeWatchList("Foo"))
	assert.NotNil(t, err)
}

func TestDeleteWatchList(t *testing.T) {
	userDb := createUserDbForTest()
	user, _ := userDb.GetUserByEmail("etakahashi@synthostech.com")

	watchList, _ := userDb.SaveWatchList(user.Id, makeWatchList("Foo"))

	// Verify this user has 1 watchlist (the one we just added above)
	savedWatchLists, _ := userDb.GetWatchLists(user.Id)
	assert.Equal(t, 1, len(savedWatchLists))

	// Pass in a bogus watchlist ID.  This should NOT generate an error,
	// but instead just result in a no-op.
	bogusWatchListId := -99999999
	err := userDb.DeleteWatchList(user.Id, bogusWatchListId)
	assert.Nil(t, err)

	// Now delete the watchlist and verify the user now has 0 watchlists.
	err = userDb.DeleteWatchList(user.Id, watchList.Id)
	assert.Nil(t, err)
	savedWatchLists, _ = userDb.GetWatchLists(user.Id)
	assert.Equal(t, 0, len(savedWatchLists))
}

func TestDeleteWatchList_nonexistentUser(t *testing.T) {
	userDb := createUserDbForTest()

	// sanity check -- make sure user actually doesn't exist
	nonexistentUserId := -99999999
	_, wasUserFound := userDb.GetUserById(nonexistentUserId)
	assert.False(t, wasUserFound)

	someWatchListId := -1111111
	err := userDb.DeleteWatchList(nonexistentUserId, someWatchListId)
	assert.NotNil(t, err)
}

func TestNextObjectId(t *testing.T) {
	userDb := NewUserDb()
	for i := 1; i < 1000; i++ {
		assert.Equal(t, i, userDb.nextObjectId())
	}
}

func TestSaveAndLoadUsers(t *testing.T) {
	id := 1
	nextId := func() int {
		id = id + 1
		return id
	}

	createFakeWatchList := func() WatchList {
		id := nextId()
		return WatchList{
			Id:          id,
			Title:       fmt.Sprintf("watchlist_%v", id),
			Description: fmt.Sprintf("Description for watchlist_%v", id),
		}
	}

	createFakeUser := func() User {
		id := nextId()
		return User{
			Id:           id,
			Email:        fmt.Sprintf("user_%v@example.com", id),
			WatchLists:   []WatchList{createFakeWatchList()},
			PasswordHash: "abcdefg-12345678",
		}
	}

	userDb := NewUserDb()
	userDb.users = []User{createFakeUser(), createFakeUser()}

	// Save the users to a file and then reload them.
	dataFile := "/tmp/TestSaveAndLoadUsers.json"
	userDb.Save(dataFile)
	userDb2 := LoadUserDb(dataFile)

	assert.Equal(t, userDb.users, userDb2.users)
	assert.Equal(t, int64(id+1), userDb2.objectId)
}

func TestFindUserBy_returnsReferenceAndNotCopy(t *testing.T) {
	userDb := UserDb{
		users: []User{
			User{Id: 100},
		},
	}

	// Pass in a matcher func that modifies the state of u
	userDb.findUserBy(func(u *User) bool {
		u.Id = 999999
		return false
	})

	// Verify that the user object modified within the findUserBy func
	// was a reference to the user in userDb.users.
	assert.Equal(t, 999999, userDb.users[0].Id)
}

//
// TEST HELPERS
//

func createUserDbForTest() *UserDb {
	userDb := NewUserDb()
	userDb.AddUser("etakahashi@synthostech.com", "cat-knuckle-sweater-59!")
	return userDb
}

// Create a WatchList populated with fake data.
func makeWatchList(title string) WatchList {
	desc := fmt.Sprintf("Description of '%v'", title)
	return WatchList{
		Title:       title,
		Description: desc,
	}
}
