package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRepositoryTokenReadsAllPagesWithoutWriting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer repository-secret" {
			t.Errorf("unexpected method or authentication")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			w.Write([]byte(`{"values":[{"id":12,"source":{"commit":{"hash":"abc"}}}]}`))
			return
		}
		w.Write([]byte(`{"values":[{"id":11}],"next":"http://` + r.Host + `/2.0/repositories/team/app/pullrequests?page=2"}`))
	}))
	defer server.Close()
	client := NewRepositoryClient("repository-secret")
	client.baseURL = server.URL + "/2.0"
	prs, err := client.ListAllOpenPullRequests("team", "app")
	if err != nil || len(prs) != 2 || prs[1].Source.Commit.Hash != "abc" {
		t.Fatalf("unexpected PR list: %+v, %v", prs, err)
	}
}

func TestRepositoryTokenDoesNotFollowPaginationOutsideAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"values":[],"next":"https://attacker.invalid/steal"}`))
	}))
	defer server.Close()
	client := NewRepositoryClient("repository-secret")
	client.baseURL = server.URL
	if _, err := client.ListAllOpenPullRequests("team", "app"); err == nil {
		t.Fatal("external pagination link must be refused")
	}
}

func TestReviewClientRefusesBitbucketWrites(t *testing.T) {
	client := NewRepositoryClient("repository-secret")
	if err := client.Post("/repositories/team/app/pullrequests/1/comments", map[string]string{"text": "comment"}, nil); err == nil {
		t.Fatal("review token client permitted an external write")
	}
}

func TestPinnedPullRequestResolvesAbbreviatedCommits(t *testing.T) {
	source := "a9272b2f945b" + strings.Repeat("a", 28)
	destination := "b693aa74f071" + strings.Repeat("b", 28)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer fixture" {
			t.Error("unexpected request")
		}
		switch r.URL.Path {
		case "/repositories/team/app/pullrequests/619":
			fmt.Fprint(w, `{"id":619,"source":{"repository":{"full_name":"team/fork"},"commit":{"hash":"a9272b2f945b"}},"destination":{"commit":{"hash":"b693aa74f071"}}}`)
		case "/repositories/team/fork/commit/a9272b2f945b":
			fmt.Fprintf(w, `{"hash":%q}`, source)
		case "/repositories/team/app/commit/b693aa74f071":
			fmt.Fprintf(w, `{"hash":%q}`, destination)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	pr, err := NewRepositoryClient("fixture").WithEndpoint(server.URL).GetPinnedPullRequest("team", "app", 619)
	if err != nil {
		t.Fatal(err)
	}
	if pr.Source.Commit.Hash != source || pr.Destination.Commit.Hash != destination {
		t.Fatalf("commits were not resolved: %+v", pr)
	}
}

func TestPinnedPullRequestRefusesMismatchedCommit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pullrequests/") {
			fmt.Fprint(w, `{"source":{"commit":{"hash":"aaaaaaaaaaaa"}},"destination":{"commit":{"hash":"bbbbbbbbbbbb"}}}`)
			return
		}
		fmt.Fprintf(w, `{"hash":%q}`, strings.Repeat("c", 40))
	}))
	defer server.Close()
	if _, err := NewRepositoryClient("fixture").WithEndpoint(server.URL).GetPinnedPullRequest("team", "app", 1); err == nil {
		t.Fatal("mismatched full commit accepted")
	}
}
