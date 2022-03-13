// Copyright (c) 2022 Hirotsuna Mizuno. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package cache

import "fmt"

// CreationError is the error type thrown when the object creation is failed.
type CreationError[K comparable] struct {
	Key K
	Err error
}

// Error implements the error interface.
func (e CreationError[_]) Error() string {
	return fmt.Sprintf("creation failure for %v: %s", e.Key, e.Err)
}

// Unwrap returns the underlying error of CreationError.
func (e CreationError[_]) Unwrap() error {
	return e.Err
}
