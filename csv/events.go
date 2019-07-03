// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package csv

// events.go processes the CSV generated by csv.go, and creates a map from metric to events.

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/chenjiacun35/battery-historian/checkinutil"
	"github.com/chenjiacun35/battery-historian/historianutils"
)

// sortByStartTime sorts events in ascending order of startTimeMs.
type sortByStartTime []Event

func (a sortByStartTime) Len() int      { return len(a) }
func (a sortByStartTime) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortByStartTime) Less(i, j int) bool {
	return a[i].Start < a[j].Start
}

// Event stores the details contained in a CSV line.
type Event struct {
	Type       string
	Start, End int64
	Value      string
	Opt        string
	AppName    string // For populating from package info.
}

// ExtractEvents returns all events matching any of the given metrics names.
// If a metric has no matching events, the map will contain a nil slice for that metric.
// If the metrics slice is nil, all events will be extracted.
// Errors encountered during parsing will be collected into an errors slice and will continue parsing remaining events.
func ExtractEvents(csvInput string, metrics []string) (map[string][]Event, []error) {
	records := checkinutil.ParseCSV(csvInput)
	if records == nil {
		return nil, []error{errors.New("nil result generated by ParseCSV")}
	}
	events := make(map[string][]Event, len(metrics))
	// Only store metrics requested.
	for _, m := range metrics {
		events[m] = nil
	}

	var errs []error
	for i, parts := range records {
		// Skip CSV header.
		if len(parts) == 0 || strings.Join(records[i], ",") == FileHeader {
			continue
		}
		desc := parts[0]
		metricEvents, ok := events[desc]
		if metrics != nil && !ok {
			// Ignore non matching metrics.
			continue
		}
		e, err := eventFromRecord(parts)
		if err != nil {
			errs = append(errs, fmt.Errorf("record %v: %v", i, err))
			continue
		}
		events[desc] = append(metricEvents, e)
	}
	return events, errs
}

// eventFromRecord parses the parts and either returns an event if in the correct format, else an error.
// Parts expected are desc,metricType,start,end,value,opt.
func eventFromRecord(parts []string) (Event, error) {
	if len(parts) != 6 {
		return Event{}, fmt.Errorf("non matching %v, len was %v", parts, len(parts))
	}
	start, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return Event{}, err
	}
	end, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return Event{}, err
	}
	return Event{
		Type:  parts[1],
		Start: start,
		End:   end,
		Value: parts[4],
		Opt:   parts[5],
	}, nil
}

// MergeEvents merges all overlapping events.
func MergeEvents(events []Event) []Event {
	if len(events) == 0 {
		return nil
	}
	// Need to sort the events by start time here,
	// because the following algorithm relies on sorted events.
	sort.Sort(sortByStartTime(events))

	var res []Event
	prev := events[0]
	for _, cur := range events[1:] {
		if prev.End < cur.Start {
			res = append(res, prev)
			prev = cur
		} else {
			prev = Event{Start: prev.Start, End: historianutils.MaxInt64(prev.End, cur.End)}
		}
	}
	res = append(res, prev)
	return res
}
