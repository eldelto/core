# Flake-Finder

A little command line tool to discover flakiness tests.

## Motivation

## Installation

To use the CLI tool you need to have [Go](https://go.dev/doc/install) >= 1.23 [installed](https://go.dev/doc/install).

Afterwards you can fetch the latest version by running the following
command in your terminal:

`go install github.com/eldelto/core/cmd/flake-finder@latest`

Now you are all set and can start identifying those pesky flaky tests
🎉

## Usage

Using it only requires setting the desired number of iterations to run
and how many workers should run the command in parallel. For example
to find flaky Elixir tests you could run the following:

`flake-finder mix test -i 100 -p 5`

This will run the command `mix test` 100 times with five instances
running in parallel at most.

Faulty iterations are printed to standard out and the full output of
the run is stored in a file named
`flake-<command-name>-<iteration>.txt`.

Please also refer to the CLI's help pages (`flake-finder -h`) for more
parameters and sub-commands.
