package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	mock "qbase/synthos/heelix_ws/mock"
	"qbase/synthos/synthos_core/json"
	server "qbase/synthos/synthos_svr"
	"strings"
	"testing"
	"time"
)

func TestGetSystemInfo(t *testing.T) {
	mgrCfg := server.EntityManagerConfig{
		TimeRanges:    []time.Duration{6 * time.Hour, 1 * time.Hour},
		ContentSource: mock.NewMockContentSource(),
	}
	mgr := server.NewEntityManager(mgrCfg)
	appCfg := AppConfig{
		TimeRanges: []time.Duration{1 * time.Hour, 8 * time.Hour, 12 * time.Hour},
	}

	systemInfoHandler := GetSystemInfo(mgr, appCfg)

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/api/system_info", nil)
	systemInfoHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	// Spot check: just make sure the parent JSON keys are present.
	response := json.ParseBytes(w.Body.Bytes())
	assert.Equal(t, "[1 8 12]", response.Get("Runtime").Get("TimeRangesInHours").AsString())
	assert.True(t, response.Get("Deployment").Exists())
	assert.True(t, response.Get("Runtime").Exists())
	assert.True(t, response.Get("Runtime").Get("ItemCounts").Exists())
	assert.True(t, response.Get("Runtime").Get("OldestContent").Exists())
	assert.True(t, response.Get("Runtime").Get("NewestContent").Exists())
}

func TestFetchEntityInfo_badHttpPath(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/api/person/xxxx", nil)
	userId := 123
	handler := FetchEntityInfo(server.PersonEntity, nil)
	handler(w, r, userId)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFetchEntityInfo_badEntityId(t *testing.T) {
	mockEntityAnnotator := &mock.MockEntityAnnotator{}

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/api/person/-5", nil)
	userId := 123
	handler := FetchEntityInfo(server.PersonEntity, mockEntityAnnotator)
	handler(w, r, userId)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFetchEntityInfo_noInfo(t *testing.T) {
	mockEntityAnnotator := &mock.MockEntityAnnotator{SimulateNoInfo: true}

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/api/person/123", nil)
	userId := 123
	handler := FetchEntityInfo(server.PersonEntity, mockEntityAnnotator)
	handler(w, r, userId)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHealthCheck(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/api/any_ol_endpoint", nil)
	healthCheckHandler := HealthCheck()
	healthCheckHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	jsonResponse := json.ParseBytes(w.Body.Bytes())
	assert.Equal(t, "true", jsonResponse.Get("ok").AsString())
}

func TestDeleteAccessToken(t *testing.T) {
	// Assign an access token to the user.
	accessToken := "FAKE_TOKEN_FOR_THIS_TEST"
	userDb := NewUserDb()
	userDb.AddUser("john@example.com", "blah-12345678")
	user, _ := userDb.GetUserByEmail("john@example.com")
	userDb.SetAccessToken(user.Id, accessToken)

	mockWriter := httptest.NewRecorder()
	mockRequest, _ := http.NewRequest("GET", fmt.Sprintf("/dummy/path?access_token=%v", accessToken), nil)

	logoutHandler := Logout(userDb)
	logoutHandler(mockWriter, mockRequest, user.Id)

	// Verify handler returned HTTP 200 response and that the access token
	// was deleted for this user.
	assert.Equal(t, http.StatusOK, mockWriter.Code)
	_, wasUserFound := userDb.GetUserByAccessToken(accessToken)
	assert.False(t, wasUserFound)
}

func TestAcceptLicenseTerms(t *testing.T) {
	userEmail := "john@example.com"
	userDb := NewUserDb()
	user, err := userDb.AddUser(userEmail, "blah-12345678")
	assert.Nil(t, err)
	handler := AcceptLicenseTerms(userDb)
	mockWriter := httptest.NewRecorder()
	handler(mockWriter, nil, user.Id)

	assert.Equal(t, http.StatusOK, mockWriter.Code)
	user, _ = userDb.GetUserByEmail("john@example.com")
	assert.True(t, user.TermsAccepted)
	assert.Equal(t, http.StatusOK, mockWriter.Code)
}

func TestFindEntities(t *testing.T) {
	mockEntitySearch := &MockEntitySearch{}

	handler := FindEntities(mockEntitySearch)
	mockWriter := httptest.NewRecorder()
	mockRequest, _ := http.NewRequest("GET", "/api/search/alpha", nil)
	fakeUserId := -1 // not used for this test, but method signature requires it
	handler(mockWriter, mockRequest, fakeUserId)

	assert.Equal(t, http.StatusOK, mockWriter.Code)

	assert.Equal(t, "alpha", mockEntitySearch.searchString)

	// The response should contain exactly 1 search result for each entity type given the
	// seach string "alpha".
	response := json.ParseBytes(mockWriter.Body.Bytes())
	assert.Equal(t, 1, len(response.Get("Person").AsList()))
	assert.Equal(t, 1, len(response.Get("Org").AsList()))
	assert.Equal(t, 1, len(response.Get("Place").AsList()))
}

// Verify that submitting a search string that's empty or only contains
// whitespace returns an empty resultset and does not actually call Find() on
// the target EntitySearch component.
func TestFindEntities_emptySearchString(t *testing.T) {
	mockEntitySearch := &MockEntitySearch{}

	handler := FindEntities(mockEntitySearch)
	mockWriter := httptest.NewRecorder()
	mockRequest, _ := http.NewRequest("GET", "/api/search/ ", nil)
	fakeUserId := -1 // not used for this test, but method signature requires it
	handler(mockWriter, mockRequest, fakeUserId)

	assert.Equal(t, http.StatusOK, mockWriter.Code)
	assert.False(t, mockEntitySearch.wasFindInvoked)

	response := json.ParseBytes(mockWriter.Body.Bytes())
	assert.Equal(t, 0, len(response.Get("Person").AsList()))
	assert.Equal(t, 0, len(response.Get("Org").AsList()))
	assert.Equal(t, 0, len(response.Get("Place").AsList()))
}

func TestGetAllEntityInfo_noFiltersApplied(t *testing.T) {
	config := server.EntityManagerConfig{
		ContentSource: mock.NewMockContentSource(),
		TimeRanges:    []time.Duration{1 * time.Hour},
	}

	entityMgr := server.NewEntityManager(config)

	// Request the "all entity info" stats with no co-occurrence filter applied
	// (indicated by an empty post body).  In this case, we expect the call to
	// return the globally-computed entity statistics
	// (i.e. entityMgr.ContentBuffer.GetLatestEntityStats()).
	w := httptest.NewRecorder()
	postBody := strings.NewReader("")
	r, _ := http.NewRequest("GET", "/api/some/path", postBody)
	userId := 123
	handler := GetAllEntityInfo(entityMgr)
	handler(w, r, userId)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAddNewUser(t *testing.T) {
	// Here's the handler we're going to be testing
	userDb := NewUserDb()
	addNewUserHandler := AddNewUser(userDb)

	postBody := "{\"Email\": \"joe@example.com\", \"Password\": \"pass123\"}"

	request, _ := http.NewRequest("POST", "/api/users", strings.NewReader(postBody))
	mockWriter := httptest.NewRecorder()
	addNewUserHandler(mockWriter, request)
	assert.Equal(t, http.StatusOK, mockWriter.Code)
	_, userExists := userDb.GetUserByEmail("joe@example.com")
	assert.True(t, userExists)

	// 2nd attempt should fail, since we're trying to add the same user twice.
	request, _ = http.NewRequest("POST", "/api/users", strings.NewReader(postBody))
	mockWriter = httptest.NewRecorder()
	addNewUserHandler(mockWriter, request)
	assert.Equal(t, http.StatusInternalServerError, mockWriter.Code)
}

func TestAddNewUser_malformedPost(t *testing.T) {
	// Here's the handler we're going to be testing
	userDb := NewUserDb()
	addNewUserHandler := AddNewUser(userDb)

	malformedPostBody := "THIS IS NOT VALID JSON!!!"

	r, _ := http.NewRequest("POST", "/api/users", strings.NewReader(malformedPostBody))
	w := httptest.NewRecorder()
	addNewUserHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPutOrDeleteWatchList(t *testing.T) {
	userDb := NewUserDb()
	userDb.AddUser("john@example.com", "blah-12345678")
	user, _ := userDb.GetUserByEmail("john@example.com")

	// Test setup: Add a WatchList to the test user and verify the watchlist exists.
	watchList, _ := userDb.SaveWatchList(user.Id, makeWatchList("WatchList_1"))
	watchLists, _ := userDb.GetWatchLists(user.Id)
	assert.Equal(t, 1, len(watchLists))

	// Here's the handler we're going to be testing
	handler := PutOrDeleteWatchList(userDb)

	// Update the title and description of the existing watchlist
	postBody := "{\"Title\": \"WatchList_1A\", \"Description\": \"Updated description\"}"
	request, _ := http.NewRequest("PUT", fmt.Sprintf("/api/watchlists/%v", watchList.Id), strings.NewReader(postBody))
	mockWriter := httptest.NewRecorder()
	handler(mockWriter, request, user.Id)
	// Verify http status ok
	assert.Equal(t, http.StatusOK, mockWriter.Code)
	// Verify # of watchlists didn't change (shouldn't since we're just updating an item)
	watchLists, _ = userDb.GetWatchLists(user.Id)
	assert.Equal(t, 1, len(watchLists))
	// Verify updated watchlist has correct state.
	updatedWatchList := watchLists[0]
	assert.Equal(t, watchList.Id, updatedWatchList.Id)
	assert.Equal(t, "WatchList_1A", updatedWatchList.Title)
	assert.Equal(t, "Updated description", updatedWatchList.Description)

	// Delete the watchList we just created and updated
	request, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/watchlists/%v", watchList.Id), nil)
	mockWriter = httptest.NewRecorder()
	handler(mockWriter, request, user.Id)
	assert.Equal(t, http.StatusOK, mockWriter.Code)

	// Verify the watchlist was indeed deleted.
	watchLists, _ = userDb.GetWatchLists(user.Id)
	assert.Equal(t, 0, len(watchLists))
}

func TestPutOrDeleteWatchList_errorCases(t *testing.T) {
	userDb := NewUserDb()
	userDb.AddUser("john@example.com", "blah-12345678")
	user, _ := userDb.GetUserByEmail("john@example.com")

	// Here's the handler we're going to be testing
	handler := PutOrDeleteWatchList(userDb)

	// A malformed request path should result in an HTTP 400 error response.
	// In this case "UNPARSEABLE_ID" obviously cannot be parsed into an integer,
	// and so an HTTP 400 BAD REQUEST response is expected.
	request, _ := http.NewRequest("DELETE", "/api/watchlists/UNPARSEABLE_ID", nil)
	mockWriter := httptest.NewRecorder()
	handler(mockWriter, request, user.Id)
	assert.Equal(t, http.StatusBadRequest, mockWriter.Code)

	request, _ = http.NewRequest("DELETE", "/api/watchlists/123", nil)
	mockWriter = httptest.NewRecorder()
	badUserId := -9999
	handler(mockWriter, request, badUserId)
	assert.Equal(t, http.StatusInternalServerError, mockWriter.Code)
}

func TestCreateUsageReport(t *testing.T) {
	userDb := NewUserDb()
	userDb.AddUser("joe1@example.com", "blah-12345678")
	userDb.AddUser("joe2@example.com", "blah-12345678")

	handler := CreateUsageReport(userDb)
	request, _ := http.NewRequest("GET", "/api/memstats", nil)
	mockWriter := httptest.NewRecorder()
	handler(mockWriter, request)
	assert.Equal(t, http.StatusOK, mockWriter.Code)

	expectedResponse := "" +
		"email, last_login, terms_accepted, watchlists\n" +
		"joe1@example.com, NEVER, false, 0\n" +
		"joe2@example.com, NEVER, false, 0\n"

	assert.Equal(t, expectedResponse, mockWriter.Body.String())
}

func TestGetMemStats(t *testing.T) {
	handler := GetMemStats()
	request, _ := http.NewRequest("GET", "/api/memstats", nil)
	mockWriter := httptest.NewRecorder()
	handler(mockWriter, request)

	assert.Equal(t, http.StatusOK, mockWriter.Code)

	jsonResponse := json.ParseBytes(mockWriter.Body.Bytes())
	assert.True(t, jsonResponse.Get("General").Exists())
	assert.True(t, jsonResponse.Get("Heap").Exists())
}

func TestParseObjectIdFromPath(t *testing.T) {
	// Nominal case: object ID is 123
	id, err := parseObjectIdFromPath("/foo/bar/123")
	assert.Nil(t, err)
	assert.Equal(t, 123, id)

	// Error case: object ID "xxx" not paresable as an int
	_, err = parseObjectIdFromPath("/foo/bar/xxx")
	assert.NotNil(t, err)

	// Error case: No object ID at all!
	_, err = parseObjectIdFromPath("/")
	assert.NotNil(t, err)
}

func TestParseWatchList_invalidJSON(t *testing.T) {
	badJson := []byte("this is not valid json!")
	_, err := parseWatchList(badJson)
	assert.NotNil(t, err)
}

func TestParseWatchList_invalidWatchList(t *testing.T) {
	invalidWatchListJson := []byte("{}") // invalid since not Title was specified
	_, err := parseWatchList(invalidWatchListJson)
	assert.NotNil(t, err)
}

func TestParseEntityStr(t *testing.T) {
	// success case
	entityType, entityId, err := parseEntityStr("Person:123")
	assert.Nil(t, err)
	assert.Equal(t, "Person", entityType)
	assert.Equal(t, 123, entityId)

	// error case: unparseable integer id portion of entity id
	_, _, err = parseEntityStr("Person:NOT_A_NUMBER")
	assert.NotNil(t, err)

	// error case: bad format (missing the ":" separator)
	_, _, err = parseEntityStr("blahblahblah")
	assert.NotNil(t, err)
}

func TestSendJsonResponse(t *testing.T) {
	mockWriter := httptest.NewRecorder()
	sendJsonResponse([]string{"a", "b"}, mockWriter)
	assert.Equal(t, http.StatusOK, mockWriter.Code)

	// This call should fail since functions cannot be marshalled as JSON.
	mockWriter = httptest.NewRecorder()
	f := func() { return }
	sendJsonResponse(f, mockWriter)
	assert.Equal(t, http.StatusInternalServerError, mockWriter.Code)
}

type MockEntitySearch struct {
	searchString   string
	wasFindInvoked bool
}

func (me *MockEntitySearch) Find(searchStr string) map[server.EntityType][]server.DisplayEntity {
	me.wasFindInvoked = true
	me.searchString = searchStr
	return map[server.EntityType][]server.DisplayEntity{
		server.PersonEntity: []server.DisplayEntity{server.DisplayEntity{Id: 1, Name: "one"}},
		server.OrgEntity:    []server.DisplayEntity{server.DisplayEntity{Id: 2, Name: "two"}},
		server.PlaceEntity:  []server.DisplayEntity{server.DisplayEntity{Id: 3, Name: "three"}},
	}
}

func NewFakeContentDAO() *server.ContentDAO {
	contentDAO := server.NewContentDAO()
	contentDAO.PersonDAO = &FakeEntityDAO{}
	contentDAO.OrgDAO = &FakeEntityDAO{}
	contentDAO.PlaceDAO = &FakeEntityDAO{}
	return contentDAO
}

type FakeEntityDAO struct {
}

func (dao *FakeEntityDAO) GetLabel(entityId int) string {
	return fmt.Sprintf("fake_entity_%v", entityId)
}
func (dao *FakeEntityDAO) Load(filePath string) error {
	panic("Not implemented!")
}
func (dao *FakeEntityDAO) Save(filePath string) error {
	panic("Not implemented!")
}
func (dao *FakeEntityDAO) Size() int {
	return 1
}
func (dao *FakeEntityDAO) Update(entities []server.Entity) {
	panic("Not implemented!")
}
