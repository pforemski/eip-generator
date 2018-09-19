/*
 * eip-generator: a new IPv6 generator for Entropy/IP
 *
 * Copyright (C) 2017-2018 Pawel Foremski <pjf@foremski.pl>
 * Licensed under GNU GPL v3
 */

package main

import "os"
import "fmt"
import "flag"
import "strings"
import "bufio"
import "sync"
import "math/rand"
import "strconv"

// command line arguments
var (
	opt_N = flag.Int("N", 1000000, "approx. number of addresses to generate")
	opt_N_prob float64 // = 1.0 / opt_N
	opt_S = flag.Float64("S", 0.0, "minimum state probability, 0 = auto")
	opt_M = flag.Int("M", 1000, "max. number of addresses per model state")

	opt_P = flag.Int("P", 4, "max. depth in model to run in parallel")
	opt_V = flag.Bool("V", false, "verbose")
	opt_p = flag.Bool("p", false, "pass stdin to stdout")
)

var zeros string = "00000000000000000000000000000000"

// gaddr represents a model state with it's probability and count of addresses within
type gaddr struct {
	state   []string
	prob    float64
	count   float64
	chance  float64
}

// dive recursively dives into given vertex
func dive(model *BNModel, vid int, state []string, prob float64, count float64, out chan *gaddr, wg *sync.WaitGroup) {
	if wg != nil { defer wg.Done() }

	// done?
	if vid >= len(model.vertices) {
		sp := &gaddr{}
		sp.prob = prob
		sp.count = count
		sp.chance = prob / count
		sp.state = make([]string, len(state))
		copy(sp.state, state)
		out <- sp
		return
	}

	// prepare
	if state == nil { state = make([]string, len(model.vertices)) }
	vertex := model.vertices[vid]

	// find cpd
	var cpd map[string]float64
	var ok bool
	if len(vertex.parents) > 0 { // depends on CPD
		pstate := make([]string, len(vertex.parents))
		for i,pid := range vertex.parents { pstate[i] = state[pid] }
		cpd, ok = vertex.cpds[strings.Join(pstate, ",")]
	} else {
		cpd, ok = vertex.cpds[""]
	}

	// TODO? on another hand, maybe it's safe to ignore such a rare state
	if !ok { return }

	// dive into each value
	for _,value := range vertex.values {
		// get probability
		vprob,ok := cpd[value]
		if !ok { vprob = cpd["*"] }
		vprob = vprob * prob

		// big enough?
		if vprob < *opt_S { continue }

		// get count
		vcount := vertex.valcounts[value]
		vcount = vcount * count

		// run in parallel?
		if vid < *opt_P {
			// copy state
			state2 := make([]string, len(state))
			copy(state2, state)
			state2[vid] = value

			// run in parallel
			wg.Add(1)
			go dive(model, vid+1, state2, vprob, vcount, out, wg)

		} else { // dive in & wait
			state[vid] = value
			dive(model, vid+1, state, vprob, vcount, out, nil)
		}
	}
}

// rewrite gaddr as an IPv6 address
func rewrite(model *BNModel, gaddr *gaddr, budget float64) {
	if budget < 1 {
		budget = 1
	} else if budget > float64(*opt_M) {
		budget = float64(*opt_M)
	}

	// TODO: if budget is close to gaddr.count,
	// then generate in systematic way instead of random
	if budget > gaddr.count {
		budget = gaddr.count
	}

	addr := make([]byte, 0, 32)
	for budget >= 1.0 {
		addr = addr[:0]

		for vid,val := range gaddr.state {
			segment := model.vertices[vid].segment
			sv := model.vertices[vid].segvals[val]

			slen := segment.stop - segment.start + 1
			valc := sv.stop - sv.start + 1

			// decode segment value
			val := sv.start
			if valc > 0 { val += rand.Int63n(valc) }
			vals := strconv.FormatInt(val, 16)

			// prepend zeros?
			diff := slen - len(vals)
			if diff > 0 { addr = append(addr, zeros[0:diff]...) }

			// add it
			addr = append(addr, vals...)
		}

		fmt.Printf("%s\n", addr)
		budget -= 1.0
	}
}

func main() {
	// parse args
	flag.Parse()

	// compute probabilities
	opt_N_prob = 1.0 / float64(*opt_N)
	if *opt_S == 0 {
		if *opt_N < 1000 {
			*opt_S = 1.0 / 1e6
		} else {
			*opt_S = opt_N_prob / 1e3
		}
	}

	// prepare storage for input data
	model_lines := make([]string, 0, 1000)
	segments := make(Segments, 0, 20)
	segvals := make(Segvals)

	// read input
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		switch line[0] {
		case '/': break
		case '>': segments = read_segment(segments, line)
		case '=': read_segval(segvals, line)
		default:
			model_lines = append(model_lines, line)
			continue
		}
		if *opt_p { fmt.Println(line) }
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
		os.Exit(1)
	}

	// parse model
	model := parse_model(strings.Join(model_lines, "\n"), segments, segvals)
	if *opt_V { model.print() }

	// dive into model
	out := make(chan *gaddr, 1000000)
	go func() {
		wg := sync.WaitGroup{}
		wg.Add(1)
		dive(model, 0, nil, 1.0, 1.0, out, &wg)
		wg.Wait()
		close(out)
	}()

	// start output reader
	n := 0
	n2 := 0
	prob := 0.0
	psum := 0.0
	for sp := range out {
		n++
		prob += sp.prob
		psum += sp.prob

		if prob >= opt_N_prob {
			n2++

			// rewrite sp.prob / opt_N_prob times (at least 1, max opt_M)
			rewrite(model, sp, sp.prob / opt_N_prob)

			prob = 0.0
		}
	}

	if *opt_V {
		fmt.Printf("generated %d combinations, printed %d, Psum=%f\n", n, n2, psum)
	}
}
