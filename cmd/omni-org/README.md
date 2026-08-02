# Worklog

A CLI tool to sync work time entries between different systems.

![Demo GIF](./media/demo.gif?raw=true "Demo GIF")

## Motivation

As many jobs require time bookings in different systems (more often
than not Jira) and I have never met anyone who cheerfully does so
(looking at you again Jira), I built this little CLI tool to time
entries from an easier to manage source
(e.g. [Org-mode](https://orgmode.org/)) and synchronizes them with
multiple other systems (called sinks).

Currently there are only a couple of supported sources and sinks but I
welcome everyone to open an issue for sources/sinks that would be
useful to them.

## Sources

Currently implemented sources include:

  - Org-mode file
  - CSV file (with columns: ticket, from, to)
  - Directory containing above mentioned file types
  - `'clockify'` fetches data from https://clockify.me (kindly
    implemented by [phkorn](https://github.com/phkorn))

## Sinks

This tool supports the two systems that I have to deal with in my day
job:

- Jira Tempo - if you provide a URL to a Jira instance
  (e.g. https://jira.acme.com)
- Personio - if you provide a URL to a Personio instance
  (e.g. https://acme.personio.de)

## Installation

To use the CLI tool you need to have [Go](https://go.dev/doc/install) >= 1.22.1 [installed](https://go.dev/doc/install).

Afterwards you can fetch the latest version by running the following
command in your terminal:

`go install github.com/eldelto/core/cmd/worklog@latest`

Now you are all set and can run your first worklog syncs 🎉

## Usage

If I want to sync my time entries of the last seven days from an
`.org` file to Jira Tempo, I can now do so by running

`worklog sync work-notes.org --sink 'https://jira.acme.com'`.

_This assumes that `https://jria.acme.com` is the Jira instance of my
company._

Please also refer to the CLI's help pages (`worklog -h`) for more
parameters and sub-commands.
