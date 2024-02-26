# Mono-repo

... for personal projects.

The following projects are currently contained i this repo in a more or less
working state:

## Blog

My personal [blog](www.eldelto.net) that more or less is a frontend for an
Emacs `.org` file that I use to manage most of my work/projects.

### TODO

- [ ] Have one .org file that represents all pages on the blog not just
      /articles
  - [ ] Move data to separate .org file
  - [ ] Add `Hidden` field to `Article` and skip in listing
  - [ ] Append parent name to path when storing in bbolt
  - [ ] Add special handling for home page
- [ ] Performance improvements
  - [ ] Save articles as HTML + meta data instead of AST
  - [ ] Redirect to pages with `.html` to be elegible for bback/forward cache

## Diatom

My take on a Forth implementation that should be aimed at simplicity and
portability.

### TODO

- [ ] Reimplement assembler in Go
- [ ] Reimplement VM in Go
- [ ] Improve VM memory introspection

## Plant Guild

A website to make picking beneficial plant neighbours easier.

Currently covers most common vegetable and herb familiers but only in German.

### TODO

- [ ] Improve CSS to be somewhat respectable
- [ ] Translate to English

## Riff Robot

A [website](https://riffrobot.eldelto.net) hat returns you a random scale each day that I use to practice scales
and music theory.

### TODO

- [ ] Add more exotic scales (e.g. Japanese, medieval)
- [ ] Only show notes on the fretboard that match the scale
- [ ] Improve CSS

## Solvent

A CRDT-based to-do list that I use regularly that desperately needs a rewrite
to get rid of the React frontend as it is a maintainance nightmare.

### TODO

- [ ] Replace React frontend with something more maintainable (HTMX?)
- [ ] Implement proper authentication (probably OAuth2 social login)

## Voltbuddy

CLI tool to calculate resistor values for a Pi-pad attentuator.

### TODO

- [ ] Move to Cobra
- [ ] Add Farad to mAh conversion
- [ ] Add LED resister calculator

