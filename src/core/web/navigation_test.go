package web

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddNavigationItem(t *testing.T) {
	navigation := NewNavigation()
	item := NavItem{Name: "TestItem", URL: "/test", Position: 1}

	navigation.Add("TestItem", item)

	addedItem, found := navigation.Item("TestItem")
	assert.True(t, found, "Item should have been found.")
	assert.Equal(t, item.URL, addedItem.URL, "Added item URL does not match.")
	assert.Nil(t, navigation.sorted, "Sorted cache should be invalid (nil) after adding an item.")
}
