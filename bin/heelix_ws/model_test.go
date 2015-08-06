package main

import (
	//	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWatchList_IsSaved(t *testing.T) {
	w := WatchList{Id: 0}
	assert.False(t, w.IsSaved())
	w.Id = 1
	assert.True(t, w.IsSaved())
}

func TestWatchList_Validate(t *testing.T) {
	w := WatchList{}
	assert.NotNil(t, w.Validate()) // No title specified
	w.Title = "Some Title"
	assert.Nil(t, w.Validate()) // Title specified now, so ok.
}
