# eip-generator: a new generator for Entropy/IP

This repository hosts a new IPv6 address generator for [Entropy/IP](http://entropy-ip.com/), which lets for probabilistic IPv6 Internet scanning.

Note that Entropy/IP is patent pending, submitted as [application number 15618303](https://patents.google.com/patent/US20170359227A1/en) by Akamai Technologies, Inc. in 2016. However, an open source Entropy/IP implementation licensed for *non-commercial, academic research purposes* is available at https://github.com/akamai/entropy-ip.

This project is independent and separate from the above, in the sense that it uses the Entropy/IP output as its own input to generate a possibly large set of IPv6 addresses matching the model. Thus, it is published under GNU GPL v3 with the intent of further popularization of Entropy/IP for IPv6 academic research.

This generator was used in the following ACM IMC2018 paper:

> [Clusters in the Expanse: Understanding and Unbiasing IPv6 Hitlists](https://ipv6hitlist.github.io/), *Oliver Gasser, Quirin Scheitle, Paweł Foremski, Qasim Lone, Maciej Korczyński, Stephen D. Strowes, Luuk Hendriks, Georg Carle*, ACM Internet Measurement Conference 2018, Boston, MA, USA

See [ipv6hitlist.github.io](https://ipv6hitlist.github.io/) for more details.

# Prerequisites

1. Install [Entropy/IP](http://entropy-ip.com/).
1. Install [Go](https://www.golang.org/), eg.:
```
$ sudo apt-get install golang-go
```
1. Build `eip-generator` by running make in this repo:
```
$ make
go build -o eip-generator eip-generator.go lib.go
```

# Basic usage

1. Build your model using Entropy/IP, store all resultant files.
2. Convert the resultant files into a file format understood by this code, by running `eip-convert.py`:
```
$ ./eip-convert.py -h
usage: eip-convert.py [-h] segments analysis cpd

positional arguments:
  segments    output of entropy-ip/a1-segments.py
  analysis    output of entropy-ip/a2-mining.py
  cpd         output of entropy-ip/a5-bayes.py

optional arguments:
  -h, --help  show this help message and exit
```
3. For instance, if your model is in `isp1` subdir, you might need to run:
```
$ ./eip-convert.py isp1/segments isp1/analysis isp1/cpd > isp1/eip.model
```
4. Finally, you are ready to start `eip-generator`:
```
$ ./eip-generator -h
Usage of ./eip-generator:
  -M int
    	max. number of addresses per model state (default 1000)
  -N int
    	approx. number of addresses to generate (default 1000000)
  -P int
    	max. depth in model to run in parallel (default 4)
  -S float
    	minimum state probability, 0 = auto
  -V	verbose
  -p	pass stdin to stdout
```
5. Assuming the converted model is in `isp1/eip.model`, you might want to run:
```
$ cat isp1/eip.model | ./eip-generator -N 10
20010db803022100000000000000023c
20010db80002256b0000000000000001
20010db804013000000000000000b1f8
20010db800028bcc0000000000000001
20010db80003fd610000000000000001
20010db80009f1290000000000000001
20010db8000863ca0000000000000001
20010db800096cc10000000000000003
```
6. Note that the output is in full hex format, i.e. we skip colons but don't skip any zeros. Depending on your next step, you probably need to convert this format back to ordinary notation. You might want to use the `ipv6-hex2addr` program published in [github.com/pforemski/entropy-clustering](https://github.com/pforemski/entropy-clustering).

## Differences vs. the Entropy/IP generator

The original goal for Entropy/IP was uncovering structures in IPv6 addressing schemes, whereas the new paper goal was scanning the IPv6 Internet. Thus, the new target generator exhaustively walks the model and prints the most probable IPv6 addresses under a scanning budget of e.g. <1M targets. The budget can be set using the `-N` option of the `eip-generator` program, which sets the *maximum* budget, but the actual number of generated targets can be much smaller, even of the order of magnitude (which depends on network).

The generator still can produce some amount of duplicates, but less frequently. It does generate addresses that were used for training. Thus, it is desirable to run an address de-duplication / filtering step before using the generator output for scanning.

Also, a mere fact that an address belongs to the "most probable" Entropy/IP target for given budget does not imply that it is easy to hit in practice. For instance, a network can have 90% of hosts within a single Entropy/IP representation (aka "encoded address"), but with billions of possible realizations (in other words, e.g. a /96 prefix covers 90% of hosts, but the last 32 bits are pseudo-random and temporal). To some degree, such difficult targets can be skipped by using small values for the `-M` option that controls the maximum number of IPv6 addresses per each "encoded" representation. By default, `-M` is set to 1000, which means the generator will touch "pseudo-random" areas of a given network quite briefly (at most 1000 tries).

# Author
Written by Paweł Foremski, [@pforemski](https://twitter.com/pforemski), 2017-2018.
