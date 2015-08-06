// This package provides a mock implementation of the EntityManager interface,
// which is useful for local development and for integration testing of
// downstream components that depend on the EntityManager interface.
package mock

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"qbase/synthos/synthos_core/unixtime"
	server "qbase/synthos/synthos_svr"
	"strconv"
	"time"
)

//
// Package-scope variables
//
var logger = log.New(os.Stderr, "[mock] ", (log.Ltime | log.Lshortfile))

var fakePersons, fakeOrgs, fakePlaces []server.Entity
var rnd = rand.New(rand.NewSource(time.Now().Unix()))

type MockContentSource struct {
	topPlaces []server.Place
}

func NewMockContentSource() *MockContentSource {
	return &MockContentSource{}
}

// This initializer runs when this file is imported for the first time.
func init() {
	logger.Printf("Initializing MockContentSource")

	fakePersons = makeFakePersons()
	fakeOrgs = makeFakeOrgs()

	filePath := "./mock/city_info.csv"
	fakePlaces = loadFakePlacesFromFile(filePath)

	logger.Printf("%d places were loaded from %s.", len(fakePlaces), filePath)
	logger.Printf("MockContentSource initialized.")
}

func (me *MockContentSource) FetchNewsArticles(startTime unixtime.Time, endTime unixtime.Time) []server.NewsArticle {
	articlesPerMin := 200 + rand.Intn(400)

	logger.Printf("Exporting docs from %v to %v", startTime, endTime)

	articlesPerSec := articlesPerMin / 60
	timeWindow := endTime.Time().Sub(startTime.Time())
	numArticlesToFetch := articlesPerSec * int(timeWindow/time.Second)
	logger.Printf("articlesPerMin=%v, timeWindow=%v => generating %v fake news articles", articlesPerMin, timeWindow, numArticlesToFetch)

	newsArticles := make([]server.NewsArticle, 0, numArticlesToFetch)
	for i := 0; i < numArticlesToFetch; i++ {
		doc := makeFakeDocument(me.nextDocumentId(), startTime, endTime)

		// Create some fake people
		numPersonsPerDoc := 0 + rnd.Intn(20)
		persons := make([]server.Entity, 0, numPersonsPerDoc)
		for j := 0; j < numPersonsPerDoc; j++ {
			persons = append(persons, fakePersons[rnd.Intn(len(fakePersons))])
		}

		// Create some fake orgs
		numOrgsPerDoc := 0 + rnd.Intn(20)
		orgs := make([]server.Entity, 0, numOrgsPerDoc)
		for j := 0; j < numOrgsPerDoc; j++ {
			orgs = append(orgs, fakeOrgs[rnd.Intn(len(fakeOrgs))])
		}

		// Create some fake places
		numPlacesPerDoc := 0 + rnd.Intn(20)
		places := make([]server.Entity, 0, numPlacesPerDoc)
		for j := 0; j < numPlacesPerDoc; j++ {
			places = append(places, fakePlaces[rnd.Intn(len(fakePlaces))])
		}

		newsArticle := server.NewsArticle{Document: doc, Persons: persons, Orgs: orgs, Places: places}
		newsArticles = append(newsArticles, newsArticle)
	}

	return newsArticles
}

// Returns the next unique document Id in the sequence.
func (me *MockContentSource) nextDocumentId() int {
	return rnd.Intn(1000000000) + 1
}

// Loads city info from a CSV file into a slice of Place structs.
// This provides some mock 'Place' entities until the real content
// is ready to be accessed from MemDB.
func loadFakePlacesFromFile(csvFilePath string) []server.Entity {
	citiesCsv, err := os.Open(csvFilePath)
	if err != nil {
		panic(fmt.Sprintf("ERROR loading city info: %v", err))
	}
	defer citiesCsv.Close()

	places := make([]server.Entity, 0, 100)

	reader := csv.NewReader(citiesCsv)
	reader.Comma = '|'
	reader.Comment = '#'
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(fmt.Sprintf("ERROR reading city info from csv: %v", err))
		}
		place := server.Place{
			Id:   string2int(record[0]),
			Name: fmt.Sprintf("%s, %s", record[1], record[2]),
			Location: server.GeoCoord{
				Lat: string2float32(record[3]),
				Lng: string2float32(record[4]),
			},
		}
		places = append(places, place)
	}

	return places
}

func makeFakeDocument(docId int, startTime unixtime.Time, endTime unixtime.Time) server.Document {
	fakeNewsSources := []string{"The Onion", "NYTimes", "Wired.com", "SciCentral", "Reuters", "BBC News"}

	// Returns a UnixTime object that falls within the specified time endpoints (inclusive).
	randomTimeBetween := func(startTime unixtime.Time, endTime unixtime.Time) unixtime.Time {
		timeDiff := endTime.Time().Unix() - startTime.Time().Unix() + 1
		randomUnixTime := int32(startTime.Time().Unix() + rnd.Int63n(timeDiff))
		return unixtime.Unix(randomUnixTime)
	}

	return server.Document{
		Id:         docId,
		Headline:   fmt.Sprintf("Document %v", docId),
		InsertDate: randomTimeBetween(startTime, endTime),
		Source:     fakeNewsSources[rnd.Intn(len(fakeNewsSources))],
		Url:        fmt.Sprintf("http://www.example.com/fake-document/%v", docId),
	}
}

func makeFakeOrgs() []server.Entity {
	fakeOrgCount := 10000
	fakeOrgs := make([]server.Entity, fakeOrgCount)
	for i := 0; i < fakeOrgCount; i++ {
		id := 100000 + i
		fakeOrgs[i] = server.DisplayEntity{Id: id, Name: fmt.Sprintf("Organization %v", id)}
	}
	return fakeOrgs
}

func makeFakePersons() []server.Entity {
	lastNames := []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Miller", "Davis", "Garcia", "Rodriguez", "Wilson",
		"Martinez", "Anderson", "Taylor", "Thomas", "Hernandez", "Moore", "Martin", "Jackson", "Thomson", "White",
		"Lopez", "Lee", "Gonzalez", "Harris", "Clark", "Lewis", "Robinson", "Walker", "Perez", "Hall",
		"Young", "Allen", "Sanchez", "Wright", "King", "Scott", "Green", "Baker", "Adams", "Nelson",
		"Hill", "Ramirez", "Campbell", "Mitchell", "Roberts", "Carter", "Philips", "Evans", "Turner", "Torres",
		"Parker", "Collins", "Edwards", "Stewart", "Flores", "Morris", "Nguyen", "Murphy", "Rivera", "Cook",
	}
	firstNames := []string{
		"John", "Anne", "Eric", "Mia", "Phil", "Rebecca", "Simon", "Carol", "Mat", "Gwen",
		"Mike", "Jessica", "Chris", "Ashley", "Matthew", "Brittany", "Joshua", "Amanda", "Daniel", "Samantha",
		"David", "Sarah", "Andrew", "Stephani", "James", "Jennifer", "Justin", "Elizabeth", "Joseph", "Lauren",
		"Ryan", "Megan", "John", "Emily", "Robert", "Nichole", "Nicholas", "Kayla", "Anthony", "Amber",
		"William", "Rachael", "Jonathan", "Courtney", "Kyle", "Danielle", "Brandon", "Heather", "Jacob", "Melissa",
		"Tyler", "Becky", "Zachary", "Michelle", "Kevin", "Tiffany", "Chad", "Chelsea", "Steven", "Christina",
	}
	middleInits := []string{
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
		"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
	}

	persons := make([]server.Entity, 0, 100000)
	personId := 1
	for _, ln := range lastNames {
		for _, fn := range firstNames {
			for _, mi := range middleInits {
				person := server.DisplayEntity{Id: personId, Name: fmt.Sprintf("%v %v. %v", fn, mi, ln)}
				persons = append(persons, person)
				personId++
			}
		}
	}

	return persons
}

// Converts a string to a float32.  Panics if a parsing error occurs.
func string2float32(s string) float32 {
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		panic(fmt.Sprintf("Error converting \"%v\" to a float32: %v", s, err))
	}
	return float32(f)
}

// Converts a string to an int64.  Panics if a parsing error occurs.
func string2int(s string) int {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Error converting \"%v\" to an int64: %v", s, err))
	}
	return int(i)
}
