package web

import (
	"sort"
	"sync"
)

// Navigation is a collection of NavItems. It is used to build the navigation bar.
// The navbar is built by calling Build. Build will call the Display function of each NavItem to determine if it should be displayed.
// The Display function is called with the current IO. The IO can be used to determine if the user is logged in or not.
//
// Navigation also sorts the NavItems by their Position.
// If two items have the same Position the order is undefined. The Navigation will cache the sorted items.
//
// Navigation is safe for concurrent use by multiple goroutines.
type Navigation struct {
	items    map[string]NavItem
	mu       sync.RWMutex
	sorted   []NavItem
	sortedMu sync.Mutex
}

// NavItem is a single item in the navigation bar. It can either be part of the Navigation or a sub item of another NavItem.
// NavItem.active will be determined by the Navigation.Build depending on the current URL.
// Set NavItem.Redirect to true if the item should redirect to NavItem.URL (e.g. for logout) otherwise the item might be an HTMX boosted link.
// In that case the item will be loaded via HTMX and the URL will be changed to NavItem.URL. However, the URL will not be reloaded.
// During Navigation.Built the Display function will be called to determine if the item should be displayed.
// Also, the NavItem.Position will be used to sort the items. Items with a lower Position will be displayed first.
type NavItem struct {
	active   bool
	Redirect bool
	URL      string
	Name     string
	Items    []NavItem
	Display  func(io IO) (bool, error)
	Position int
}

// Active returns true if the item is active. An item is active if the current URL matches the item URL.
func (i *NavItem) Active() bool {
	return i.active
}

// NewNavigation returns a new Navigation with an empty but allocated map of NavItems.
// The sorted cache is neither allocated nor initialized.
func NewNavigation() *Navigation {
	return &Navigation{
		items: make(map[string]NavItem),
	}
}

// Add adds a new NavItem to the Navigation and invalidates the sorted cache.
func (n *Navigation) Add(name string, item NavItem) {
	n.mu.Lock()
	n.items[name] = item
	n.mu.Unlock()

	n.sortedMu.Lock()
	n.sorted = nil
	n.sortedMu.Unlock()
}

// Remove removes a NavItem from the Navigation and invalidates the sorted cache.
func (n *Navigation) Remove(name string) {
	n.mu.Lock()
	delete(n.items, name)
	n.mu.Unlock()

	n.sortedMu.Lock()
	n.sorted = nil
	n.sortedMu.Unlock()
}

// Item returns the NavItem with the given name and a boolean indicating if the item was found.
// It can be used as a Lookup function.
func (n *Navigation) Item(name string) (NavItem, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	item, ok := n.items[name]
	return item, ok
}

// Items returns a slice of all NavItems in the Navigation sorted by their Position.
// The slice is cached and will not be recalculated until the Navigation is modified.
func (n *Navigation) Items() []NavItem {
	n.sortedMu.Lock()
	defer n.sortedMu.Unlock()

	if n.sorted != nil {
		return n.sorted
	}

	n.sorted = make([]NavItem, 0, len(n.items))

	n.mu.RLock()
	items := n.items
	n.mu.RUnlock()
	for _, item := range items {
		n.sorted = append(n.sorted, item)
	}

	sort.Slice(n.sorted, func(i, j int) bool {
		return n.sorted[i].Position < n.sorted[j].Position
	})

	return n.sorted
}

// Build builds a slice of NavItems that should be displayed based on the web.IO as the current context.
// Build internally calls BuildNavigation on the Navigation items.
// Therefore, the NavItems will be sorted by their Position and then evaluated based on the BuildNavigation function. (see BuildNavigation for more details)
func (n *Navigation) Build(io IO) ([]NavItem, error) {
	return BuildNavigation(n.Items(), io)
}

// BuildNavigation builds a slice of NavItems that should be displayed based on the web.IO as the current context.
// It accepts exactly one parent as a parameter. The parent NavItem will be set to active if any of its children is active.
// BuildNavigation is used by Navigation.Build to build the navigation bar.
//
// BuildNavigation will NOT sort the NavItems by their Position. This has to be done before calling BuildNavigation.
// Automatically, the items are sorted when calling Navigation.Items or Navigation.Build as this calls Navigation.Items internally before calling BuildNavigation.
//
// BuildNavigation will call the Display function of each NavItem to determine if it should be displayed.
// Also, NavItems with a non-empty Items slice will be recursively evaluated and the current item will be set to active if any of its children is active.
// An Item will also be set to active if its URL matches the current URL.Path.
func BuildNavigation(navigation []NavItem, io IO, parent ...*NavItem) ([]NavItem, error) {
	// TODO show profile/login/logout links on the right (maybe this is a separate navigation bar?)

	parents := len(parent)
	if parents > 1 {
		panic("child can only have one parent Navigation item")
	}

	var singleParent *NavItem
	if parents == 1 {
		singleParent = parent[0]
	}

	var nav []NavItem

	for _, item := range navigation {
		display, err := item.Display(io)
		if err != nil {
			return nil, err
		}

		if !display {
			continue
		}

		if item.URL == io.Request().URL.Path {
			item.active = true

			if singleParent != nil {
				singleParent.active = true
			}
		}

		if len(item.Items) > 0 {
			subNavigation, err := BuildNavigation(item.Items, io, &item)
			if err != nil {
				return nil, err
			}

			item.Items = subNavigation
		}

		nav = append(nav, item)
	}

	return nav, nil
}
