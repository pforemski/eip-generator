/*
 * eip-generator: a new IPv6 generator for Entropy/IP
 * a library of various functions
 *
 * Copyright (C) 2017-2018 Pawel Foremski <pjf@foremski.pl>
 * Licensed under GNU GPL v3
 */

package main

import "os"
import "fmt"
import "strings"
import "strconv"
import "encoding/json"

// ---------------- bit segments --------------------

// describes a bit segment
type Segment struct {
	name  string  // id
	start int     // first char
	stop  int     // last char
	vals  int     // number of values
}

// describes all bit segments in given input
type Segments []*Segment

// parse segment description from line of text
// XXX: we assume the segments are processed in order A-Z
func read_segment(segments Segments, line string) Segments {
	name := line[1:2]

	start, err := strconv.Atoi(strings.TrimSpace(line[4:6]))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing line '%s', segment #%s: %s\n", line, name, err)
		os.Exit(1)
	}

	stop, err := strconv.Atoi(strings.TrimSpace(line[7:9]))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing line '%s', segment #%s: %s\n", line, name, err)
		os.Exit(1)
	}

	return append(segments, &Segment{name, start, stop, 0})
}

// ---------------- segment values --------------------

// describes a segment value
type Segval struct {
	code    string
	source  string
	start   int64
	stop    int64
	freq    float64
}

// references values of all segments
type Segvals map[string][]*Segval

// converts segment id to segment name
var Segid2name = map[int]string{
	 0:"A",  1:"B",  2:"C",  3:"D",  4:"E",  5:"F",  6:"G",  7:"H",
	 8:"I",  9:"J", 10:"K", 11:"L", 12:"M", 13:"N", 14:"O", 15:"P",
	16:"Q", 17:"R", 18:"S", 19:"T", 20:"U", 21:"V", 22:"W", 23:"X",
	24:"Y", 25:"Z",
}

// parse segment value description from line of text
func read_segval(segvals Segvals, line string) {
	// parse
	code    := strings.TrimSpace(line[1:4])
	segname := code[0:1]
	source  := strings.TrimSpace(line[5:12])

	freq,_  := strconv.ParseFloat(strings.TrimSpace(line[13:19]), 64)
	freq    /= 100.0

	startstop := strings.SplitN(strings.TrimSpace(line[21:]), "-", 2)
	start,_   := strconv.ParseInt(startstop[0], 16, 64)
	stop      := start
	if len(startstop) > 1 {
		stop,_ = strconv.ParseInt(startstop[1], 16, 64)
	}

	// summarize
	segval := &Segval{
		code:   code,
		source: source,
		start:  start,
		stop:   stop,
		freq:   freq,
	}

	// first segval?
	slice, ok := segvals[segname]
	if !ok {
		slice = make([]*Segval, 0, 10)
	}

	// append
	segvals[segname] = append(slice, segval)
}

// ---------------- bayes network model --------------------

type BNModel struct {
	vertices      []*BNVertex
}

type BNVertex struct {
	vid         int
	name        string
	parents     []int
	values      []string
	cpds        map[string]map[string]float64

	segment     *Segment
	segvals     map[string]*Segval
	valcounts   map[string]float64
}

func parse_model(input string, segments Segments, segvals Segvals) *BNModel {
	// parse JSON
	parsed := map[string]map[string]interface{}{}
	err := json.Unmarshal([]byte(input), &parsed)
	if err != nil { panic(err) }

	// re-write
	model := &BNModel{}
	model.vertices = make([]*BNVertex, len(parsed))
	for vname,vdata := range parsed {
		vid := int(vname[0] - 'A')
		if vid < 0 || vid >= len(model.vertices) { panic(vname) }

		vertex := &BNVertex{}
		vertex.vid = vid
		vertex.name = vname
		vertex.segment = segments[vid]
		model.vertices[vid] = vertex

		// get parents
		vertex.parents = make([]int, 0)
		for _,p := range vdata["parents"].([]interface{}) {
			pid := int(p.(string)[0] - 'A')
			if pid < 0 { panic(pid) }
			vertex.parents = append(vertex.parents, pid)
		}

		// get values
		vertex.values = make([]string, 0)
		for _,v := range vdata["values"].([]interface{}) {
			vertex.values = append(vertex.values, v.(string))
		}

		// get counts of addresses behind segment values
		vertex.segvals = make(map[string]*Segval)
		vertex.valcounts = make(map[string]float64)
		for svid,sv := range segvals[vname] {
			key := strconv.FormatInt(int64(svid), 10)
			vertex.segvals[key] = sv
			vertex.valcounts[key] = float64(sv.stop - sv.start + 1)
		}

		// get cpds for various parent states
		vertex.cpds = make(map[string]map[string]float64)
		for property,data := range vdata {
			if property == "parents" || property == "values" { continue }

			// rewrite cpd for given parent state (property)
			vertex.cpds[property] = make(map[string]float64)
			sum := 0.0
			rem := float64(len(vertex.values))
			for val,prob := range data.(map[string]interface{}) {
				vertex.cpds[property][val] = prob.(float64)
				sum += prob.(float64)
				rem -= 1.0
			}

			// prob. for each of remaining values
			if rem != 0.0 {
				vertex.cpds[property]["*"] = (1.0 - sum) / rem
			}
		}
	}

	return model
}

func (model *BNModel) print() {
	for _,vertex := range model.vertices {
		fmt.Printf("%s:\n", vertex.name)
		fmt.Printf("  parents: %d\n", vertex.parents)
		fmt.Printf("  values: %s\n", vertex.values)
		for pval,cpd := range vertex.cpds {
			fmt.Printf("  %s: %v\n", pval, cpd)
		}
	}
}
