// Copyright 2015 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package uid

import (
	"math/rand"
	"time"
)

// A UidRange structure used to set uidshift and its range.
type UidRange struct {
	UidShift uint64
	UidCount uint64
}

func NewUidRange(uidshift uint64, uidcount uint64) (*UidRange) {
	ur := &UidRange{
		uidshift,
		uidcount,
	}
	return ur
}

func GenerateUidRange(containerid uint64, uidcount uint64) (*UidRange) {
	var containerId uint64 = containerid
	rand.Seed(time.Now().UnixNano())
	containerUid := rand.Intn(0x0000FFFF)

	if containerId < 0x00010000 {
		rand.Seed(time.Now().UnixNano())
		containerId = uint64(rand.Int63n(0xFFFFFFFF) & 0xFFFF0000)
	} 

	containerId = containerId | uint64(containerUid)

	ur := NewUidRange(containerId, uidcount)
	return ur
}
