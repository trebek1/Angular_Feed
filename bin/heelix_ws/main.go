package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	finch "qbase/synthos/gofinch"
	migrate "qbase/synthos/heelix_ws/datamigrate"
	mock "qbase/synthos/heelix_ws/mock"
	"qbase/synthos/synthos_core/cache"
	"qbase/synthos/synthos_core/unixtime"
	"qbase/synthos/synthos_core/webapp"
	server "qbase/synthos/synthos_svr"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

//
// Global variables
//

// global logging object
var logger = log.New(os.Stderr, "[heelix_ws] ", (log.Ldate | log.Ltime | log.Lshortfile))

// Mutex that prevents the content buffer from getting saved to disk while the global
// stats are being refreshed, and vice-versa.
var refreshStatsLock sync.Locker

// Fires up the EntityManager service.  Once started, the service will pull
// the latest content (documents and entities) from the configured ContentSource
// calculate some aggreate stats on them, and then make the stats available to
// this web app.
func startEntityManager(cfg AppConfig, contentSource server.ContentSource) *server.EntityManager {
	// Create and configure a new EntityManager object.
	entityManager := server.NewEntityManager(server.EntityManagerConfig{
		TimeRanges:    cfg.TimeRanges,
		ContentSource: contentSource,
	})

	now := unixtime.Now()

	// If it exists, load global content saved prior to previous app shutdown.
	dataDir := cfg.DataDir
	if server.FileExists(dataDir) {
		logger.Printf("Loading saved content from %v", dataDir)
		entityManager.LoadState(dataDir, now)
		entityManager.ContentDAO.Load(dataDir)
	} else {
		logger.Printf("Loading recent content directly from content source into content buffer.")
		entityManager.PreFill(now.Subtract(cfg.DataPreFetchWindow), now)
	}

	logger.Printf("Calculating entity stats for the first time...")
	entityManager.RefreshStats(now)

	refreshStatsLock = &sync.Mutex{}

	logger.Printf("Starting MemDb content polling loop.")
	go func() {
		// The show must go on.  Trap any panics, log them as an error, and then
		// continue to attempt to get more data from MemDb service.
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Printf("ERROR: unexpected application error: \n%v\n\n", r)
						debug.PrintStack()
					}
				}()

				ticker := time.NewTicker(cfg.RefreshInterval)
				for _ = range ticker.C {
					refreshStatsLock.Lock()
					startTime := time.Now()
					entityManager.FetchMoreContent(unixtime.Now())
					logger.Printf("entityManager.FetchMoreContent() took %v", time.Since(startTime))
					refreshStatsLock.Unlock()
				}
			}()
		}
	}()

	return entityManager
}

func createEntityAnnotator(useMockData bool, finchDb finch.DB) server.EntityAnnotator {
	var entityAnnotator server.EntityAnnotator
	if useMockData {
		entityAnnotator = &mock.MockEntityAnnotator{}
	} else {
		entityAnnotator = server.NewMemDbEntityAnnotator(finchDb)
	}

	return entityAnnotator
}

func createEntitySearch(useMockData bool, contentDAO *server.ContentDAO, contentBuffer *server.ContentBuffer, finchDb finch.DB) server.EntitySearch {
	contentBufferEntitySearch := server.NewContentBufferEntitySearch(contentDAO, contentBuffer)
	return contentBufferEntitySearch
	// if useMockData {
	// 	return contentBufferEntitySearch
	// } else {
	// 	return server.NewMemDbEntitySearch(qr, contentBufferEntitySearch)
	// }
}

func createContentSource(useMockData bool, finchDb finch.DB) server.ContentSource {
	var rawContentSource server.ContentSource
	if useMockData {
		logger.Printf("No MemDb connection string provided, so using mock data.")
		rawContentSource = mock.NewMockContentSource()
	} else {
		// Use the real production components
		rawContentSource = server.NewMemDbContentSource(finchDb)
	}

	return &server.ValidatingContentSource{TargetContentSource: rawContentSource}
}

// Creates an instance of the application user db, which stores all
// user-specific data (user's personal info, watchlists, etc.).
func createUserDb(dataDir string) (userDb *UserDb) {
	userDataFilePath := filepath.Join(dataDir, "user_data.json")

	if server.FileExists(userDataFilePath) {
		userDb = LoadUserDb(userDataFilePath)
	} else {
		logger.Printf("%v doesn't exist, so loading hardcoded test users", userDataFilePath)
		userDb = NewUserDb()
		createHardcodedUsers(userDb)
	}

	return userDb
}

// Hardcoded test users for non-production environments.
func createHardcodedUsers(userDb *UserDb) {

	hardcodedUserNames := []string{
		"ahunt@synthostech.com",
		"calagno@synthostech.com",
		"cszlucha@synthostech.com",
		"dnguyen@synthostech.com",
		"emgill@synthostech.com",
		"etakahashi@synthostech.com",
		"gmilstein@synthostech.com",
		"gsofo@synthostech.com",
		"mJaeggli@synthostech.com",
		"mwuerstl@qbase.com",
		"pluther@synthostech.com",
		"pviswanathan@synthostech.com",
		"support@synthostech.com",
	}

	defaultPassword := "cat-knuckle-sweater-59!"

	for _, userName := range hardcodedUserNames {
		_, err := userDb.AddUser(userName, defaultPassword)
		if err != nil {
			panic(err)
		}
	}
}

// This is the starting point of the application.
func main() {
	appConfig := MakeAppConfig()

	// Handle command line options
	runMigrationPtr := flag.Bool("migrate", false, "Run the data migration.")
	flag.Parse()
	if *runMigrationPtr {
		migrate.Migrate(appConfig.DataDir)
		os.Exit(0)
	}

	// Set max # of OS threads that may execute user-level Golang code
	runtime.GOMAXPROCS(4)

	logger.Printf("Starting up Heelix Web Service...")
	logger.Printf("AppConfig = %v", fmt.Sprintf("%+v", appConfig))
	logger.Printf("General Info:\n"+
		"   ------------------------------------------\n"+
		"    Golang Version: %v\n"+
		"    NumCPU: %v\n"+
		"    NumGoroutine: %v\n"+
		"   ------------------------------------------", runtime.Version(), runtime.NumCPU(), runtime.NumGoroutine())

	//
	// Create the business objects needed by the web app
	//

	useMockData := appConfig.UseMockData()

	// Application users and their associated user-specific content is stored here.
	userDb := createUserDb(appConfig.DataDir)
	// Handles user authentication and authorization.
	auth := NewAuthenticator(userDb)
	// Issues queries to the Finch database.
	var finchDb finch.DB
	if !appConfig.UseMockData() {
		finchDb = finch.NewFinchDB(appConfig.MemDbConn)
	}
	// Provides info about a 'Person' entities (used for "Baseball Card" feature in app).
	entityAnnotator := createEntityAnnotator(useMockData, finchDb)
	// Provides access to news documents and their associated entities.
	contentSource := createContentSource(useMockData, finchDb)
	// Fire up the EntityManager component, which will periodically talk to the
	// MemDB server to obtain the latest content.
	entityMgr := startEntityManager(appConfig, contentSource)
	// Finds entities given a search string.
	entitySearch := createEntitySearch(useMockData, entityMgr.ContentDAO, entityMgr.ContentBuffer(), finchDb)

	// This is a global (i.e. not user-specific) calculation, which only needs to
	// be recalculated every few minutes or so.
	memoizedHotEntityCalc := cache.NewMemoizingFunc(30*time.Second, func() interface{} {
		hotEntities := entityMgr.ContentBufferForTimeRange(8 * time.Hour).CalcHotEntities()
		annotatedHotEntities := map[string][]server.Entity{
			"Person": annotateEntities(entityMgr.ContentDAO.PersonDAO, hotEntities[server.PersonEntity]),
			"Org":    annotateEntities(entityMgr.ContentDAO.OrgDAO, hotEntities[server.OrgEntity]),
			"Place":  annotateEntities(entityMgr.ContentDAO.PlaceDAO, hotEntities[server.PlaceEntity]),
		}
		return annotatedHotEntities
	})

	//
	// Create an HTTP request handler and define the <url path> -> <handler func> mapping.
	//

	appRouteHandler := http.NewServeMux()

	appRouteHandler.HandleFunc("/robots.txt", webapp.HandleRobots)

	// Web service endpoints (open to the world)
	appRouteHandler.HandleFunc("/api/health_check", HealthCheck())
	appRouteHandler.HandleFunc("/api/system_info", GetSystemInfo(entityMgr, appConfig))

	// Web service endpoints (require user authentication/authorization)
	appRouteHandler.HandleFunc("/api/authenticate", webapp.PostOnly(auth.AuthenticateUser()))
	appRouteHandler.HandleFunc("/api/accept_terms", auth.AuthorizeUser(AcceptLicenseTerms(userDb)))
	appRouteHandler.HandleFunc("/api/logout", auth.AuthorizeUser(Logout(userDb)))
	appRouteHandler.HandleFunc("/api/all_entity_info", webapp.PostOnly(auth.AuthorizeUser(GetAllEntityInfo(entityMgr))))
	appRouteHandler.HandleFunc("/api/person/", auth.AuthorizeUser(FetchEntityInfo(server.PersonEntity, entityAnnotator)))
	appRouteHandler.HandleFunc("/api/org/", auth.AuthorizeUser(FetchEntityInfo(server.OrgEntity, entityAnnotator)))
	appRouteHandler.HandleFunc("/api/watchlists", auth.AuthorizeUser(GetOrPostWatchLists(userDb)))
	appRouteHandler.HandleFunc("/api/watchlists/", auth.AuthorizeUser(PutOrDeleteWatchList(userDb)))
	appRouteHandler.HandleFunc("/api/search/", auth.AuthorizeUser(FindEntities(entitySearch)))
	appRouteHandler.HandleFunc("/api/hot_entities", auth.AuthorizeUser(CalcHotEntities(memoizedHotEntityCalc)))

	// WEb service endpoints (can only be called on localhost)
	appRouteHandler.HandleFunc("/api/users", webapp.LocalOnly(webapp.PostOnly(AddNewUser(userDb))))
	appRouteHandler.HandleFunc("/api/users/report", webapp.LocalOnly(CreateUsageReport(userDb)))
	appRouteHandler.HandleFunc("/api/save_global_data", webapp.LocalOnly(SaveGlobalData(entityMgr, appConfig)))
	appRouteHandler.HandleFunc("/api/save_user_data", webapp.LocalOnly(SaveUserData(userDb, appConfig)))
	appRouteHandler.HandleFunc("/api/memstats", webapp.LocalOnly(GetMemStats()))

	// If deployment environment has an HTTPS reverse proxy, we need to redirect
	// all non-HTTPS requests to back to the HTTPS reverse proxy.
	hasReverseProxy := (appConfig.HttpsRedirectUrl != "")
	var requestHandler http.Handler
	if hasReverseProxy {
		requestHandler = webapp.NewHttpsRedirectHandler(appRouteHandler, appConfig.HttpsRedirectUrl)
	} else {
		requestHandler = appRouteHandler
	}

	// Enables Cross Origin Resource Sharing (CORS) so that web clients running in
	// a separate domain are allowed to talk to this web service.
	requestHandler = webapp.NewCORSRequestHandler(requestHandler)
	// Global error trapper
	requestHandler = webapp.NewHttpErrorHandler(requestHandler, logger)
	// HTTP request logger
	requestHandler = webapp.NewHttpLogHandler(logger, requestHandler)

	//
	// Start the app server!
	//
	port := ":8081"
	logger.Printf("Server ready and listening on port %s", port)
	logger.Fatal(http.ListenAndServe(port, requestHandler))
}
