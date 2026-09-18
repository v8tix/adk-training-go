package main

import (
	"context"
	"reflect"
	"sync"
	"testing"
)

// TestCart_AddItem_AccumulatesInOrder proves items accumulate in the order
// added, across multiple calls — the direct proof of statefulness this
// module's whole lesson is about.
func TestCart_AddItem_AccumulatesInOrder(t *testing.T) {
	c := &cart{}

	tests := []struct {
		item     string
		wantCart []string
	}{
		{item: "milk", wantCart: []string{"milk"}},
		{item: "eggs", wantCart: []string{"milk", "eggs"}},
		{item: "bread", wantCart: []string{"milk", "eggs", "bread"}},
	}

	for _, tt := range tests {
		_, got, err := c.addItem(t.Context(), nil, AddItemArgs{Item: tt.item})
		if err != nil {
			t.Fatalf("addItem(%q) error = %v", tt.item, err)
		}
		if got.Status != "success" {
			t.Errorf("addItem(%q) Status = %q, want %q", tt.item, got.Status, "success")
		}
		if !reflect.DeepEqual(got.Cart, tt.wantCart) {
			t.Errorf("addItem(%q) Cart = %v, want %v", tt.item, got.Cart, tt.wantCart)
		}
	}
}

// TestCart_View_EmptyByDefault proves a fresh cart's view returns an empty
// (not nil-panicking, not pre-populated) list.
func TestCart_View_EmptyByDefault(t *testing.T) {
	c := &cart{}

	_, got, err := c.view(t.Context(), nil, ViewCartArgs{})
	if err != nil {
		t.Fatalf("view() error = %v", err)
	}
	if len(got.Items) != 0 {
		t.Errorf("view() on a fresh cart = %v, want empty", got.Items)
	}
}

// TestCart_View_ReflectsAddedItems proves view() reflects items added
// beforehand, in order.
func TestCart_View_ReflectsAddedItems(t *testing.T) {
	c := &cart{}
	for _, item := range []string{"milk", "eggs"} {
		if _, _, err := c.addItem(t.Context(), nil, AddItemArgs{Item: item}); err != nil {
			t.Fatalf("addItem(%q) error = %v", item, err)
		}
	}

	_, got, err := c.view(t.Context(), nil, ViewCartArgs{})
	if err != nil {
		t.Fatalf("view() error = %v", err)
	}
	want := []string{"milk", "eggs"}
	if !reflect.DeepEqual(got.Items, want) {
		t.Errorf("view() = %v, want %v", got.Items, want)
	}
}

// TestCart_AddItem_ConcurrentSafe proves the mutex genuinely protects
// concurrent addItem calls — run with -race, this is the test that would
// actually catch a missing lock.
func TestCart_AddItem_ConcurrentSafe(t *testing.T) {
	c := &cart{}
	const n = 50

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Go(func() {
			if _, _, err := c.addItem(context.Background(), nil, AddItemArgs{Item: "item"}); err != nil {
				t.Errorf("addItem() error = %v", err)
			}
		})
	}
	wg.Wait()

	_, got, err := c.view(t.Context(), nil, ViewCartArgs{})
	if err != nil {
		t.Fatalf("view() error = %v", err)
	}
	if len(got.Items) != n {
		t.Errorf("len(view().Items) = %d, want %d — a race would drop items", len(got.Items), n)
	}
}
