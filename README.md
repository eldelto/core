# Mono-repo for Personal Projects

## Blog

### Atom Feed

- [x] Play around with Atom readers
  - [x] New York Times has an Atom [feed](https://rss.nytimes.com/services/xml/rss/nyt/Technology.xml)
- [x] Checkout the [specification](https://validator.w3.org/feed/docs/atom.html)
- [ ] Implement basic `atom` package
  - [x] Data structure
  - [x] XML serialization
  - [x] Validate required fields
  - [ ] Tests
  - [ ] Check against [validtor](https://validator.w3.org/feed/check.cgi)
- [ ] Implement `Service.AtomFeed()`
- [ ] Implement `/atom/feed.xml` route with content-type `application/atom+xml`

