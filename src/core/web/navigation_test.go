package web

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

// helper function to create a mock request
func newMockIO(url string) IO {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	return &HIO{request: req}
}

func TestAddNavigationItem(t *testing.T) {
	navigation := NewNavigation()
	item := NavItem{Name: "TestItem", URL: "/test", Position: 1}

	navigation.Add("TestItem", item)

	addedItem, found := navigation.Item("TestItem")
	assert.True(t, found)
	assert.Equal(t, item.URL, addedItem.URL)
	assert.Nil(t, navigation.sorted)
}

func TestRemoveNavigationItem(t *testing.T) {
	navigation := NewNavigation()
	item := NavItem{Name: "TestItem", URL: "/test", Position: 1}

	navigation.Add("TestItem", item)
	navigation.Remove("TestItem")

	_, found := navigation.Item("TestItem")
	assert.False(t, found)

	assert.Nil(t, navigation.sorted)
}

func TestNavigationItems(t *testing.T) {
	navigation := NewNavigation()

	navigation.Add("Item1", NavItem{Name: "Item1", Position: 2})
	navigation.Add("Item2", NavItem{Name: "Item2", Position: 1})

	items := navigation.Items()
	assert.Len(t, items, 2)
	assert.Equal(t, "Item2", items[0].Name)
	assert.Equal(t, "Item1", items[1].Name)

	// test cache on add
	assert.NotNil(t, navigation.sorted)

	navigation.Add("Item3", NavItem{Name: "Item3", Position: 3})
	assert.Nil(t, navigation.sorted)

	items = navigation.Items()
	assert.Len(t, items, 3)
	assert.Equal(t, "Item2", items[0].Name)

	// test cache on remove
	assert.NotNil(t, navigation.sorted)

	navigation.Remove("Item2")
	assert.Nil(t, navigation.sorted)

	items = navigation.Items()
	assert.Len(t, items, 2)
	assert.Equal(t, "Item1", items[0].Name)
}

func TestBuildNavigation_Display(t *testing.T) {
	io := newMockIO("/test")

	item1 := NavItem{
		URL:      "/test",
		Display:  func(io IO) (bool, error) { return true, nil },
		Position: 1,
	}
	item2 := NavItem{
		URL:      "/no-display",
		Display:  func(io IO) (bool, error) { return false, nil },
		Position: 2,
	}

	items := []NavItem{item1, item2}
	builtItems, err := BuildNavigation(items, io)

	require.NoError(t, err)
	assert.Len(t, builtItems, 1)
	assert.Equal(t, "/test", builtItems[0].URL)
}

func TestBuildNavigation_ErrorHandling(t *testing.T) {
	io := newMockIO("/test")

	expectedError := errors.New("display error")
	item := NavItem{
		URL:      "/test",
		Display:  func(io IO) (bool, error) { return false, expectedError },
		Position: 1,
	}

	items := []NavItem{item}
	_, err := BuildNavigation(items, io)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestBuildNavigation_ActiveState(t *testing.T) {
	io := newMockIO("/test")

	item := NavItem{
		URL:      "/test",
		Display:  func(io IO) (bool, error) { return true, nil },
		Position: 1,
	}

	items := []NavItem{item}
	builtItems, _ := BuildNavigation(items, io)

	assert.True(t, builtItems[0].Active())
}

func TestBuildNavigation_Recursive(t *testing.T) {
	io := newMockIO("/subitem")

	parent := NavItem{
		URL:      "/test",
		Display:  func(io IO) (bool, error) { return true, nil },
		Position: 1,
	}

	child := NavItem{
		URL:      "/subitem",
		Display:  func(io IO) (bool, error) { return true, nil },
		Position: 1,
	}

	parent.Items = []NavItem{child}

	items := []NavItem{parent}
	builtItems, err := BuildNavigation(items, io)

	require.NoError(t, err)
	assert.Len(t, builtItems, 1, "Parent item should be included")
	assert.Len(t, builtItems[0].Items, 1, "Child item should be included and built recursively")
	assert.True(t, builtItems[0].Items[0].Active(), "Child NavItem should be active when its URL matches the request path")
	assert.True(t, builtItems[0].Active(), "Parent NavItem should be active if any of its children are active")
}

func TestBuildNavigation_PanicOnMultipleParents(t *testing.T) {
	io := newMockIO("/test")

	assert.Panics(t, func() { BuildNavigation(nil, io, &NavItem{}, &NavItem{}) })
}

func TestBuildNavigation_NotOrdered(t *testing.T) {
	io := newMockIO("/")

	item1 := NavItem{
		URL:      "/first",
		Display:  func(io IO) (bool, error) { return true, nil },
		Position: 2,
	}
	item2 := NavItem{
		URL:      "/second",
		Display:  func(io IO) (bool, error) { return true, nil },
		Position: 1,
	}

	items := []NavItem{item1, item2}
	builtItems, err := BuildNavigation(items, io)

	require.NoError(t, err)
	assert.Equal(t, "/first", builtItems[0].URL) // items are NOT sorted by BuildNavigation, only by Navigation.Items
	assert.Equal(t, "/second", builtItems[1].URL)
}
