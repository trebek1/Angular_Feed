package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	migrate "qbase/synthos/heelix_ws/datamigrate"
	"qbase/synthos/synthos_core/cache"
	"qbase/synthos/synthos_core/strutil"
	"qbase/synthos/synthos_core/webapp"
	server "qbase/synthos/synthos_svr"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Sets the 'TermsAccepted' attribute of the authenticated user to true.
func AcceptLicenseTerms(userDb *UserDb) webapp.UserHttpHandler {
	return func(w http.ResponseWriter, r *http.Request, userId int) {
		w.Header().Set("Content-Type", "application/json")

		user, _ := userDb.GetUserById(userId)
		if !user.TermsAccepted {
			logger.Printf("Adding default watchlists for User:%v ('%v')", userId, user.Email)
			watchLists := createDefaultWatchlists()
			for _, watchList := range watchLists {
				_, err := userDb.SaveWatchList(userId, watchList)
				if err != nil {
					panic(err)
				}
			}
		}

		userDb.SetTermsAccepted(userId)
	}
}

// Deletes the authenticated user's access token.  This is called by the client
// to log the authenticated user out.
func Logout(userDb *UserDb) webapp.UserHttpHandler {
	return func(w http.ResponseWriter, r *http.Request, userId int) {
		w.Header().Set("Content-Type", "application/json")

		userDb.SetAccessToken(userId, "")
	}
}

// Clients (most likely a load balancer) can ping this handler to verify the system is healthy.
func HealthCheck() webapp.HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := map[string]interface{}{
			"ok": true,
		}
		sendJsonResponse(response, w)
	}
}

// Returns configuration and runtime information about the deployed application.
func GetSystemInfo(mgr *server.EntityManager, cfg AppConfig) webapp.HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		contentBuffer := mgr.ContentBuffer()
		entityStats := contentBuffer.LatestEntityStats()

		// If no content received from the content source within the last minute
		// or so, we deem it "offline".
		timeSinceLatestContentReceived := time.Since(entityStats.NewestContent.Time())
		isContentSourceOnline := timeSinceLatestContentReceived <= (1 * time.Minute)

		timeRangesInHours := []int{}
		for _, duration := range cfg.TimeRanges {
			timeRangesInHours = append(timeRangesInHours, int(duration/time.Hour))
		}

		itemCounts := map[string]int{
			"Documents": contentBuffer.DocumentCount(),
			"Persons":   contentBuffer.PersonGraph.EntityCount(),
			"Orgs":      contentBuffer.OrgGraph.EntityCount(),
			"Places":    contentBuffer.PlaceGraph.EntityCount(),
		}

		runtimeInfo := map[string]interface{}{
			"Errors":              mgr.RecentErrors(),
			"ItemCounts":          itemCounts,
			"ContentSourceOnline": isContentSourceOnline,
			"TimeRangesInHours":   timeRangesInHours,
			"OldestContent":       entityStats.OldestContent.String(),
			"NewestContent":       entityStats.NewestContent.String(),
		}

		deploymentInfo := map[string]interface{}{
			"Version":       GIT_VERSION,
			"BuildDate":     BUILD_DATE,
			"MemDbEndpoint": cfg.MemDbConn,
		}

		systemIinfo := map[string]interface{}{
			"Runtime":    runtimeInfo,
			"Deployment": deploymentInfo,
		}

		sendJsonResponse(systemIinfo, w)
	}
}

// Returns information about a specific 'Person' entity (e.g. Barack Obama).
// The entity's ID is expected to be the last token in the slash-delimited
// URL path (e.g. /api/person/12345).
func FetchEntityInfo(entityType server.EntityType, entityAnnotator server.EntityAnnotator) webapp.UserHttpHandler {
	return func(w http.ResponseWriter, r *http.Request, userId int) {
		w.Header().Set("Content-Type", "application/json")

		entityId, err := parseObjectIdFromPath(r.URL.Path)
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not determine entity ID from '%v': %v", r.URL.Path, err), http.StatusBadRequest)
			return
		}

		entityInfo, err := entityAnnotator.FetchEntityInfo(entityType, entityId)

		if err != nil {
			http.Error(w, fmt.Sprintf("Error getting info for %v:%v: %v", entityType, entityId, err), http.StatusInternalServerError)
			return
		}

		foundEntityInfo := (entityInfo != nil)
		if !foundEntityInfo {
			http.Error(w, fmt.Sprintf("No info for entityType=%v, entityId=%v", entityType, entityId), http.StatusNotFound)
		}

		sendJsonResponse(entityInfo, w)
	}
}

// Add a new user.
func AddNewUser(userDb *UserDb) webapp.HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		getHttpRequestBody(w, r, func(postedData []byte) {
			type newUserInfo struct {
				Email    string
				Password string
			}

			var userInfo newUserInfo
			if err := json.Unmarshal(postedData, &userInfo); err != nil {
				http.Error(w, fmt.Sprintf("Error parsing userInfo: %v", err), http.StatusBadRequest)
				return
			}

			user, err := userDb.AddUser(userInfo.Email, userInfo.Password)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error creating new user: %v", err), http.StatusInternalServerError)
				return
			}

			sendJsonResponse(user, w)
		})
	}
}

func GetOrPostWatchLists(userDb *UserDb) webapp.UserHttpHandler {
	return func(w http.ResponseWriter, r *http.Request, userId int) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case "GET":
			watchLists, err := userDb.GetWatchLists(userId)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error getting watchlists for User:%v: %v", userId, err), http.StatusInternalServerError)
			} else {
				sendJsonResponse(watchLists, w)
			}
		case "POST":
			getHttpRequestBody(w, r, func(postBody []byte) {
				watchList, err := parseWatchList(postBody)
				if err != nil {
					http.Error(w, fmt.Sprintf("Error getting WatchList data from request for User:%v: %v", userId, err), http.StatusInternalServerError)
				} else {
					watchList, err = userDb.SaveWatchList(userId, watchList)
					if err != nil {
						http.Error(w, fmt.Sprintf("Error saving WatchList to datastore for User:%v: %v", userId, err), http.StatusInternalServerError)
					} else {
						sendJsonResponse(watchList, w)
					}
				}
			})
		default:
			http.Error(w, fmt.Sprintf("WatchList: User:%v; unsupported HTTP Verb '%v'", userId, r.Method), http.StatusMethodNotAllowed)
		}
	}
}

func PutOrDeleteWatchList(userDb *UserDb) webapp.UserHttpHandler {
	return func(w http.ResponseWriter, r *http.Request, userId int) {
		w.Header().Set("Content-Type", "application/json")

		watchListId, err := parseObjectIdFromPath(r.URL.Path)
		if err != nil {
			http.Error(w, fmt.Sprintf("Could not determine WatchList Id from '%v': %v", r.URL.Path, err), http.StatusBadRequest)
			return
		}

		switch r.Method {
		case "PUT":
			getHttpRequestBody(w, r, func(postBody []byte) {
				watchList, err := parseWatchList(postBody)
				if err != nil {
					http.Error(w, fmt.Sprintf("Error parsing WatchList data from request for User:%v: %v", userId, err), http.StatusInternalServerError)
				} else {
					watchList.Id = watchListId
					_, err = userDb.SaveWatchList(userId, watchList)
					if err != nil {
						http.Error(w, fmt.Sprintf("Error updating WatchList for User:%v: %v", userId, err), http.StatusInternalServerError)
					}
				}
			})
		case "DELETE":
			err = userDb.DeleteWatchList(userId, watchListId)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error deleting WatchList:%v for User:%v: %v", watchListId, userId, err), http.StatusInternalServerError)
			}
		default:
			http.Error(w, fmt.Sprintf("WatchList: User:%v; unsupported HTTP Verb '%v'", userId, r.Method), http.StatusMethodNotAllowed)
		}
	}
}

// Returns all entities in the EntityManager that match a given search term.
// Matching logic is currently just a substring match.
func FindEntities(entitySearch server.EntitySearch) webapp.UserHttpHandler {
	return func(w http.ResponseWriter, r *http.Request, userId int) {
		w.Header().Set("Content-Type", "application/json")

		searchString := strings.TrimSpace(strings.ToLower(strutil.LastToken(r.URL.Path, "/")))

		var response map[string][]server.DisplayEntity
		if !strutil.IsEmpty(searchString) {
			searchResults := entitySearch.Find(searchString)
			response = map[string][]server.DisplayEntity{
				"Person": searchResults[server.PersonEntity],
				"Org":    searchResults[server.OrgEntity],
				"Place":  searchResults[server.PlaceEntity],
			}
		} else {
			noEntities := []server.DisplayEntity{}
			response = map[string][]server.DisplayEntity{
				"Person": noEntities,
				"Org":    noEntities,
				"Place":  noEntities,
			}
		}

		sendJsonResponse(response, w)
	}
}

// PA-241/PA-198: Support for disjunctive querying.
func GetAllEntityInfo(mgr *server.EntityManager) webapp.UserHttpHandler {
	return func(w http.ResponseWriter, r *http.Request, userId int) {
		w.Header().Set("Content-Type", "application/json")

		entityStr2entityType := map[string]server.EntityType{
			"Person": server.PersonEntity,
			"Org":    server.OrgEntity,
			"Place":  server.PlaceEntity,
		}

		// This function calculates the 'AND' co-occurence of the conjunct expression
		// of the form: ["and", "<entityType>:<id>", "<entityType:id", ...]
		calcConjunctExpr := func(g *server.ContentBuffer, expr ConjunctiveExpr) (*server.ContentBuffer, error) {
			logger.Printf("Calculating co-occurences for conjuncts: %v", expr.And)
			for _, entity := range expr.And {
				logger.Printf("Filtering on %v", entity.Id)
				entityTypeStr, entityId, err := parseEntityStr(entity.Id)
				if err != nil {
					return nil, err
				}

				entityType, isValidEntityType := entityStr2entityType[entityTypeStr]
				if !isValidEntityType {
					return nil, errors.New(fmt.Sprintf("Unknown entity type: '%v'", entityTypeStr))
				}

				g = g.FilterOnEntity(entityType, entityId)
			}

			return g, nil
		}

		// Parse the posted data into the filter query in JSON format.
		// This query is essentially a LISP expression of the form:
		//
		//     (op arg1 arg2 ...)
		//
		// where op is (for now) an 'or' operator (i.e. a disjunction).
		// Each arg is a nested LISP conjunct expression of the form (and arg1 arg2...),
		// where each arg is a string representing a distinct entity of the form
		// "<entity type>:<entity id>" (e.g. "Person:9283742").
		getHttpRequestBody(w, r, func(postedData []byte) {
			var stats server.EntityStats
			var err error
			var finalContentBuffer *server.ContentBuffer

			// Default filter query: no time range specified, and no entity filters specified.
			filterQuery := FilterQuery{}

			httpRequestContainsData := !strutil.IsEmpty(string(postedData))
			if httpRequestContainsData {
				if err = json.Unmarshal(postedData, &filterQuery); err != nil {
					http.Error(w, fmt.Sprintf("User %v: Error parsing posted JSON: %v", userId, err), http.StatusBadRequest)
					return
				}
			}

			var baseContentBuffer *server.ContentBuffer
			if filterQuery.IsTimeRangeSpecified() {
				timeRange := time.Duration(filterQuery.TimeRangeInHours) * time.Hour
				logger.Printf("Getting content buffer for timeRange=%v", timeRange)
				baseContentBuffer = mgr.ContentBufferForTimeRange(timeRange)
			} else {
				logger.Printf("Getting global content buffer")
				baseContentBuffer = mgr.ContentBuffer()
			}

			if filterQuery.IsEntityFilterSpecified() {
				logger.Printf("Calculating entity co-occurrences with disjunct query: %+V", filterQuery.Or)
				tmpContentBuffer := baseContentBuffer
				finalContentBuffer = server.NewContentBuffer()
				// Each disjunct is itself a conjunct expression of the form (and arg1 arg2 ...)
				for _, conjunctiveExpr := range filterQuery.Or {
					tmpContentBuffer, err = calcConjunctExpr(baseContentBuffer, conjunctiveExpr)
					if err != nil {
						http.Error(w, fmt.Sprintf("User %v: Error processing conjunct expression: %v", userId, err), http.StatusBadRequest)
						return
					}
					finalContentBuffer = finalContentBuffer.Union(tmpContentBuffer)
				}

				stats = finalContentBuffer.CalcEntityStats()
			} else {
				// Use the global stats, since this user has no filter set.
				logger.Printf("No entity filter provided, so using global entity stats.")
				finalContentBuffer = baseContentBuffer
				stats = finalContentBuffer.LatestEntityStats()
			}

			topEntities := map[string][]server.Entity{
				"Person": annotateEntities(mgr.ContentDAO.PersonDAO, stats.TopPersons),
				"Org":    annotateEntities(mgr.ContentDAO.OrgDAO, stats.TopOrgs),
				"Place":  annotateEntities(mgr.ContentDAO.PlaceDAO, stats.TopPlaces),
			}

			entityTimes, entityValues := stats.EntityTrend.Data()

			latestNews := make([]server.NewsArticle, 0, len(stats.LatestNews))
			for _, doc := range stats.LatestNews {
				newsArticle := server.NewsArticle{
					Document: doc,
					Persons:  annotateEntities(mgr.ContentDAO.PersonDAO, makeEntities(finalContentBuffer.PersonGraph.EntityIdsForDocument(doc.Id).Items())),
					Orgs:     annotateEntities(mgr.ContentDAO.OrgDAO, makeEntities(finalContentBuffer.OrgGraph.EntityIdsForDocument(doc.Id).Items())),
					Places:   annotateEntities(mgr.ContentDAO.PlaceDAO, makeEntities(finalContentBuffer.PlaceGraph.EntityIdsForDocument(doc.Id).Items())),
				}
				latestNews = append(latestNews, newsArticle)
			}

			response := map[string]interface{}{
				"EntityTrend": EntityTrend{entityTimes, entityValues},
				"TopEntities": topEntities,
				"LatestNews":  latestNews,
			}

			sendJsonResponse(response, w)
		}) // End getHttpRequestBody()
	}
}

// Saves the global data (a.k.a. the "content buffer") to disk.
func SaveGlobalData(entityMgr *server.EntityManager, cfg AppConfig) webapp.HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		logger.Printf("\n\n========================\n" +
			"SAVING GLOBAL DATA!\n" +
			"========================\n\n")

		dataDir := cfg.DataDir

		createDataDirIfNotExists(dataDir)

		refreshStatsLock.Lock()
		entityMgr.ContentBuffer().SaveState(dataDir)
		entityMgr.ContentDAO.Save(dataDir)
		refreshStatsLock.Unlock()

		logger.Printf("\n\n========================\n" +
			"GLOBAL DATA SAVED!\n" +
			"========================\n\n")
	}
}

func GetMemStats() webapp.HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		response := map[string]interface{}{
			"General": map[string]uint64{
				"Alloc":      memStats.Alloc,
				"TotalAlloc": memStats.TotalAlloc,
				"Sys":        memStats.Sys,
				"Lookups":    memStats.Lookups,
			},
			"Heap": map[string]uint64{
				"HeapAlloc":   memStats.HeapAlloc,
				"HeapSys":     memStats.HeapSys,
				"HeapIdle":    memStats.HeapIdle,
				"HeapInuse":   memStats.HeapInuse,
				"HeapObjects": memStats.HeapObjects,
			},
		}

		sendJsonResponse(response, w)
	}
}

func CalcHotEntities(hotEntityCalc *cache.MemoizingFunc) webapp.UserHttpHandler {
	return func(w http.ResponseWriter, r *http.Request, userId int) {
		w.Header().Set("Content-Type", "application/json")
		logger.Printf("User:%v requesting hot entities", userId)

		hotEntities := hotEntityCalc.Call().(map[string][]server.Entity)
		sendJsonResponse(hotEntities, w)
	}
}

func CreateUsageReport(userDb *UserDb) webapp.HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")

		headerRow := "email, last_login, terms_accepted, watchlists"
		fmt.Fprintln(w, headerRow)

		// data rows
		userDb.ForEachUser(func(user User) {
			lastLogin := fmt.Sprintf("%v", user.LastLogin)
			if user.LastLogin.IsEmpty() {
				lastLogin = "NEVER"
			}

			dataRow := fmt.Sprintf("%v, %v, %v, %v", user.Email, lastLogin, user.TermsAccepted, len(user.WatchLists))
			fmt.Fprintln(w, dataRow)
		})
	}
}

// Saves all user-specific data to a data file named "user_data.dat" within
// the directory specified by the SYNTHOS_DATA_DIR config (default directory
// is /tmp/synthos/data).
func SaveUserData(userDb *UserDb, cfg AppConfig) webapp.HttpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		dataDir := cfg.DataDir
		userDataFilePath := filepath.Join(dataDir, "user_data.json")
		logger.Printf("Saving user data to %v", userDataFilePath)

		createDataDirIfNotExists(dataDir)

		err := userDb.Save(userDataFilePath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error saving user data to %v: %v", userDataFilePath, err), http.StatusInternalServerError)
			return
		}

		logger.Printf("User data saved to %v.", userDataFilePath)
	}
}

func createDataDirIfNotExists(dataDir string) {
	if !server.FileExists(dataDir) {
		logger.Printf("%v doesn't exist, so creating it and creating data version file.", dataDir)
		server.Must(os.MkdirAll(dataDir, 0755))
		migrate.WriteDataVersion(dataDir)
	}
}

// Given a URL path (e.g. "/api/persons/123"), parse out the "123" and
// return it as an int.
func parseObjectIdFromPath(urlPath string) (id int, err error) {
	pathParts := strings.Split(urlPath, "/")
	entityIdStr := pathParts[len(pathParts)-1]
	objectId, err := strutil.ParseInt(entityIdStr)
	if err != nil {
		return -1, err
	}
	return objectId, nil
}

func parseWatchList(watchListJson []byte) (WatchList, error) {
	var watchList WatchList
	if err := json.Unmarshal(watchListJson, &watchList); err != nil {
		return WatchList{}, err
	}

	if err := watchList.Validate(); err != nil {
		return WatchList{}, err
	}

	return watchList, nil
}

// Parses a string of the form "<entity type>:<entity id>" and returns the
// constituent entity type string and entity id.  The browser client sends
// entity IDs in this format.
//
// Example: parseEntityStr("Foo:123") // returns ("foo", 123)
//
func parseEntityStr(entityStr string) (entityType string, entityId int, err error) {
	tokens := strings.Split(entityStr, ":")
	if len(tokens) != 2 {
		return "", -1, errors.New(fmt.Sprintf("Malformed entity id, expected 'type:id' format, but got: '%v'", entityStr))
	}

	entityType = tokens[0]

	entityId, err = strconv.Atoi(tokens[1])
	if err != nil {
		return "", -1, err
	}

	return entityType, entityId, nil
}

func makeEntities(entityIds []int) []server.Entity {
	entities := make([]server.Entity, 0, len(entityIds))
	for _, entityId := range entityIds {
		entities = append(entities, server.DisplayEntity{Id: entityId})
	}
	return entities
}

func annotateEntities(entityDAO server.EntityDAO, entities []server.Entity) []server.Entity {
	annotatedEntities := make([]server.Entity, 0, len(entities))
	for _, entity := range entities {
		entityId := entity.GetId()
		entityLabel := entityDAO.GetLabel(entityId)
		annotatedEntity := server.DisplayEntity{
			Id:    entityId,
			Name:  entityLabel,
			Score: entity.GetScore(),
		}
		annotatedEntities = append(annotatedEntities, annotatedEntity)
	}

	return annotatedEntities
}

// Helper function that formats the specified object into a JSON message
// and then writes it to the specified writer.  Any errors that occur result
// in an HTTP 500 response with the text of the error.
func sendJsonResponse(object interface{}, w http.ResponseWriter) {
	b, err := json.MarshalIndent(object, "", "\t")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, string(b))
}

// Wraps an existing function, handling various IO error checking prior to
// passing the request body into said function.  Example usage:
//
//     getHttpRequestBody(w, r, func(body []byte) {
//         // Do something with the request body, like...
//         fmt.Printf("The request is: %v", string(body))
//     })
//
func getHttpRequestBody(w http.ResponseWriter, r *http.Request, f func(body []byte)) {
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading HTTP request: %v", err), http.StatusInternalServerError)
	} else {
		f(requestBody)
	}
}

// Hardcoded watchlists that are associated with a new user upon their
// initial login to the application.
func createDefaultWatchlists() []WatchList {
	return []WatchList{
		WatchList{
			Title:       "Tech CEOs",
			Description: "Technology CEOs.",
			Filter: FilterQuery{
				Or: []ConjunctiveExpr{
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:175952", Label: "Tim Cook"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:563938", Label: "John Donahoe"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:624426", Label: "Larry Page"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:688332", Label: "Travis Kalanick"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:3830116", Label: "Marissa Ann Mayer"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:603586", Label: "Ben Silbermann"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:587931", Label: "Jack Dorsey"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:5312055", Label: "Evan Spiegel"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:814543", Label: "Drew Houston"}},
					},
				},
			},
		},
		WatchList{
			Title:       "Tennis Players",
			Description: "News about tennis.",
			Filter: FilterQuery{
				Or: []ConjunctiveExpr{
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:326675", Label: "John McEnroe"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:618581", Label: "Nick Kyrgios"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:3752349", Label: "Tomas Berdych"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:570162", Label: "Andy Murray"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:166172", Label: "Novak Djokovic"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:628511", Label: "Serena Williams"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:294253", Label: "Rafael Nadal"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:3577000", Label: "Rodney George Laver"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Person:5215795", Label: "Chris Evert-Lloyd"}},
					},
				},
			},
		},
		WatchList{
			Title:       "Aerospace Industry",
			Description: "People and news about aerospace.",
			Filter: FilterQuery{
				Or: []ConjunctiveExpr{
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:70654120", Label: "BAE-Systems"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:20141986", Label: "Fokker"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:43266861", Label: "Mid-Western Aircraft Syst Inc"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:70428708", Label: "AlliedSignal Aerospace Company"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:33836283", Label: "Bombardier Transportation Inc"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:70514677", Label: "Lockheed Martin Aeronautics"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:20005309", Label: "Airbus"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{
							FilterItem{Id: "Org:22198950", Label: "Boeing"},
							FilterItem{Id: "Person:558942", Label: "James McNerney"},
						},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:20132759", Label: "United Technologies Corporation"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:44190737", Label: "Northrop Grumman Corp"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:20143499", Label: "Embraer"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:20131811", Label: "Finmeccanica"}},
					},
					ConjunctiveExpr{
						And: []FilterItem{FilterItem{Id: "Org:33153478", Label: "Ball Aerospace and Technologies"}},
					},
				},
			},
		},
	}
}
