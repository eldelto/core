# Diatom

A stack VM & Forth-like language implementation.

## Goals

The goal of this project is to create a minimal viable programming language to
learn more about the fundamentals of computing but also to see how featureful a
language really needs to be to create useful software.

The priorities are:

1. Simplicity - simple to use but also simple to implement
2. Portability - be the most portable virtual machine there is
3. Correctness - the obvious way to do something should lead towards
                 predictable results
4. Efficiency - be reasonable efficient in both time and space requirements but
                always provide an escape hatch to the host language

## Problems

The `exit` instruction is equal to 0 which means whenever rogue code jumps to a
wrong memory address the VM just exists gracefully, which is not exactly what
we want. Maybe we can introduce an additional instruction `abort` for value 0
and move `exit` to 1 so we can distinguish this.

## TODO

- [ ] Reorder instructions so `ret` & `call` have well-defined values
- [ ] Implement debugging tool
- [ ] Start the repl with `diatom repl`
- [ ] Read programs from stdin instead of files
- [ ] Bootstrap Forth interpreter
- [ ] Use it in a _real world_ project

