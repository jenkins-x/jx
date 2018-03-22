package jira

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestBoardService_GetAllBoards(t *testing.T) {
	setup()
	defer teardown()
	testAPIEdpoint := "/rest/agile/1.0/board"

	raw, err := ioutil.ReadFile("./mocks/all_boards.json")
	if err != nil {
		t.Error(err.Error())
	}
	testMux.HandleFunc(testAPIEdpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEdpoint)
		fmt.Fprint(w, string(raw))
	})

	projects, _, err := testClient.Board.GetAllBoards(nil)
	if projects == nil {
		t.Error("Expected boards list. Boards list is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

// Test with params
func TestBoardService_GetAllBoards_WithFilter(t *testing.T) {
	setup()
	defer teardown()
	testAPIEdpoint := "/rest/agile/1.0/board"

	raw, err := ioutil.ReadFile("./mocks/all_boards_filtered.json")
	if err != nil {
		t.Error(err.Error())
	}
	testMux.HandleFunc(testAPIEdpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEdpoint)
		fmt.Fprint(w, string(raw))
	})

	boardsListOptions := &BoardListOptions{
		BoardType:      "scrum",
		Name:           "Test",
		ProjectKeyOrID: "TE",
	}
	boardsListOptions.StartAt = 1
	boardsListOptions.MaxResults = 10

	projects, _, err := testClient.Board.GetAllBoards(boardsListOptions)
	if projects == nil {
		t.Error("Expected boards list. Boards list is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestBoardService_GetBoard(t *testing.T) {
	setup()
	defer teardown()
	testAPIEdpoint := "/rest/agile/1.0/board/1"

	testMux.HandleFunc(testAPIEdpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEdpoint)
		fmt.Fprint(w, `{"id":4,"self":"https://test.jira.org/rest/agile/1.0/board/1","name":"Test Weekly","type":"scrum"}`)
	})

	board, _, err := testClient.Board.GetBoard(1)
	if board == nil {
		t.Error("Expected board list. Board list is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestBoardService_GetBoard_WrongID(t *testing.T) {
	setup()
	defer teardown()
	testAPIEndpoint := "/rest/api/2/board/99999999"

	testMux.HandleFunc(testAPIEndpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEndpoint)
		fmt.Fprint(w, nil)
	})

	board, resp, err := testClient.Board.GetBoard(99999999)
	if board != nil {
		t.Errorf("Expected nil. Got %s", err)
	}

	if resp.Status == "404" {
		t.Errorf("Expected status 404. Got %s", resp.Status)
	}
	if err == nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestBoardService_CreateBoard(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/agile/1.0/board", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testRequestURL(t, r, "/rest/agile/1.0/board")

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":17,"self":"https://test.jira.org/rest/agile/1.0/board/17","name":"Test","type":"kanban"}`)
	})

	b := &Board{
		Name:     "Test",
		Type:     "kanban",
		FilterID: 17,
	}
	issue, _, err := testClient.Board.CreateBoard(b)
	if issue == nil {
		t.Error("Expected board. Board is nil")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestBoardService_DeleteBoard(t *testing.T) {
	setup()
	defer teardown()
	testMux.HandleFunc("/rest/agile/1.0/board/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		testRequestURL(t, r, "/rest/agile/1.0/board/1")

		w.WriteHeader(http.StatusNoContent)
		fmt.Fprint(w, `{}`)
	})

	_, resp, err := testClient.Board.DeleteBoard(1)
	if resp.StatusCode != 204 {
		t.Error("Expected board not deleted.")
	}
	if err != nil {
		t.Errorf("Error given: %s", err)
	}
}

func TestBoardService_GetAllSprints(t *testing.T) {
	setup()
	defer teardown()

	testAPIEndpoint := "/rest/agile/1.0/board/123/sprint"

	raw, err := ioutil.ReadFile("./mocks/sprints.json")
	if err != nil {
		t.Error(err.Error())
	}

	testMux.HandleFunc(testAPIEndpoint, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testRequestURL(t, r, testAPIEndpoint)
		fmt.Fprint(w, string(raw))
	})

	sprints, _, err := testClient.Board.GetAllSprints("123")

	if err != nil {
		t.Errorf("Got error: %v", err)
	}

	if sprints == nil {
		t.Error("Expected sprint list. Got nil.")
	}

	if len(sprints) != 4 {
		t.Errorf("Expected 4 transitions. Got %d", len(sprints))
	}
}
