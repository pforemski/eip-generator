#!/usr/bin/env python
#
# eip-convert.py: convert entropy-ip model to eip-generator format
#
# Copyright (C) Pawel Foremski <pjf@foremski.pl>, 2018
# Licensed under GNU GPL v3
#

from __future__ import print_function
import json

def rewrite_segments(segpath):
	entropies = []
	segments = []
	for line in open(segpath):
		line = line.strip()
		if line[0].isdigit():
			d = line.split("\t")
			entropies.append(float(d[3]))
		elif line.startswith("# segment"):
			d = line.split("\t")
			start = int(d[2])
			stop = int(d[3])
			segments.append({
				"name": chr(ord("A")+len(segments)),
				"start_nybble": start/4,
				"stop_nybble": (stop-1)/4,
				"start_bit": start+1,
				"stop_bit": stop
			})

	for i in range(0, 4):
		print("/%-4d: %s" % (
			(i+1)*32, 
			" ".join(["%.5f"%x for x in entropies[i*8:(i+1)*8]])
		))

	for s in segments:
		print(">%s: %2d-%-2d (bits %3d-%-3d)" % (
			s["name"],
			s["start_nybble"], s["stop_nybble"],
			s["start_bit"], s["stop_bit"])
		)

def rewrite_analysis(anpath):
	segment = "?"
	segval = 0
	for line in open(anpath):
		if line[0].isalpha():
			segment = line[0]
			segval = 0
		else:
			d = line[2:-2].split(" ")
			val = d[0]
			pct = d[-1]
			print("=%s%-2d convert %6s%% %s" % (
				segment, segval,
				pct, val)
			)
			segval += 1

def rewrite_cpd(cpdpath):
	model = eval(open(cpdpath).read())

	vertices = []
	for vname, vdata in model.items():
		# get vertex CPDs
		cpds = []
		for priorv,cpd in vdata["cpds"].items():
			if priorv == None: continue
			if vdata["pars"]:
				key = ",".join([str(v) for v in priorv])
			else:
				key = ""

			vals = ", ".join(['"%d":%.4f'%(v,p) for v,p in cpd.items() if v != None])
			cpds.append('  "%s": { %s }' % (key, vals))

		# append vertex data
		vertex  = '"%s": {\n' % (vname)
		vertex += '  "parents": [ %s ],\n' % (", ".join(['"%s"'%(x) for x in vdata["pars"]]))
		vertex += '  "values": [ %s ],\n' % (", ".join(['"%d"'%(int(x)-1) for x in vdata["vals"]]))
		vertex += ",\n".join(cpds)
		vertex += "\n}"
		vertices.append(vertex)

	# print vertices with their CPDs
	print("{")
	print(",\n".join(vertices))
	print("}")

def main():
	import argparse

	# parse arguments
	p = argparse.ArgumentParser()
	p.add_argument('segments', help='output of entropy-ip/a1-segments.py')
	p.add_argument('analysis', help='output of entropy-ip/a2-mining.py')
	p.add_argument('cpd', help='output of entropy-ip/a5-bayes.py')
	args = p.parse_args()

	rewrite_segments(args.segments)
	rewrite_analysis(args.analysis)
	rewrite_cpd(args.cpd)

if __name__ == "__main__":
	main()
