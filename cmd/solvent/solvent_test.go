package main

import (
	"fmt"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type toDoListsDto struct {
	Identifier   string        `json:"identifier"`
	LiveSet      []interface{} `json:"liveSet"`
	TombstoneSet []interface{} `json:"tombstoneSet"`
}

type responseDto struct {
	ID        uuid.UUID    `json:"id"`
	ToDoLists toDoListsDto `json:"toDoLists"`
	CreatedAt int64        `json:"createdAt"`
}

func wireTestServer(t *testing.T) *TestServer {
	r := mux.NewRouter()
	mainController.RegisterRoutes(r)
	return NewTestServer(t, r)
}

func TestCreateNotebook(t *testing.T) {
	ts := wireTestServer(t)
	defer ts.Close()

	response := ts.POST("/api/notebook", "")
	AssertEquals(t, 200, response.StatusCode, "response.StatusCode")

	var responseBody responseDto
	err := response.Decode(&responseBody)
	AssertEquals(t, nil, err, "response.Decode error")

	AssertNotEquals(t, nil, responseBody.ID, "responseBody.ID")
	AssertEquals(t, []interface{}{}, responseBody.ToDoLists.LiveSet, "responseBody.ToDoLists.Liveset")
	AssertEquals(t, []interface{}{}, responseBody.ToDoLists.TombstoneSet, "responseBody.ToDoLists.TombstoneSet")
	AssertNotEquals(t, nil, responseBody.CreatedAt, "responseBody.CreatedAt")
	AssertNotEquals(t, 0, responseBody.CreatedAt, "responseBody.CreatedAt")
}

func TestFetchNotebook(t *testing.T) {
	ts := wireTestServer(t)
	defer ts.Close()

	response := ts.POST("/api/notebook", "")
	var responseBody responseDto
	err := response.Decode(&responseBody)
	AssertEquals(t, nil, err, "PUT response.Decode error")

	response = ts.GET("/api/notebook/" + responseBody.ID.String())
	AssertEquals(t, 200, response.StatusCode, "response.StatusCode")
	err = response.Decode(&responseBody)
	AssertEquals(t, nil, err, "GET response.Decode error")

	AssertNotEquals(t, nil, responseBody.ID, "responseBody.ID")
	AssertEquals(t, []interface{}{}, responseBody.ToDoLists.LiveSet, "responseBody.ToDoLists.Liveset")
	AssertEquals(t, []interface{}{}, responseBody.ToDoLists.TombstoneSet, "responseBody.ToDoLists.TombstoneSet")
	AssertNotEquals(t, nil, responseBody.CreatedAt, "responseBody.CreatedAt")
	AssertNotEquals(t, 0, responseBody.CreatedAt, "responseBody.CreatedAt")
}

func TestUpdateNotebook(t *testing.T) {
	ts := wireTestServer(t)
	defer ts.Close()

	response := ts.POST("/api/notebook", "")
	var postResponseBody responseDto
	err := response.Decode(&postResponseBody)
	AssertEquals(t, nil, err, "POST response.Decode error")

	requestBody := fmt.Sprintf(`{
		"id": "%s",
		"toDoLists": {
			"identifier": "ToDoListPSet",
			"liveSet": [
				{
					"id": "c8745289-c064-4759-815d-172261eaff8b",
					"title": {
						"value": "list0",
						"updatedAt": 1590829861033038682
					},
					"toDoItems": {
						"liveSet": [],
						"tombstoneSet": []
					},
					"createdAt": 1590829861033039336
				}
			],
			"tombstoneSet": []
		},
		"createdAt": %d 
	}`, postResponseBody.ID.String(), postResponseBody.CreatedAt)
	response = ts.PUT("/api/notebook", requestBody)
	AssertEquals(t, 200, response.StatusCode, "response.StatusCode")
	var putResponseBody responseDto
	err = response.Decode(&putResponseBody)
	AssertEquals(t, nil, err, "PUT response.Decode error")

	response = ts.GET("/api/notebook/" + postResponseBody.ID.String())
	var getResponseBody responseDto
	err = response.Decode(&getResponseBody)
	AssertEquals(t, nil, err, "GET response.Decode error")

	AssertEquals(t, putResponseBody, getResponseBody, "getResponseBody")
	AssertEquals(t, postResponseBody.ID, putResponseBody.ID, "putResponseBody.ID")
	AssertEquals(t, 1, len(putResponseBody.ToDoLists.LiveSet), "len(responseBody.ToDoLists.Liveset)")
	AssertEquals(t, []interface{}{}, putResponseBody.ToDoLists.TombstoneSet, "responseBody.ToDoLists.TombstoneSet")
}

func TestRemoveNotebook(t *testing.T) {
	ts := wireTestServer(t)
	defer ts.Close()

	response := ts.POST("/api/notebook", "")
	var responseBody responseDto
	err := response.Decode(&responseBody)
	AssertEquals(t, nil, err, "response.Decode error")

	response = ts.DELETE("/api/notebook/" + responseBody.ID.String())
	AssertEquals(t, 204, response.StatusCode, "DELETE response.StatusCode")

	response = ts.GET("/api/notebook/" + responseBody.ID.String())
	AssertEquals(t, 404, response.StatusCode, "GET response.StatusCode")
}
