package datamigrate

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	server "qbase/synthos/synthos_svr"
	"strings"
)

// Each time the migration code is changed, this version stamp needs to
// be incremented!
var DATA_VERSION = "1.0.0"

//
// Package-scope variables
//
var logger = log.New(os.Stderr, "[datamigrate] ", (log.Ltime | log.Lshortfile))

func WriteDataVersion(dataDir string) {
	server.Must(ioutil.WriteFile(getVersionFilePath(dataDir), []byte(DATA_VERSION), 0644))
}

func Migrate(dataDir string) {
	if !server.FileExists(dataDir) {
		logger.Printf("Data directory '%v' doesn't appear to exist, so aborting migration.", dataDir)
		return
	}

	logger.Printf("Migrating data in %v to version %v", dataDir, DATA_VERSION)
	deployedDataVersion := readDataVersion(dataDir)
	if deployedDataVersion == DATA_VERSION {
		logger.Printf("Actually, data format was already up-to-date, so skipping migration.")
		return
	}

	migrateEntityGraph(dataDir, "person")
	migrateEntityGraph(dataDir, "org")
	migrateEntityGraph(dataDir, "place")

	server.Must(os.Remove(filepath.Join(dataDir, "personGraph.dat")))
	server.Must(os.Remove(filepath.Join(dataDir, "orgGraph.dat")))
	server.Must(os.Remove(filepath.Join(dataDir, "placeGraph.dat")))

	server.Must(os.Rename(filepath.Join(dataDir, "personGraph.dat.tmp"), filepath.Join(dataDir, "personGraph.dat")))
	server.Must(os.Rename(filepath.Join(dataDir, "orgGraph.dat.tmp"), filepath.Join(dataDir, "orgGraph.dat")))
	server.Must(os.Rename(filepath.Join(dataDir, "placeGraph.dat.tmp"), filepath.Join(dataDir, "placeGraph.dat")))

	WriteDataVersion(dataDir)

	logger.Printf("Data in %v successfully migrated to version %v", dataDir, DATA_VERSION)
}

func migrateEntityGraph(dataDir string, entityType string) {
	logger.Printf("Migrating '%v' graph", entityType)

	isPlaceGraph := (entityType == "place")

	sourceFile, err := os.Open(filepath.Join(dataDir, fmt.Sprintf("%vGraph.dat", entityType)))
	if err != nil {
		panic(err)
	}
	defer sourceFile.Close()

	destEntityFile, err := os.Create(filepath.Join(dataDir, fmt.Sprintf("%vInfo.dat", entityType)))
	if err != nil {
		panic(err)
	}
	defer destEntityFile.Close()

	destGraphFile, err := os.Create(filepath.Join(dataDir, fmt.Sprintf("%vGraph.dat.tmp", entityType)))
	if err != nil {
		panic(err)
	}
	defer destGraphFile.Close()

	var geoCoordOut *server.StreamWriter
	if isPlaceGraph {
		f, err := os.Create(filepath.Join(dataDir, "geoCoords.dat"))
		if err != nil {
			panic(err)
		}
		defer f.Close()
		geoCoordOut = server.NewStreamWriter(bufio.NewWriter(f))
	}

	in := server.NewStreamReader(bufio.NewReader(sourceFile))
	bufOut := bufio.NewWriter(destEntityFile)
	out := server.NewStreamWriter(bufOut)

	persistedEntityType := in.GetString() // entity type (e.g. "synthos_svr.Person")
	logger.Printf("    persistedEntityType = %v", persistedEntityType)
	entityCount := in.GetInt()
	logger.Printf("    entityCount = %v", entityCount)
	out.PutInt(entityCount)
	if isPlaceGraph {
		geoCoordOut.PutInt(entityCount)
	}

	for i := 0; i < entityCount; i++ {
		id := in.GetInt()
		in.GetInt() // skip score
		label := in.GetString()

		out.PutInt(id)
		out.PutString(label)

		if isPlaceGraph {
			geoCoordOut.PutInt(id)
			geoCoordOut.PutFloat32(in.GetFloat32()) // lat
			geoCoordOut.PutFloat32(in.GetFloat32()) // lng
		}
	}
	bufOut.Flush()

	bufOut = bufio.NewWriter(destGraphFile)
	out = server.NewStreamWriter(bufOut)

	// Output: doc -> [related entity ID] links
	docCount := in.GetInt()
	logger.Printf("    Migrating %v document->entity links", docCount)
	tempContainer := map[int]*server.IntSet{}
	numDocsWithNoEntities := 0
	for i := 0; i < docCount; i++ {
		docId := in.GetInt()
		relatedEntityCount := in.GetInt32()
		if relatedEntityCount == 0 {
			numDocsWithNoEntities++
		} else {
			entityIds := server.NewIntSet()
			for j := 0; j < int(relatedEntityCount); j++ {
				entityIds.Put(in.GetInt())
			}
			tempContainer[docId] = entityIds
		}
	}
	logger.Printf("    %v documents had no associated entities", numDocsWithNoEntities)
	logger.Printf("    Writing %v document->entity links to file", len(tempContainer))
	out.PutInt(len(tempContainer))
	for docId, relatedEntityIds := range tempContainer {
		out.PutInt(docId)
		out.PutInt32(int32(relatedEntityIds.Size()))
		relatedEntityIds.ForEach(func(entityId int) {
			out.PutInt(entityId)
		})
	}

	// Output: entity -> [related doc ID] links
	entityCount = in.GetInt()
	out.PutInt(entityCount)
	logger.Printf("    Migrating %v entity->document links", entityCount)
	for i := 0; i < entityCount; i++ {
		out.PutInt(in.GetInt()) // entity id
		relatedDocCount := in.GetInt32()
		if relatedDocCount == 0 {
			logger.Printf(">>>> WARN: i=%v: entity had no associated entities", i)
		}
		out.PutInt32(relatedDocCount)
		for j := 0; j < int(relatedDocCount); j++ {
			out.PutInt(in.GetInt())
		}
	}

	bufOut.Flush()
}

func readDataVersion(dataDir string) string {
	bytes, err := ioutil.ReadFile(getVersionFilePath(dataDir))
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(string(bytes))
}

func getVersionFilePath(dataDir string) string {
	return filepath.Join(dataDir, "version.txt")
}
