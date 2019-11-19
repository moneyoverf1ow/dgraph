/*
 * Copyright 2019 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package posting

import (
	"math"
	"reflect"
	"unsafe"

	"github.com/dgraph-io/dgo/v2/protos/api"
	"github.com/dgraph-io/dgraph/protos/pb"
)

const sizeOfBucket = 144

// DeepSize computes the memory taken by a Posting List
func (l *List) DeepSize() int {
	var size int
	if l == nil {
		return size
	}

	size = 4 * 8 // safe mutex consists of 4 words.

	size += 1 * 8 // plist pointer consists of 1 word.

	size += 1 * 8 // mutation map pointer  consists of 1 word.

	size += 2 * 8 // minTs and maxTs take 1 word each.

	size += 3 * 8 // array take 3 words. so key array is 3 words.

	size += 1 * 8 // So far 11 words, in order to round the slab we're adding one more word.
	// so far basic struct layout has been calculated.
	// Add each entry size of key array.
	size += int(cap(l.key))

	// add the posting list size.
	size += calculatePostingListSize(l.plist)
	if l.mutationMap != nil {
		// add the List.mutationMap size.
		// map has maptype and hmap
		// maptype is defined at compile time and is hardcoded in the compiled code.
		// Hence, it doesn't consume any extra memory.
		// Ref: https://bit.ly/2NQU8Jq
		// Now, let's look at hmap struct.
		// size of hmap struct
		// Ref: https://golang.org/src/runtime/map.go?#L114
		size += 6 * 8
		// we'll calculate the number of buckets based on pointer arithmetic in hmap struct.
		// reflect value give us access to the hmap struct.
		hmap := reflect.ValueOf(l.mutationMap)
		numBucket := int(math.Pow(2, float64((*(*uint8)(unsafe.Pointer(hmap.Pointer() + uintptr(9)))))))
		numOldBuckets := (*(*uint16)(unsafe.Pointer(hmap.Pointer() + uintptr(10))))
		size += int(numOldBuckets * sizeOfBucket)
		if len(l.mutationMap) > 0 || numBucket > 1 {
			size += int(numBucket * sizeOfBucket)
		}
	}
	// adding the size of all the entries in the map.
	for _, v := range l.mutationMap {
		size += calculatePostingListSize(v)
	}

	return size
}

// calculatePostingListSize is used to calculate the size of posting list
func calculatePostingListSize(list *pb.PostingList) int {
	var size int

	if list == nil {
		return size
	}

	size += 1 * 8 // Pack consists of 1 word.

	size += 3 * 8 // Postings array consists of 3 words.

	size += 1 * 8 // CommitTs consists of 1 word.

	size += 3 * 8 // Splits array consists of 3 words.

	size += 0 * 8 // XXX_NoUnkeyedLiteral consists of 0 words. because it is empty struct.

	size += 3 * 8 // XXX_unrecognized array consists of 3 words.

	size += 1 * 8 // XXX_sizecache consists of 1 word.

	// add pack size.
	size += calculatePackSize(list.Pack)

	// Each entry take one word.
	// Adding each entry reference allocation.
	size += int(cap(list.Postings)) * 8
	for _, p := range list.Postings {
		// add the size of each posting.
		size += calculatePostingSize(p)
	}

	// Each entry take one word.
	// Adding each entry size.
	size += int(cap(list.Splits)) * 8

	// XXX_unrecognized take one byte.
	// Adding size of each entry.
	size += int(cap(list.XXX_unrecognized))

	return size
}

// calculatePostingSize is used to calculate the size of a posting
func calculatePostingSize(posting *pb.Posting) int {
	var size int

	if posting == nil {
		return size
	}

	size += 1 * 8 // Uid consists of 1 word.

	size += 3 * 8 // Value byte array take 3 words.

	size += 1 * 8 // ValType consists 1 word.

	size += 1 * 8 // PostingType consists of 1 word.

	size += 3 * 8 // LangTag array consists of 3 words.

	size += 1 * 8 // Label consists of 1 word.

	size += 3 * 8 // Facets array consists of 3 word.

	size += 1 * 8 // Op consists of 1 word.

	size += 1 * 8 // StartTs consists of 1 word.

	size += 1 * 8 // CommitTs consists of 1 word.

	size += 0 * 8 // XXX_NoUnkeyedLiteral consists of 0 word. Because, it is empty struct.

	size += 3 * 8 // XXX_unrecognized array consists of 3 words.

	size += 1 * 8 // XXX_sizecache consists of 1 word.

	// Adding the size of each entry in Value array.
	size += int(cap(posting.Value))

	// Adding the size of each entry in LangTag array.
	size += int(cap(posting.LangTag))

	// Adding the size of each entry in Lables array.
	size += int(len(posting.Label))

	for _, f := range posting.Facets {
		// Add the size of each facet.
		size += calcuateFacet(f)
	}

	// Add the size of each entry in XXX_unrecognized array.
	size += int(cap(posting.XXX_unrecognized))

	return size
}

// calculatePackSize is used to calculate the size of a uidpack
func calculatePackSize(pack *pb.UidPack) int {
	var size int

	if pack == nil {
		return size
	}

	// BlockSize consists of 1 word.
	size += 1 * 8
	// Blocks array consists of 3 words.
	size += 3 * 8
	// XXX_NoUnkeyedLiteral consists of 0 word. Because, it
	// is empty struct.
	size += 0 * 8
	// XXX_unrecognized array consists of 3 words.
	size += 3 * 8
	// XXX_sizecache consists of 1 word.
	size += 1 * 8

	// Adding size of each entry in Blocks array.
	// Each entry consumes 1 word.
	size += int(cap(pack.Blocks)) * 8
	for _, block := range pack.Blocks {
		// Adding the size of UIDBlock.
		size += calculateUIDBlock(block)
	}
	// Adding the size each entry in XXX_unrecognized array.
	// Each entry consumes 1 word.
	size += int(cap(pack.XXX_unrecognized))

	return size
}

// calculateUIDBlock is used to calculate UidBlock
func calculateUIDBlock(block *pb.UidBlock) int {
	var size int

	if block == nil {
		return size
	}

	size += 1 * 8 // Base consists of 1 word.

	size += 3 * 8 // Delta array consists of 3 words.

	size += 1 * 8 // NumUids consists of 1 word.

	size += 0 * 8 // XXX_NoUnkeyedLiteral consists of 0 word. Because, It is empty struct.

	size += 3 * 8 // XXX_unrecognized array consists of 3 words.

	size += 1 * 8 // XXX_sizecache consistss of 1 word.

	size += 1 * 8 // Rounding it to 10 words by adding 1.

	// Adding the size of each entry in Deltas array.
	size += int(cap(block.Deltas))

	// Adding the size of each entry in XXX_unrecognized array.
	size += int(cap(block.XXX_unrecognized))

	return size
}

// calcuateFacet is used to calculate size of a facet.
func calcuateFacet(facet *api.Facet) int {
	var size int
	if facet == nil {
		return size
	}
	// Key consists of 1 word.
	size += 1 * 8
	// Value array consists of 3 words.
	size += 3 * 8
	// ValType consists of 1 word.
	size += 1 * 8
	// Tokens array consists of 3 words.
	size += 3 * 8
	// Alias consists of 1 word.
	size += 1 * 8
	// XXX_NoUnkeyedLiteral consists of 0 word. Because it is empty
	// struct.
	size += 0 * 8
	// XXX_unrecognized array consists of 3 word.
	size += 3 * 8
	// XXX_sizecache consists of 1 word.
	size += 1 * 8
	// size of struct 1 + 3 + 1 + 3 + 1 + 0 + 3 + 1 = 13
	// rounding to 16 so adding 3
	size += 3 * 8
	// Adding size of each entry in Key array.
	size += int(len(facet.Key))
	// Adding size of each entry in Value array.
	size += int(cap(facet.Value))

	for _, token := range facet.Tokens {
		// Adding size of each token.
		size += int(len(token))
	}
	// Adding size of each entry in Alias Array.
	size += int(len(facet.Alias))
	// Adding size of each entry in XXX_unrecognized array.
	size += int(len(facet.XXX_unrecognized))
	return size
}