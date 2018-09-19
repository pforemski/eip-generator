default: all
all: eip-generator

eip-generator: eip-generator.go lib.go
	go build -o $@ $^

clean:
	rm -f eip-generator
