
# Solvent

A minimalistic CRDT-based To-Do list.

- [Solvent](#solvent)
  - [Introduction](#introduction)
    - [Adding / Removing](#adding--removing)
    - [Checking / Unchecking](#checking--unchecking)
    - [Re-Ordering](#re-ordering)
  - [Getting Started](#getting-started)
  - [To-Do](#to-do)
  - [Screens](#screens)

## Introduction

The CRDTs (Conflictfree-Replicated-Data-Types) representing the to-do list have to
cover three basic operations:
  * Adding / removing items
  * Checking / unchecking items
  * Re-ordering items

### Adding / Removing

Can be represented with a 2P-Set consisting of two G-Sets (append only sets).
One tracking all the added items (called `liveSet`) and another one tracking 
all the removed items (called `tombstoneSet`).

An item is visible if it is contained in the `liveSet` set and not in the
`tombstoneSet`.

Renaming of items is treated as deleting the old item and creating a new one
with the changed name.

### Checking / Unchecking

The items themself hold the current checked state as simple boolean flag. Items
can only be checked but not unchecked on their own. Unchecking will be modeled
as an item deletion followed by creating a new and unchecked item with the same
title.

### Re-Ordering

Each item will be assigned an ordering value representing its order in the
to-do list. When an item gets moved the new order value will be the the average
of the two adjacent items. For the last position the order value will be the
order value of the second to last item plus 10.

## Getting Started

To run Solvent locally make sure you have Go, NPM and Docker-Compose installed
your system.

```shell
git clone https://github.com/eldelto/solvent.git

cd solvent

// Setup a local PostgreSQL DB
docker-compose up

// Launch backend server
go run web/main.go

// Launch frontend client
cd react-client
npm install
npm start

// Build Docker image
./docker_build.sh
```

## To-Do

- [x] Frontend rework
- [x] Mark lists as done when all items are checked
- [x] Properly handle errors in controllers
- [ ] Implement user handling
- [ ] Implement list removal
- [ ] Use Websockets instead of polling
- [ ] Fix potential DB race condition on update
- [ ] Register a service worker to cache data on reloads
- [ ] Send delta in update request / response instead of the everything
- [ ] Implement search functionality

## Screens

![List View](https://raw.githubusercontent.com/eldelto/solvent/master/docs/resources/list-view.png)

![Detail View](https://raw.githubusercontent.com/eldelto/solvent/master/docs/resources/detail-view.png)

