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
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// A UidRange structure used to set uidshift and its range.
type UidRange struct {
	UidShift uint64
	UidCount uint64
}

func NewUidRange(uidshift uint64, uidcount uint64) *UidRange {
	ur := &UidRange{
		uidshift,
		uidcount,
	}
	return ur
}

func GenerateContainerID(containerid uint64) uint64 {
	var containerID uint64 = containerid

	if containerID < 0x00010000 {
		rand.Seed(time.Now().UnixNano())
		containerID = uint64(rand.Int63n(0xFFFDFFFF)) + 0X00010000
	}

	return containerID
}

func GenerateUidShift(containerid uint64) uint64 {
	var uidShift uint64
	rand.Seed(time.Now().UnixNano())
	containerUid := rand.Intn(0x0000FFFF)

	uidShift = GenerateContainerID(containerid) | uint64(containerUid)

	return uidShift
}

func GenerateUidRange(containerid uint64, uidcount uint64) *UidRange {
	uidShift := GenerateUidShift(containerid)
	ur := NewUidRange(uidShift, uidcount)
	return ur
}

func SetUidRange(uidrange *UidRange, containerid uint64, uidcount uint64) {
	uidShift := GenerateUidShift(containerid)
	uidrange.UidShift = uidShift
	uidrange.UidCount = uidcount
}

func ParseSerializedUidRange(uidrange string) (uidShift uint64, uidCount uint64, err error) {
	if len(uidrange) == 0 {
		return 0, 0, nil
	}

	uidrangeArr := strings.Split(uidrange, ":")
	if len(uidrangeArr) < 2 {
		return 0, 0, errors.New("uidrange format error")
	}

	uidShift, err = strconv.ParseUint(uidrangeArr[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	uidCount, err = strconv.ParseUint(uidrangeArr[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return
}

func SerializeUidRange(uidrange UidRange) string {
	var uidrangeStr string = ""

	uidshift := strconv.FormatUint(uidrange.UidShift, 10)
	uidcount := strconv.FormatUint(uidrange.UidCount, 10)

	uidrangeStr = uidshift + ":" + uidcount

	return uidrangeStr
}

func UnserializeUidRange(uidrange string) (UidRange, error) {
	ur := UidRange{0, 0}

	uidshift, uidcount, err := ParseSerializedUidRange(uidrange)
	if err != nil {
		return ur, err
	}

	ur.UidShift = uidshift
	ur.UidCount = uidcount

	return ur, nil
}
