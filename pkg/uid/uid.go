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

// For how the uidshift and uidcount are generated please check:
// http://cgit.freedesktop.org/systemd/systemd/commit/?id=03cfe0d51499e86b1573d1

package uid

import (
	"fmt"
	"math/rand"
	"time"
)

// A UidRange structure used to set uidshift and its range.
type UidRange struct {
	UidShift uint32
	UidCount uint32
}

func NewUidRange(uidshift uint32, uidcount uint32) *UidRange {
	ur := &UidRange{
		uidshift,
		uidcount,
	}
	return ur
}

func GenerateUidShift() uint32 {
	rand.Seed(time.Now().UnixNano())
	// we force the MSB to 0 because devpts parses the uid,gid options as int
	// instead of as uint.
	// http://lxr.free-electrons.com/source/fs/devpts/inode.c?v=4.1#L189
	n := rand.Intn(0x7FFF) + 1
	uidShift := uint32(n << 16)

	return uidShift
}

func SetUidRange(uidrange *UidRange, uidcount uint32) {
	uidShift := GenerateUidShift()
	uidrange.UidShift = uidShift
	uidrange.UidCount = uidcount
}

func (uidrange UidRange) String() string {
	return fmt.Sprintf("%d:%d", uidrange.UidShift, uidrange.UidCount)
}

func UnserializeUidRange(uidrange string) (UidRange, error) {
	ur := UidRange{0, 0}

	n, err := fmt.Sscanf(uidrange, "%d:%d", &ur.UidShift, &ur.UidCount)
	if err != nil {
		return ur, err
	}

	return ur, nil
}
