# Mono-repo for Personal Projects

This repository contains most of the things I'm currently working on plus a
bunch of helper libraries.

It contains the following somewhat finished projects:

## Plant Guild

A small website to help you in choosing beneficial neighbours for your next
planting project.

Please see the project's ![README](/cmd/plantguild/README.md) for more information.

## Blog

### Atom Feed

- [x] Play around with Atom readers
  - [x] New York Times has an Atom [feed](https://rss.nytimes.com/services/xml/rss/nyt/Technology.xml)
- [x] Checkout the [specification](https://validator.w3.org/feed/docs/atom.html)
- [x] Implement basic `atom` package
  - [x] Data structure
  - [x] XML serialization
  - [x] Validate required fields
  - [x] Tests
  - [x] Check against [validtor](https://validator.w3.org/feed/check.cgi)
- [x] Implement `Service.AtomFeed()`
- [x] Implement `/atom/feed.xml` route with content-type `application/atom+xml`

