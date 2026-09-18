.PHONY: build test fuzz lint trace ci regen-fixtures dist demo

build:
	script/build

test:
	script/test

fuzz:
	script/fuzz

lint:
	script/lint

trace:
	script/trace

ci:
	script/ci

regen-fixtures:
	script/regen-fixtures

dist:
	script/dist $(VERSION)

demo:
	script/demo
