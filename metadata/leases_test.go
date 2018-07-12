/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package metadata

import (
	"testing"

	"github.com/boltdb/bolt"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/leases"
	"github.com/pkg/errors"
)

func TestLeases(t *testing.T) {
	ctx, db, cancel := testEnv(t)
	defer cancel()

	testCases := []struct {
		ID    string
		Cause error
	}{
		{
			ID: "tx1",
		},
		{
			ID:    "tx1",
			Cause: errdefs.ErrAlreadyExists,
		},
		{
			ID: "tx2",
		},
	}

	var ll []leases.Lease

	for _, tc := range testCases {
		if err := db.Update(func(tx *bolt.Tx) error {
			lease, err := NewLeaseManager(tx).Create(ctx, leases.WithID(tc.ID))
			if err != nil {
				if tc.Cause != nil && errors.Cause(err) == tc.Cause {
					return nil
				}
				return err
			}
			ll = append(ll, lease)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}

	var listed []leases.Lease
	// List leases, check same
	if err := db.View(func(tx *bolt.Tx) error {
		var err error
		listed, err = NewLeaseManager(tx).List(ctx)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if len(listed) != len(ll) {
		t.Fatalf("Expected %d lease, got %d", len(ll), len(listed))
	}
	for i := range listed {
		if listed[i].ID != ll[i].ID {
			t.Fatalf("Expected lease ID %s, got %s", ll[i].ID, listed[i].ID)
		}
		if listed[i].CreatedAt != ll[i].CreatedAt {
			t.Fatalf("Expected lease created at time %s, got %s", ll[i].CreatedAt, listed[i].CreatedAt)
		}
	}

	for _, tc := range testCases {
		if err := db.Update(func(tx *bolt.Tx) error {
			return NewLeaseManager(tx).Delete(ctx, leases.Lease{
				ID: tc.ID,
			})
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := db.View(func(tx *bolt.Tx) error {
		var err error
		listed, err = NewLeaseManager(tx).List(ctx)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	if len(listed) > 0 {
		t.Fatalf("Expected no leases, found %d: %v", len(listed), listed)
	}
}
