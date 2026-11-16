// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"sort"

	"github.com/kastrul/gowsdl/pkg/parser"
)

// contentElements returns the elements of the lists, which together hold
// the element declarations of one content model, in schema document order,
// so that encoding/xml writes them in the order an xs:sequence requires.
// The parser flattens nested sequences and choices into separate lists;
// XSDElement.Position restores their order. Lists not built by the parser,
// whose elements have no position, keep their order. An element held by two
// lists is returned once.
func contentElements(lists ...[]*parser.XSDElement) []*parser.XSDElement {
	var out []*parser.XSDElement
	seen := make(map[*parser.XSDElement]bool)
	for _, list := range lists {
		for _, el := range list {
			if !seen[el] {
				seen[el] = true
				out = append(out, el)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Position < out[j].Position
	})
	return out
}
