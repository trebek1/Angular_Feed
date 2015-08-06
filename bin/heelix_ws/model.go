package main

import (
	"errors"
	"qbase/synthos/synthos_core/unixtime"
)

// An Synthos application user.
type User struct {
	Id            int
	Email         string
	PasswordHash  string
	PasswordSalt  string
	AccessToken   string `json:"-"` // Don't export this to JSON
	LastLogin     unixtime.Time
	TermsAccepted bool

	WatchLists []WatchList
}

// A WatchList is basically a named set of entities that can be used as a
// filter to restrict content to only those entities that co-occur with the
// ones in the watchlist.
type WatchList struct {
	Id          int
	Title       string
	Description string
	Filter      FilterQuery
}

func (me *WatchList) IsSaved() bool {
	return me.Id != 0
}

func (me *WatchList) Validate() error {
	if me.Title == "" {
		return errors.New("Title was empty")
	}

	return nil
}

// Represents an entity filter query in Disjunctive Normal Form (DNF).
// In terms of LISP, this struct represents statements like this:
//
//  (or
// 	    (and arg1 arg2 ...)
// 	    (and arg1 arg2 ...)
// 	    ...
//  );
type FilterQuery struct {
	// Restricts the content (documents and entities) by time range.
	// For example, if TimeRangeInHours=24, this means "only include content
	// from from the past 24 hours".
	TimeRangeInHours int

	// Entity
	Or []ConjunctiveExpr
}

func (me *FilterQuery) IsTimeRangeSpecified() bool {
	return me.TimeRangeInHours > 0
}

func (me *FilterQuery) IsEntityFilterSpecified() bool {
	return len(me.Or) > 0
}

// Represents a conjunctive expression of the form (and entity1 entity2 ...).
// The 'and' operator is implied by the name of this type.
type ConjunctiveExpr struct {
	And []FilterItem
}

// In atomic term representing a specific entity of a given type.
type FilterItem struct {
	// Entity ID that is globally unique across all entity types.  The Id is
	// expected to have the format: "<entity type>:<entity id>".
	Id string
	// The entity's display label
	Label string
}

// Represents the time series data points for an entity type in the JSON format
// expected by the UI client.
type EntityTrend struct {
	Times  []int
	Values []int
}
