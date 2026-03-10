# Solvent

Efficient, self-hostable, no-fuzz to-do list.

A hosted instance is running at [solvent.eldelto.net](https://solvent.eldelto.net).

![Screenshot](../../internal/solvent/server/assets/screens/list.png?raw=true "Screenshot")

## Motivation

Back in 2020 I had troubles with some to-do list applications I used
because they wouldn't sync state between devices fast enough. The
first implementation was based on React and CRDT data-structures that
worked alright but still had the same issues for me as the other
applications I was familiar with.

I kept using it for the next 4 years and then finally decided to give
it some love, implement some new features, a proper login process and
radically simplify everything so it is easy to maintain and host.

As simple as it is, it is still, to this day, my most used hobby
project and I'm quite fond of it and hope that it might also be of
some use to fellow strangers on the internet.

## User Guide

A bit of info about different features in depth.

### General

The application tries to immediately sync every user action with the
backend server. This has the advantage that the user's data is always
up to date, no matter which device they choose. The obvious downside
is that a internet connection is required for the application to work.

Most actions require very little bandwith though and a somewhat flaky
internet connection is usually enough for the application to be
useable.

Theoretically a read-only mode could be implemented rather easily but
is currently not in place.

### Item Deletion

Single items can be deleted by long-pressing on the checkbox icon.

A caveat to this is that it also triggers the browser's long-press
action and it is not very discoverable. This will most likely be
changed in the future (for the better hopefully).

### Bulk Editing

The bulk edit mode can be reached via the *cog* icon -> *Bulk edit*.

In this mode the whole to-do list is treated as plain text. The first
line represents the headline and all subsequent lines are to-do
items. To-do items can optionally be prefixed with `- [ ]` or `- [x]`
to designate their intended state.

### Sharing

Lists can be shared with other people via clicking on the *cog* icon
-> *Share*.  A share link will be created that can be distributed to
persons that should be able to edit/view the list.

Everyone who has access to this link also has full edit permissions,
regardless if they have a Solvent account or not. If the other user
has a Solvent account though, the shared to-do list will be added to
their account and it will be displayed in their list view.

### Keyboard Shortcuts

Solvent supports a basic set of keyboard shortcuts to make editing
more convinient.

| Shortcut      | Context   | Description                   |
| ------------- | --------- | ----------------------------- |
| ctrl + a      | List view | Focuses the *add item* field. |
| ctrl + e      | List view | Switches into bulk edit mode. |
| ctrl + enter  | Bulk edit | Confirms the current changes. |
| shift + enter | Bulk edit | Confirms the current changes. |

## Self-Hosting

Solvent can easily be self-hosted as it is a single binary that can be
run on any architecture/OS combination that is supported by the Go
toolchain.

### Prerequisites

You need a installed Go toolchain with version 1.21 or higher. Please
refer to the official Go [installation guide](https://go.dev/doc/install).

### Building the Binary

With Go installed you can issue `go install
github.com/eldelto/core/cmd/solvent@latest` to install the binary in
your `$GOPATH/bin` directory (usually `~/go`).

There are currently no tagged releases for Solvent as it is too much
overhead for me. Generally you should be fine by using the latest
commit from `main` as this is also what I run at
`solvent.eldelto.net`.

### Environment Variables

Solvent can be configured via a couple of environment variables. It
will still start without issues if none of them are set but for
example E-mail sending will only work once you have configured the
respective env vars.

| Env var       | Description                                                           |
| ------------- | --------------------------------------------------------------------- |
| PORT          | The listening port for the HTTP server.                               |
| HOST          | The public-viewable domain name + protocol (e.g. https://to-do.list). |
| SMTP_USER     | The SMTP user to use for E-mailing.                                   |
| SMTP_PASSWORD | The SMTP password to use for E-mailing.                               |
| SMTP_HOST     | The SMTP server's host name.                                          |
| SMTP_PORT     | The SMTP server's port.                                               |
